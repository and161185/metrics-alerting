package main

import (
	"net/http"

	"github.com/and161185/metrics-alerting/cmd/server/handlers"
	"github.com/and161185/metrics-alerting/storage"
)

func main() {

	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {

	storage := storage.NewMemStorage()

	http.HandleFunc(`/update/`, handlers.UpdateMetricHandler(storage))
	return http.ListenAndServe(`:8080`, nil)
}
