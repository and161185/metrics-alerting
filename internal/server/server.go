package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/and161185/metrics-alerting/cmd/server/metrics"
	"github.com/and161185/metrics-alerting/internal/config"
	"github.com/and161185/metrics-alerting/internal/server/middleware"
	"github.com/and161185/metrics-alerting/model"
	"github.com/and161185/metrics-alerting/storage"
	chiMiddleware "github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
)

type Storage interface {
	Save(metric *model.Metric) error
	Get(metric *model.Metric) (*model.Metric, error)
	GetAll() (map[string]*model.Metric, error)
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

func (srv *Server) Run() error {

	router := chi.NewRouter()
	router.Use(chiMiddleware.StripSlashes)
	router.Use(middleware.LogMiddelware(srv.config.Logger))
	router.Post("/update/{type}/{name}/{value}", srv.UpdateMetricHandler)
	router.Post("/update", srv.UpdateMetricHandlerJSON)
	router.Get("/value/{type}/{name}", srv.GetMetricHandler)
	router.Post("/value", srv.GetMetricHandlerJSON)
	router.Get("/", srv.ListMetricsHandler)

	return http.ListenAndServe(srv.config.Addr, router)
}

func (srv *Server) UpdateMetricHandler(w http.ResponseWriter, r *http.Request) {
	typ := chi.URLParam(r, "type")
	name := chi.URLParam(r, "name")
	val := chi.URLParam(r, "value")

	metric, err := metrics.NewMetric(typ, name, val)
	if err != nil {
		log.Printf("failed to create metric [type=%s, name=%s]: %v", typ, name, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = srv.storage.Save(metric)
	if err != nil {
		log.Printf("failed to save metric [name=%s]: %v", name, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (srv *Server) UpdateMetricHandlerJSON(w http.ResponseWriter, r *http.Request) {

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

	err = srv.storage.Save(&metric)
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

func (srv *Server) GetMetricHandler(w http.ResponseWriter, r *http.Request) {
	typ := chi.URLParam(r, "type")
	name := chi.URLParam(r, "name")

	metric, err := metrics.NewEmptyMetric(typ, name)
	if err != nil {
		log.Printf("failed to create metric [type=%s, name=%s]: %v", typ, name, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	storedMetric, err := srv.storage.Get(metric)
	if err != nil {
		if errors.Is(err, storage.ErrMetricNotFound) {
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
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "unsupported content type", http.StatusUnsupportedMediaType)
		return
	}

	var reqMetric model.Metric
	if err := json.NewDecoder(r.Body).Decode(&reqMetric); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	stored, err := srv.storage.Get(&reqMetric)
	if err != nil {
		if errors.Is(err, storage.ErrMetricNotFound) {
			http.NotFound(w, r)
		} else {
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stored); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func (srv *Server) ListMetricsHandler(w http.ResponseWriter, r *http.Request) {

	all, err := srv.storage.GetAll()
	if err != nil {
		log.Printf("failed to get all metrics from storage: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
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
