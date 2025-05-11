package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/and161185/metrics-alerting/cmd/agent/collector"
	"github.com/and161185/metrics-alerting/internal/config"
	"github.com/and161185/metrics-alerting/model"
)

type Storage interface {
	Save(metric *model.Metric) error
	GetAll() (map[string]*model.Metric, error)
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

func (clnt *Client) Run() error {

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
				err := store.Save(&m)
				if err != nil {
					log.Printf("failed to save metric [type=%s, name=%s]: %v", m.Type, m.ID, err)
				}
			}
		}

		if tics%reportInterval == 0 {
			if err := clnt.SendToServer(); err != nil {
				log.Printf("failed to send metrics: %v", err)
				continue
			}
			log.Println("SendToServer success")
			collector.ResetPollCount()
		}
	}
}

func (clnt *Client) SendToServer() error {

	store := clnt.storage
	serverAddr := clnt.config.ServerAddr
	httpClient := clnt.httpClient

	all, err := store.GetAll()
	if err != nil {
		return fmt.Errorf("internal error: %w", err)
	}

	for _, metric := range all {
		/*
			url := fmt.Sprintf(
				"%s/update/%s/%s/%v",
				serverAddr,
				metric.Type,
				metric.ID,
				metric.Value,
			)
		*/

		url := fmt.Sprintf("%s/update/", serverAddr)

		body, _ := json.Marshal(metric)
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("creating request for %s: %w", metric.ID, err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("sending %s: %w", metric.ID, err)
		}
		_, err = io.Copy(io.Discard, resp.Body)
		if err != nil {
			return fmt.Errorf("getting response %s: %w", metric.ID, err)
		}

		err = resp.Body.Close()
		if err != nil {
			return fmt.Errorf("response body closing %s: %w", metric.ID, err)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status for %s: %d", metric.ID, resp.StatusCode)
		}
	}

	return nil
}
