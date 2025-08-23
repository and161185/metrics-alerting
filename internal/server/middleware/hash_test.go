package middleware

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
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

func gz(b []byte) *bytes.Buffer {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, _ = zw.Write(b)
	_ = zw.Close()
	return &buf
}

func TestVerifyHashMiddleware_SkipWhenNoKey(t *testing.T) {
	cfg := &config.ServerConfig{Key: ""}
	h := VerifyHashMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(`{}`)))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	require.Equal(t, 200, rr.Code)
}

func TestVerifyHashMiddleware_Ok(t *testing.T) {
	cfg := &config.ServerConfig{Key: "k"}
	body, _ := json.Marshal(map[string]int{"x": 1})
	gbuf := gz(body)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(gbuf.Bytes()))
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HashSHA256", utils.CalculateHash(gbuf.Bytes(), cfg.Key))

	h := VerifyHashMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// тело должно быть доступно после проверки
		var m map[string]int
		gr, _ := gzip.NewReader(r.Body)
		b, _ := io.ReadAll(gr)
		_ = gr.Close()
		_ = json.Unmarshal(b, &m)
		require.Equal(t, 1, m["x"])
		w.WriteHeader(200)
	}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	require.Equal(t, 200, rr.Code)
}

func TestVerifyHashMiddleware_Bad(t *testing.T) {
	cfg := &config.ServerConfig{Key: "k"}
	body := []byte(`{"x":1}`)
	gbuf := gz(body)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(gbuf.Bytes()))
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("HashSHA256", "bad")

	rr := httptest.NewRecorder()
	VerifyHashMiddleware(cfg)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).ServeHTTP(rr, req)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}
