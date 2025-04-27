package server

import (
	"io"
	"log"
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
			defer func() {
				if err := response.Body.Close(); err != nil {
					log.Fatalf("failed to close response body for url %s: %v", v.url, err)
				}
			}()

			assert.Equal(t, v.wantStatus, response.StatusCode)

		})
	}
}

func TestGetMetricHandler(t *testing.T) {
	st := storage.NewMemStorage()

	m := model.Metric{ID: "test", Type: model.Gauge, Value: 42.0}
	err := st.Save(m)
	if err != nil {
		t.Fatalf("Save in storage metric %s %f failed: %v", m.ID, m.Value, err)
	}
	server := Server{storage: st}

	router := chi.NewRouter()
	router.Get("/value/{type}/{name}", server.GetMetricHandler)

	url := "/value/gauge/test"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	response := w.Result()
	defer func() {
		if err := response.Body.Close(); err != nil {
			log.Fatalf("failed to close response body for url %s: %v", url, err)
		}
	}()

	assert.Equal(t, http.StatusOK, response.StatusCode)

	body, _ := io.ReadAll(response.Body)
	assert.Equal(t, "42", strings.TrimSpace(string(body)))
}

func TestListMetricsHandler(t *testing.T) {
	st := storage.NewMemStorage()

	m1 := model.Metric{ID: "foo", Type: model.Gauge, Value: 1.23}
	err := st.Save(m1)
	if err != nil {
		t.Fatalf("Save in storage metric %s %f failed: %v", m1.ID, m1.Value, err)
	}

	m2 := model.Metric{ID: "bar", Type: model.Counter, Value: 10}
	err = st.Save(m2)
	if err != nil {
		t.Fatalf("Save in storage metric %s %f failed: %v", m2.ID, m2.Value, err)
	}
	server := Server{storage: st}

	router := chi.NewRouter()
	router.Get("/", server.ListMetricsHandler)

	url := "/"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	response := w.Result()
	defer func() {
		if err := response.Body.Close(); err != nil {
			log.Fatalf("failed to close response body for url %s: %v", url, err)
		}
	}()

	assert.Equal(t, http.StatusOK, response.StatusCode)

	body, _ := io.ReadAll(response.Body)
	assert.Contains(t, string(body), "foo")
	assert.Contains(t, string(body), "bar")
}
