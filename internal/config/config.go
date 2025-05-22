package config

import (
	"flag"
	"log"
	"os"
	"strconv"
	"strings"

	"go.uber.org/zap"
)

type ClientConfig struct {
	ServerAddr     string
	ReportInterval int
	PollInterval   int
	ClientTimeout  int
}

type ServerConfig struct {
	Addr            string
	Logger          *zap.SugaredLogger
	StoreInterval   int
	FileStoragePath string
	Restore         bool
}

func NewServerConfig() *ServerConfig {
	logCfg := zap.NewProductionConfig()
	logCfg.OutputPaths = []string{"stdout", "server.log"}

	logger := zap.Must(logCfg.Build())

	cfg := &ServerConfig{}
	flag.StringVar(&cfg.Addr, "a", "localhost:8080", "HTTP server address")
	flag.IntVar(&cfg.StoreInterval, "i", 300, "store interval")
	flag.StringVar(&cfg.FileStoragePath, "f", "./tmp/metrics-db.json", "path to metrics file")
	flag.BoolVar(&cfg.Restore, "r", true, "load metrics from last file")
	flag.Parse()

	cfg.Logger = logger.Sugar()

	ReadServerEnvironment(cfg)

	return cfg
}

func ReadServerEnvironment(cfg *ServerConfig) {
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

	restoreEnv := os.Getenv("RESTORE")
	if restoreEnv != "" {
		v, err := strconv.ParseBool(restoreEnv)
		if err == nil {
			cfg.Restore = v
		} else {
			log.Printf("invalid RESTORE env var: %v", err)
		}
	}
}

func NewClientConfig() *ClientConfig {
	cfg := &ClientConfig{}
	flag.StringVar(&cfg.ServerAddr, "a", "http://localhost:8080", "HTTP server address (must include http(s)://)")
	flag.IntVar(&cfg.ReportInterval, "r", 10, "report interval")
	flag.IntVar(&cfg.PollInterval, "p", 2, "poll interval")
	flag.IntVar(&cfg.ClientTimeout, "t", 10, "client timeout")
	flag.Parse()

	ReadClientEnvironment(cfg)

	if !strings.HasPrefix(cfg.ServerAddr, "http://") && !strings.HasPrefix(cfg.ServerAddr, "https://") {
		cfg.ServerAddr = "http://" + cfg.ServerAddr
	}

	return cfg
}

func ReadClientEnvironment(cfg *ClientConfig) {
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
}
