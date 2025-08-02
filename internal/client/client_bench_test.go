package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/and161185/metrics-alerting/internal/config"
	"github.com/and161185/metrics-alerting/internal/utils"
	"github.com/and161185/metrics-alerting/model"
	"github.com/and161185/metrics-alerting/storage/inmemory"
)

func setupClient(ctx context.Context) *Client {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	st := inmemory.NewMemStorage(ctx)
	st.Save(ctx, &model.Metric{ID: "TestMetric", Type: model.Gauge, Value: utils.F64Ptr(42.0)})

	return &Client{
		storage:    st,
		config:     &config.ClientConfig{ServerAddr: ts.URL},
		httpClient: &http.Client{Timeout: 1 * time.Second},
	}
}

func BenchmarkSendToServer(b *testing.B) {
	ctx := context.Background()
	client := setupClient(ctx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.SendToServer(ctx)
	}
}

func BenchmarkSendMetricToServer(b *testing.B) {
	ctx := context.Background()
	client := setupClient(ctx)
	metric := &model.Metric{ID: "TestMetric", Type: model.Gauge, Value: utils.F64Ptr(42.0)}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.SendMetricToServer(ctx, metric)
	}
}
