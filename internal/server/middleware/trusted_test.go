package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestTrustedCIDR_Empty_AllowsAll(t *testing.T) {
	h := TrustedCIDR("")(okHandler())

	req := httptest.NewRequest(http.MethodPost, "/any", nil)
	// без X-Real-IP
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
}

func TestTrustedCIDR_Inside_OK(t *testing.T) {
	h := TrustedCIDR("10.0.0.0/24")(okHandler())

	req := httptest.NewRequest(http.MethodPost, "/any", nil)
	req.Header.Set("X-Real-IP", "10.0.0.42")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
}

func TestTrustedCIDR_Outside_Forbidden(t *testing.T) {
	h := TrustedCIDR("10.0.0.0/24")(okHandler())

	req := httptest.NewRequest(http.MethodPost, "/any", nil)
	req.Header.Set("X-Real-IP", "192.168.1.10")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d", rr.Code)
	}
}

func TestTrustedCIDR_InvalidCIDR_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on invalid CIDR")
		}
	}()
	_ = TrustedCIDR("wtf") // должен паникнуть на старте
}
