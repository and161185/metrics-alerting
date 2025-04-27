package client

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
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

	ReadEnvironment(cfg)

	if !strings.HasPrefix(cfg.serverAddr, "http://") && !strings.HasPrefix(cfg.serverAddr, "https://") {
		cfg.serverAddr = "http://" + cfg.serverAddr
	}

	return cfg
}

func ReadEnvironment(cfg *Config) {
	if addr := os.Getenv("ADDRESS"); addr != "" {
		cfg.serverAddr = addr
	}

	reportIntervalEnv := os.Getenv("REPORT_INTERVAL")
	if reportIntervalEnv != "" {
		v, err := strconv.Atoi(reportIntervalEnv)
		if err == nil {
			cfg.reportInterval = v
		} else {
			log.Printf("invalid REPORT_INTERVAL env var: %v", err)
		}
	}

	pollIntervallEnv := os.Getenv("POLL_INTERVAL")
	if pollIntervallEnv != "" {
		v, err := strconv.Atoi(pollIntervallEnv)
		if err == nil {
			cfg.pollInterval = v
		} else {
			log.Printf("invalid POLL_INTERVAL env var: %v", err)
		}
	}
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
				err := store.Save(m)
				if err != nil {
					log.Printf("failed to save metric [type=%s, name=%s]: %v", m.Type, m.ID, err)
				}
			}
		}

		if tics%reportInterval == 0 {
			if err := c.SendToServer(); err != nil {
				log.Printf("failed to send metrics: %v", err)
				continue
			}
			fmt.Println("SendToServer success")
			collector.ResetPollCount()
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
