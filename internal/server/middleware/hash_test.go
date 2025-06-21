package middleware

import (
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/and161185/metrics-alerting/internal/config"
	"github.com/and161185/metrics-alerting/internal/utils"
)

func gzipBody(data []byte) []byte {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, _ = zw.Write(data)
	_ = zw.Close()
	return buf.Bytes()
}

func TestVerifyHashMiddleware(t *testing.T) {
	key := "supersecret"
	cfg := &config.ServerConfig{Key: key}

	handler := VerifyHashMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("response"))
	}))

	t.Run("valid hash", func(t *testing.T) {
		raw := []byte(`{"id":"Alloc","type":"gauge","value":123}`)
		compressed := gzipBody(raw)
		hash := utils.CalculateHash(compressed, key)

		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(compressed))
		req.Header.Set("Content-Encoding", "gzip")
		req.Header.Set("HashSHA256", hash)
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}

		if rr.Header().Get("HashSHA256") == "" {
			t.Errorf("expected HashSHA256 in response")
		}
	})

	t.Run("invalid hash", func(t *testing.T) {
		raw := []byte(`{"id":"Alloc","type":"gauge","value":123}`)
		compressed := gzipBody(raw)

		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(compressed))
		req.Header.Set("Content-Encoding", "gzip")
		req.Header.Set("HashSHA256", "invalidhash")
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("no hash header", func(t *testing.T) {
		raw := []byte(`{"id":"Alloc","type":"gauge","value":123}`)
		compressed := gzipBody(raw)

		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(compressed))
		req.Header.Set("Content-Encoding", "gzip")
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200 without hash, got %d", rr.Code)
		}
	})
}
