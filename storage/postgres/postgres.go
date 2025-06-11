package postgres

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/and161185/metrics-alerting/internal/errs"
	"github.com/and161185/metrics-alerting/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStorage struct {
	db *pgxpool.Pool
}

const initSchemaQuery = `
	CREATE TABLE IF NOT EXISTS metrics (
		id TEXT PRIMARY KEY,
		mtype TEXT NOT NULL,
		delta BIGINT,
		value DOUBLE PRECISION
	);`

const mergeMetricsQuery = `INSERT INTO metrics (id, mtype, delta, value)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE
		SET mtype = EXCLUDED.mtype,
			delta = EXCLUDED.delta,
			value = EXCLUDED.value;`

const getMetricQuery = `SELECT id, mtype, delta, value FROM metrics WHERE id = $1`

const getAllMetricsQuery = `SELECT id, mtype, delta, value FROM metrics`

func NewPostgresStorage(ctx context.Context, DatabaseDsn string) (*PostgresStorage, error) {
	db, err := pgxpool.New(ctx, DatabaseDsn)
	if err != nil {
		return nil, err
	}

	storage := &PostgresStorage{db: db}

	if err := storage.Ping(ctx); err != nil {
		return nil, err
	}

	if err := storage.initSchema(ctx); err != nil {
		return nil, err
	}

	return storage, nil
}

func (store *PostgresStorage) initSchema(ctx context.Context) error {
	_, err := store.db.Exec(ctx, initSchemaQuery)
	return err
}

func (store *PostgresStorage) calculateDelta(ctx context.Context, m *model.Metric, getFn func(context.Context, *model.Metric) (*model.Metric, error)) (*int64, error) {
	if m.Type != model.Counter {
		return m.Delta, nil
	}

	currentMetric, err := getFn(ctx, m)
	if err != nil && err != errs.ErrMetricNotFound {
		return m.Delta, err
	}

	if currentMetric != nil && currentMetric.Delta != nil && m.Delta != nil {
		v := *currentMetric.Delta + *m.Delta
		return &v, nil
	}

	return m.Delta, nil
}

func (store *PostgresStorage) Save(ctx context.Context, m *model.Metric) error {

	delta, err := store.calculateDelta(ctx, m, store.Get)
	if err != nil {
		return err
	}

	_, err = store.db.Exec(ctx, mergeMetricsQuery, m.ID, string(m.Type), delta, m.Value)
	if err != nil {
		return err
	}

	m.Delta = delta

	return nil
}

func (store *PostgresStorage) SaveBatch(ctx context.Context, metrics []model.Metric) error {
	tx, err := store.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
				log.Printf("failed to rollback transaction: %v", rollbackErr)
			}
		}
	}()

	for _, m := range metrics {
		delta, err := store.calculateDelta(ctx, &m, func(ctx context.Context, m *model.Metric) (*model.Metric, error) {
			return GetWithTx(ctx, tx, m)
		})
		if err != nil {
			return err
		}

		_, err = tx.Exec(ctx, mergeMetricsQuery, m.ID, string(m.Type), delta, m.Value)
		if err != nil {
			return fmt.Errorf("failed to save metric %s: %w", m.ID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (store *PostgresStorage) Get(ctx context.Context, m *model.Metric) (*model.Metric, error) {
	return getMetric(ctx, store.db, m)
}

func GetWithTx(ctx context.Context, tx pgx.Tx, m *model.Metric) (*model.Metric, error) {
	return getMetric(ctx, tx, m)
}

func getMetric(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...interface{}) pgx.Row
}, m *model.Metric) (*model.Metric, error) {
	row := q.QueryRow(ctx, getMetricQuery, m.ID)

	var val model.Metric
	var mtype string
	err := row.Scan(&val.ID, &mtype, &val.Delta, &val.Value)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.ErrMetricNotFound
		}
		return nil, err
	}
	val.Type = model.MetricType(mtype)

	return &val, nil
}

func (store *PostgresStorage) GetAll(ctx context.Context) (map[string]*model.Metric, error) {
	rows, err := store.db.Query(ctx, getAllMetricsQuery)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

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

func (store *PostgresStorage) Ping(ctx context.Context) error {
	return store.db.Ping(ctx)
}
