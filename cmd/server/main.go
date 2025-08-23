package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/and161185/metrics-alerting/internal/config"
	"github.com/and161185/metrics-alerting/internal/server"
	"github.com/and161185/metrics-alerting/storage/inmemory"
	"github.com/and161185/metrics-alerting/storage/postgres"
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func printBuildInfo() {
	v := buildVersion
	if v == "" {
		v = "N/A"
	}
	d := buildDate
	if d == "" {
		d = "N/A"
	}
	c := buildCommit
	if c == "" {
		c = "N/A"
	}

	fmt.Printf("Build version: %s\n", v)
	fmt.Printf("Build date: %s\n", d)
	fmt.Printf("Build commit: %s\n", c)
}

func main() {
	printBuildInfo()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	config := config.NewServerConfig()

	var (
		storage server.Storage
		err     error
	)
	if config.DatabaseDsn != "" {
		storage, err = postgres.NewPostgresStorage(ctx, config.DatabaseDsn)
		if err != nil {
			config.Logger.Fatal(err)
		}
	} else {
		storage = inmemory.NewMemStorage(ctx)
	}

	config.Logger.Infof("Server config: Addr=%s, StoreInterval=%d, FileStoragePath=%q, Restore=%t, DatabaseDSN set=%t",
		config.Addr,
		config.StoreInterval,
		config.FileStoragePath,
		config.Restore,
		config.DatabaseDsn != "",
	)

	srv := server.NewServer(storage, config)
	if err := srv.Run(ctx); err != nil {
		config.Logger.Fatal(err)
	}
}
