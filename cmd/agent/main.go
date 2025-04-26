package main

import (
	"github.com/and161185/metrics-alerting/internal/client"
	"github.com/and161185/metrics-alerting/storage"
)

func main() {

	c := client.NewClient(storage.NewMemStorage())

	if err := c.Run(); err != nil {
		panic(err)
	}
}
