package main

import (
	"context"
	"log"

	"github.com/and161185/metrics-alerting/internal/client"
	"github.com/and161185/metrics-alerting/internal/config"
	"github.com/and161185/metrics-alerting/storage/inmemory"
)

func main() {

	ctx := context.Background()

	config := config.NewClientConfig()
	storage := inmemory.NewMemStorage(ctx)
	clnt := client.NewClient(storage, config)

	log.Printf("Client config: ServerAddr=%s, ReportInterval=%d, PollInterval=%d, Timeout=%d",
		config.ServerAddr, config.ReportInterval, config.PollInterval, config.ClientTimeout)

	if err := clnt.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
