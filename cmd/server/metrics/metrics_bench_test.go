package metrics

import (
	"testing"

	"github.com/and161185/metrics-alerting/model"
	"github.com/stretchr/testify/require"
)

func runNewEmptyMetricTest(b testing.TB, typ, id string, wantErr error) {
	b.Helper()
	_, err := NewEmptyMetric(typ, id)

	if wantErr != nil {
		require.ErrorIs(b, err, wantErr)
	} else {
		require.NoError(b, err)
	}
}

func BenchmarkNewEmptyMetric(b *testing.B) {
	for i := 0; i < b.N; i++ {
		runNewEmptyMetricTest(b, "gauge", "TestMetric", nil)
	}
}

func runNewMetricTest(b testing.TB, typ, id, val string, wantErr error, wantType model.MetricType, wantVal float64) {
	b.Helper()
	m, err := NewMetric(typ, id, val)

	if wantErr != nil {
		require.Error(b, err)
		require.Contains(b, err.Error(), wantErr.Error())
		return
	}

	require.NoError(b, err)
	require.Equal(b, id, m.ID)
	require.Equal(b, wantType, m.Type)
	require.NotNil(b, m.Value)
	require.Equal(b, wantVal, *m.Value)
}

func BenchmarkNewMetric(b *testing.B) {
	for i := 0; i < b.N; i++ {
		runNewMetricTest(b, "gauge", "cpu", "12.3", nil, model.Gauge, 12.3)
	}
}
