package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

func DecompressMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Encoding") != "gzip" {
			next.ServeHTTP(w, r)
			return
		}

		gr, err := gzip.NewReader(r.Body)
		if err != nil {
			http.Error(w, "bad gzip", http.StatusBadRequest)
			return
		}
		defer gr.Close()
		r.Body = io.NopCloser(gr)
		next.ServeHTTP(w, r)
	})
}

func CompressMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		gzw := &gzipResponseWriter{ResponseWriter: w}
		next.ServeHTTP(gzw, r)
	})
}

type gzipResponseWriter struct {
	http.ResponseWriter
	buf     *bytes.Buffer
	written bool
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	if !w.written {
		ct := w.Header().Get("Content-Type")
		if strings.Contains(ct, "application/json") || strings.Contains(ct, "text/html") {
			w.Header().Set("Content-Encoding", "gzip")
			var compressed bytes.Buffer
			gz := gzip.NewWriter(&compressed)
			if _, err := gz.Write(b); err != nil {
				return 0, err
			}
			gz.Close()
			w.written = true
			return w.ResponseWriter.Write(compressed.Bytes())
		}
	}
	w.written = true
	return w.ResponseWriter.Write(b)
}

func LogMiddleware(logger *zap.SugaredLogger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				logger.Errorf("failed to read request body: %v", err)
			}
			r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

			lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(lrw, r)

			duration := time.Since(start)

			logger.Infof(
				"method=%s uri=%s status=%d size=%d duration=%s body=%s",
				r.Method, r.RequestURI, lrw.statusCode, lrw.size, duration, string(bodyBytes),
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
