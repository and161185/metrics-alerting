package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/and161185/metrics-alerting/model"
	"github.com/and161185/metrics-alerting/storage/inmemory"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStorage struct {
	db *pgxpool.Pool
	ms *inmemory.MemStorage
}

func NewPostgresStorage(ctx context.Context, DatabaseDsn string) (*PostgresStorage, error) {
	db, err := pgxpool.New(ctx, DatabaseDsn)
	if err != nil {
		return nil, err
	}

	storage := &PostgresStorage{db: db, ms: inmemory.NewMemStorage(ctx)}

	if err := storage.Ping(ctx); err != nil {
		return nil, err
	}

	if err := storage.initSchema(ctx); err != nil {
		return nil, err
	}

	return storage, nil
}

func (store *PostgresStorage) initSchema(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS metrics (
		id TEXT PRIMARY KEY,
		mtype TEXT NOT NULL,
		delta BIGINT,
		value DOUBLE PRECISION
	);`
	_, err := store.db.Exec(ctx, query)
	return err
}

func (store *PostgresStorage) Save(ctx context.Context, m *model.Metric) error {
	_, err := store.db.Exec(ctx, `INSERT INTO metrics (id, mtype, delta, value)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE
		SET mtype = EXCLUDED.mtype,
			delta = EXCLUDED.delta,
			value = EXCLUDED.value;`, m.ID, string(m.Type), m.Delta, m.Value)

	return err
}

func (store *PostgresStorage) Get(ctx context.Context, m *model.Metric) (*model.Metric, error) {
	row := store.db.QueryRow(ctx, `SELECT id, mtype, delta, value FROM metrics 
		WHERE id = $1`, m.ID)

	var val model.Metric
	var mtype string
	err := row.Scan(&val.ID, &mtype, &val.Delta, &val.Value)
	if err != nil {
		return nil, err
	}

	val.Type = model.MetricType(mtype)

	return &val, nil
}

func (store *PostgresStorage) GetAll(ctx context.Context) (map[string]*model.Metric, error) {
	rows, err := store.db.Query(ctx, `SELECT id, mtype, delta, value FROM metrics`)
	defer rows.Close()

	if err != nil {
		return nil, err
	}

	result := make(map[string]*model.Metric)
	for rows.Next() {
		var m model.Metric
		var mtype string

		err := rows.Scan(&m.ID, &mtype, &m.Delta, &m.Value)
		if err != nil {
			return nil, err
		}

		m.Type = model.MetricType(mtype)

		copy := m
		result[m.ID] = &copy
	}

	return result, nil
}

func (store *PostgresStorage) SaveToFile(ctx context.Context, filePath string) error {
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

func (store *PostgresStorage) LoadFromFile(ctx context.Context, filePath string) error {
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

func (store *PostgresStorage) Ping(ctx context.Context) error {
	return store.db.Ping(ctx)
}
