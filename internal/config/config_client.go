// Package config provides application configuration structures and helpers.
package config

import (
	"flag"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
)

// ClientConfig holds the configuration settings for the agent.
type ClientConfig struct {
	ServerAddr     string // Server address
	ReportInterval int    // Interval for sending metrics (in seconds)
	PollInterval   int    // Interval for collecting metrics (in seconds)
	ClientTimeout  int    // HTTP client timeout (in seconds)
	Key            string // Key for hash generation
	RateLimit      int    // Limit on simultaneous outgoing requests
	CryptoKeyPath  string // Path to public key

	GRPCEnable      bool
	GRPCAddr        string
	GRPCTLS         bool
	GRPCDialTimeout int
	GRPCCallTimeout int
}

// NewClientConfig creates and returns a new ClientConfig by parsing flags and environment variables.
func NewClientConfig() *ClientConfig {
	cfg := &ClientConfig{
		ServerAddr:     "http://localhost:8080",
		ReportInterval: 10,
		PollInterval:   2,
		ClientTimeout:  10,
		RateLimit:      runtime.NumCPU(),

		GRPCEnable:      false,
		GRPCAddr:        "localhost:9090",
		GRPCTLS:         false,
		GRPCDialTimeout: 5,
		GRPCCallTimeout: 5,
	}

	var fAddr, fKey, fCrypto, fConf strFlag
	var fRep, fPoll, fTO, fRate intFlag
	var fGRPCEnable boolFlag
	var fGRPCAddr strFlag
	fGRPCAddr.v = cfg.GRPCAddr
	var fGRPCTLS boolFlag
	var fGRPCDialTO intFlag
	fGRPCDialTO.v = cfg.GRPCDialTimeout
	var fGRPCCallTO intFlag
	fGRPCCallTO.v = cfg.GRPCCallTimeout

	flag.Var(&fAddr, "a", "HTTP server address (must include http(s)://)")
	flag.Var(&fRep, "r", "report interval (seconds)")
	flag.Var(&fPoll, "p", "poll interval (seconds)")
	flag.Var(&fTO, "t", "client timeout (seconds)")
	flag.Var(&fKey, "k", "Hash key string")
	flag.Var(&fRate, "l", "rate limit")
	flag.Var(&fCrypto, "crypto-key", "Path to public key")
	flag.Var(&fConf, "c", "Path to JSON config file")
	flag.Var(&fConf, "config", "Path to JSON config file (alias)")

	flag.Var(&fGRPCEnable, "grpc", "enable gRPC client")
	flag.Var(&fGRPCAddr, "grpc-addr", "gRPC server address (host:port)")
	flag.Var(&fGRPCTLS, "grpc-tls", "use TLS for gRPC")
	flag.Var(&fGRPCDialTO, "grpc-dial-timeout", "gRPC dial timeout (seconds)")
	flag.Var(&fGRPCCallTO, "grpc-call-timeout", "gRPC call timeout (seconds)")
	flag.Parse()

	if fAddr.set {
		cfg.ServerAddr = fAddr.v
	}
	if fRep.set {
		cfg.ReportInterval = fRep.v
	}
	if fPoll.set {
		cfg.PollInterval = fPoll.v
	}
	if fTO.set {
		cfg.ClientTimeout = fTO.v
	}
	if fKey.set {
		cfg.Key = fKey.v
	}
	if fRate.set {
		cfg.RateLimit = fRate.v
	}
	if fCrypto.set {
		cfg.CryptoKeyPath = fCrypto.v
	}

	if fGRPCEnable.set {
		cfg.GRPCEnable = fGRPCEnable.v
	}
	if fGRPCAddr.set {
		cfg.GRPCAddr = fGRPCAddr.v
	}
	if fGRPCTLS.set {
		cfg.GRPCTLS = fGRPCTLS.v
	}
	if fGRPCDialTO.set {
		cfg.GRPCDialTimeout = fGRPCDialTO.v
	}
	if fGRPCCallTO.set {
		cfg.GRPCCallTimeout = fGRPCCallTO.v
	}

	if fConf.v == "" {
		if v := os.Getenv("CONFIG"); v != "" {
			fConf.v = v
		}
	}
	if fConf.v != "" {
		if js, err := loadClientJSON(fConf.v); err == nil {
			if js.Address != nil && !fAddr.set {
				cfg.ServerAddr = *js.Address
			}
			if js.ReportInterval != nil && !fRep.set {
				if sec, err := parseDurationSeconds(*js.ReportInterval); err == nil {
					cfg.ReportInterval = sec
				}
			}
			if js.PollInterval != nil && !fPoll.set {
				if sec, err := parseDurationSeconds(*js.PollInterval); err == nil {
					cfg.PollInterval = sec
				}
			}
			if js.CryptoKey != nil && !fCrypto.set {
				cfg.CryptoKeyPath = *js.CryptoKey
			}

			if js.GRPC != nil && !fGRPCEnable.set {
				cfg.GRPCEnable = *js.GRPC
			}
			if js.GRPCAddr != nil && !fGRPCAddr.set {
				cfg.GRPCAddr = *js.GRPCAddr
			}
			if js.GRPCTLS != nil && !fGRPCTLS.set {
				cfg.GRPCTLS = *js.GRPCTLS
			}
			if js.GRPCDialTimeout != nil && !fGRPCDialTO.set {
				cfg.GRPCDialTimeout = *js.GRPCDialTimeout
			}
			if js.GRPCCallTimeout != nil && !fGRPCCallTO.set {
				cfg.GRPCCallTimeout = *js.GRPCCallTimeout
			}
		}
	}

	readClientEnvironment(cfg)

	// normalize address
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

	if rateLimit := os.Getenv("RATE_LIMIT"); rateLimit != "" {
		if i, err := strconv.Atoi(rateLimit); err == nil {
			cfg.RateLimit = i
		}
	}

	if key := os.Getenv("KEY"); key != "" {
		cfg.Key = key
	}

	if cryptokey := os.Getenv("CRYPTO_KEY"); cryptokey != "" {
		cfg.CryptoKeyPath = cryptokey
	}

	if v := os.Getenv("GRPC"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.GRPCEnable = b
		}
	}
	if v := os.Getenv("GRPC_ADDR"); v != "" {
		cfg.GRPCAddr = v
	}
	if v := os.Getenv("GRPC_TLS"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.GRPCTLS = b
		}
	}
	if v := os.Getenv("GRPC_DIAL_TIMEOUT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.GRPCDialTimeout = n
		}
	}
	if v := os.Getenv("GRPC_CALL_TIMEOUT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.GRPCCallTimeout = n
		}
	}
}
