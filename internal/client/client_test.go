package client

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/and161185/metrics-alerting/internal/config"
	"github.com/and161185/metrics-alerting/internal/utils"
	"github.com/and161185/metrics-alerting/model"
	"github.com/and161185/metrics-alerting/storage/inmemory"
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

	st := inmemory.NewMemStorage()
	m := model.Metric{ID: "TestMetric", Type: model.Gauge, Value: utils.F64Ptr(42.0)}
	err := st.Save(&m)
	if err != nil {
		t.Fatalf("Save in storage metric %s failed: %v", m.ID, err)
	}

	client := &Client{
		storage:    st,
		config:     &config.ClientConfig{ServerAddr: ts.URL},
		httpClient: &http.Client{Timeout: 2 * time.Second},
	}

	err = client.SendToServer()
	if err != nil {
		t.Errorf("SendToServer failed: %v", err)
	}
}
