package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/and161185/metrics-alerting/internal/config"
	"github.com/and161185/metrics-alerting/internal/server"
	"github.com/and161185/metrics-alerting/storage/inmemory"
	"github.com/and161185/metrics-alerting/storage/postgres"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	config := config.NewServerConfig()

	var storage server.Storage
	var err error

	if config.DatabaseDsn != "" {
		storage, err = postgres.NewPostgresStorage(ctx, config.DatabaseDsn)
	} else {
		storage = inmemory.NewMemStorage(ctx)
	}

	if err != nil {
		config.Logger.Fatal(err)
	}

	srv := server.NewServer(storage, config)
	if err := srv.Run(ctx); err != nil {
		config.Logger.Fatal(err)
	}
}
