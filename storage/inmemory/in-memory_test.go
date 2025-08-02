// inmemory_test.go — только тесты
package inmemory

import (
	"context"
	"os"
	"testing"

	"github.com/and161185/metrics-alerting/internal/utils"
	"github.com/and161185/metrics-alerting/model"
)

func TestSaveGauge(t *testing.T) {
	ctx := context.Background()
	st := NewMemStorage(ctx)
	m := &model.Metric{ID: "TestGauge", Type: model.Gauge, Value: utils.F64Ptr(42.0)}
	requireNoErr(t, st.Save(ctx, m))
	all, _ := st.GetAll(ctx)
	if got := *all["TestGauge"].Value; got != 42.0 {
		t.Errorf("want 42.0, got %v", got)
	}
}

func TestOverwriteGauge(t *testing.T) {
	ctx := context.Background()
	st := NewMemStorage(ctx)
	st.Save(ctx, &model.Metric{ID: "TestGauge", Type: model.Gauge, Value: utils.F64Ptr(42.0)})
	st.Save(ctx, &model.Metric{ID: "TestGauge", Type: model.Gauge, Value: utils.F64Ptr(100.0)})
	all, _ := st.GetAll(ctx)
	if got := *all["TestGauge"].Value; got != 100.0 {
		t.Errorf("overwrite failed: want 100.0, got %v", got)
	}
}

func TestAccumulateCounter(t *testing.T) {
	ctx := context.Background()
	st := NewMemStorage(ctx)
	st.Save(ctx, &model.Metric{ID: "TestCounter", Type: model.Counter, Delta: utils.I64Ptr(10)})
	st.Save(ctx, &model.Metric{ID: "TestCounter", Type: model.Counter, Delta: utils.I64Ptr(5)})
	all, _ := st.GetAll(ctx)
	if got := *all["TestCounter"].Delta; got != 15 {
		t.Errorf("accumulate failed: want 15, got %v", got)
	}
}

func TestSaveAndLoad(t *testing.T) {
	ctx := context.Background()
	file := "test_metrics.json"
	defer os.Remove(file)

	storage := NewMemStorage(ctx)
	m := model.Metric{ID: "test", Type: "gauge", Value: utils.F64Ptr(123.45)}
	_ = storage.Save(ctx, &m)
	requireNoErr(t, storage.SaveToFile(ctx, file))

	newStorage := NewMemStorage(ctx)
	m2 := model.Metric{ID: "other", Type: "gauge", Value: utils.F64Ptr(999.99)}
	_ = newStorage.Save(ctx, &m2)
	requireNoErr(t, newStorage.LoadFromFile(ctx, file))

	restored, err := newStorage.Get(ctx, &m)
	if err != nil || restored.Value == nil || *restored.Value != 123.45 {
		t.Errorf("wrong restored metric: %+v, err: %v", restored, err)
	}
	restored2, _ := newStorage.Get(ctx, &model.Metric{ID: "other", Type: "gauge"})
	if restored2.Value == nil || *restored2.Value != 999.99 {
		t.Errorf("existing metric lost: %+v", restored2)
	}
}

func requireNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
