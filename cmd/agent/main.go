package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/and161185/metrics-alerting/internal/client"
	"github.com/and161185/metrics-alerting/internal/config"
	"github.com/and161185/metrics-alerting/storage/inmemory"
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config := config.NewClientConfig()
	storage := inmemory.NewMemStorage(ctx)
	clnt := client.NewClient(storage, config)

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
