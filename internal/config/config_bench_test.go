package config

import (
	"os"
	"testing"
)

func BenchmarkReadServerEnvironment(b *testing.B) {
	_ = os.Setenv("ADDRESS", "127.0.0.1:9999")
	_ = os.Setenv("STORE_INTERVAL", "5")
	_ = os.Setenv("FILE_STORAGE_PATH", "/tmp/testfile.json")
	_ = os.Setenv("RESTORE", "false")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfg := &ServerConfig{}
		readServerEnvironment(cfg)
	}
}

func BenchmarkReadClientEnvironment(b *testing.B) {
	_ = os.Setenv("ADDRESS", "127.0.0.1:9999")
	_ = os.Setenv("REPORT_INTERVAL", "5")
	_ = os.Setenv("POLL_INTERVAL", "1")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfg := &ClientConfig{}
		readClientEnvironment(cfg)
	}
}
