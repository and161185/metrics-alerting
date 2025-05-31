package main

import (
	"context"
	"encoding/json"
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

	b, _ := json.MarshalIndent(config, "", "  ")
	log.Printf("Server config:\n%s", string(b))

	if err := clnt.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
