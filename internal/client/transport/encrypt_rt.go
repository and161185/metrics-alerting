package transport

import (
	"bytes"
	"crypto/rsa"
	"io"
	"net/http"

	"github.com/and161185/metrics-alerting/internal/crypto"
)

type EncryptRoundTripper struct {
	Base   http.RoundTripper
	PubKey *rsa.PublicKey // импортируй "crypto/rsa" тут, это норм
}

func (e *EncryptRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	rt := e.Base
	if rt == nil {
		rt = http.DefaultTransport
	}
	if e.PubKey == nil || req.Body == nil {
		return rt.RoundTrip(req)
	}

	plain, err := io.ReadAll(req.Body) // здесь уже gzipped JSON
	if err != nil {
		return nil, err
	}
	_ = req.Body.Close()

	envBytes, err := crypto.EncryptEnvelope(e.PubKey, plain)
	if err != nil {
		return nil, err
	}

	req.Body = io.NopCloser(bytes.NewReader(envBytes))
	req.ContentLength = int64(len(envBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Encrypted", "v1")
	req.Header.Del("Content-Encoding") // gzip внутри

	return rt.RoundTrip(req)
}
