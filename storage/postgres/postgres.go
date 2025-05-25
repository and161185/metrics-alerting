package postgres

import (
	"context"

	"github.com/and161185/metrics-alerting/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStorage struct {
	db *pgxpool.Pool
}

func NewPostgresStorage(DatabaseDsn string) (*PostgresStorage, error) {
	db, err := pgxpool.New(context.Background(), DatabaseDsn)
	if err != nil {
		return nil, err
	}
	return &PostgresStorage{db: db}, nil
}

func (store *PostgresStorage) Save(m *model.Metric) error {
	return nil
}

func (store *PostgresStorage) Get(m *model.Metric) (*model.Metric, error) {
	return nil, nil
}

func (store *PostgresStorage) GetAll() (map[string]*model.Metric, error) {
	return nil, nil
}

func (store *PostgresStorage) SaveToFile(filePath string) error {
	return nil
}

func (store *PostgresStorage) LoadFromFile(filePath string) error {
	return nil
}

func (store *PostgresStorage) Ping(ctx context.Context) error {
	return store.db.Ping(ctx)
}
