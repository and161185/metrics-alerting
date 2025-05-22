package config

import (
	"os"
	"testing"
)

func TestReadServerEnvironment(t *testing.T) {
	// очистим и установим переменные окружения
	os.Setenv("ADDRESS", "127.0.0.1:9999")
	os.Setenv("STORE_INTERVAL", "5")
	os.Setenv("FILE_STORAGE_PATH", "/tmp/testfile.json")
	os.Setenv("RESTORE", "false")
	defer os.Clearenv()

	// сбрасываем стандартный флаг-парсер
	cfg := &ServerConfig{}
	ReadServerEnvironment(cfg)

	if cfg.Addr != "127.0.0.1:9999" {
		t.Errorf("expected addr from env, got %s", cfg.Addr)
	}
	if cfg.StoreInterval != 5 {
		t.Errorf("expected interval=5, got %d", cfg.StoreInterval)
	}
	if cfg.FileStoragePath != "/tmp/testfile.json" {
		t.Errorf("wrong path: %s", cfg.FileStoragePath)
	}
	if cfg.Restore != false {
		t.Errorf("expected restore=false, got %v", cfg.Restore)
	}
}

func TestReadClientEnvironment(t *testing.T) {
	// очистим и установим переменные окружения
	os.Setenv("ADDRESS", "127.0.0.1:9999")
	os.Setenv("REPORT_INTERVAL", "5")
	os.Setenv("POLL_INTERVAL", "1")
	defer os.Clearenv()

	// сбрасываем стандартный флаг-парсер
	cfg := &ClientConfig{}
	ReadClientEnvironment(cfg)

	if cfg.ServerAddr != "127.0.0.1:9999" {
		t.Errorf("expected addr from env, got %s", cfg.ServerAddr)
	}
	if cfg.ReportInterval != 5 {
		t.Errorf("expected interval=5, got %d", cfg.ReportInterval)
	}
	if cfg.PollInterval != 1 {
		t.Errorf("wrong path: %d", cfg.PollInterval)
	}
}
