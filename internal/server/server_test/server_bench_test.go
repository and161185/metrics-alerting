package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/and161185/metrics-alerting/internal/server/testutils"
	"github.com/and161185/metrics-alerting/internal/utils"
	"github.com/and161185/metrics-alerting/model"
	"github.com/go-chi/chi/v5"
)

func BenchmarkUpdateMetricHandler(b *testing.B) {
	ctx := context.Background()
	srv := testutils.NewTestServer(ctx)
	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", srv.UpdateMetricHandler)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/loadavg/42.5", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}

func BenchmarkUpdateMetricHandlerJSON(b *testing.B) {
	ctx := context.Background()
	srv := testutils.NewTestServer(ctx)
	r := chi.NewRouter()
	r.Post("/update/", srv.UpdateMetricHandlerJSON)

	m := model.Metric{ID: "cpu", Type: "gauge", Value: utils.F64Ptr(12.3)}
	body, _ := json.Marshal(m)
	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}

func BenchmarkGetMetricHandler(b *testing.B) {
	ctx := context.Background()
	srv := testutils.NewTestServer(ctx)
	_ = srv.Storage.Save(ctx, &model.Metric{ID: "cpu", Type: "gauge", Value: utils.F64Ptr(42.0)})

	r := chi.NewRouter()
	r.Get("/value/{type}/{name}", srv.GetMetricHandler)

	req := httptest.NewRequest(http.MethodGet, "/value/gauge/cpu", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}

func BenchmarkListMetricsHandler(b *testing.B) {
	ctx := context.Background()
	srv := testutils.NewTestServer(ctx)
	_ = srv.Storage.Save(ctx, &model.Metric{ID: "foo", Type: "gauge", Value: utils.F64Ptr(1.23)})
	_ = srv.Storage.Save(ctx, &model.Metric{ID: "bar", Type: "counter", Delta: utils.I64Ptr(10)})

	r := chi.NewRouter()
	r.Get("/", srv.ListMetricsHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}
