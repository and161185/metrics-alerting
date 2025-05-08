package collector

import (
	"testing"

	"github.com/and161185/metrics-alerting/model"
	"github.com/magiconair/properties/assert"
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

	assert.Equal(t, poll1+1, poll2, "PollCount test failed")
}

func getMetricValue(metrics []model.Metric, id string) float64 {
	for _, m := range metrics {
		if m.ID == id {
			return m.Value
		}
	}
	return -1 // или panic если хочешь жёстче
}
