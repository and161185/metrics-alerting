package testutils

import (
	"context"

	"github.com/and161185/metrics-alerting/internal/config"
	"github.com/and161185/metrics-alerting/internal/server"
	"github.com/and161185/metrics-alerting/storage/inmemory"
	"go.uber.org/zap"
)

func NewTestServer(ctx context.Context) server.Server {
	return server.Server{
		Storage: inmemory.NewMemStorage(ctx),
		Config: &config.ServerConfig{
			StoreInterval:   1,
			FileStoragePath: "./dev-null",
			Logger:          zap.NewNop().Sugar(),
		},
	}
}
