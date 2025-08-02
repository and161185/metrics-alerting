package middleware

import (
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/and161185/metrics-alerting/internal/config"
	"github.com/and161185/metrics-alerting/internal/utils"
	"github.com/stretchr/testify/require"
)

func gzipBody(data []byte) []byte {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, _ = zw.Write(data)
	_ = zw.Close()
	return buf.Bytes()
}

func makeRequest(t *testing.T, body []byte, key, hash string) *httptest.ResponseRecorder {
	cfg := &config.ServerConfig{Key: key}

	handler := VerifyHashMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("response"))
	}))

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Content-Type", "application/json")
	if hash != "" {
		req.Header.Set("HashSHA256", hash)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

func TestVerifyHashMiddleware(t *testing.T) {
	raw := []byte(`{"id":"Alloc","type":"gauge","value":123}`)
	compressed := gzipBody(raw)
	key := "supersecret"

	t.Run("valid hash", func(t *testing.T) {
		hash := utils.CalculateHash(compressed, key)
		rr := makeRequest(t, compressed, key, hash)
		require.Equal(t, http.StatusOK, rr.Code)
		require.NotEmpty(t, rr.Header().Get("HashSHA256"))
	})

	t.Run("invalid hash", func(t *testing.T) {
		rr := makeRequest(t, compressed, key, "invalidhash")
		require.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("no hash header", func(t *testing.T) {
		rr := makeRequest(t, compressed, key, "")
		require.Equal(t, http.StatusOK, rr.Code)
	})
}
