package transport

import (
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/and161185/metrics-alerting/internal/server/middleware"
	"github.com/go-chi/chi/v5"
)

func genKey(t *testing.T) (*rsa.PrivateKey, *rsa.PublicKey) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("gen key: %v", err)
	}
	return priv, &priv.PublicKey
}

func gz(b []byte) []byte {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, _ = zw.Write(b)
	_ = zw.Close()
	return buf.Bytes()
}

func TestEncryptRoundTripper_EndToEnd(t *testing.T) {

	priv, pub := genKey(t)

	r := chi.NewRouter()
	r.Use(middleware.DecryptMiddleware(priv, true))
	r.Post("/update", func(w http.ResponseWriter, r *http.Request) {

		gzBody, _ := io.ReadAll(r.Body)
		gr, _ := gzip.NewReader(bytes.NewReader(gzBody))
		defer gr.Close()
		data, _ := io.ReadAll(gr)

		var got map[string]any
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("bad json: %v", err)
		}
		if got["ok"] != true {
			t.Fatalf("unexpected: %v", got)
		}
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(r)
	defer srv.Close()

	cl := &http.Client{
		Transport: &EncryptRoundTripper{Base: http.DefaultTransport, PubKey: pub},
	}

	payload := gz([]byte(`{"ok":true}`))
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/update", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")

	resp, err := cl.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestEncryptRoundTripper_NoKey_PassThrough(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Encrypted") != "" {
			t.Fatalf("should not set X-Encrypted without key")
		}

		body, _ := io.ReadAll(r.Body)
		if len(body) == 0 {
			t.Fatal("empty body")
		}
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(h)
	defer srv.Close()

	cl := &http.Client{Transport: &EncryptRoundTripper{Base: http.DefaultTransport, PubKey: nil}}

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, _ = zw.Write([]byte(`{"x":1}`))
	_ = zw.Close()

	req, _ := http.NewRequest(http.MethodPost, srv.URL, bytes.NewReader(buf.Bytes()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")

	resp, err := cl.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}
