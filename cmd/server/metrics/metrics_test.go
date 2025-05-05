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
			_, err := NewEmptyMetric(tt.typ, tt.id)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
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
				return
			}

			require.NoError(t, err)
			if tc.id != m.ID {
				t.Errorf("metric mismatch: need %s get %s", tc.id, m.ID)
			}
			require.NoError(t, err)
			if tc.id != m.ID {
				t.Errorf("metric type mismatch: need %s get %s", tc.wantType, m.Type)
			}
			require.NoError(t, err)
			if tc.id != m.ID {
				t.Errorf("metric value error: need %f get %f", tc.wantVal, m.Value)
			}
		})
	}

}
