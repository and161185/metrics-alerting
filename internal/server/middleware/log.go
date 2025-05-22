package middleware

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

func LogMiddleware(logger *zap.SugaredLogger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				logger.Errorf("failed to read request body: %v", err)
			}
			r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			headers := fmt.Sprintf("%v", r.Header)

			loggerBody := "<skipped>"
			if len(bodyBytes) > 0 && isProbablyText(bodyBytes) {
				loggerBody = string(bodyBytes)
			}

			lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(lrw, r)

			duration := time.Since(start)

			logger.Infof(
				"method=%s uri=%s status=%d size=%d duration=%s body=%s headers=%s",
				r.Method, r.RequestURI, lrw.statusCode, lrw.size, duration, loggerBody, headers,
			)
		})
	}
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
	n, err := lrw.ResponseWriter.Write(b)
	lrw.size += n
	return n, err
}

func isProbablyText(b []byte) bool {
	for _, c := range b {
		if c == 0 || c > 127 {
			return false
		}
	}
	return true
}
