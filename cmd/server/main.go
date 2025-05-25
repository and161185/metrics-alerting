package main

import (
	"github.com/and161185/metrics-alerting/internal/config"
	"github.com/and161185/metrics-alerting/internal/server"
	"github.com/and161185/metrics-alerting/storage/postgres"
)

func main() {

	config := config.NewServerConfig()
	//storage := inmemory.NewMemStorage()
	storage, err := postgres.NewPostgresStorage(config.DatabaseDsn)
	if err != nil {
		config.Logger.Fatal(err)
	}

	srv := server.NewServer(storage, config)
	if err := srv.Run(); err != nil {
		config.Logger.Fatal(err)
	}
}
