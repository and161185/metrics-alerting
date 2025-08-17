package collector

import (
	"testing"

	"github.com/and161185/metrics-alerting/model"
	"github.com/stretchr/testify/require"
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

	for _, m := range metrics {
		// Проверка типа
		require.Condition(t, func() bool {
			return m.Type == model.Gauge || m.Type == model.Counter
		}, "invalid metric type: %s", m.Type)

		// Помечаем найденные
		if _, ok := requiredMetrics[m.ID]; ok {
			requiredMetrics[m.ID] = true
		}
	}

	// Проверка что все обязательные метрики найдены
	for id, found := range requiredMetrics {
		require.True(t, found, "required metric %s not found", id)
	}

	// Проверка инкремента PollCount
	poll1 := getMetricValue(metrics, "PollCount")
	metrics2 := CollectRuntimeMetrics()
	poll2 := getMetricValue(metrics2, "PollCount")

	require.Equal(t, poll1+1, poll2, "PollCount should increment by 1")
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
