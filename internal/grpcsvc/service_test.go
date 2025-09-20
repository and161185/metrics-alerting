package grpcsvc

import (
	"context"
	"testing"
	"time"

	metricsv1 "github.com/and161185/metrics-alerting/internal/api/metricsv1"
)

type fakeStore struct {
	g       map[string]float64
	c       map[string]int64
	pingErr error
}

func newFake() *fakeStore { return &fakeStore{g: map[string]float64{}, c: map[string]int64{}} }

func (f *fakeStore) Ping(ctx context.Context) error { return f.pingErr }
func (f *fakeStore) UpdateGauge(ctx context.Context, id string, v float64) error {
	f.g[id] = v
	return nil
}
func (f *fakeStore) UpdateCounter(ctx context.Context, id string, d int64) error {
	f.c[id] += d
	return nil
}
func (f *fakeStore) GetGauge(ctx context.Context, id string) (float64, bool, error) {
	v, ok := f.g[id]
	return v, ok, nil
}
func (f *fakeStore) GetCounter(ctx context.Context, id string) (int64, bool, error) {
	v, ok := f.c[id]
	return v, ok, nil
}
func (f *fakeStore) GetAll(ctx context.Context) (map[string]float64, map[string]int64, error) {
	return f.g, f.c, nil
}

func TestService_Update_and_Get(t *testing.T) {
	s := NewService(newFake())

	// gauge
	_, err := s.UpdateMetric(context.Background(), &metricsv1.Metric{
		Id:    "temp",
		Kind:  metricsv1.MetricKind_GAUGE,
		Value: &metricsv1.Metric_Gauge{Gauge: 36.6},
	})
	if err != nil {
		t.Fatalf("update gauge: %v", err)
	}

	// counter
	_, err = s.UpdateMetric(context.Background(), &metricsv1.Metric{
		Id:    "hits",
		Kind:  metricsv1.MetricKind_COUNTER,
		Value: &metricsv1.Metric_Counter{Counter: 2},
	})
	if err != nil {
		t.Fatalf("update counter: %v", err)
	}

	// get gauge
	mg, err := s.GetMetric(context.Background(), &metricsv1.GetRequest{Id: "temp", Kind: metricsv1.MetricKind_GAUGE})
	if err != nil {
		t.Fatalf("get gauge: %v", err)
	}
	if mg.GetGauge() != 36.6 {
		t.Fatalf("gauge = %v", mg.GetGauge())
	}

	// get counter
	mc, err := s.GetMetric(context.Background(), &metricsv1.GetRequest{Id: "hits", Kind: metricsv1.MetricKind_COUNTER})
	if err != nil {
		t.Fatalf("get counter: %v", err)
	}
	if mc.GetCounter() != 2 {
		t.Fatalf("counter = %v", mc.GetCounter())
	}

	// get not found
	if _, err := s.GetMetric(context.Background(), &metricsv1.GetRequest{Id: "nope", Kind: metricsv1.MetricKind_GAUGE}); err == nil {
		t.Fatal("want not found")
	}

	// get all
	all, err := s.GetAll(context.Background(), &metricsv1.Empty{})
	if err != nil {
		t.Fatalf("get all: %v", err)
	}
	if len(all.Items) != 2 {
		t.Fatalf("items = %d", len(all.Items))
	}
}

func TestService_UpdateBatch_and_Ping(t *testing.T) {
	fs := newFake()
	s := NewService(fs)

	// batch
	_, err := s.UpdateBatch(context.Background(), &metricsv1.MetricsBatch{
		Items: []*metricsv1.Metric{
			{Id: "a", Kind: metricsv1.MetricKind_GAUGE, Value: &metricsv1.Metric_Gauge{Gauge: 1.23}},
			{Id: "b", Kind: metricsv1.MetricKind_COUNTER, Value: &metricsv1.Metric_Counter{Counter: 10}},
		},
	})
	if err != nil {
		t.Fatalf("update batch: %v", err)
	}

	mg, _ := s.GetMetric(context.Background(), &metricsv1.GetRequest{Id: "a", Kind: metricsv1.MetricKind_GAUGE})
	if mg.GetGauge() != 1.23 {
		t.Fatalf("gauge = %v", mg.GetGauge())
	}
	mc, _ := s.GetMetric(context.Background(), &metricsv1.GetRequest{Id: "b", Kind: metricsv1.MetricKind_COUNTER})
	if mc.GetCounter() != 10 {
		t.Fatalf("counter = %v", mc.GetCounter())
	}

	// ping OK
	if _, err := s.Ping(context.Background(), &metricsv1.Empty{}); err != nil {
		t.Fatalf("ping ok: %v", err)
	}

	// ping timeout maps to DeadlineExceeded
	fs.pingErr = context.DeadlineExceeded
	if _, err := s.Ping(context.Background(), &metricsv1.Empty{}); err == nil {
		t.Fatal("want ping error")
	}
}

func TestService_ValidateInputs(t *testing.T) {
	s := NewService(newFake())

	// empty metric
	if _, err := s.UpdateMetric(context.Background(), &metricsv1.Metric{}); err == nil {
		t.Fatal("want invalid argument on empty metric")
	}
	// wrong kind/value mismatch
	_, err := s.UpdateMetric(context.Background(), &metricsv1.Metric{
		Id: "x", Kind: metricsv1.MetricKind_GAUGE, Value: &metricsv1.Metric_Counter{Counter: 1},
	})
	if err == nil {
		t.Fatal("want invalid argument on kind/value mismatch")
	}

	// empty batch
	if _, err := s.UpdateBatch(context.Background(), &metricsv1.MetricsBatch{}); err == nil {
		t.Fatal("want invalid argument on empty batch")
	}
}

func TestService_ContextCancel(t *testing.T) {
	// Просто дергаем с коротким контекстом, чтобы зацепить ветки обработки ошибок/контекста.
	s := NewService(newFake())
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	cancel()
	_, _ = s.UpdateMetric(ctx, &metricsv1.Metric{Id: "a", Kind: metricsv1.MetricKind_GAUGE, Value: &metricsv1.Metric_Gauge{Gauge: 1}})
	_, _ = s.GetAll(ctx, &metricsv1.Empty{})
}
