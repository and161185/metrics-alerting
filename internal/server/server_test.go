package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/and161185/metrics-alerting/model"
	"github.com/and161185/metrics-alerting/storage"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func TestUpdateMetricHandler(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		url        string
		wantStatus int
	}{
		{"invalid method", http.MethodGet, "/update/gauge/test/1.23", http.StatusMethodNotAllowed},
		{"valid gauge", http.MethodPost, "/update/gauge/test/1.23", http.StatusOK},
		{"valid counter", http.MethodPost, "/update/counter/testCounter/1", http.StatusOK},
		{"invalid counter value", http.MethodPost, "/update/counter/testCounter/1.2", http.StatusBadRequest},
		{"invalid type", http.MethodPost, "/update/type/testCounter/1", http.StatusBadRequest},
		{"invalid url", http.MethodPost, "/update/gauge/gauge", http.StatusNotFound},
	}

	for _, v := range tests {
		t.Run(v.name, func(t *testing.T) {

			r := chi.NewRouter()
			server := Server{storage: storage.NewMemStorage()}
			r.Post("/update/{type}/{name}/{value}", server.UpdateMetricHandler)

			req := httptest.NewRequest(v.method, v.url, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			response := w.Result()
			defer response.Body.Close()

			assert.Equal(t, v.wantStatus, response.StatusCode)

		})
	}
}

func TestGetMetricHandler(t *testing.T) {
	st := storage.NewMemStorage()
	st.Save(model.Metric{ID: "test", Type: model.Gauge, Value: 42.0})
	server := Server{storage: st}

	router := chi.NewRouter()
	router.Get("/value/{type}/{name}", server.GetMetricHandler)

	req := httptest.NewRequest(http.MethodGet, "/value/gauge/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "42", strings.TrimSpace(string(body)))
}

func TestListMetricsHandler(t *testing.T) {
	st := storage.NewMemStorage()
	st.Save(model.Metric{ID: "foo", Type: model.Gauge, Value: 1.23})
	st.Save(model.Metric{ID: "bar", Type: model.Counter, Value: 10})
	server := Server{storage: st}

	router := chi.NewRouter()
	router.Get("/", server.ListMetricsHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "foo")
	assert.Contains(t, string(body), "bar")
}
