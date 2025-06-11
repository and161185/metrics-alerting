package postgres

import (
	"context"

	"github.com/and161185/metrics-alerting/model"
	"github.com/and161185/metrics-alerting/storage/inmemory"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStorage struct {
	db *pgxpool.Pool
	ms *inmemory.MemStorage
}

func NewPostgresStorage(DatabaseDsn string) (*PostgresStorage, error) {
	db, err := pgxpool.New(context.Background(), DatabaseDsn)
	if err != nil {
		return nil, err
	}
	return &PostgresStorage{db: db, ms: inmemory.NewMemStorage()}, nil
}

func (store *PostgresStorage) Save(m *model.Metric) error {
	return store.ms.Save(m)
}

func (store *PostgresStorage) Get(m *model.Metric) (*model.Metric, error) {
	return store.ms.Get(m)
}

func (store *PostgresStorage) GetAll() (map[string]*model.Metric, error) {
	return store.ms.GetAll()
}

func (store *PostgresStorage) SaveToFile(filePath string) error {
	return store.ms.SaveToFile(filePath)
}

func (store *PostgresStorage) LoadFromFile(filePath string) error {
	return store.ms.LoadFromFile(filePath)
}

func (store *PostgresStorage) Ping(ctx context.Context) error {
	return store.db.Ping(ctx)
}
