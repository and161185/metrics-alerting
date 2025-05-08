package storage

import (
	"testing"

	"github.com/and161185/metrics-alerting/model"
)

func TestMemStorage_SaveGauge(t *testing.T) {
	st := NewMemStorage()
	metric := model.Metric{ID: "TestGauge", Type: model.Gauge, Value: 42.0}

	if err := st.Save(&metric); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	all, _ := st.GetAll()
	if got := all["TestGauge"].Value; got != 42.0 {
		t.Errorf("want 42.0, got %v", got)
	}
}

func TestMemStorage_OverwriteGauge(t *testing.T) {
	st := NewMemStorage()
	st.Save(&model.Metric{ID: "TestGauge", Type: model.Gauge, Value: 42.0})
	st.Save(&model.Metric{ID: "TestGauge", Type: model.Gauge, Value: 100.0})

	all, _ := st.GetAll()
	if got := all["TestGauge"].Value; got != 100.0 {
		t.Errorf("overwrite failed: want 100.0, got %v", got)
	}
}

func TestMemStorage_AccumulateCounter(t *testing.T) {
	st := NewMemStorage()
	st.Save(&model.Metric{ID: "TestCounter", Type: model.Counter, Value: 10})
	st.Save(&model.Metric{ID: "TestCounter", Type: model.Counter, Value: 5})

	all, _ := st.GetAll()
	if got := all["TestCounter"].Value; got != 15.0 {
		t.Errorf("accumulate failed: want 15.0, got %v", got)
	}
}
