package storage

import (
	"testing"

	"github.com/and161185/metrics-alerting/model"
)

func TestMemStorage(t *testing.T) {
	st := NewMemStorage()

	// gauge
	g := model.Metric{ID: "TestGauge", Type: model.Gauge, Value: 42.0}
	err := st.Save(g)
	if err != nil {
		t.Fatalf("Save in storage metric %s %f failed: %v", g.ID, g.Value, err)
	}

	all, err := st.GetAll()
	if err != nil {
		t.Errorf("internal error")
		return
	}

	if got := all["TestGauge"].Value; got != 42.0 {
		t.Errorf("Gauge save failed: got %v, want 42", got)
	}

	g2 := model.Metric{ID: "TestGauge", Type: model.Gauge, Value: 100.0}
	err = st.Save(g2)
	if err != nil {
		t.Fatalf("Save in storage metric %s %f failed: %v", g2.ID, g2.Value, err)
	}

	all, err = st.GetAll()
	if err != nil {
		t.Errorf("internal error")
		return
	}

	if got := all["TestGauge"].Value; got != 100.0 {
		t.Errorf("Gauge overwrite failed: got %v, want 100", got)
	}

	// counter
	c := model.Metric{ID: "TestCounter", Type: model.Counter, Value: 10}
	err = st.Save(c)
	if err != nil {
		t.Fatalf("Save in storage metric %s %f failed: %v", g2.ID, g2.Value, err)
	}

	c2 := model.Metric{ID: "TestCounter", Type: model.Counter, Value: 5}
	err = st.Save(c2)
	if err != nil {
		t.Fatalf("Save in storage metric %s %f failed: %v", g2.ID, g2.Value, err)
	}

	all, err = st.GetAll()
	if err != nil {
		t.Errorf("internal error")
		return
	}

	if got := all["TestCounter"].Value; got != 15.0 {
		t.Errorf("Counter accumulate failed: got %v, want 15", got)
	}
}
