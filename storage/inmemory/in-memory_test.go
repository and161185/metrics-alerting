package inmemory

import (
	"os"
	"testing"

	"github.com/and161185/metrics-alerting/internal/utils"
	"github.com/and161185/metrics-alerting/model"
)

func TestMemStorage_SaveGauge(t *testing.T) {
	st := NewMemStorage()

	metric := model.Metric{ID: "TestGauge", Type: model.Gauge, Value: utils.F64Ptr(42.0)}

	if err := st.Save(&metric); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	all, _ := st.GetAll()
	if got := *all["TestGauge"].Value; got != 42.0 {
		t.Errorf("want 42.0, got %v", got)
	}
}

func TestMemStorage_OverwriteGauge(t *testing.T) {
	st := NewMemStorage()
	st.Save(&model.Metric{ID: "TestGauge", Type: model.Gauge, Value: utils.F64Ptr(42.0)})
	st.Save(&model.Metric{ID: "TestGauge", Type: model.Gauge, Value: utils.F64Ptr(100.0)})

	all, _ := st.GetAll()
	if got := *all["TestGauge"].Value; got != 100.0 {
		t.Errorf("overwrite failed: want 100.0, got %v", got)
	}
}

func TestMemStorage_AccumulateCounter(t *testing.T) {
	st := NewMemStorage()
	st.Save(&model.Metric{ID: "TestCounter", Type: model.Counter, Delta: utils.I64Ptr(10)})
	st.Save(&model.Metric{ID: "TestCounter", Type: model.Counter, Delta: utils.I64Ptr(5)})

	all, _ := st.GetAll()
	if got := *all["TestCounter"].Delta; got != 15 {
		t.Errorf("accumulate failed: want 15.0, got %v", got)
	}
}

func TestSaveAndLoad(t *testing.T) {
	file := "test_metrics.json"
	defer os.Remove(file)

	storage := NewMemStorage()

	// сохраняем одну метрику
	m := model.Metric{
		ID:    "test",
		Type:  "gauge",
		Value: utils.F64Ptr(123.45),
	}
	_ = storage.Save(&m)

	if err := storage.SaveToFile(file); err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	// создаём новое хранилище и загружаем
	newStorage := NewMemStorage()
	m2 := model.Metric{
		ID:    "other",
		Type:  "gauge",
		Value: utils.F64Ptr(999.99),
	}
	_ = newStorage.Save(&m2)

	if err := newStorage.LoadFromFile(file); err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	restored, err := newStorage.Get(&m)
	if err != nil {
		t.Fatalf("metric not restored: %v", err)
	}

	if restored.Value == nil || *restored.Value != 123.45 {
		t.Errorf("wrong value: got %+v", restored.Value)
	}

	restored2, _ := newStorage.Get(&model.Metric{ID: "other", Type: "gauge"})
	if restored2.Value == nil || *restored2.Value != 999.99 {
		t.Errorf("existing metric lost: %+v", restored2)
	}

}
