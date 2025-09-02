// Package server implements the HTTP server for metrics handling.
package server

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/and161185/metrics-alerting/cmd/server/metrics"
	"github.com/and161185/metrics-alerting/internal/config"
	"github.com/and161185/metrics-alerting/internal/errs"
	"github.com/and161185/metrics-alerting/internal/server/middleware"
	"github.com/and161185/metrics-alerting/internal/utils"
	"github.com/and161185/metrics-alerting/model"
	chiMiddleware "github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
)

// Storage provides metric storage operations.
type Storage interface {
	// Save stores a single metric.
	Save(ctx context.Context, metric *model.Metric) error
	// SaveBatch stores a batch of metrics.
	SaveBatch(ctx context.Context, metrics []model.Metric) error
	// Get retrieves a metric by ID and type.
	Get(ctx context.Context, metric *model.Metric) (*model.Metric, error)
	// GetAll returns all stored metrics.
	GetAll(ctx context.Context) (map[string]*model.Metric, error)
	// Ping checks the availability of the storage.
	Ping(ctx context.Context) error
}

type fileBackedStore interface {
	SaveToFile(ctx context.Context, path string) error
	LoadFromFile(ctx context.Context, path string) error
}

// Server serves the metrics HTTP API.
type Server struct {
	Storage    Storage
	Config     *config.ServerConfig
	FileStore  fileBackedStore
	PrivateKey *rsa.PrivateKey
}

// NewServer creates a new server instance with the given storage and configuration.
func NewServer(storage Storage, config *config.ServerConfig, priv *rsa.PrivateKey) *Server {
	fileStore, _ := storage.(fileBackedStore)

	return &Server{
		Storage:    storage,
		Config:     config,
		FileStore:  fileStore,
		PrivateKey: priv,
	}
}

func (srv *Server) buildRouter() http.Handler {
	router := chi.NewRouter()
	router.Use(chiMiddleware.StripSlashes)
	router.Use(middleware.DecryptMiddleware(srv.PrivateKey, true))
	router.Use(middleware.DecompressMiddleware)
	router.Use(middleware.VerifyHashMiddleware(srv.Config))
	router.Use(middleware.LogMiddleware(srv.Config.Logger))
	router.Use(middleware.CompressMiddleware)
	router.Post("/update/{type}/{name}/{value}", srv.UpdateMetricHandler)
	router.Post("/update", srv.UpdateMetricHandlerJSON)
	router.Post("/updates", srv.UpdateArrayMetricHandlerJSON)
	router.Get("/value/{type}/{name}", srv.GetMetricHandler)
	router.Post("/value", srv.GetMetricHandlerJSON)
	router.Get("/", srv.ListMetricsHandler)
	router.Get("/ping", srv.PingHandler)
	return router
}

// Run starts the HTTP server and, if configured, periodically saves metrics to a file.
func (srv *Server) Run(ctx context.Context) error {
	router := srv.buildRouter()

	server := &http.Server{
		Addr:    srv.Config.Addr,
		Handler: router,
	}

	if srv.Config.Restore && srv.FileStore != nil {
		if err := srv.FileStore.LoadFromFile(ctx, srv.Config.FileStoragePath); err != nil {
			srv.Config.Logger.Warnf("failed to restore metrics from file: %v", err)
		}
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			srv.Config.Logger.Fatalf("server error: %v", err)
		}
	}()

	if srv.Config.StoreInterval > 0 && srv.FileStore != nil {
		ticker := time.NewTicker(time.Duration(srv.Config.StoreInterval) * time.Second)
		go func() {
			for range ticker.C {
				if err := srv.FileStore.SaveToFile(ctx, srv.Config.FileStoragePath); err != nil {
					srv.Config.Logger.Errorf("auto-save failed: %v", err)
				}
			}
		}()
	}

	<-ctx.Done()

	if srv.Config.FileStoragePath != "" && srv.FileStore != nil {
		if err := srv.FileStore.SaveToFile(ctx, srv.Config.FileStoragePath); err != nil {
			log.Printf("save failed: %v", err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return server.Shutdown(shutdownCtx)
}

// UpdateMetricHandler handles updating a metric via URL parameters.
func (srv *Server) UpdateMetricHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	typ := chi.URLParam(r, "type")
	name := chi.URLParam(r, "name")
	val := chi.URLParam(r, "value")

	metric, err := metrics.NewMetric(typ, name, val)
	if err != nil {
		log.Printf("failed to create metric [type=%s, name=%s]: %v", typ, name, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = utils.WithRetry(ctx, func() error {
		return srv.saveToStorage(ctx, metric)
	})

	if err != nil {
		log.Printf("failed to save metric [name=%s]: %v", metric.ID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// UpdateMetricHandlerJSON handles updating a single metric via a JSON payload.
func (srv *Server) UpdateMetricHandlerJSON(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "unsupported content type", http.StatusUnsupportedMediaType)
		return
	}

	var metric model.Metric
	err := json.NewDecoder(r.Body).Decode(&metric)
	if err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	err = metrics.CheckMetric(&metric)
	if err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	err = utils.WithRetry(ctx, func() error {
		return srv.saveToStorage(ctx, &metric)
	})

	if err != nil {
		log.Printf("failed to save metric [name=%s]: %v", metric.ID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(metric); err != nil {
		log.Printf("failed to write response JSON: %v", err)
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// UpdateArrayMetricHandlerJSON handles updating multiple metrics via a JSON array.
func (srv *Server) UpdateArrayMetricHandlerJSON(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "unsupported content type", http.StatusUnsupportedMediaType)
		return
	}

	var metricsArray []model.Metric

	err := json.NewDecoder(r.Body).Decode(&metricsArray)
	if err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if len(metricsArray) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	for _, metric := range metricsArray {
		err = metrics.CheckMetric(&metric)
		if err != nil {
			msg := fmt.Sprintf("invalid JSON: %v", err)
			http.Error(w, msg, http.StatusUnprocessableEntity)
			return
		}
	}

	err = utils.WithRetry(ctx, func() error {
		return srv.saveBatchToStorage(ctx, metricsArray)
	})

	if err != nil {
		log.Printf("failed to save metrics: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (srv *Server) saveToStorage(ctx context.Context, metric *model.Metric) error {

	err := srv.Storage.Save(ctx, metric)
	if err != nil {
		return err
	}

	if srv.Config.StoreInterval == 0 && srv.FileStore != nil {
		if err := srv.FileStore.SaveToFile(ctx, srv.Config.FileStoragePath); err != nil {
			srv.Config.Logger.Errorf("failed to save file %s: %v", srv.Config.FileStoragePath, err)
		}
	}

	return nil
}

func (srv *Server) saveBatchToStorage(ctx context.Context, metricsArray []model.Metric) error {
	err := srv.Storage.SaveBatch(ctx, metricsArray)
	if err != nil {
		return err
	}

	if srv.Config.StoreInterval == 0 && srv.FileStore != nil {
		if err := srv.FileStore.SaveToFile(ctx, srv.Config.FileStoragePath); err != nil {
			srv.Config.Logger.Errorf("failed to save file %s: %v", srv.Config.FileStoragePath, err)
		}
	}

	return nil
}

// GetMetricHandler returns the value of a metric as a plain string (gauge/counter).
func (srv *Server) GetMetricHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	typ := chi.URLParam(r, "type")
	name := chi.URLParam(r, "name")

	metric, err := metrics.NewEmptyMetric(typ, name)
	if err != nil {
		log.Printf("failed to create metric [type=%s, name=%s]: %v", typ, name, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var storedMetric *model.Metric
	err = utils.WithRetry(ctx, func() error {
		var getErr error
		storedMetric, getErr = srv.Storage.Get(ctx, metric)
		return getErr
	})

	if err != nil {
		if errors.Is(err, errs.ErrMetricNotFound) {
			log.Printf("metric not found [type=%s, name=%s]: %v", typ, name, err)
			http.NotFound(w, r)
			return
		}
		log.Printf("failed to get metric from storage [name=%s]: %v", name, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	switch typ {
	case string(model.Gauge):
		if storedMetric.Value == nil {
			http.NotFound(w, r)
			return
		}
		_, err = fmt.Fprintf(w, "%v", *storedMetric.Value)

	case string(model.Counter):
		if storedMetric.Delta == nil {
			http.NotFound(w, r)
			return
		}
		_, err = fmt.Fprintf(w, "%v", *storedMetric.Delta)

	default:
		http.Error(w, "unsupported metric type", http.StatusBadRequest)
		return
	}

	if err != nil {
		log.Printf("failed to write response body for metric [name=%s]: %v", name, err)
	}
}

// GetMetricHandlerJSON returns the value of a metric in JSON format.
func (srv *Server) GetMetricHandlerJSON(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "unsupported content type", http.StatusUnsupportedMediaType)
		return
	}

	var reqMetric model.Metric
	if err := json.NewDecoder(r.Body).Decode(&reqMetric); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	var storedMetric *model.Metric
	err := utils.WithRetry(ctx, func() error {
		var err error
		storedMetric, err = srv.Storage.Get(ctx, &reqMetric)
		return err
	})

	if err != nil {
		if errors.Is(err, errs.ErrMetricNotFound) {
			http.NotFound(w, r)
		} else {
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(storedMetric); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// ListMetricsHandler returns a list of all stored metrics in HTML format.
func (srv *Server) ListMetricsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var all map[string]*model.Metric
	err := utils.WithRetry(ctx, func() error {
		var err error
		all, err = srv.Storage.GetAll(ctx)
		return err
	})

	if err != nil {
		log.Printf("failed to get all metrics from storage: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, err = fmt.Fprintln(w, "<html><body><ul>")
	if err != nil {
		log.Printf("failed to start response body for list metrics: %v", err)
	}

	for _, m := range all {
		_, err = fmt.Fprintf(w, "<li>%s (%s): %v</li>", m.ID, m.Type, m.Value)
		if err != nil {
			log.Printf("failed to write response body for list metrics for metric [name=%s]: %v", m.ID, err)
		}
	}

	_, err = fmt.Fprintln(w, "</ul></body></html>")
	if err != nil {
		log.Printf("failed to end response body for list metrics: %v", err)
	}
}

// PingHandler checks the availability of the database.
func (srv *Server) PingHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := utils.WithRetry(ctx, func() error {
		return srv.Storage.Ping(ctx)
	})

	if err != nil {
		http.Error(w, "db not available", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
