package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func setEnvAndRun(t *testing.T, env map[string]string, fn func()) {
	t.Helper()

	// сохранить старые переменные
	backup := map[string]string{}
	for k := range env {
		backup[k] = os.Getenv(k)
	}

	// установить новые
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

func TestReadServerEnvironment(t *testing.T) {
	env := map[string]string{
		"ADDRESS":           "127.0.0.1:9999",
		"STORE_INTERVAL":    "5",
		"FILE_STORAGE_PATH": "/tmp/testfile.json",
		"RESTORE":           "false",
	}

	setEnvAndRun(t, env, func() {
		cfg := &ServerConfig{}
		readServerEnvironment(cfg)

		require.Equal(t, "127.0.0.1:9999", cfg.Addr)
		require.Equal(t, 5, cfg.StoreInterval)
		require.Equal(t, "/tmp/testfile.json", cfg.FileStoragePath)
		require.False(t, cfg.Restore)
	})
}

func TestReadClientEnvironment(t *testing.T) {
	env := map[string]string{
		"ADDRESS":         "127.0.0.1:9999",
		"REPORT_INTERVAL": "5",
		"POLL_INTERVAL":   "1",
	}

	setEnvAndRun(t, env, func() {
		cfg := &ClientConfig{}
		readClientEnvironment(cfg)

		require.Equal(t, "127.0.0.1:9999", cfg.ServerAddr)
		require.Equal(t, 5, cfg.ReportInterval)
		require.Equal(t, 1, cfg.PollInterval)
	})
}
