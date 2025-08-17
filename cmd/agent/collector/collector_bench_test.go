package collector

import (
	"testing"
)

func BenchmarkCollectRuntimeMetrics(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = CollectRuntimeMetrics()
	}
}
