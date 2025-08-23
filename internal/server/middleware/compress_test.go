package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type errBody struct{}

func (e errBody) Read(_ []byte) (int, error) { return 0, io.ErrUnexpectedEOF }
func (e errBody) Close() error               { return nil }

func TestDecompressMiddleware_PassThrough_NoGzip(t *testing.T) {
	h := DecompressMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		require.Equal(t, "plain", string(b))
		w.WriteHeader(200)
	}))
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("plain"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	require.Equal(t, 200, rr.Code)
}

func TestDecompressMiddleware_ValidGzip(t *testing.T) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, _ = zw.Write([]byte("hello"))
	_ = zw.Close()

	h := DecompressMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		require.Equal(t, "hello", string(data))
		w.WriteHeader(200)
	}))
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(buf.Bytes()))
	req.Header.Set("Content-Encoding", "gzip")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	require.Equal(t, 200, rr.Code)
}

func TestDecompressMiddleware_InvalidGzip_StillPassBody(t *testing.T) {
	raw := []byte("not-gzip")
	h := DecompressMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		require.Equal(t, raw, data)
		w.WriteHeader(200)
	}))
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(raw))
	req.Header.Set("Content-Encoding", "gzip")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	require.Equal(t, 200, rr.Code)
}

func TestDecompressMiddleware_BadBodyRead_Returns400(t *testing.T) {
	h := DecompressMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called on body read error")
	}))
	req := httptest.NewRequest(http.MethodPost, "/", errBody{})
	req.Header.Set("Content-Encoding", "gzip")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCompressMiddleware_SkipWhenNoAcceptGzip(t *testing.T) {
	h := CompressMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("plain"))
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	require.Equal(t, "", rr.Header().Get("Content-Encoding"))
	require.Equal(t, "plain", rr.Body.String())
}

func TestCompressMiddleware_GzipResponse(t *testing.T) {
	h := CompressMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("he"))
		_, _ = w.Write([]byte("llo"))
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	require.Equal(t, "gzip", rr.Header().Get("Content-Encoding"))

	gr, err := gzip.NewReader(bytes.NewReader(rr.Body.Bytes()))
	require.NoError(t, err)
	defer gr.Close()
	decompressed, _ := io.ReadAll(gr)
	require.Equal(t, "hello", string(decompressed))
}

func TestGzipResponseWriter_Close_NoWrites(t *testing.T) {
	var rw httptest.ResponseRecorder
	grw := &gzipResponseWriter{ResponseWriter: &rw}
	require.NoError(t, grw.Close())
	require.Equal(t, "", rw.Header().Get("Content-Encoding"))
}
