package middleware

import (
	"bytes"
	"crypto/rsa"
	"io"
	"net/http"

	"github.com/and161185/metrics-alerting/internal/crypto"
)

const encryptedHeader = "X-Encrypted"

func DecryptMiddleware(priv *rsa.PrivateKey, require bool) func(http.Handler) http.Handler {
	if priv == nil {
		return func(next http.Handler) http.Handler { return next }
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ver := r.Header.Get(encryptedHeader)
			if ver == "" {
				if require {
					http.Error(w, "encryption required", http.StatusBadRequest)
					return
				}
				next.ServeHTTP(w, r)
				return
			}
			if ver != "v1" {
				http.Error(w, "unsupported encryption version", http.StatusBadRequest)
				return
			}

			envBytes, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "read body failed", http.StatusBadRequest)
				return
			}
			_ = r.Body.Close()

			plain, err := crypto.DecryptEnvelope(priv, envBytes)
			if err != nil {
				http.Error(w, "decrypt failed", http.StatusBadRequest)
				return
			}

			// plain — это gzipped JSON (если агент так отправляет).
			r.Body = io.NopCloser(bytes.NewReader(plain))
			r.ContentLength = int64(len(plain))
			r.Header.Set("Content-Encoding", "gzip")
			r.Header.Del(encryptedHeader)

			next.ServeHTTP(w, r)
		})
	}
}
