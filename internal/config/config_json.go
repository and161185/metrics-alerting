// internal/config/json.go
package config

import (
	"encoding/json"
	"os"
	"time"
)

type serverJSON struct {
	Address       *string `json:"address"`
	Restore       *bool   `json:"restore"`
	StoreInterval *string `json:"store_interval"` // "1s"
	StoreFile     *string `json:"store_file"`
	DatabaseDSN   *string `json:"database_dsn"`
	CryptoKey     *string `json:"crypto_key"`
	TrustedSubnet *string `json:"trusted_subnet"`
}

type clientJSON struct {
	Address        *string `json:"address"`
	ReportInterval *string `json:"report_interval"`
	PollInterval   *string `json:"poll_interval"`
	CryptoKey      *string `json:"crypto_key"`
}

func loadServerJSON(path string) (*serverJSON, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg serverJSON
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func loadClientJSON(path string) (*clientJSON, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c clientJSON
	return &c, json.Unmarshal(b, &c)
}

func parseDurationSeconds(s string) (int, error) {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, err
	}
	return int(d / time.Second), nil
}
