// inmemory_bench_test.go — только бенчмарки
package inmemory

import (
	"context"
	"testing"

	"github.com/and161185/metrics-alerting/internal/utils"
	"github.com/and161185/metrics-alerting/model"
)

func BenchmarkSaveGauge(b *testing.B) {
	ctx := context.Background()
	st := NewMemStorage(ctx)
	m := &model.Metric{ID: "TestGauge", Type: model.Gauge, Value: utils.F64Ptr(42.0)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = st.Save(ctx, m)
	}
}

func BenchmarkSaveCounter(b *testing.B) {
	ctx := context.Background()
	st := NewMemStorage(ctx)
	m := &model.Metric{ID: "TestCounter", Type: model.Counter, Delta: utils.I64Ptr(1)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = st.Save(ctx, m)
	}
}

func BenchmarkGetAll(b *testing.B) {
	ctx := context.Background()
	st := NewMemStorage(ctx)
	for i := 0; i < 100; i++ {
		id := "metric" + string(rune(i))
		_ = st.Save(ctx, &model.Metric{ID: id, Type: model.Gauge, Value: utils.F64Ptr(float64(i))})
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = st.GetAll(ctx)
	}
}
