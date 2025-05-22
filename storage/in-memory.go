package storage

import (
	"errors"

	"github.com/and161185/metrics-alerting/model"
)

var ErrMetricNotFound = errors.New("metric not found")

type MemStorage struct {
	metrics map[string]*model.Metric
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		metrics: make(map[string]*model.Metric),
	}
}

// TODO: sync access if used concurrently

func (store *MemStorage) Save(m *model.Metric) error {
	existing, ok := store.metrics[m.ID]
	if !ok {
		store.metrics[m.ID] = m
	} else if m.Type == model.Gauge {
		store.metrics[m.ID] = m
	} else if m.Type == model.Counter && m.Delta != nil {
		if existing.Delta != nil {
			*existing.Delta += *m.Delta
		} else {
			v := *m.Delta
			existing.Delta = &v
		}
	}
	return nil
}

func (store *MemStorage) Get(m *model.Metric) (*model.Metric, error) {
	val, ok := store.metrics[m.ID]

	if !ok {
		return m, ErrMetricNotFound
	}
	return val, nil
}

func (store *MemStorage) GetAll() (map[string]*model.Metric, error) {
	result := make(map[string]*model.Metric, len(store.metrics))
	for k, v := range store.metrics {
		result[k] = v
	}
	return result, nil
}
