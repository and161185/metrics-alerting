package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/and161185/metrics-alerting/model"
	"github.com/and161185/metrics-alerting/storage"
)

func TestSendToServer(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/update/") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s, want POST", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	st := storage.NewMemStorage()
	st.Save(model.Metric{ID: "TestMetric", Type: model.Gauge, Value: 42.0})

	err := SendToServer(st, ts.URL)
	if err != nil {
		t.Errorf("SendToServer failed: %v", err)
	}
}
