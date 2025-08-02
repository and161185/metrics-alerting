package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/and161185/metrics-alerting/internal/config"
	"github.com/and161185/metrics-alerting/internal/utils"
)

func BenchmarkVerifyHashMiddleware_Valid(b *testing.B) {
	raw := []byte(`{"id":"Alloc","type":"gauge","value":123}`)
	compressed := gzipBody(raw)
	key := "supersecret"
	hash := utils.CalculateHash(compressed, key)

	cfg := &config.ServerConfig{Key: key}
	handler := VerifyHashMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(compressed))
		req.Header.Set("Content-Encoding", "gzip")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("HashSHA256", hash)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}
}
