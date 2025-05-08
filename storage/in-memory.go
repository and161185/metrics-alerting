package storage

import (
	"github.com/and161185/metrics-alerting/model"
)

type MemStorage struct {
	metrics map[string]float64
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		metrics: make(map[string]float64),
	}
}

func (s *MemStorage) Save(m *model.Metric) error {
	_, ok := s.metrics[m.ID]
	if !ok {
		s.metrics[m.ID] = m.Value
	} else if m.Type == model.Gauge {
		s.metrics[m.ID] = m.Value
	} else if m.Type == model.Counter {
		s.metrics[m.ID] += m.Value
	}
	return nil
}
