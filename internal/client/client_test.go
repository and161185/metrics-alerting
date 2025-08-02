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
	"github.com/stretchr/testify/require"
)

func TestSendToServer(t *testing.T) {
	ctx := context.Background()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/updates/" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if r.Header.Get("Content-Encoding") == "gzip" {
			gr, err := gzip.NewReader(r.Body)
			require.NoError(t, err)
			defer gr.Close()
			_, _ = io.ReadAll(gr)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	st := inmemory.NewMemStorage(ctx)
	m := model.Metric{ID: "TestMetric", Type: model.Gauge, Value: utils.F64Ptr(42.0)}
	require.NoError(t, st.Save(ctx, &m))

	client := &Client{
		storage:    st,
		config:     &config.ClientConfig{ServerAddr: ts.URL},
		httpClient: &http.Client{Timeout: 2 * time.Second},
	}

	require.NoError(t, client.SendToServer(ctx))
}

func TestSendMetricToServer(t *testing.T) {
	ctx := context.Background()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/update/" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if r.Header.Get("Content-Encoding") != "gzip" {
			t.Errorf("missing gzip encoding")
		}
		gr, err := gzip.NewReader(r.Body)
		require.NoError(t, err)
		defer gr.Close()
		_, _ = io.ReadAll(gr)

		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	st := inmemory.NewMemStorage(ctx)
	m := &model.Metric{ID: "TestMetric", Type: model.Gauge, Value: utils.F64Ptr(42.0)}

	client := &Client{
		storage:    st,
		config:     &config.ClientConfig{ServerAddr: ts.URL},
		httpClient: &http.Client{Timeout: 2 * time.Second},
	}

	require.NoError(t, client.SendMetricToServer(ctx, m))
}
