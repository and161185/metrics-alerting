package client

import (
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/and161185/metrics-alerting/internal/config"
	"github.com/and161185/metrics-alerting/internal/utils"
	"github.com/and161185/metrics-alerting/model"
	"github.com/and161185/metrics-alerting/storage/inmemory"
)

func TestSendToServer(t *testing.T) {
	ctx := context.Background()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/updates/" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s, want POST", r.Method)
		}
		if r.Header.Get("Content-Encoding") == "gzip" {
			gr, err := gzip.NewReader(r.Body)
			if err != nil {
				t.Errorf("gzip decode error: %v", err)
				return
			}
			defer gr.Close()
			body, _ := io.ReadAll(gr)
			t.Logf("received: %s", string(body))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	st := inmemory.NewMemStorage(ctx)
	m := model.Metric{ID: "TestMetric", Type: model.Gauge, Value: utils.F64Ptr(42.0)}
	err := st.Save(ctx, &m)
	if err != nil {
		t.Fatalf("Save in storage metric %s failed: %v", m.ID, err)
	}

	client := &Client{
		storage:    st,
		config:     &config.ClientConfig{ServerAddr: ts.URL},
		httpClient: &http.Client{Timeout: 2 * time.Second},
	}

	err = client.SendToServer(ctx)
	if err != nil {
		t.Errorf("SendToServer failed: %v", err)
	}
}
