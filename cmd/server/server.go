package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/and161185/metrics-alerting/cmd/server/logic"
	"github.com/and161185/metrics-alerting/storage"
	"github.com/go-chi/chi/v5"
)

type Server struct {
	storage storage.Storage
}

var ErrInvalidUrl = errors.New("invalid url")

func (s *Server) UpdateMetricHandler(w http.ResponseWriter, r *http.Request) {
	typ := chi.URLParam(r, "type")
	name := chi.URLParam(r, "name")
	val := chi.URLParam(r, "value")

	metric, err := logic.NewMetric(typ, name, val)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.storage.Save(metric)
	w.WriteHeader(http.StatusOK)
}

func (s *Server) GetMetricHandler(w http.ResponseWriter, r *http.Request) {
	typ := chi.URLParam(r, "type")
	name := chi.URLParam(r, "name")

	metric, err := logic.NewEmptyMetric(typ, name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	storedMetric, err := s.storage.Get(metric)
	if err != nil {
		if errors.Is(err, storage.ErrMetricNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "%v", storedMetric.Value)
}

func (s *Server) ListMetricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "<html><body><ul>")

	all, err := s.storage.GetAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, m := range all {
		fmt.Fprintf(w, "<li>%s (%s): %v</li>", m.ID, m.Type, m.Value)
	}

	fmt.Fprintln(w, "</ul></body></html>")
}
