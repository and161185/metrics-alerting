package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/and161185/metrics-alerting/model"
	"github.com/and161185/metrics-alerting/storage"
)

var ErrInvalidUrl = errors.New("invalid url")
var ErrInvalidValue = errors.New("invalid value")
var ErrInvalidType = errors.New("invalid metric type")
var ErrInvalidName = errors.New("invalid metric name")

func UpdateMetricHandler(storage storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		urlPath := strings.Trim(r.URL.Path, "/")

		metrics, err := getMetrics(urlPath)
		if err != nil {
			if errors.Is(err, ErrInvalidUrl) {
				http.NotFound(w, r)
				return
			} else {
				http.Error(w, err.Error(), http.StatusBadRequest)
			}
			return
		}

		storage.Save(metrics)
		w.WriteHeader(http.StatusOK)
	}
}

func getMetrics(urlPath string) (*model.Metric, error) {

	urlParts := strings.Split(urlPath, "/")
	if len(urlParts) != 4 {
		return nil, ErrInvalidUrl
	}

	metricsType := model.MetricType(urlParts[1])
	if invalidMetricsType(metricsType) {
		return nil, ErrInvalidType
	}

	metricsName := urlParts[2]
	if invalidMetricsName(metricsName) {
		return nil, ErrInvalidName
	}

	metricsValue, err := getMetricsValue(urlParts[3], metricsType)
	if err != nil {
		return nil, fmt.Errorf("invalid value: %w", err)
	}

	result := model.Metric{ID: metricsName, Type: metricsType, Value: metricsValue}
	return &result, nil
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
