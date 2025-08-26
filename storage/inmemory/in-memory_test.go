// inmemory_test.go — только тесты
package inmemory

import (
	"context"
	"os"
	"testing"

	"github.com/and161185/metrics-alerting/internal/errs"
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

func TestGet_NotFound(t *testing.T) {
	ctx := context.Background()
	st := NewMemStorage(ctx)
	_, err := st.Get(ctx, &model.Metric{ID: "nope", Type: model.Gauge})
	if err == nil || err != errs.ErrMetricNotFound {
		t.Fatalf("want ErrMetricNotFound, got %v", err)
	}
}

func TestCounterFirstSetWhenExistingNilDelta(t *testing.T) {
	ctx := context.Background()
	st := NewMemStorage(ctx)

	requireNoErr(t, st.Save(ctx, &model.Metric{ID: "C", Type: model.Counter}))

	requireNoErr(t, st.Save(ctx, &model.Metric{ID: "C", Type: model.Counter, Delta: utils.I64Ptr(7)}))

	got, err := st.Get(ctx, &model.Metric{ID: "C", Type: model.Counter})
	requireNoErr(t, err)
	if got.Delta == nil || *got.Delta != 7 {
		t.Fatalf("want 7, got %v", got.Delta)
	}
}

func TestSaveBatch_SavesAll(t *testing.T) {
	ctx := context.Background()
	st := NewMemStorage(ctx)
	ms := []model.Metric{
		{ID: "g1", Type: model.Gauge, Value: utils.F64Ptr(1)},
		{ID: "c1", Type: model.Counter, Delta: utils.I64Ptr(2)},
	}
	requireNoErr(t, st.SaveBatch(ctx, ms))

	all, _ := st.GetAll(ctx)
	if _, ok := all["g1"]; !ok {
		t.Fatal("g1 not saved")
	}
	if _, ok := all["c1"]; !ok {
		t.Fatal("c1 not saved")
	}
}

func TestGetAll_ReturnsMapCopy(t *testing.T) {
	ctx := context.Background()
	st := NewMemStorage(ctx)
	requireNoErr(t, st.Save(ctx, &model.Metric{ID: "keep", Type: model.Gauge, Value: utils.F64Ptr(1)}))

	m1, _ := st.GetAll(ctx)
	delete(m1, "keep")

	_, err := st.Get(ctx, &model.Metric{ID: "keep", Type: model.Gauge})
	requireNoErr(t, err)
}

func TestSaveToFile_NoMetrics_NoFile(t *testing.T) {
	ctx := context.Background()
	st := NewMemStorage(ctx)

	tmp := "empty.json"
	_ = os.Remove(tmp)
	requireNoErr(t, st.SaveToFile(ctx, tmp))

	if _, err := os.Stat(tmp); err == nil {
		t.Fatalf("file should not be created for empty metrics")
	}
}

func TestLoadFromFile_NoFile_OK(t *testing.T) {
	ctx := context.Background()
	st := NewMemStorage(ctx)
	requireNoErr(t, st.LoadFromFile(ctx, "no_such_file_123456.json"))
}

func TestLoadFromFile_BadJSON(t *testing.T) {
	ctx := context.Background()
	tmp := "bad.json"
	_ = os.WriteFile(tmp, []byte("{not json]"), 0644)
	defer os.Remove(tmp)

	st := NewMemStorage(ctx)
	if err := st.LoadFromFile(ctx, tmp); err == nil {
		t.Fatal("want error on bad json")
	}
}

func TestPing_OK(t *testing.T) {
	ctx := context.Background()
	st := NewMemStorage(ctx)
	if err := st.Ping(ctx); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}
