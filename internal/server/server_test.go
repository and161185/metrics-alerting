package server

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/and161185/metrics-alerting/internal/utils"
	"github.com/and161185/metrics-alerting/model"
	"github.com/and161185/metrics-alerting/storage"
	"github.com/go-chi/chi/v5"
)

func TestUpdateMetricHandler(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		url        string
		wantStatus int
	}{
		{"invalid_method", http.MethodGet, "/update/gauge/test/1.23", http.StatusMethodNotAllowed},
		{"valid_gauge", http.MethodPost, "/update/gauge/test/1.23", http.StatusOK},
		{"valid_counter", http.MethodPost, "/update/counter/testCounter/1", http.StatusOK},
		{"invalid_counter_value", http.MethodPost, "/update/counter/testCounter/1.2", http.StatusBadRequest},
		{"invalid_type", http.MethodPost, "/update/type/testCounter/1", http.StatusBadRequest},
		{"invalid_url", http.MethodPost, "/update/gauge/gauge", http.StatusNotFound},
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

			if v.wantStatus != response.StatusCode {
				log.Fatalf("wrong response status: want %d get %d", v.wantStatus, response.StatusCode)
			}
		})
	}
}

func TestUpdateMetricHandlerJSON(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		metric     model.Metric
		wantStatus int
	}{
		{"valid_gauge", http.MethodPost, model.Metric{ID: "TestGauge", Type: "gauge", Value: utils.F64Ptr(42.0)}, http.StatusOK},
		{"valid_counter", http.MethodPost, model.Metric{ID: "TestCounter", Type: "counter", Delta: utils.I64Ptr(1)}, http.StatusOK},
		{"invalid_counter_value", http.MethodPost, model.Metric{ID: "TestCounter", Type: "counter"}, http.StatusBadRequest},
		{"invalid_type", http.MethodPost, model.Metric{ID: "TestGauge", Type: "type_invalid", Value: utils.F64Ptr(42.0)}, http.StatusBadRequest},
	}

	for _, v := range tests {
		t.Run(v.name, func(t *testing.T) {

			r := chi.NewRouter()
			server := Server{storage: storage.NewMemStorage()}
			r.Post("/update/", server.UpdateMetricHandlerJSON)

			body, _ := json.Marshal(v.metric)
			req := httptest.NewRequest(v.method, "/update/", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			response := w.Result()
			defer func() {
				if err := response.Body.Close(); err != nil {
					log.Fatalf("%s: failed to close response body for url %s: %v", v.name, "/update/", err)
				}
			}()

			if v.wantStatus != response.StatusCode {
				log.Fatalf("%s: wrong response status: want %d get %d", v.name, v.wantStatus, response.StatusCode)
			}

			if v.wantStatus == http.StatusOK {
				var respMetric model.Metric
				if err := json.NewDecoder(response.Body).Decode(&respMetric); err != nil {
					t.Fatalf("%s: failed to decode response JSON: %v", v.name, err)
				}

				if respMetric.ID != v.metric.ID || respMetric.Type != v.metric.Type {
					t.Errorf("%s: wrong metric returned: got %+v", v.name, respMetric)
				}

				switch v.metric.Type {
				case "gauge":
					if respMetric.Value == nil {
						t.Errorf("%s: expected non-nil Value", v.name)
					}
				case "counter":
					if respMetric.Delta == nil {
						t.Errorf("%s: expected non-nil Delta", v.name)
					}
				}
			}

		})
	}
}

func TestGetMetricHandler(t *testing.T) {
	st := storage.NewMemStorage()

	m := model.Metric{ID: "test", Type: model.Gauge, Value: utils.F64Ptr(42.0)}
	err := st.Save(&m)
	if err != nil {
		t.Fatalf("Save in storage metric %s %f failed: %v", m.ID, *m.Value, err)
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

	if http.StatusOK != response.StatusCode {
		log.Fatalf("wrong response status: want %d get %d", http.StatusOK, response.StatusCode)
	}

	body, _ := io.ReadAll(response.Body)
	if strings.TrimSpace(string(body)) != "42" {
		t.Errorf("wrong response body: want %s, got %s", "42", string(body))
	}
}

func TestGetMetricHandlerJSON(t *testing.T) {
	st := storage.NewMemStorage()

	m := model.Metric{ID: "test", Type: model.Gauge, Value: utils.F64Ptr(42.0)}
	err := st.Save(&m)
	if err != nil {
		t.Fatalf("Save in storage metric %s %f failed: %v", m.ID, *m.Value, err)
	}
	server := Server{storage: st}

	router := chi.NewRouter()
	router.Post("/value/", server.GetMetricHandlerJSON)

	url := "/value/"
	body, _ := json.Marshal(model.Metric{ID: "test", Type: model.Gauge})
	req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	response := w.Result()
	defer func() {
		if err := response.Body.Close(); err != nil {
			log.Fatalf("failed to close response body for url %s: %v", url, err)
		}
	}()

	if http.StatusOK != response.StatusCode {
		log.Fatalf("wrong response status: want %d get %d", http.StatusOK, response.StatusCode)
	}

	var mResponse model.Metric
	err = json.NewDecoder(response.Body).Decode(&mResponse)
	if err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	if m.Value == nil || *m.Value != 42 {
		t.Errorf("unexpected metric value: want 42, got %v", m.Value)
	}
}

func TestListMetricsHandler(t *testing.T) {
	st := storage.NewMemStorage()

	m1 := model.Metric{ID: "foo", Type: model.Gauge, Value: utils.F64Ptr(1.23)}
	err := st.Save(&m1)
	if err != nil {
		t.Fatalf("Save in storage metric %s %f failed: %v", m1.ID, *m1.Value, err)
	}

	m2 := model.Metric{ID: "bar", Type: model.Counter, Delta: utils.I64Ptr(10)}
	err = st.Save(&m2)
	if err != nil {
		t.Fatalf("Save in storage metric %s %f failed: %v", m2.ID, *m2.Value, err)
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

	if http.StatusOK != response.StatusCode {
		log.Fatalf("wrong response status: want %d get %d", http.StatusOK, response.StatusCode)
	}

	body, _ := io.ReadAll(response.Body)
	if !strings.Contains(string(body), "foo") {
		t.Errorf(`response body doesn't contain "foo": %s`, string(body))
	}
	if !strings.Contains(string(body), "bar") {
		t.Errorf(`response body doesn't contain "bar": %s`, string(body))
	}

}
