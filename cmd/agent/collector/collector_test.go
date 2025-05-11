package collector

import (
	"testing"

	"github.com/and161185/metrics-alerting/model"
)

func TestCollectRuntimeMetrics(t *testing.T) {

	requiredMetrics := map[string]bool{
		"Alloc":       false,
		"GCSys":       false,
		"HeapAlloc":   false,
		"PollCount":   false,
		"RandomValue": false,
	}

	metrics := CollectRuntimeMetrics()

	for _, v := range metrics {
		if _, ok := requiredMetrics[v.ID]; ok {
			requiredMetrics[v.ID] = true
		}

		if v.Type != model.Counter && v.Type != model.Gauge {
			t.Errorf("invalid type: %s for metric %s", v.Type, v.ID)
		}
	}

	for id, found := range requiredMetrics {
		if !found {
			t.Errorf("required metric %s not found", id)
		}
	}

	poll1 := getMetricValue(metrics, "PollCount")

	metrics2 := CollectRuntimeMetrics()
	poll2 := getMetricValue(metrics2, "PollCount")

	if poll1+1 != poll2 {
		t.Errorf("pollCount test failed: need %f get %f", poll1+1, poll2)
	}
}

func getMetricValue(metrics []model.Metric, id string) float64 {
	for _, m := range metrics {
		if m.ID != id {
			continue
		}
		if m.Type == model.Gauge && m.Value != nil {
			return *m.Value
		}
		if m.Type == model.Counter && m.Delta != nil {
			return float64(*m.Delta)
		}

	}
	return -1
}
