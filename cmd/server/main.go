package main

import (
	"log"

	"github.com/and161185/metrics-alerting/internal/config"
	"github.com/and161185/metrics-alerting/internal/server"
	"github.com/and161185/metrics-alerting/storage"
)

func main() {

	config := config.NewServerConfig()
	storage := storage.NewMemStorage()

	srv := server.NewServer(storage, config)
	if err := srv.Run(); err != nil {
		log.Fatal(err)
	}
}
