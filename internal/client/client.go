package client

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/and161185/metrics-alerting/cmd/agent/collector"
	"github.com/and161185/metrics-alerting/internal/config"
	"github.com/and161185/metrics-alerting/internal/utils"
	"github.com/and161185/metrics-alerting/model"
)

type Storage interface {
	Save(ctx context.Context, metric *model.Metric) error
	GetAll(ctx context.Context) (map[string]*model.Metric, error)
}

type Client struct {
	storage    Storage
	config     *config.ClientConfig
	httpClient *http.Client
}

func NewClient(storage Storage, config *config.ClientConfig) *Client {

	return &Client{
		storage:    storage,
		config:     config,
		httpClient: &http.Client{Timeout: time.Duration(config.ClientTimeout) * time.Second},
	}
}

func (clnt *Client) Run(ctx context.Context) error {

	store := clnt.storage
	pollInterval := clnt.config.PollInterval
	reportInterval := clnt.config.ReportInterval
	rateLimit := clnt.config.RateLimit

	go RuntimeCollector(ctx, store, time.Duration(pollInterval))

	go GopsutilCollector(ctx, store, time.Duration(pollInterval))

	metricsChan := make(chan *model.Metric, rateLimit)

	go DispatchMetrics(ctx, store, metricsChan, time.Duration(reportInterval))

	for i := 0; i < rateLimit; i++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case m := <-metricsChan:
					if err := clnt.SendMetricToServer(ctx, m); err != nil {
						log.Printf("failed to send metric: %v", err)
						continue
					}
					log.Println("Send success")
				}
			}
		}()
	}

	return nil
}

func RuntimeCollector(ctx context.Context, store Storage, interval time.Duration) {

	collectAndSave(ctx, store, collector.CollectRuntimeMetrics, "runtime")

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			collectAndSave(ctx, store, collector.CollectRuntimeMetrics, "runtime")
		}
	}
}

func GopsutilCollector(ctx context.Context, store Storage, interval time.Duration) {

	collectAndSave(ctx, store, collector.CollectGopsutilMetrics, "gopsutil")

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			collectAndSave(ctx, store, collector.CollectGopsutilMetrics, "gopsutil")
		}
	}
}

func collectAndSave(ctx context.Context, store Storage, collect func() []model.Metric, label string) {
	for _, m := range collect() {
		if err := store.Save(ctx, &m); err != nil {
			log.Printf("failed to save metric [%s][%s]: %v", label, m.ID, err)
		}
	}
}

func DispatchMetrics(ctx context.Context, store Storage, ch chan<- *model.Metric, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			metrics, err := store.GetAll(ctx)
			if err != nil {
				log.Printf("failed to get metrics: %v", err)
				continue
			}
			for _, m := range metrics {
				select {
				case ch <- m:
				case <-ctx.Done():
					return
				}
			}
		}
	}
}

func (clnt *Client) SendMetricToServer(ctx context.Context, m *model.Metric) error {
	serverAddr := clnt.config.ServerAddr
	httpClient := clnt.httpClient

	bodyRaw, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	var body bytes.Buffer
	zw := gzip.NewWriter(&body)
	if _, err := zw.Write(bodyRaw); err != nil {
		return fmt.Errorf("gzip write: %w", err)
	}
	if err := zw.Close(); err != nil {
		return fmt.Errorf("gzip close: %w", err)
	}

	url := fmt.Sprintf("%s/update/", serverAddr)
	req, err := http.NewRequest(http.MethodPost, url, &body)
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")

	if clnt.config.Key != "" {
		req.Header.Set("HashSHA256", utils.CalculateHash(body.Bytes(), clnt.config.Key))
	}

	var statusCode int
	err = utils.WithRetry(ctx, func() error {
		resp, err := httpClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		_, err = io.Copy(io.Discard, resp.Body)
		if err != nil {
			return err
		}

		statusCode = resp.StatusCode
		return nil
	})

	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}

	if statusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", statusCode)
	}

	return nil
}

func (clnt *Client) SendToServer(ctx context.Context) error {
	store := clnt.storage
	serverAddr := clnt.config.ServerAddr
	httpClient := clnt.httpClient

	all, err := store.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("internal error: %w", err)
	}

	if len(all) == 0 {
		return nil
	}

	metrics := make([]model.Metric, 0, len(all))
	for _, m := range all {
		metrics = append(metrics, *m)
	}

	bodyRaw, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	var body bytes.Buffer
	zw := gzip.NewWriter(&body)
	if _, err := zw.Write(bodyRaw); err != nil {
		return fmt.Errorf("gzip write: %w", err)
	}
	if err := zw.Close(); err != nil {
		return fmt.Errorf("gzip close: %w", err)
	}

	url := fmt.Sprintf("%s/updates/", serverAddr)
	req, err := http.NewRequest(http.MethodPost, url, &body)
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")

	if clnt.config.Key != "" {
		req.Header.Set("HashSHA256", utils.CalculateHash(body.Bytes(), clnt.config.Key))
	}

	var statusCode int
	err = utils.WithRetry(ctx, func() error {
		resp, err := httpClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		_, err = io.Copy(io.Discard, resp.Body)
		if err != nil {
			return err
		}

		statusCode = resp.StatusCode
		return nil
	})

	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}

	if statusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", statusCode)
	}

	return nil
}
