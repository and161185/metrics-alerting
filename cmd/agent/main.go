package main

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/and161185/metrics-alerting/cmd/agent/collector"
	"github.com/and161185/metrics-alerting/storage"
)

func main() {

	serverAddr := "http://localhost:8080"

	if err := run(serverAddr); err != nil {
		panic(err)
	}
}

func run(serverAddr string) error {
	storage := storage.NewMemStorage()
	tics := 0

	for {
		time.Sleep(1 * time.Second)
		tics++

		if tics%2 == 0 {
			for _, m := range collector.CollectRuntimeMetrics() {
				storage.Save(m)
			}
		}

		if tics%10 == 0 {
			if err := SendToServer(storage, serverAddr); err == nil {
				collector.ResetPollCount()
			}
		}
	}
}

func SendToServer(s *storage.MemStorage, serverAddr string) error {
	client := &http.Client{Timeout: 2 * time.Second}

	all, err := s.GetAll()
	if err != nil {
		return fmt.Errorf("internal error: %w", err)
	}

	for _, metric := range all {
		url := fmt.Sprintf(
			"%s/update/%s/%s/%v",
			serverAddr,
			metric.Type,
			metric.ID,
			metric.Value,
		)

		req, err := http.NewRequest(http.MethodPost, url, nil)
		if err != nil {
			return fmt.Errorf("creating request for %s: %w", metric.ID, err)
		}
		req.Header.Set("Content-Type", "text/plain")

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("sending %s: %w", metric.ID, err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status for %s: %d", metric.ID, resp.StatusCode)
		}
	}

	return nil
}
