package storage

import "github.com/and161185/metrics-alerting/model"

type Storage interface {
	Save(metric model.Metric) error
	Get(metric model.Metric) (model.Metric, error)
	GetAll() (map[string]model.Metric, error)
}
