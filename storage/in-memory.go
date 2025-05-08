package storage

import (
	"github.com/and161185/metrics-alerting/model"
)

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

func (s *MemStorage) GetAll() map[string]model.Metric {
	result := make(map[string]model.Metric, len(s.metrics))
	for k, v := range s.metrics {
		result[k] = v
	}
	return result
}
