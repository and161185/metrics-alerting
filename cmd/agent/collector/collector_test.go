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
		require.Condition(t, func() bool {
			return m.Type == model.Gauge || m.Type == model.Counter
		}, "invalid metric type: %s", m.Type)

		if _, ok := requiredMetrics[m.ID]; ok {
			requiredMetrics[m.ID] = true
		}
	}

	for id, found := range requiredMetrics {
		require.True(t, found, "required metric %s not found", id)
	}

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

func TestResetPollCount(t *testing.T) {
	ResetPollCount()
	metrics := CollectRuntimeMetrics()

	poll := getMetricValue(metrics, "PollCount")
	require.Equal(t, float64(1), poll, "после ResetPollCount первый вызов должен дать PollCount=1")
}

func TestCollectGopsutilMetrics_Smoke(t *testing.T) {
	t.Parallel()

	metrics := CollectGopsutilMetrics()

	seen := map[string]struct{}{}
	for _, m := range metrics {

		if _, ok := seen[m.ID]; ok {
			t.Fatalf("duplicate metric id: %s", m.ID)
		}
		seen[m.ID] = struct{}{}

		require.True(t, m.Type == model.Gauge || m.Type == model.Counter)
		if m.Type == model.Gauge {
			require.NotNil(t, m.Value)
		} else {
			require.NotNil(t, m.Delta)
		}
	}
}
