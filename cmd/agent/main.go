package main

import (
	"log"

	"github.com/and161185/metrics-alerting/internal/client"
	"github.com/and161185/metrics-alerting/internal/config"
	"github.com/and161185/metrics-alerting/storage/inmemory"
)

func main() {

	config := config.NewClientConfig()
	storage := inmemory.NewMemStorage()
	clnt := client.NewClient(storage, config)

	if err := clnt.Run(); err != nil {
		log.Fatal(err)
	}
}
