package metrics

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/and161185/metrics-alerting/model"
)

var ErrInvalidValue = errors.New("invalid value")
var ErrInvalidType = errors.New("invalid metric type")
var ErrInvalidName = errors.New("invalid metric name")

func NewEmptyMetric(typ, name string) (*model.Metric, error) {

	metricsType := model.MetricType(typ)
	if invalidMetricsType(metricsType) {
		return &model.Metric{}, ErrInvalidType
	}

	return &model.Metric{ID: name, Type: metricsType}, nil
}

func NewMetric(typ, name, val string) (*model.Metric, error) {

	metric, err := NewEmptyMetric(typ, name)
	if err != nil {
		return &model.Metric{}, err
	}

	metricsValue, err := getMetricsValue(val, metric.Type)
	if err != nil {
		return &model.Metric{}, fmt.Errorf("invalid value: %w", err)
	}

	metric.Value = &metricsValue

	return metric, nil
}

func CheckMetric(m *model.Metric) error {
	if m.Type == model.Counter && m.Delta == nil {
		return errors.New("delta required for counter")
	}
	if m.Type == model.Gauge && m.Value == nil {
		return errors.New("value required for gauge")
	}
	if m.Type != model.Gauge && m.Type != model.Counter {
		return errors.New("invalid type")
	}
	return nil
}

func invalidMetricsType(typ model.MetricType) bool {
	result := typ != model.Gauge && typ != model.Counter
	return result
}

func getMetricsValue(strValue string, metricsType model.MetricType) (float64, error) {
	val, err := strconv.ParseFloat(strValue, 64)
	if err != nil {
		return 0, err
	}

	if metricsType == model.Counter && val != float64(int64(val)) {
		return 0, errors.New("value of counter metric should be int64")
	}

	return val, nil
}
