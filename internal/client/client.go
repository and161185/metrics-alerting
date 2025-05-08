package client

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/and161185/metrics-alerting/cmd/agent/collector"
	"github.com/and161185/metrics-alerting/storage"
)

type Client struct {
	storage    storage.Storage
	config     *Config
	httpClient *http.Client
}

type Config struct {
	serverAddr     string
	reportInterval int
	pollInterval   int
	clientTimeout  int
}

func NewClient(storage storage.Storage) *Client {

	config := NewConfig()

	return &Client{
		storage:    storage,
		config:     config,
		httpClient: &http.Client{Timeout: time.Duration(config.clientTimeout) * time.Second},
	}
}

func NewConfig() *Config {
	cfg := &Config{}
	flag.StringVar(&cfg.serverAddr, "a", "http://localhost:8080", "HTTP server address (must include http(s)://)")
	flag.IntVar(&cfg.reportInterval, "r", 10, "report interval")
	flag.IntVar(&cfg.pollInterval, "p", 2, "poll interval")
	flag.IntVar(&cfg.clientTimeout, "t", 10, "client timeout")
	flag.Parse()

	if !strings.HasPrefix(cfg.serverAddr, "http://") && !strings.HasPrefix(cfg.serverAddr, "https://") {
		cfg.serverAddr = "http://" + cfg.serverAddr
	}

	return cfg
}

func (c *Client) Run() error {

	store := c.storage
	pollInterval := c.config.pollInterval
	reportInterval := c.config.reportInterval

	tics := 0

	for {
		time.Sleep(1 * time.Second)
		tics++

		fmt.Println("Tick:", tics)

		if tics%pollInterval == 0 {
			for _, m := range collector.CollectRuntimeMetrics() {
				store.Save(m)
			}
		}

		if tics%reportInterval == 0 {
			if err := c.SendToServer(); err == nil {
				fmt.Println("SendToServer success")

				collector.ResetPollCount()
			} else {
				fmt.Println("SendToServer error:", err)

			}
		}
	}
}

func (c *Client) SendToServer() error {

	store := c.storage
	serverAddr := c.config.serverAddr
	httpClient := c.httpClient

	all, err := store.GetAll()
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

		resp, err := httpClient.Do(req)
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
