package server

import (
	"context"
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

type Storage interface {
	Save(ctx context.Context, metric *model.Metric) error
	Get(ctx context.Context, metric *model.Metric) (*model.Metric, error)
	GetAll(ctx context.Context) (map[string]*model.Metric, error)
	SaveToFile(ctx context.Context, filePath string) error
	LoadFromFile(ctx context.Context, filePath string) error
	Ping(ctx context.Context) error
}

type Server struct {
	storage Storage
	config  *config.ServerConfig
}

func NewServer(storage Storage, config *config.ServerConfig) *Server {
	return &Server{
		storage: storage,
		config:  config,
	}
}

func (srv *Server) buildRouter() http.Handler {
	router := chi.NewRouter()
	router.Use(chiMiddleware.StripSlashes)
	router.Use(middleware.LogMiddleware(srv.config.Logger))
	router.Use(middleware.DecompressMiddleware)
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

func (srv *Server) Run(ctx context.Context) error {
	router := srv.buildRouter()

	server := &http.Server{
		Addr:    srv.config.Addr,
		Handler: router,
	}

	if srv.config.Restore {
		if err := srv.storage.LoadFromFile(ctx, srv.config.FileStoragePath); err != nil {
			srv.config.Logger.Warnf("failed to restore metrics from file: %v", err)
		}
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			srv.config.Logger.Fatalf("server error: %v", err)
		}
	}()

	if srv.config.StoreInterval > 0 {
		ticker := time.NewTicker(time.Duration(srv.config.StoreInterval) * time.Second)
		go func() {
			for range ticker.C {
				if err := srv.storage.SaveToFile(ctx, srv.config.FileStoragePath); err != nil {
					srv.config.Logger.Errorf("auto-save failed: %v", err)
				}
			}
		}()
	}

	<-ctx.Done()

	if srv.config.FileStoragePath != "" {
		if err := srv.storage.SaveToFile(ctx, srv.config.FileStoragePath); err != nil {
			log.Printf("save failed: %v", err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return server.Shutdown(shutdownCtx)
}

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
		return srv.SaveToStorage(ctx, metric)
	})

	if err != nil {
		log.Printf("failed to save metric [name=%s]: %v", metric.ID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

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
		return srv.SaveToStorage(ctx, &metric)
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
		return srv.SaveBatchToStorage(ctx, metricsArray)
	})

	if err != nil {
		log.Printf("failed to save metrics: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (srv *Server) SaveToStorage(ctx context.Context, metric *model.Metric) error {

	err := srv.storage.Save(ctx, metric)
	if err != nil {
		return err
	}

	if srv.config.StoreInterval == 0 {
		if err := srv.storage.SaveToFile(ctx, srv.config.FileStoragePath); err != nil {
			srv.config.Logger.Errorf("failed to-save file %s: %v", srv.config.FileStoragePath, err)
		}
	}

	return nil
}

func (srv *Server) SaveBatchToStorage(ctx context.Context, metricsArray []model.Metric) error {

	for _, metric := range metricsArray {
		err := srv.storage.Save(ctx, &metric)
		if err != nil {
			return err
		}
	}

	if srv.config.StoreInterval == 0 {
		if err := srv.storage.SaveToFile(ctx, srv.config.FileStoragePath); err != nil {
			srv.config.Logger.Errorf("failed to-save file %s: %v", srv.config.FileStoragePath, err)
		}
	}

	return nil
}

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
		var err error
		storedMetric, err = srv.storage.Get(ctx, metric)
		return err
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
		storedMetric, err = srv.storage.Get(ctx, &reqMetric)
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

func (srv *Server) ListMetricsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var all map[string]*model.Metric
	err := utils.WithRetry(ctx, func() error {
		var err error
		all, err = srv.storage.GetAll(ctx)
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

func (srv *Server) PingHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := utils.WithRetry(ctx, func() error {
		return srv.storage.Ping(ctx)
	})

	if err != nil {
		http.Error(w, "db not available", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
