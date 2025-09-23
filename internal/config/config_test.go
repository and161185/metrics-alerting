package config

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func setEnvAndRun(t *testing.T, env map[string]string, fn func()) {
	t.Helper()

	backup := map[string]string{}
	for k := range env {
		backup[k] = os.Getenv(k)
	}

	for k, v := range env {
		require.NoError(t, os.Setenv(k, v))
	}
	defer func() {
		for k := range env {
			_ = os.Unsetenv(k)
			if old, ok := backup[k]; ok {
				_ = os.Setenv(k, old)
			}
		}
	}()

	fn()
}

func withFreshFlagSet(t *testing.T, fn func()) {
	t.Helper()
	old := flag.CommandLine
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	defer func() { flag.CommandLine = old }()
	fn()
}

func TestReadServerEnvironment(t *testing.T) {
	env := map[string]string{
		"ADDRESS":           "127.0.0.1:9999",
		"STORE_INTERVAL":    "5",
		"FILE_STORAGE_PATH": "/tmp/testfile.json",
		"RESTORE":           "false",

		// gRPC
		"GRPC":          "true",
		"GRPC_ADDR":     ":9090",
		"GRPC_TLS":      "true",
		"GRPC_CERT":     "/s.crt",
		"GRPC_KEY":      "/s.key",
		"GRPC_MAX_RECV": "1048576",
		"GRPC_MAX_SEND": "2097152",
	}

	setEnvAndRun(t, env, func() {
		withFreshFlagSet(t, func() {
			cfg := &ServerConfig{}
			readServerEnvironment(cfg)

			require.Equal(t, "127.0.0.1:9999", cfg.Addr)
			require.Equal(t, 5, cfg.StoreInterval)
			require.Equal(t, "/tmp/testfile.json", cfg.FileStoragePath)
			require.False(t, cfg.Restore)

			// gRPC
			require.True(t, cfg.GRPCEnable)
			require.Equal(t, ":9090", cfg.GRPCAddr)
			require.True(t, cfg.GRPCTLS)
			require.Equal(t, "/s.crt", cfg.GRPCCert)
			require.Equal(t, "/s.key", cfg.GRPCKey)
			require.Equal(t, 1048576, cfg.GRPCMaxRecvMsg)
			require.Equal(t, 2097152, cfg.GRPCMaxSendMsg)
		})
	})
}

func TestReadClientEnvironment(t *testing.T) {
	env := map[string]string{
		"ADDRESS":         "127.0.0.1:9999",
		"REPORT_INTERVAL": "5",
		"POLL_INTERVAL":   "1",

		// gRPC
		"GRPC":              "true",
		"GRPC_ADDR":         "localhost:9090",
		"GRPC_TLS":          "false",
		"GRPC_DIAL_TIMEOUT": "7",
		"GRPC_CALL_TIMEOUT": "11",
	}

	setEnvAndRun(t, env, func() {
		withFreshFlagSet(t, func() {
			cfg := &ClientConfig{}
			readClientEnvironment(cfg)

			require.Equal(t, "127.0.0.1:9999", cfg.ServerAddr)
			require.Equal(t, 5, cfg.ReportInterval)
			require.Equal(t, 1, cfg.PollInterval)

			// gRPC
			require.True(t, cfg.GRPCEnable)
			require.Equal(t, "localhost:9090", cfg.GRPCAddr)
			require.False(t, cfg.GRPCTLS)
			require.Equal(t, 7, cfg.GRPCDialTimeout)
			require.Equal(t, 11, cfg.GRPCCallTimeout)
		})
	})
}

func TestReadServerEnvironment_AllAndInvalid(t *testing.T) {
	env := map[string]string{
		"ADDRESS":           "0.0.0.0:9090",
		"STORE_INTERVAL":    "bad", // invalid
		"FILE_STORAGE_PATH": "/tmp/x.json",
		"RESTORE":           "nope", // invalid
		"DATABASE_DSN":      "postgres://u:p@h/db",
		"KEY":               "secret",
	}
	setEnvAndRun(t, env, func() {
		withFreshFlagSet(t, func() {
			cfg := &ServerConfig{}
			readServerEnvironment(cfg)
			require.Equal(t, "0.0.0.0:9090", cfg.Addr)
			require.Equal(t, "/tmp/x.json", cfg.FileStoragePath)
			require.Equal(t, "postgres://u:p@h/db", cfg.DatabaseDsn)
			require.Equal(t, "secret", cfg.Key)
		})
	})
}

func TestReadClientEnvironment_All(t *testing.T) {
	env := map[string]string{
		"ADDRESS":         "srv:8081",
		"REPORT_INTERVAL": "7",
		"POLL_INTERVAL":   "3",
		"KEY":             "k",
	}
	setEnvAndRun(t, env, func() {
		withFreshFlagSet(t, func() {
			cfg := &ClientConfig{}
			readClientEnvironment(cfg)
			require.Equal(t, "srv:8081", cfg.ServerAddr)
			require.Equal(t, 7, cfg.ReportInterval)
			require.Equal(t, 3, cfg.PollInterval)
			require.Equal(t, "k", cfg.Key)
		})
	})
}

func TestNewClientConfig_AddsHTTPPrefix(t *testing.T) {
	env := map[string]string{"ADDRESS": "srv:9090"}
	setEnvAndRun(t, env, func() {
		withFreshFlagSet(t, func() {
			cfg := NewClientConfig()
			require.Equal(t, "http://srv:9090", cfg.ServerAddr)
		})
	})
}

func TestNewServerConfig_BuildsLoggerAndReadsEnv(t *testing.T) {
	env := map[string]string{
		"ADDRESS":           "127.0.0.1:7070",
		"FILE_STORAGE_PATH": "/tmp/s.json",
		"DATABASE_DSN":      "dsn",
		"KEY":               "s",
	}
	setEnvAndRun(t, env, func() {
		withFreshFlagSet(t, func() {
			cfg := NewServerConfig()
			require.NotNil(t, cfg.Logger)
			require.Equal(t, "127.0.0.1:7070", cfg.Addr)
			require.Equal(t, "/tmp/s.json", cfg.FileStoragePath)
			require.Equal(t, "dsn", cfg.DatabaseDsn)
			require.Equal(t, "s", cfg.Key)
		})
	})
}

func withArgs(args []string, fn func()) {
	old := os.Args
	os.Args = append([]string{}, args...)
	defer func() { os.Args = old }()
	fn()
}

func writeJSON(t *testing.T, dir, name string, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	p := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(p, b, 0o644))
	return p
}

// -------- SERVER --------

func TestServer_JSONLowPriority_FlagsWin(t *testing.T) {
	td := t.TempDir()
	cfgPath := writeJSON(t, td, "srv.json", map[string]any{
		"address":        "json:8080",
		"restore":        false,
		"store_interval": "1s",
		"store_file":     "/json.db",
		"database_dsn":   "json-dsn",
		"crypto_key":     "/json.key",
	})

	setEnvAndRun(t, nil, func() {
		withFreshFlagSet(t, func() {
			withArgs([]string{"cmd",
				"-a", "flag:9000",
				"-i", "13",
				"-f", "/flag.db",
				"-r=false",
				"-d", "flag-dsn",
				"-crypto-key", "/flag.key",
				"-c", cfgPath,
			}, func() {
				cfg := NewServerConfig()
				require.Equal(t, "flag:9000", cfg.Addr)
				require.Equal(t, 13, cfg.StoreInterval)
				require.Equal(t, "/flag.db", cfg.FileStoragePath)
				require.Equal(t, false, cfg.Restore)
				require.Equal(t, "flag-dsn", cfg.DatabaseDsn)
				require.Equal(t, "/flag.key", cfg.CryptoKeyPath)
			})
		})
	})
}

func TestServer_ENVHighest_OverridesFlagsAndJSON(t *testing.T) {
	td := t.TempDir()
	cfgPath := writeJSON(t, td, "srv.json", map[string]any{
		"address":        "json:8080",
		"store_interval": "2s",
	})

	env := map[string]string{
		"ADDRESS":        "env:7000",
		"STORE_INTERVAL": "7",
		"STORE_FILE":     "/env.db",
		"RESTORE":        "true",
		"DATABASE_DSN":   "env-dsn",
		"CRYPTO_KEY":     "/env.key",
	}
	setEnvAndRun(t, env, func() {
		withFreshFlagSet(t, func() {
			withArgs([]string{"cmd", "-a", "flag:9000", "-i", "3", "-f", "/flag.db", "-r=false", "-d", "flag", "-crypto-key", "/flag.key", "-c", cfgPath}, func() {
				cfg := NewServerConfig()
				require.Equal(t, "env:7000", cfg.Addr)
				require.Equal(t, 7, cfg.StoreInterval)
				require.Equal(t, "/env.db", cfg.FileStoragePath)
				require.Equal(t, true, cfg.Restore)
				require.Equal(t, "env-dsn", cfg.DatabaseDsn)
				require.Equal(t, "/env.key", cfg.CryptoKeyPath)
			})
		})
	})
}

func TestServer_ConfigPathFromENV_CONFIG(t *testing.T) {
	td := t.TempDir()
	cfgPath := writeJSON(t, td, "srv.json", map[string]any{
		"address":        "json-only:8080",
		"store_interval": "5s",
	})
	env := map[string]string{"CONFIG": cfgPath}
	setEnvAndRun(t, env, func() {
		withFreshFlagSet(t, func() {
			withArgs([]string{"cmd"}, func() {
				cfg := NewServerConfig()
				require.Equal(t, "json-only:8080", cfg.Addr)
				require.Equal(t, 5, cfg.StoreInterval)
			})
		})
	})
}

func TestServer_InvalidJSONDuration_Ignored(t *testing.T) {
	td := t.TempDir()
	cfgPath := writeJSON(t, td, "srv.json", map[string]any{
		"store_interval": "wtf",
	})
	setEnvAndRun(t, nil, func() {
		withFreshFlagSet(t, func() {
			withArgs([]string{"cmd", "-c", cfgPath}, func() {
				cfg := NewServerConfig()
				require.Equal(t, 300, cfg.StoreInterval) // дефолт остался
			})
		})
	})
}

// -------- CLIENT --------

func TestClient_JSONLowPriority_FlagsWin(t *testing.T) {
	td := t.TempDir()
	cfgPath := writeJSON(t, td, "agent.json", map[string]any{
		"address":         "json:8080",
		"report_interval": "1s",
		"poll_interval":   "1s",
		"crypto_key":      "/json.pub",
	})

	setEnvAndRun(t, nil, func() {
		withFreshFlagSet(t, func() {
			withArgs([]string{"cmd",
				"-a", "flag:9000",
				"-r", "11",
				"-p", "3",
				"-t", "9",
				"-l", "123",
				"-crypto-key", "/flag.pub",
				"-c", cfgPath,
			}, func() {
				cfg := NewClientConfig()
				require.Equal(t, "http://flag:9000", cfg.ServerAddr)
				require.Equal(t, 11, cfg.ReportInterval)
				require.Equal(t, 3, cfg.PollInterval)
				require.Equal(t, 9, cfg.ClientTimeout)
				require.Equal(t, 123, cfg.RateLimit)
				require.Equal(t, "/flag.pub", cfg.CryptoKeyPath)
			})
		})
	})
}

func TestServer_JSONLowPriority_FlagsWin_GRPC(t *testing.T) {
	td := t.TempDir()
	cfgPath := writeJSON(t, td, "srv.json", map[string]any{
		"grpc":          false,
		"grpc_addr":     "json:9091",
		"grpc_tls":      true,
		"grpc_cert":     "/json.crt",
		"grpc_key":      "/json.key",
		"grpc_max_recv": 1,
		"grpc_max_send": 2,
	})

	withFreshFlagSet(t, func() {
		withArgs([]string{"cmd",
			"-grpc=true",
			"-grpc-addr", "flag:9092",
			"-grpc-tls=false",
			"-grpc-cert", "/flag.crt",
			"-grpc-key", "/flag.key",
			"-grpc-max-recv", "3",
			"-grpc-max-send", "4",
			"-c", cfgPath,
		}, func() {
			cfg := NewServerConfig()
			require.True(t, cfg.GRPCEnable)
			require.Equal(t, "flag:9092", cfg.GRPCAddr)
			require.False(t, cfg.GRPCTLS)
			require.Equal(t, "/flag.crt", cfg.GRPCCert)
			require.Equal(t, "/flag.key", cfg.GRPCKey)
			require.Equal(t, 3, cfg.GRPCMaxRecvMsg)
			require.Equal(t, 4, cfg.GRPCMaxSendMsg)
		})
	})
}

func TestClient_JSONLowPriority_FlagsWin_GRPC(t *testing.T) {
	td := t.TempDir()
	cfgPath := writeJSON(t, td, "agent.json", map[string]any{
		"grpc":              false,
		"grpc_addr":         "json:9090",
		"grpc_tls":          true,
		"grpc_dial_timeout": 1,
		"grpc_call_timeout": 2,
	})

	withFreshFlagSet(t, func() {
		withArgs([]string{"cmd",
			"-grpc=true",
			"-grpc-addr", "flag:9099",
			"-grpc-tls=false",
			"-grpc-dial-timeout", "7",
			"-grpc-call-timeout", "8",
			"-c", cfgPath,
		}, func() {
			cfg := NewClientConfig()
			require.True(t, cfg.GRPCEnable)
			require.Equal(t, "flag:9099", cfg.GRPCAddr)
			require.False(t, cfg.GRPCTLS)
			require.Equal(t, 7, cfg.GRPCDialTimeout)
			require.Equal(t, 8, cfg.GRPCCallTimeout)
		})
	})
}

func TestServer_ENVHighest_OverridesFlagsAndJSON_GRPC(t *testing.T) {
	td := t.TempDir()
	cfgPath := writeJSON(t, td, "srv.json", map[string]any{
		"grpc":      false,
		"grpc_addr": "json:1",
		"grpc_tls":  false,
		"grpc_cert": "/json.crt",
		"grpc_key":  "/json.key",
	})

	env := map[string]string{
		"GRPC":          "true",
		"GRPC_ADDR":     "env:2",
		"GRPC_TLS":      "true",
		"GRPC_CERT":     "/env.crt",
		"GRPC_KEY":      "/env.key",
		"GRPC_MAX_RECV": "5",
		"GRPC_MAX_SEND": "6",
	}
	setEnvAndRun(t, env, func() {
		withFreshFlagSet(t, func() {
			withArgs([]string{"cmd",
				"-grpc=false",
				"-grpc-addr", "flag:3",
				"-grpc-tls=false",
				"-grpc-cert", "/flag.crt",
				"-grpc-key", "/flag.key",
				"-c", cfgPath,
			}, func() {
				cfg := NewServerConfig()
				require.True(t, cfg.GRPCEnable)
				require.Equal(t, "env:2", cfg.GRPCAddr)
				require.True(t, cfg.GRPCTLS)
				require.Equal(t, "/env.crt", cfg.GRPCCert)
				require.Equal(t, "/env.key", cfg.GRPCKey)
				require.Equal(t, 5, cfg.GRPCMaxRecvMsg)
				require.Equal(t, 6, cfg.GRPCMaxSendMsg)
			})
		})
	})
}

func TestClient_ENVHighest_GRPC(t *testing.T) {
	env := map[string]string{
		"GRPC":              "true",
		"GRPC_ADDR":         "env:9090",
		"GRPC_TLS":          "true",
		"GRPC_DIAL_TIMEOUT": "9",
		"GRPC_CALL_TIMEOUT": "12",
	}
	setEnvAndRun(t, env, func() {
		withFreshFlagSet(t, func() {
			withArgs([]string{"cmd",
				"-grpc=false",
				"-grpc-addr", "flag:1",
				"-grpc-tls=false",
				"-grpc-dial-timeout", "1",
				"-grpc-call-timeout", "1",
			}, func() {
				cfg := NewClientConfig()
				require.True(t, cfg.GRPCEnable)
				require.Equal(t, "env:9090", cfg.GRPCAddr)
				require.True(t, cfg.GRPCTLS)
				require.Equal(t, 9, cfg.GRPCDialTimeout)
				require.Equal(t, 12, cfg.GRPCCallTimeout)
			})
		})
	})
}

func TestDefaults_GRPC(t *testing.T) {

	withFreshFlagSet(t, func() {
		withArgs([]string{"cmd"}, func() {
			cfgS := NewServerConfig()
			require.Equal(t, ":9090", cfgS.GRPCAddr)
			require.False(t, cfgS.GRPCEnable)
		})
	})

	withFreshFlagSet(t, func() {
		withArgs([]string{"cmd"}, func() {
			cfgC := NewClientConfig()
			require.Equal(t, "localhost:9090", cfgC.GRPCAddr)
			require.False(t, cfgC.GRPCEnable)
			require.Equal(t, 5, cfgC.GRPCDialTimeout)
			require.Equal(t, 5, cfgC.GRPCCallTimeout)
		})
	})
}

func TestClient_ENVHighest(t *testing.T) {
	env := map[string]string{
		"ADDRESS":         "env:7777",
		"REPORT_INTERVAL": "17",
		"POLL_INTERVAL":   "9",
		"RATE_LIMIT":      "256",
		"CRYPTO_KEY":      "/env.pub",
	}
	setEnvAndRun(t, env, func() {
		withFreshFlagSet(t, func() {
			withArgs([]string{"cmd", "-a", "flag", "-r", "1", "-p", "1", "-l", "1", "-crypto-key", "/flag.pub"}, func() {
				cfg := NewClientConfig()
				require.Equal(t, "http://env:7777", cfg.ServerAddr)
				require.Equal(t, 17, cfg.ReportInterval)
				require.Equal(t, 9, cfg.PollInterval)
				require.Equal(t, 256, cfg.RateLimit)
				require.Equal(t, "/env.pub", cfg.CryptoKeyPath)
			})
		})
	})
}

func TestClient_ConfigPathFromENV_CONFIG(t *testing.T) {
	td := t.TempDir()
	cfgPath := writeJSON(t, td, "agent.json", map[string]any{
		"address":         "json:6060",
		"report_interval": "2s",
	})
	env := map[string]string{"CONFIG": cfgPath}
	setEnvAndRun(t, env, func() {
		withFreshFlagSet(t, func() {
			withArgs([]string{"cmd"}, func() {
				cfg := NewClientConfig()
				require.Equal(t, "http://json:6060", cfg.ServerAddr)
				require.Equal(t, 2, cfg.ReportInterval)
			})
		})
	})
}

func TestClient_AddsHTTPPrefix_OnlyWhenMissing(t *testing.T) {
	setEnvAndRun(t, map[string]string{"ADDRESS": "https://already"}, func() {
		withFreshFlagSet(t, func() {
			withArgs([]string{"cmd"}, func() {
				cfg := NewClientConfig()
				require.Equal(t, "https://already", cfg.ServerAddr)
			})
		})
	})

	setEnvAndRun(t, map[string]string{"ADDRESS": "srv:8080"}, func() {
		withFreshFlagSet(t, func() {
			withArgs([]string{"cmd"}, func() {
				cfg := NewClientConfig()
				require.Equal(t, "http://srv:8080", cfg.ServerAddr)
			})
		})
	})
}

func TestDurationParser_OK(t *testing.T) {
	sec, err := parseDurationSeconds("1500ms")
	require.NoError(t, err)
	require.Equal(t, int((1500*time.Millisecond)/time.Second), sec)
}
