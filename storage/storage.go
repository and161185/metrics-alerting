package storage

import "github.com/and161185/metrics-alerting/model"

type Storage interface {
	Save(metric *model.Metric) error
	//Get(id string) (*model.Metric, error)
}
