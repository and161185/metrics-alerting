package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/and161185/metrics-alerting/internal/buildinfo"
	"github.com/and161185/metrics-alerting/internal/client"
	"github.com/and161185/metrics-alerting/internal/config"
	"github.com/and161185/metrics-alerting/storage/inmemory"
)

func main() {
	buildinfo.PrintBuildInfo()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	config := config.NewClientConfig()
	storage := inmemory.NewMemStorage(ctx)
	clnt, err := client.NewClient(storage, config)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Client config: ServerAddr=%s, ReportInterval=%d, PollInterval=%d, Timeout=%d",
		config.ServerAddr, config.ReportInterval, config.PollInterval, config.ClientTimeout)

	if err := clnt.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatal(err)
	}
	log.Println("shutting down")
}
