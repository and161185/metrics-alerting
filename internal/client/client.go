// Package client provides functions for interacting with the metrics server.
package client

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/and161185/metrics-alerting/cmd/agent/collector"
	"github.com/and161185/metrics-alerting/internal/client/transport"
	"github.com/and161185/metrics-alerting/internal/config"
	"github.com/and161185/metrics-alerting/internal/crypto"
	"github.com/and161185/metrics-alerting/internal/utils"
	"github.com/and161185/metrics-alerting/model"
)

type storage interface {
	Save(ctx context.Context, metric *model.Metric) error
	GetAll(ctx context.Context) (map[string]*model.Metric, error)
}

// Client implements an agent that sends metrics to the server.
type Client struct {
	storage    storage
	config     *config.ClientConfig
	httpClient *http.Client
	realIP     string
}

// NewClient creates a new client instance with the given storage and configuration.
func NewClient(s storage, cfg *config.ClientConfig) (*Client, error) {
	hc, err := NewHTTPClient(cfg)
	if err != nil {
		return nil, err
	}
	return NewClientWithHTTP(s, cfg, hc), nil
}

func detectOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer conn.Close()
	if la, ok := conn.LocalAddr().(*net.UDPAddr); ok {
		return la.IP.String()
	}
	return ""
}

// DI: ready http.Client
func NewClientWithHTTP(s storage, cfg *config.ClientConfig, hc *http.Client) *Client {
	return &Client{storage: s, config: cfg, httpClient: hc, realIP: detectOutboundIP()}
}

// fabric http-client
func NewHTTPClient(cfg *config.ClientConfig) (*http.Client, error) {
	hc := &http.Client{Timeout: time.Duration(cfg.ClientTimeout) * time.Second}
	rt := http.DefaultTransport
	if cfg.CryptoKeyPath != "" {
		pub, err := crypto.LoadPublicKey(cfg.CryptoKeyPath)
		if err != nil {
			return nil, fmt.Errorf("load public key: %w", err)
		}
		rt = &transport.EncryptRoundTripper{Base: rt, PubKey: pub}
	}
	hc.Transport = rt
	return hc, nil
}

// Run starts collecting metrics and sending them to the server in the background.
func (clnt *Client) Run(ctx context.Context) error {
	store := clnt.storage
	poll := time.Duration(clnt.config.PollInterval) * time.Second
	report := time.Duration(clnt.config.ReportInterval) * time.Second
	rl := clnt.config.RateLimit

	var wg sync.WaitGroup

	// collectors
	wg.Add(1)
	go func() { defer wg.Done(); runtimeCollector(ctx, store, poll) }()
	wg.Add(1)
	go func() { defer wg.Done(); gopsutilCollector(ctx, store, poll) }()

	metricsCh := make(chan *model.Metric, rl)

	wg.Add(1)
	go func() {
		defer wg.Done()
		dispatchMetrics(ctx, store, metricsCh, report)
		close(metricsCh)
	}()

	// workers
	for i := 0; i < rl; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for m := range metricsCh {
				reqCtx, cancel := context.WithTimeout(context.Background(),
					time.Duration(clnt.config.ClientTimeout)*time.Second)
				_ = clnt.sendMetricToServer(reqCtx, m)
				cancel()
			}
		}()
	}

	<-ctx.Done()
	wg.Wait()
	return context.Canceled
}

func runtimeCollector(ctx context.Context, store storage, interval time.Duration) {
	if interval <= 0 {
		return
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			collectAndSave(ctx, store, collector.CollectRuntimeMetrics, "runtime")
		}
	}
}

func gopsutilCollector(ctx context.Context, store storage, interval time.Duration) {
	if interval <= 0 {
		return
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			collectAndSave(ctx, store, collector.CollectGopsutilMetrics, "gopsutil")
		}
	}
}

func collectAndSave(ctx context.Context, store storage, collect func() []model.Metric, label string) {
	if ctx.Err() != nil {
		return
	}
	for _, m := range collect() {
		if ctx.Err() != nil {
			return
		}
		mm := m
		if err := store.Save(ctx, &mm); err != nil {
			log.Printf("failed to save metric [%s][%s]: %v", label, mm.ID, err)
			if errors.Is(err, context.Canceled) {
				return
			}
		}
	}
}

func dispatchMetrics(ctx context.Context, store storage, ch chan<- *model.Metric, interval time.Duration) {
	if interval <= 0 {
		<-ctx.Done()

		if metrics, err := store.GetAll(context.Background()); err == nil {
			for _, m := range metrics {
				ch <- m
			}
		}
		return
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			metrics, err := store.GetAll(ctx)
			if err != nil {
				log.Printf("get: %v", err)
				continue
			}
			for _, m := range metrics {
				ch <- m
			} // без select с ctx
		case <-ctx.Done():
			if metrics, err := store.GetAll(context.Background()); err == nil {
				for _, m := range metrics {
					ch <- m
				}
			}
			return
		}
	}
}

func (clnt *Client) postGzipJSON(ctx context.Context, path string, payload any) (int, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("marshal: %w", err)
	}

	var body bytes.Buffer
	zw := gzip.NewWriter(&body)
	if _, err = zw.Write(raw); err != nil {
		return 0, fmt.Errorf("gzip write: %w", err)
	}
	if err = zw.Close(); err != nil {
		return 0, fmt.Errorf("gzip close: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, clnt.config.ServerAddr+path, &body)
	if err != nil {
		return 0, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	if clnt.realIP != "" {
		req.Header.Set("X-Real-IP", clnt.realIP)
	}
	if clnt.config.Key != "" {
		req.Header.Set("HashSHA256", utils.CalculateHash(body.Bytes(), clnt.config.Key))
	}

	var code int
	err = utils.WithRetry(ctx, func() error {
		resp, e := clnt.httpClient.Do(req)
		if e != nil {
			return e
		}
		defer resp.Body.Close()
		_, _ = io.Copy(io.Discard, resp.Body)
		code = resp.StatusCode
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("send request: %w", err)
	}
	return code, nil
}

func (clnt *Client) sendMetricToServer(ctx context.Context, m *model.Metric) error {
	code, err := clnt.postGzipJSON(ctx, "/update/", m)
	if err != nil {
		return err
	}
	if code != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", code)
	}
	return nil
}

func (clnt *Client) sendToServer(ctx context.Context) error {
	all, err := clnt.storage.GetAll(ctx)
	if err != nil || len(all) == 0 {
		return err
	}

	metrics := make([]model.Metric, 0, len(all))
	for _, m := range all {
		metrics = append(metrics, *m)
	}

	code, err := clnt.postGzipJSON(ctx, "/updates/", metrics)
	if err != nil {
		return err
	}
	if code != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", code)
	}
	return nil
}
