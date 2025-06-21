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

	tics := 0

	for {
		time.Sleep(1 * time.Second)
		tics++

		log.Println("Tick:", tics)

		if tics%pollInterval == 0 {
			for _, m := range collector.CollectRuntimeMetrics() {
				err := store.Save(ctx, &m)
				if err != nil {
					log.Printf("failed to save metric [type=%s, name=%s]: %v", m.Type, m.ID, err)
				}
			}
		}

		if tics%reportInterval == 0 {
			if err := clnt.SendToServer(ctx); err != nil {
				log.Printf("failed to send metrics: %v", err)
				continue
			}
			log.Println("SendToServer success")
			collector.ResetPollCount()
		}
	}
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
