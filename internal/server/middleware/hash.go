package middleware

import (
	"bytes"
	"io"
	"net/http"

	"github.com/and161185/metrics-alerting/internal/config"
	"github.com/and161185/metrics-alerting/internal/utils"
)

func VerifyHashMiddleware(cfg *config.ServerConfig) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.Key == "" {
				next.ServeHTTP(w, r)
				return
			}

			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "bad body", http.StatusBadRequest)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

			headerHashSHA256 := r.Header.Get("HashSHA256")
			if headerHashSHA256 != "" && headerHashSHA256 != utils.CalculateHash(bodyBytes, cfg.Key) {
				http.Error(w, "invalid hash", http.StatusBadRequest)
				return
			}

			capture := &responseCapture{ResponseWriter: w}
			next.ServeHTTP(capture, r)

			if cfg.Key != "" {
				w.Header().Set("HashSHA256", utils.CalculateHash(capture.body.Bytes(), cfg.Key))
			}
		})
	}
}

type responseCapture struct {
	http.ResponseWriter
	body bytes.Buffer
}

func (r *responseCapture) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}
