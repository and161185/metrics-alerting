package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestLogMiddleware_TextBody(t *testing.T) {
	core, obs := observer.New(zap.InfoLevel)
	logger := zap.New(core).Sugar()

	h := LogMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString("hello"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code)
	require.Equal(t, "ok", rr.Body.String())
	require.NotEmpty(t, obs.All())
}

func TestLogMiddleware_BinaryBodyAndError(t *testing.T) {
	core, obs := observer.New(zap.InfoLevel)
	logger := zap.New(core).Sugar()

	h := LogMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("resp"))
	}))

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte{0xff, 0x01, 0x02}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	require.Equal(t, "resp", rr.Body.String())
	require.NotEmpty(t, obs.All())
}

func TestIsProbablyText(t *testing.T) {
	require.True(t, isProbablyText([]byte("abc")))
	require.False(t, isProbablyText([]byte{0xff}))
	require.False(t, isProbablyText([]byte{0x00}))
}
