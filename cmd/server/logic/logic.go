package logic

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/and161185/metrics-alerting/model"
)

var ErrInvalidValue = errors.New("invalid value")
var ErrInvalidType = errors.New("invalid metric type")
var ErrInvalidName = errors.New("invalid metric name")

func NewEmptyMetric(typ, name string) (model.Metric, error) {

	metricsType := model.MetricType(typ)
	if invalidMetricsType(metricsType) {
		return model.Metric{}, ErrInvalidType
	}

	if invalidMetricsName(name) {
		return model.Metric{}, ErrInvalidName
	}

	return model.Metric{ID: name, Type: metricsType, Value: 0}, nil
}

func NewMetric(typ, name, val string) (model.Metric, error) {

	metric, err := NewEmptyMetric(typ, name)
	if err != nil {
		return model.Metric{}, err
	}

	metricsValue, err := getMetricsValue(val, metric.Type)
	if err != nil {
		return model.Metric{}, fmt.Errorf("invalid value: %w", err)
	}

	metric.Value = metricsValue

	return metric, nil
}

func invalidMetricsType(t model.MetricType) bool {
	result := t != model.Gauge && t != model.Counter
	return result
}

func invalidMetricsName(name string) bool {
	return false
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
