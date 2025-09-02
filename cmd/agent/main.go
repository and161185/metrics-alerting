package main

import (
	"context"
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config := config.NewClientConfig()
	storage := inmemory.NewMemStorage(ctx)
	clnt, err := client.NewClient(storage, config)
	if err != nil {
		log.Fatalf("client constructor error: %v", err)
	}

	log.Printf("Client config: ServerAddr=%s, ReportInterval=%d, PollInterval=%d, Timeout=%d",
		config.ServerAddr, config.ReportInterval, config.PollInterval, config.ClientTimeout)

	go func() {
		if err := clnt.Run(ctx); err != nil {
			log.Fatal(err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Println("shutting down...")
}
