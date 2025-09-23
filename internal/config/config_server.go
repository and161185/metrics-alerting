// Package config provides application configuration structures and helpers.
package config

import (
	"flag"
	"log"
	"os"
	"strconv"

	"go.uber.org/zap"
)

// ServerConfig holds the configuration settings for the server.
type ServerConfig struct {
	Addr            string // Server address
	Logger          *zap.SugaredLogger
	StoreInterval   int    // Interval for storing metrics to file (in seconds)
	FileStoragePath string // Path to the file for metric storage
	Restore         bool   // Whether to restore metrics from file on startup
	DatabaseDsn     string // Data Source Name for PostgreSQL
	Key             string // Key for hash verification
	CryptoKeyPath   string // Path to private key
	TrustedSubnet   string // CIDR, ex. "192.168.1.0/24"

	GRPCEnable     bool
	GRPCAddr       string
	GRPCTLS        bool
	GRPCCert       string
	GRPCKey        string
	GRPCMaxRecvMsg int // bytes, 0 = default
	GRPCMaxSendMsg int // bytes, 0 = default
}

// NewServerConfig creates and returns a new ServerConfig by parsing flags and environment variables.
func NewServerConfig() *ServerConfig {
	logCfg := zap.NewProductionConfig()
	logCfg.OutputPaths = []string{"stdout", "server.log"}
	logger := zap.Must(logCfg.Build())

	// 0) defaults
	cfg := &ServerConfig{
		Addr:            "localhost:8080",
		StoreInterval:   300,
		FileStoragePath: "./tmp/metrics-db.json",
		Restore:         true,

		GRPCAddr:   ":9090",
		GRPCEnable: false,
	}

	// 1) flags
	var fAddr strFlag
	fAddr.v = cfg.Addr
	var fStoreI intFlag
	fStoreI.v = cfg.StoreInterval
	var fFile strFlag
	fFile.v = cfg.FileStoragePath
	var fRestore boolFlag
	fRestore.v = cfg.Restore
	var fDSN strFlag
	var fKey strFlag
	var fCrypto strFlag
	var fConf strFlag // -c / -config
	var fTrustedSubnet strFlag

	var fGRPCEnable boolFlag
	var fGRPCAddr strFlag
	fGRPCAddr.v = cfg.GRPCAddr
	var fGRPCTLS boolFlag
	var fGRPCCert strFlag
	var fGRPCKey strFlag
	var fGRPCMaxRecv intFlag
	var fGRPCMaxSend intFlag

	flag.Var(&fAddr, "a", "HTTP server address")
	flag.Var(&fStoreI, "i", "store interval (seconds)")
	flag.Var(&fFile, "f", "path to metrics file")
	flag.Var(&fRestore, "r", "restore from file")
	flag.Var(&fDSN, "d", "DB connection string")
	flag.Var(&fKey, "k", "Hash key string")
	flag.Var(&fCrypto, "crypto-key", "Path to private key")
	flag.Var(&fConf, "c", "Path to JSON config file")
	flag.Var(&fConf, "config", "Path to JSON config file (alias)")
	flag.Var(&fTrustedSubnet, "t", "trusted subnet")

	flag.Var(&fGRPCEnable, "grpc", "enable gRPC server")
	flag.Var(&fGRPCAddr, "grpc-addr", "gRPC listen address")
	flag.Var(&fGRPCTLS, "grpc-tls", "enable TLS for gRPC")
	flag.Var(&fGRPCCert, "grpc-cert", "gRPC TLS cert path")
	flag.Var(&fGRPCKey, "grpc-key", "gRPC TLS key path")
	flag.Var(&fGRPCMaxRecv, "grpc-max-recv", "gRPC max recv msg bytes (0=default)")
	flag.Var(&fGRPCMaxSend, "grpc-max-send", "gRPC max send msg bytes (0=default)")
	flag.Parse()

	cfg.Addr = fAddr.v
	cfg.StoreInterval = fStoreI.v
	cfg.FileStoragePath = fFile.v
	cfg.Restore = fRestore.v
	cfg.DatabaseDsn = fDSN.v
	cfg.Key = fKey.v
	cfg.CryptoKeyPath = fCrypto.v
	cfg.TrustedSubnet = fTrustedSubnet.v

	cfg.GRPCEnable = fGRPCEnable.v
	cfg.GRPCAddr = fGRPCAddr.v
	cfg.GRPCTLS = fGRPCTLS.v
	cfg.GRPCCert = fGRPCCert.v
	cfg.GRPCKey = fGRPCKey.v
	cfg.GRPCMaxRecvMsg = fGRPCMaxRecv.v
	cfg.GRPCMaxSendMsg = fGRPCMaxSend.v

	// 3) JSON (lowest priority)
	if fConf.v == "" {
		if v := os.Getenv("CONFIG"); v != "" {
			fConf.v = v
		}
	}

	if fConf.v != "" {
		if js, err := loadServerJSON(fConf.v); err == nil {
			if js.Address != nil && !fAddr.set {
				cfg.Addr = *js.Address
			}
			if js.Restore != nil && !fRestore.set {
				cfg.Restore = *js.Restore
			}
			if js.StoreInterval != nil && !fStoreI.set {
				if sec, err := parseDurationSeconds(*js.StoreInterval); err == nil {
					cfg.StoreInterval = sec
				}
			}
			if js.StoreFile != nil && !fFile.set {
				cfg.FileStoragePath = *js.StoreFile
			}
			if js.DatabaseDSN != nil && !fDSN.set {
				cfg.DatabaseDsn = *js.DatabaseDSN
			}
			if js.CryptoKey != nil && !fCrypto.set {
				cfg.CryptoKeyPath = *js.CryptoKey
			}
			if js.TrustedSubnet != nil && !fTrustedSubnet.set {
				cfg.TrustedSubnet = *js.TrustedSubnet
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
			if js.GRPCCert != nil && !fGRPCCert.set {
				cfg.GRPCCert = *js.GRPCCert
			}
			if js.GRPCKey != nil && !fGRPCKey.set {
				cfg.GRPCKey = *js.GRPCKey
			}
			if js.GRPCMaxRecvMsg != nil && !fGRPCMaxRecv.set {
				cfg.GRPCMaxRecvMsg = *js.GRPCMaxRecvMsg
			}
			if js.GRPCMaxSendMsg != nil && !fGRPCMaxSend.set {
				cfg.GRPCMaxSendMsg = *js.GRPCMaxSendMsg
			}
		}
	}

	readServerEnvironment(cfg)

	cfg.Logger = logger.Sugar()
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
	} else if fsp := os.Getenv("STORE_FILE"); fsp != "" {
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

	if cryptokey := os.Getenv("CRYPTO_KEY"); cryptokey != "" {
		cfg.CryptoKeyPath = cryptokey
	}

	if trustedSubnet := os.Getenv("TRUSTED_SUBNET"); trustedSubnet != "" {
		cfg.TrustedSubnet = trustedSubnet
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
	if v := os.Getenv("GRPC_CERT"); v != "" {
		cfg.GRPCCert = v
	}
	if v := os.Getenv("GRPC_KEY"); v != "" {
		cfg.GRPCKey = v
	}
	if v := os.Getenv("GRPC_MAX_RECV"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.GRPCMaxRecvMsg = n
		}
	}
	if v := os.Getenv("GRPC_MAX_SEND"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.GRPCMaxSendMsg = n
		}
	}
}
