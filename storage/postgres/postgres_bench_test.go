// postgres_bench_test.go — бенчмарки для моков
package postgres

import (
	"context"
	"testing"

	"github.com/and161185/metrics-alerting/model"
	"github.com/golang/mock/gomock"
)

func BenchmarkMockStorage_Save(b *testing.B) {
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()
	mockStorage := NewMockStorage(ctrl)
	metric := &model.Metric{ID: "bench", Type: model.Gauge}
	mockStorage.EXPECT().Save(gomock.Any(), metric).Return(nil).AnyTimes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = mockStorage.Save(context.Background(), metric)
	}
}

func BenchmarkMockStorage_Get(b *testing.B) {
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()
	mockStorage := NewMockStorage(ctrl)
	metric := &model.Metric{ID: "bench", Type: model.Gauge}
	mockStorage.EXPECT().Get(gomock.Any(), metric).Return(nil, nil).AnyTimes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mockStorage.Get(context.Background(), metric)
	}
}

func BenchmarkMockStorage_GetAll(b *testing.B) {
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()
	mockStorage := NewMockStorage(ctrl)
	mockStorage.EXPECT().GetAll(gomock.Any()).Return(nil, nil).AnyTimes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mockStorage.GetAll(context.Background())
	}
}

func BenchmarkMockStorage_Ping(b *testing.B) {
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()
	mockStorage := NewMockStorage(ctrl)
	mockStorage.EXPECT().Ping(gomock.Any()).Return(nil).AnyTimes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = mockStorage.Ping(context.Background())
	}
}
