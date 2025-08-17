package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/and161185/metrics-alerting/internal/config"
	"github.com/and161185/metrics-alerting/storage/inmemory"
	"github.com/go-chi/chi/v5"
)

func ExampleServer_UpdateMetricHandler() {
	ctx := context.Background()
	st := inmemory.NewMemStorage(ctx)
	srv := Server{Storage: st, Config: config.NewServerConfig()}

	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", srv.UpdateMetricHandler)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/test/1.23", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	fmt.Println(resp.StatusCode)
	// Output: 200
}

func ExampleServer_GetMetricHandler() {
	ctx := context.Background()
	st := inmemory.NewMemStorage(ctx)
	srv := Server{Storage: st}

	updateReq := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/123.45", nil)
	updateW := httptest.NewRecorder()
	srv.UpdateMetricHandler(updateW, updateReq)

	req := httptest.NewRequest(http.MethodGet, "/value/gauge/Alloc", nil)
	w := httptest.NewRecorder()

	srv.GetMetricHandler(w, req)

	fmt.Println(w.Code)
}

func ExampleServer_ListMetricsHandler() {
	ctx := context.Background()
	st := inmemory.NewMemStorage(ctx)
	srv := Server{Storage: st}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	srv.ListMetricsHandler(w, req)

	fmt.Println(w.Code)
}

func ExampleServer_PingHandler() {
	ctx := context.Background()
	st := inmemory.NewMemStorage(ctx)
	srv := Server{Storage: st}
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()

	srv.PingHandler(w, req)

	fmt.Println(w.Code)
}
