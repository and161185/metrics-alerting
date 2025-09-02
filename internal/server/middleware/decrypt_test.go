package middleware

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

	"github.com/and161185/metrics-alerting/internal/crypto"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

func genKey(t *testing.T) (*rsa.PrivateKey, *rsa.PublicKey) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("gen key: %v", err)
	}
	return priv, &priv.PublicKey
}

func ungz(b []byte) []byte {
	r, _ := gzip.NewReader(bytes.NewReader(b))
	defer r.Close()
	out, _ := io.ReadAll(r)
	return out
}

func gzb(b []byte) []byte {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, _ = zw.Write(b)
	_ = zw.Close()
	return buf.Bytes()
}

func TestDecryptMiddleware_HappyPath(t *testing.T) {
	priv, pub := genKey(t)

	r := chi.NewRouter()
	r.Use(chimw.StripSlashes)
	r.Use(DecryptMiddleware(priv, true))
	r.Post("/update", func(w http.ResponseWriter, r *http.Request) {

		gzBody, _ := io.ReadAll(r.Body)
		body := ungz(gzBody)

		var got map[string]any
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("bad json after decrypt: %v", err)
		}
		if got["hello"] != "world" {
			t.Fatalf("unexpected payload: %v", got)
		}
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(r)
	defer srv.Close()

	plain := gzb([]byte(`{"hello":"world","n":123}`))

	env, err := crypto.EncryptEnvelope(pub, plain)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	got, err := crypto.DecryptEnvelope(priv, env)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	if !bytes.Equal(got, plain) {
		t.Fatalf("mismatch: got %dB, want %dB", len(got), len(plain))
	}

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/update", bytes.NewReader(env))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Encrypted", "v1")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestDecryptMiddleware_Require_RejectsPlain(t *testing.T) {
	priv, _ := genKey(t)

	r := chi.NewRouter()
	r.Use(DecryptMiddleware(priv, true))
	r.Post("/update", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	srv := httptest.NewServer(r)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/update", bytes.NewReader([]byte("plain")))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", resp.StatusCode)
	}
}

func TestDecryptMiddleware_BadEnvelope(t *testing.T) {
	priv, _ := genKey(t)

	r := chi.NewRouter()
	r.Use(DecryptMiddleware(priv, true))
	r.Post("/", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	srv := httptest.NewServer(r)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/", bytes.NewReader([]byte(`{"v":1,"alg":"X","enc":"Y"}`)))
	req.Header.Set("X-Encrypted", "v1")
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", resp.StatusCode)
	}
}
