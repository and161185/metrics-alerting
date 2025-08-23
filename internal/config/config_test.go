package config

import (
	"flag"
	"os"
	"testing"

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
