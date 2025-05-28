package inmemory

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/and161185/metrics-alerting/internal/errs"
	"github.com/and161185/metrics-alerting/model"
)

type MemStorage struct {
	metrics map[string]*model.Metric
	mu      sync.Mutex
}

func NewMemStorage(ctx context.Context) *MemStorage {
	return &MemStorage{
		metrics: make(map[string]*model.Metric),
	}
}

// TODO: sync access if used concurrently

func (store *MemStorage) Save(ctx context.Context, m *model.Metric) error {
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

func (store *MemStorage) Get(ctx context.Context, m *model.Metric) (*model.Metric, error) {
	val, ok := store.metrics[m.ID]

	if !ok {
		return m, errs.ErrMetricNotFound
	}
	return val, nil
}

func (store *MemStorage) GetAll(ctx context.Context) (map[string]*model.Metric, error) {
	result := make(map[string]*model.Metric, len(store.metrics))
	for k, v := range store.metrics {
		result[k] = v
	}
	return result, nil
}

func (store *MemStorage) SaveToFile(ctx context.Context, filePath string) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	metrics, err := store.GetAll(ctx)

	if err != nil {
		return fmt.Errorf("failed to get metrics: %w", err)
	}

	if len(metrics) == 0 {
		return nil
	}

	data, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	log.Printf("saved to %s", filePath)

	return nil
}

func (store *MemStorage) LoadFromFile(ctx context.Context, filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read file: %w", err)
	}

	var metrics map[string]*model.Metric
	if err := json.Unmarshal(data, &metrics); err != nil {
		return fmt.Errorf("failed to unmarshal metrics: %w", err)
	}

	for _, m := range metrics {
		if err := store.Save(ctx, m); err != nil {
			return fmt.Errorf("failed to restore metric %s: %w", m.ID, err)
		}
	}

	log.Printf("loaded from %s", filePath)

	return nil
}

func (store *MemStorage) Ping(ctx context.Context) error {
	return nil
}
