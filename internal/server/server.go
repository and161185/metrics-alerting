package server

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/and161185/metrics-alerting/cmd/server/logic"
	"github.com/and161185/metrics-alerting/storage"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
)

type Config struct {
	Addr string
}

type Server struct {
	storage storage.Storage
	config  *Config
}

func NewServer(storage storage.Storage) *Server {
	return &Server{
		storage: storage,
		config:  NewConfig(),
	}
}

func NewConfig() *Config {
	cfg := &Config{}
	flag.StringVar(&cfg.Addr, "a", "localhost:8080", "HTTP server address")
	flag.Parse()

	ReadEnvironment(cfg)

	return cfg
}

func ReadEnvironment(cfg *Config) {
	if addr := os.Getenv("ADDRESS"); addr != "" {
		cfg.Addr = addr
	}
}

func (s *Server) Run() error {

	router := chi.NewRouter()
	router.Use(middleware.StripSlashes)
	router.Post("/update/{type}/{name}/{value}", s.UpdateMetricHandler)
	router.Get("/value/{type}/{name}", s.GetMetricHandler)
	router.Get("/", s.ListMetricsHandler)

	return http.ListenAndServe(s.config.Addr, router)
}

func (s *Server) UpdateMetricHandler(w http.ResponseWriter, r *http.Request) {
	typ := chi.URLParam(r, "type")
	name := chi.URLParam(r, "name")
	val := chi.URLParam(r, "value")

	metric, err := logic.NewMetric(typ, name, val)
	if err != nil {
		log.Printf("failed to create metric [type=%s, name=%s]: %v", typ, name, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = s.storage.Save(metric)
	if err != nil {
		log.Printf("failed to save metric [name=%s]: %v", name, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) GetMetricHandler(w http.ResponseWriter, r *http.Request) {
	typ := chi.URLParam(r, "type")
	name := chi.URLParam(r, "name")

	metric, err := logic.NewEmptyMetric(typ, name)
	if err != nil {
		log.Printf("failed to create metric [type=%s, name=%s]: %v", typ, name, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	storedMetric, err := s.storage.Get(metric)
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

	w.WriteHeader(http.StatusOK)
	_, err = fmt.Fprintf(w, "%v", storedMetric.Value)
	if err != nil {
		log.Printf("failed to write response body for metric [name=%s]: %v", name, err)
	}
}

func (s *Server) ListMetricsHandler(w http.ResponseWriter, r *http.Request) {

	all, err := s.storage.GetAll()
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
