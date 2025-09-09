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
	}

	setEnvAndRun(t, env, func() {
		withFreshFlagSet(t, func() {
			cfg := &ServerConfig{}
			readServerEnvironment(cfg)

			require.Equal(t, "127.0.0.1:9999", cfg.Addr)
			require.Equal(t, 5, cfg.StoreInterval)
			require.Equal(t, "/tmp/testfile.json", cfg.FileStoragePath)
			require.False(t, cfg.Restore)
		})
	})
}

func TestReadClientEnvironment(t *testing.T) {
	env := map[string]string{
		"ADDRESS":         "127.0.0.1:9999",
		"REPORT_INTERVAL": "5",
		"POLL_INTERVAL":   "1",
	}

	setEnvAndRun(t, env, func() {
		withFreshFlagSet(t, func() {
			cfg := &ClientConfig{}
			readClientEnvironment(cfg)

			require.Equal(t, "127.0.0.1:9999", cfg.ServerAddr)
			require.Equal(t, 5, cfg.ReportInterval)
			require.Equal(t, 1, cfg.PollInterval)
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
