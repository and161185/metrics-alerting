package main

import (
	"github.com/and161185/metrics-alerting/internal/server"
	"github.com/and161185/metrics-alerting/storage"
)

func main() {

	s := server.NewServer(storage.NewMemStorage())
	if err := s.Run(); err != nil {
		panic(err)
	}
}
