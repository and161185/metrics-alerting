package storage

import (
	"errors"

	"github.com/and161185/metrics-alerting/model"
)

var ErrMetricNotFound = errors.New("metric not found")

type MemStorage struct {
	metrics map[string]model.Metric
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		metrics: make(map[string]model.Metric),
	}
}

func (s *MemStorage) Save(m model.Metric) error {
	_, ok := s.metrics[m.ID]
	if !ok {
		s.metrics[m.ID] = m
	} else if m.Type == model.Gauge {
		s.metrics[m.ID] = m
	} else if m.Type == model.Counter {
		existing := s.metrics[m.ID]
		existing.Value += m.Value
		s.metrics[m.ID] = existing
	}
	return nil
}

func (s *MemStorage) Get(m model.Metric) (model.Metric, error) {
	val, ok := s.metrics[m.ID]

	if !ok {
		return m, ErrMetricNotFound
	}
	return val, nil
}

func (s *MemStorage) GetAll() (map[string]model.Metric, error) {
	result := make(map[string]model.Metric, len(s.metrics))
	for k, v := range s.metrics {
		result[k] = v
	}
	return result, nil
}
