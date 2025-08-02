// server_test.go — переписанные тесты
package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/and161185/metrics-alerting/internal/config"
	"github.com/and161185/metrics-alerting/internal/utils"
	"github.com/and161185/metrics-alerting/model"
	"github.com/and161185/metrics-alerting/storage/inmemory"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTestServer() Server {
	ctx := context.Background()
	return Server{
		storage: inmemory.NewMemStorage(ctx),
		config: &config.ServerConfig{
			StoreInterval:   1,
			FileStoragePath: "./dev-null",
			Logger:          zap.NewNop().Sugar(),
		},
	}
}

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

	r := chi.NewRouter()
	srv := newTestServer()
	r.Post("/update/{type}/{name}/{value}", srv.UpdateMetricHandler)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.url, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			require.Equal(t, tc.wantStatus, w.Result().StatusCode)
		})
	}
}

func TestUpdateMetricHandlerJSON(t *testing.T) {
	tests := []struct {
		name       string
		metric     model.Metric
		wantStatus int
	}{
		{"valid_gauge", model.Metric{ID: "TestGauge", Type: "gauge", Value: utils.F64Ptr(42.0)}, http.StatusOK},
		{"valid_counter", model.Metric{ID: "TestCounter", Type: "counter", Delta: utils.I64Ptr(1)}, http.StatusOK},
		{"invalid_counter_value", model.Metric{ID: "TestCounter", Type: "counter"}, http.StatusBadRequest},
		{"invalid_type", model.Metric{ID: "TestGauge", Type: "invalid", Value: utils.F64Ptr(42.0)}, http.StatusBadRequest},
	}

	r := chi.NewRouter()
	srv := newTestServer()
	r.Post("/update/", srv.UpdateMetricHandlerJSON)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.metric)
			req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			require.Equal(t, tc.wantStatus, w.Result().StatusCode)
		})
	}
}

func TestGetMetricHandler(t *testing.T) {
	ctx := context.Background()
	st := inmemory.NewMemStorage(ctx)
	_ = st.Save(ctx, &model.Metric{ID: "test", Type: model.Gauge, Value: utils.F64Ptr(42.0)})
	srv := Server{storage: st}
	r := chi.NewRouter()
	r.Get("/value/{type}/{name}", srv.GetMetricHandler)

	req := httptest.NewRequest(http.MethodGet, "/value/gauge/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	resp := w.Result()
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	require.Equal(t, "42", strings.TrimSpace(string(body)))
}

func TestGetMetricHandlerJSON(t *testing.T) {
	ctx := context.Background()
	st := inmemory.NewMemStorage(ctx)
	_ = st.Save(ctx, &model.Metric{ID: "test", Type: model.Gauge, Value: utils.F64Ptr(42.0)})
	srv := Server{storage: st}
	r := chi.NewRouter()
	r.Post("/value/", srv.GetMetricHandlerJSON)

	body, _ := json.Marshal(model.Metric{ID: "test", Type: model.Gauge})
	req := httptest.NewRequest(http.MethodPost, "/value/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var m model.Metric
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&m))
	require.NotNil(t, m.Value)
	require.Equal(t, float64(42), *m.Value)
}

func TestListMetricsHandler(t *testing.T) {
	ctx := context.Background()
	st := inmemory.NewMemStorage(ctx)
	_ = st.Save(ctx, &model.Metric{ID: "foo", Type: model.Gauge, Value: utils.F64Ptr(1.23)})
	_ = st.Save(ctx, &model.Metric{ID: "bar", Type: model.Counter, Delta: utils.I64Ptr(10)})
	srv := Server{storage: st}
	r := chi.NewRouter()
	r.Get("/", srv.ListMetricsHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	require.Contains(t, string(body), "foo")
	require.Contains(t, string(body), "bar")
}

func TestUpdateArrayMetricHandlerJSON(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name        string
		metrics     []model.Metric
		contentType string
		body        []byte
		wantStatus  int
	}{
		{
			"valid_metrics",
			[]model.Metric{{ID: "g", Type: "gauge", Value: utils.F64Ptr(42)}, {ID: "c", Type: "counter", Delta: utils.I64Ptr(1)}},
			"application/json", nil, http.StatusOK,
		},
		{"invalid_content_type", []model.Metric{{ID: "g", Type: "gauge", Value: utils.F64Ptr(42)}}, "text/plain", nil, http.StatusUnsupportedMediaType},
		{"invalid_json", nil, "application/json", []byte("{invalid}"), http.StatusBadRequest},
		{"empty_array", []model.Metric{}, "application/json", nil, http.StatusBadRequest},
		{"invalid_counter", []model.Metric{{ID: "c", Type: "counter"}}, "application/json", nil, http.StatusUnprocessableEntity},
	}

	r := chi.NewRouter()
	srv := newTestServer()
	r.Post("/updates/", srv.UpdateArrayMetricHandlerJSON)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var body []byte
			var err error
			if tc.body != nil {
				body = tc.body
			} else {
				body, err = json.Marshal(tc.metrics)
				require.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(body))
			req.Header.Set("Content-Type", tc.contentType)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			resp := w.Result()
			defer resp.Body.Close()

			require.Equal(t, tc.wantStatus, resp.StatusCode)
			if tc.wantStatus == http.StatusOK {
				for _, m := range tc.metrics {
					stored, err := srv.storage.Get(ctx, &model.Metric{ID: m.ID, Type: m.Type})
					require.NoError(t, err)
					switch m.Type {
					case "gauge":
						require.Equal(t, *m.Value, *stored.Value)
					case "counter":
						require.Equal(t, *m.Delta, *stored.Delta)
					}
				}
			}
		})
	}
}
