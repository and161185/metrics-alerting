package storage

import (
	"testing"

	"github.com/and161185/metrics-alerting/model"
)

func TestMemStorage(t *testing.T) {
	st := NewMemStorage()

	// gauge
	g := model.Metric{ID: "TestGauge", Type: model.Gauge, Value: 42.0}
	st.Save(g)

	if got := st.GetAll()["TestGauge"].Value; got != 42.0 {
		t.Errorf("Gauge save failed: got %v, want 42", got)
	}

	g2 := model.Metric{ID: "TestGauge", Type: model.Gauge, Value: 100.0}
	st.Save(g2)

	if got := st.GetAll()["TestGauge"].Value; got != 100.0 {
		t.Errorf("Gauge overwrite failed: got %v, want 100", got)
	}

	// counter
	c := model.Metric{ID: "TestCounter", Type: model.Counter, Value: 10}
	st.Save(c)

	c2 := model.Metric{ID: "TestCounter", Type: model.Counter, Value: 5}
	st.Save(c2)

	if got := st.GetAll()["TestCounter"].Value; got != 15.0 {
		t.Errorf("Counter accumulate failed: got %v, want 15", got)
	}
}
