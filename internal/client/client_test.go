package client

import (
	"compress/gzip"
	"context"
	"errors"
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

type errStorage struct{}

func (e *errStorage) Save(_ context.Context, _ *model.Metric) error {
	return errors.New("save failed")
}
func (e *errStorage) GetAll(_ context.Context) (map[string]*model.Metric, error) {
	return nil, errors.New("getall failed")
}

func TestSendToServer_OK(t *testing.T) {
	ctx := context.Background()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/updates/", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)
		if r.Header.Get("Content-Encoding") == "gzip" {
			gr, err := gzip.NewReader(r.Body)
			require.NoError(t, err)
			_, _ = io.ReadAll(gr)
			_ = gr.Close()
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	st := inmemory.NewMemStorage(ctx)
	m := model.Metric{ID: "TestMetric", Type: model.Gauge, Value: utils.F64Ptr(42)}
	require.NoError(t, st.Save(ctx, &m))

	c, err := NewClient(st, &config.ClientConfig{ServerAddr: ts.URL, ClientTimeout: 1})
	if err != nil {
		t.Fatalf("client constructor error: %v", err)
	}
	require.NoError(t, c.sendToServer(ctx))
}

func TestSendToServer_ErrorStatus(t *testing.T) {
	ctx := context.Background()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	st := inmemory.NewMemStorage(ctx)
	m := model.Metric{ID: "x", Type: model.Gauge, Value: utils.F64Ptr(1)}
	_ = st.Save(ctx, &m)

	c, err := NewClient(st, &config.ClientConfig{ServerAddr: ts.URL, ClientTimeout: 1})
	if err != nil {
		t.Fatalf("client constructor error: %v", err)
	}
	err2 := c.sendToServer(ctx)
	require.Error(t, err2)
	require.Contains(t, err2.Error(), "unexpected status")
}

func TestSendMetricToServer_OK(t *testing.T) {
	ctx := context.Background()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/update/", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "gzip", r.Header.Get("Content-Encoding"))
		gr, err := gzip.NewReader(r.Body)
		require.NoError(t, err)
		_, _ = io.ReadAll(gr)
		_ = gr.Close()
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	st := inmemory.NewMemStorage(ctx)
	m := &model.Metric{ID: "X", Type: model.Gauge, Value: utils.F64Ptr(1)}

	c, err := NewClient(st, &config.ClientConfig{ServerAddr: ts.URL, ClientTimeout: 1})
	if err != nil {
		t.Fatalf("client constructor error: %v", err)
	}
	require.NoError(t, c.sendMetricToServer(ctx, m))
}

func TestSendMetricToServer_Error(t *testing.T) {
	ctx := context.Background()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot) // 418
	}))
	defer ts.Close()

	st := inmemory.NewMemStorage(ctx)
	m := &model.Metric{ID: "X", Type: model.Gauge, Value: utils.F64Ptr(1)}

	c, err := NewClient(st, &config.ClientConfig{ServerAddr: ts.URL, ClientTimeout: 1})
	if err != nil {
		t.Fatalf("client constructor errer: %v", err)
	}
	err2 := c.sendMetricToServer(ctx, m)
	require.Error(t, err2)
	require.Contains(t, err2.Error(), "unexpected status")
}

func TestCollectAndSave(t *testing.T) {
	ctx := context.Background()
	st := inmemory.NewMemStorage(ctx)
	collect := func() []model.Metric {
		return []model.Metric{{ID: "m1", Type: model.Gauge, Value: utils.F64Ptr(10)}}
	}
	collectAndSave(ctx, st, collect, "label")
	all, _ := st.GetAll(ctx)
	require.Contains(t, all, "m1")
}

func TestCollectAndSave_SaveError(t *testing.T) {
	ctx := context.Background()
	st := &errStorage{}
	collectAndSave(ctx, st, func() []model.Metric {
		return []model.Metric{{ID: "bad", Type: model.Gauge, Value: utils.F64Ptr(1)}}
	}, "test")
}

func TestDispatchMetrics_OK(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	st := inmemory.NewMemStorage(ctx)
	m := model.Metric{ID: "m1", Type: model.Gauge, Value: utils.F64Ptr(1)}
	_ = st.Save(ctx, &m)
	ch := make(chan *model.Metric, 1)

	go dispatchMetrics(ctx, st, ch, 10*time.Millisecond)

	select {
	case got := <-ch:
		require.Equal(t, "m1", got.ID)
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout")
	}
}

func TestDispatchMetrics_GetAllError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	st := &errStorage{}
	ch := make(chan *model.Metric, 1)
	go dispatchMetrics(ctx, st, ch, 10*time.Millisecond)
	<-ctx.Done()
}

func TestRuntimeCollector_Saves(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	st := inmemory.NewMemStorage(ctx)
	go runtimeCollector(ctx, st, 10*time.Millisecond)
	time.Sleep(20 * time.Millisecond)
	all, _ := st.GetAll(ctx)
	require.NotEmpty(t, all)
}

func TestGopsutilCollector_Saves(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	st := inmemory.NewMemStorage(ctx)
	go gopsutilCollector(ctx, st, 10*time.Millisecond)
	time.Sleep(20 * time.Millisecond)
	all, _ := st.GetAll(ctx)

	_ = all
}

func TestClientRun_StartsAndStops(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	st := inmemory.NewMemStorage(ctx)
	cfg := &config.ClientConfig{PollInterval: 1, ReportInterval: 1, RateLimit: 1, ClientTimeout: 1}
	c, err := NewClient(st, cfg)
	if err != nil {
		t.Fatalf("client constructor errer: %v", err)
	}
	require.NoError(t, c.Run(ctx))
}
