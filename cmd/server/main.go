package main

import (
	"net/http"

	"github.com/and161185/metrics-alerting/storage"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
)

func main() {

	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {

	server := Server{storage: storage.NewMemStorage()}

	router := chi.NewRouter()
	router.Use(middleware.StripSlashes)
	router.Post("/update/{type}/{name}/{value}", server.UpdateMetricHandler)
	router.Get("/value/{type}/{name}", server.GetMetricHandler)
	router.Get("/", server.ListMetricsHandler)

	return http.ListenAndServe(":8080", router)
}
