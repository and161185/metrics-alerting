// metrics_test.go — исправленные тесты
package metrics

import (
	"testing"

	"github.com/and161185/metrics-alerting/model"
	"github.com/stretchr/testify/require"
)

func TestNewEmptyMetric(t *testing.T) {
	tests := []struct {
		name    string
		typ     string
		id      string
		wantErr error
	}{
		{"valid_gauge", "gauge", "TestMetric", nil},
		{"valid_counter", "counter", "CounterMetric", nil},
		{"invalid_type", "float", "Invalid", ErrInvalidType},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := NewEmptyMetric(tt.typ, tt.id)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Equal(t, "", m.ID)
				require.Equal(t, model.MetricType(""), m.Type)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.id, m.ID)
			require.Equal(t, model.MetricType(tt.typ), m.Type)
		})
	}
}

func TestNewMetric(t *testing.T) {
	tests := []struct {
		name     string
		typ      string
		id       string
		val      string
		wantErr  error
		wantType model.MetricType
		wantVal  float64
	}{
		{"valid_gauge", "gauge", "cpu", "12.3", nil, model.Gauge, 12.3},
		{"valid_counter", "counter", "ops", "5", nil, model.Counter, 5},
		{"invalid_type", "lol", "x", "1", ErrInvalidType, "", 0},
		{"invalid_counter value", "counter", "x", "1.1", ErrInvalidValue, "", 0},
		{"invalid_value format", "gauge", "x", "abc", ErrInvalidValue, "", 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m, err := NewMetric(tc.typ, tc.id, tc.val)

			if tc.wantErr != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr.Error())
				require.Equal(t, "", m.ID)
				require.Equal(t, model.MetricType(""), m.Type)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.id, m.ID)
			require.Equal(t, tc.wantType, m.Type)
			if tc.wantType == model.Gauge {
				require.NotNil(t, m.Value)
				require.InEpsilon(t, tc.wantVal, *m.Value, 0.0001)
			} else if tc.wantType == model.Counter {
				require.NotNil(t, m.Delta)
				require.EqualValues(t, tc.wantVal, *m.Delta)
			}
		})
	}
}
