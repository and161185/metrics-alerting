package inmemory

import (
	"context"
	"os"
	"testing"

	"github.com/and161185/metrics-alerting/internal/utils"
	"github.com/and161185/metrics-alerting/model"
)

func TestMemStorage_SaveGauge(t *testing.T) {
	ctx := context.Background()

	st := NewMemStorage(ctx)

	metric := model.Metric{ID: "TestGauge", Type: model.Gauge, Value: utils.F64Ptr(42.0)}

	if err := st.Save(ctx, &metric); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	all, _ := st.GetAll(ctx)
	if got := *all["TestGauge"].Value; got != 42.0 {
		t.Errorf("want 42.0, got %v", got)
	}
}

func TestMemStorage_OverwriteGauge(t *testing.T) {
	ctx := context.Background()

	st := NewMemStorage(ctx)
	st.Save(ctx, &model.Metric{ID: "TestGauge", Type: model.Gauge, Value: utils.F64Ptr(42.0)})
	st.Save(ctx, &model.Metric{ID: "TestGauge", Type: model.Gauge, Value: utils.F64Ptr(100.0)})

	all, _ := st.GetAll(ctx)
	if got := *all["TestGauge"].Value; got != 100.0 {
		t.Errorf("overwrite failed: want 100.0, got %v", got)
	}
}

func TestMemStorage_AccumulateCounter(t *testing.T) {
	ctx := context.Background()

	st := NewMemStorage(ctx)
	st.Save(ctx, &model.Metric{ID: "TestCounter", Type: model.Counter, Delta: utils.I64Ptr(10)})
	st.Save(ctx, &model.Metric{ID: "TestCounter", Type: model.Counter, Delta: utils.I64Ptr(5)})

	all, _ := st.GetAll(ctx)
	if got := *all["TestCounter"].Delta; got != 15 {
		t.Errorf("accumulate failed: want 15.0, got %v", got)
	}
}

func TestSaveAndLoad(t *testing.T) {
	ctx := context.Background()

	file := "test_metrics.json"
	defer os.Remove(file)

	storage := NewMemStorage(ctx)

	// сохраняем одну метрику
	m := model.Metric{
		ID:    "test",
		Type:  "gauge",
		Value: utils.F64Ptr(123.45),
	}
	_ = storage.Save(ctx, &m)

	if err := storage.SaveToFile(ctx, file); err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	// создаём новое хранилище и загружаем
	newStorage := NewMemStorage(ctx)
	m2 := model.Metric{
		ID:    "other",
		Type:  "gauge",
		Value: utils.F64Ptr(999.99),
	}
	_ = newStorage.Save(ctx, &m2)

	if err := newStorage.LoadFromFile(ctx, file); err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	restored, err := newStorage.Get(ctx, &m)
	if err != nil {
		t.Fatalf("metric not restored: %v", err)
	}

	if restored.Value == nil || *restored.Value != 123.45 {
		t.Errorf("wrong value: got %+v", restored.Value)
	}

	restored2, _ := newStorage.Get(ctx, &model.Metric{ID: "other", Type: "gauge"})
	if restored2.Value == nil || *restored2.Value != 999.99 {
		t.Errorf("existing metric lost: %+v", restored2)
	}

}
