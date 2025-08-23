// Package config provides application configuration structures and helpers.
package config

import (
	"flag"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"

	"go.uber.org/zap"
)

// ClientConfig holds the configuration settings for the agent.
type ClientConfig struct {
	ServerAddr     string // Server address
	ReportInterval int    // Interval for sending metrics (in seconds)
	PollInterval   int    // Interval for collecting metrics (in seconds)
	ClientTimeout  int    // HTTP client timeout (in seconds)
	Key            string // Key for hash generation
	RateLimit      int    // Limit on simultaneous outgoing requests
}

// ServerConfig holds the configuration settings for the server.
type ServerConfig struct {
	Addr            string // Server address
	Logger          *zap.SugaredLogger
	StoreInterval   int    // Interval for storing metrics to file (in seconds)
	FileStoragePath string // Path to the file for metric storage
	Restore         bool   // Whether to restore metrics from file on startup
	DatabaseDsn     string // Data Source Name for PostgreSQL
	Key             string // Key for hash verification
}

// NewServerConfig creates and returns a new ServerConfig by parsing flags and environment variables.
func NewServerConfig() *ServerConfig {
	logCfg := zap.NewProductionConfig()
	logCfg.OutputPaths = []string{"stdout", "server.log"}

	logger := zap.Must(logCfg.Build())

	cfg := &ServerConfig{}
	flag.StringVar(&cfg.Addr, "a", "localhost:8080", "HTTP server address")
	flag.IntVar(&cfg.StoreInterval, "i", 300, "store interval")
	flag.StringVar(&cfg.FileStoragePath, "f", "./tmp/metrics-db.json", "path to metrics file")
	flag.BoolVar(&cfg.Restore, "r", true, "load metrics from last file")
	flag.StringVar(&cfg.DatabaseDsn, "d", "", "DB connection string")
	flag.StringVar(&cfg.Key, "k", "", "Hash key string")
	flag.Parse()

	cfg.Logger = logger.Sugar()

	readServerEnvironment(cfg)

	return cfg
}

func readServerEnvironment(cfg *ServerConfig) {
	if addr := os.Getenv("ADDRESS"); addr != "" {
		cfg.Addr = addr
	}

	storeIntervalEnv := os.Getenv("STORE_INTERVAL")
	if storeIntervalEnv != "" {
		v, err := strconv.Atoi(storeIntervalEnv)
		if err == nil {
			cfg.StoreInterval = v
		} else {
			log.Printf("invalid STORE_INTERVAL env var: %v", err)
		}
	}

	if fsp := os.Getenv("FILE_STORAGE_PATH"); fsp != "" {
		cfg.FileStoragePath = fsp
	}

	if dbDsn := os.Getenv("DATABASE_DSN"); dbDsn != "" {
		cfg.DatabaseDsn = dbDsn
	}

	restoreEnv := os.Getenv("RESTORE")
	if restoreEnv != "" {
		v, err := strconv.ParseBool(restoreEnv)
		if err == nil {
			cfg.Restore = v
		} else {
			log.Printf("invalid RESTORE env var: %v", err)
		}
	}

	if key := os.Getenv("KEY"); key != "" {
		cfg.Key = key
	}
}

// NewClientConfig creates and returns a new ClientConfig by parsing flags and environment variables.
func NewClientConfig() *ClientConfig {
	cfg := &ClientConfig{}
	flag.StringVar(&cfg.ServerAddr, "a", "http://localhost:8080", "HTTP server address (must include http(s)://)")
	flag.IntVar(&cfg.ReportInterval, "r", 10, "report interval")
	flag.IntVar(&cfg.PollInterval, "p", 2, "poll interval")
	flag.IntVar(&cfg.ClientTimeout, "t", 10, "client timeout")
	flag.StringVar(&cfg.Key, "k", "", "Hash key string")
	flag.IntVar(&cfg.RateLimit, "l", runtime.NumCPU(), "rate limit")
	flag.Parse()

	readClientEnvironment(cfg)

	if !strings.HasPrefix(cfg.ServerAddr, "http://") && !strings.HasPrefix(cfg.ServerAddr, "https://") {
		cfg.ServerAddr = "http://" + cfg.ServerAddr
	}

	return cfg
}

func readClientEnvironment(cfg *ClientConfig) {
	if addr := os.Getenv("ADDRESS"); addr != "" {
		cfg.ServerAddr = addr
	}

	reportIntervalEnv := os.Getenv("REPORT_INTERVAL")
	if reportIntervalEnv != "" {
		v, err := strconv.Atoi(reportIntervalEnv)
		if err == nil {
			cfg.ReportInterval = v
		} else {
			log.Printf("invalid REPORT_INTERVAL env var: %v", err)
		}
	}

	pollIntervallEnv := os.Getenv("POLL_INTERVAL")
	if pollIntervallEnv != "" {
		v, err := strconv.Atoi(pollIntervallEnv)
		if err == nil {
			cfg.PollInterval = v
		} else {
			log.Printf("invalid POLL_INTERVAL env var: %v", err)
		}
	}

	if key := os.Getenv("KEY"); key != "" {
		cfg.Key = key
	}
}
