package main

import (
	"context"
	"crypto/rsa"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/and161185/metrics-alerting/internal/buildinfo"
	"github.com/and161185/metrics-alerting/internal/config"
	"github.com/and161185/metrics-alerting/internal/crypto"
	"github.com/and161185/metrics-alerting/internal/server"
	"github.com/and161185/metrics-alerting/storage/inmemory"
	"github.com/and161185/metrics-alerting/storage/postgres"
)

func main() {
	buildinfo.PrintBuildInfo()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	config := config.NewServerConfig()
	defer func() { _ = config.Logger.Sync() }()

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

	var priv *rsa.PrivateKey
	if config.CryptoKeyPath != "" {
		var err error
		priv, err = crypto.LoadPrivateKey(config.CryptoKeyPath)
		if err != nil {
			log.Fatalf("failed to load private key: %v", err)
		}
	}

	srv := server.NewServer(storage, config, priv)
	if err := srv.Run(ctx); err != nil {
		config.Logger.Fatal(err)
	}
}
