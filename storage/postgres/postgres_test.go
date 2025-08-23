// postgres_test.go — тесты для моков
package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/and161185/metrics-alerting/internal/errs"
	"github.com/and161185/metrics-alerting/model"
	"github.com/golang/mock/gomock"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestMockStorage_Save(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStorage := NewMockStorage(ctrl)

	metric := &model.Metric{ID: "test", Type: model.Gauge}
	mockStorage.EXPECT().Save(gomock.Any(), metric).Return(nil)

	if err := mockStorage.Save(context.Background(), metric); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMockStorage_Get(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStorage := NewMockStorage(ctrl)

	metric := &model.Metric{ID: "test", Type: model.Gauge}
	mockStorage.EXPECT().Get(gomock.Any(), metric).Return(nil, nil)

	_, err := mockStorage.Get(context.Background(), metric)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMockStorage_GetAll(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStorage := NewMockStorage(ctrl)

	mockStorage.EXPECT().GetAll(gomock.Any()).Return(nil, nil)

	_, err := mockStorage.GetAll(context.Background())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMockStorage_Ping(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStorage := NewMockStorage(ctrl)

	mockStorage.EXPECT().Ping(gomock.Any()).Return(nil)

	if err := mockStorage.Ping(context.Background()); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

type fakeRow struct {
	scan func(dest ...any) error
}

func (r fakeRow) Scan(dest ...any) error { return r.scan(dest...) }

type fakeQuerier struct {
	row pgx.Row
}

func (q fakeQuerier) QueryRow(context.Context, string, ...any) pgx.Row { return q.row }

func Test_getMetric_GaugeOK(t *testing.T) {
	q := fakeQuerier{
		row: fakeRow{scan: func(dest ...any) error {
			*(dest[0].(*string)) = "g1"
			*(dest[1].(*string)) = "gauge"
			// delta NULL
			*(dest[2].(**int64)) = nil
			// value = 1.5
			v := 1.5
			*(dest[3].(**float64)) = &v
			return nil
		}},
	}
	m, err := getMetric(context.Background(), q, &model.Metric{ID: "g1"})
	require.NoError(t, err)
	require.Equal(t, "g1", m.ID)
	require.Equal(t, model.Gauge, m.Type)
	require.Nil(t, m.Delta)
	require.NotNil(t, m.Value)
	require.InDelta(t, 1.5, *m.Value, 1e-9)
}

func Test_getMetric_CounterOK(t *testing.T) {
	q := fakeQuerier{
		row: fakeRow{scan: func(dest ...any) error {
			*(dest[0].(*string)) = "c1"
			*(dest[1].(*string)) = "counter"
			d := int64(7)
			*(dest[2].(**int64)) = &d
			*(dest[3].(**float64)) = nil
			return nil
		}},
	}
	m, err := getMetric(context.Background(), q, &model.Metric{ID: "c1"})
	require.NoError(t, err)
	require.Equal(t, "c1", m.ID)
	require.Equal(t, model.Counter, m.Type)
	require.NotNil(t, m.Delta)
	require.EqualValues(t, 7, *m.Delta)
	require.Nil(t, m.Value)
}

func Test_getMetric_NotFound(t *testing.T) {
	q := fakeQuerier{
		row: fakeRow{scan: func(dest ...any) error {
			return pgx.ErrNoRows
		}},
	}
	_, err := getMetric(context.Background(), q, &model.Metric{ID: "absent"})
	require.ErrorIs(t, err, errs.ErrMetricNotFound)
}

func Test_getMetric_ScanError(t *testing.T) {
	q := fakeQuerier{
		row: fakeRow{scan: func(dest ...any) error {
			return errors.New("boom")
		}},
	}
	_, err := getMetric(context.Background(), q, &model.Metric{ID: "x"})
	require.Error(t, err)
}

func Test_calculateDelta_NonCounter(t *testing.T) {
	ps := &PostgresStorage{}
	v := int64(3)
	m := &model.Metric{Type: model.Gauge, Delta: &v}
	got, err := ps.calculateDelta(context.Background(), m, func(context.Context, *model.Metric) (*model.Metric, error) {
		t.Fatal("should not be called for gauge")
		return nil, nil
	})
	require.NoError(t, err)
	require.Equal(t, &v, got)
}

func Test_calculateDelta_Counter_AddsExisting(t *testing.T) {
	ps := &PostgresStorage{}
	curD := int64(10)
	newD := int64(2)
	m := &model.Metric{Type: model.Counter, Delta: &newD}
	got, err := ps.calculateDelta(context.Background(), m, func(context.Context, *model.Metric) (*model.Metric, error) {
		return &model.Metric{Type: model.Counter, Delta: &curD}, nil
	})
	require.NoError(t, err)
	require.NotNil(t, got)
	require.EqualValues(t, 12, *got)
}

func Test_calculateDelta_Counter_NoExistingOrNil(t *testing.T) {
	ps := &PostgresStorage{}
	newD := int64(5)
	m := &model.Metric{Type: model.Counter, Delta: &newD}
	got, err := ps.calculateDelta(context.Background(), m, func(context.Context, *model.Metric) (*model.Metric, error) {
		return nil, errs.ErrMetricNotFound
	})
	require.NoError(t, err)
	require.Equal(t, &newD, got)
}

func Test_calculateDelta_PropagatesUnexpectedError(t *testing.T) {
	ps := &PostgresStorage{}
	m := &model.Metric{Type: model.Counter}
	_, err := ps.calculateDelta(context.Background(), m, func(context.Context, *model.Metric) (*model.Metric, error) {
		return nil, errors.New("db down")
	})
	require.Error(t, err)
}
