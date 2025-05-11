package collector

import (
	"math/rand/v2"
	"runtime"

	"github.com/and161185/metrics-alerting/internal/utils"
	"github.com/and161185/metrics-alerting/model"
)

var pollCount int64

func CollectRuntimeMetrics() []model.Metric {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	pollCount++

	res := []model.Metric{
		{ID: "Alloc", Type: model.Gauge, Value: utils.F64Ptr(float64(m.Alloc))},
		{ID: "BuckHashSys", Type: model.Gauge, Value: utils.F64Ptr(float64(m.BuckHashSys))},
		{ID: "Frees", Type: model.Gauge, Value: utils.F64Ptr(float64(m.Frees))},
		{ID: "GCCPUFraction", Type: model.Gauge, Value: utils.F64Ptr(m.GCCPUFraction)},
		{ID: "GCSys", Type: model.Gauge, Value: utils.F64Ptr(float64(m.GCSys))},
		{ID: "HeapAlloc", Type: model.Gauge, Value: utils.F64Ptr(float64(m.HeapAlloc))},
		{ID: "HeapIdle", Type: model.Gauge, Value: utils.F64Ptr(float64(m.HeapIdle))},
		{ID: "HeapInuse", Type: model.Gauge, Value: utils.F64Ptr(float64(m.HeapInuse))},
		{ID: "HeapObjects", Type: model.Gauge, Value: utils.F64Ptr(float64(m.HeapObjects))},
		{ID: "HeapReleased", Type: model.Gauge, Value: utils.F64Ptr(float64(m.HeapReleased))},
		{ID: "HeapSys", Type: model.Gauge, Value: utils.F64Ptr(float64(m.HeapSys))},
		{ID: "LastGC", Type: model.Gauge, Value: utils.F64Ptr(float64(m.LastGC))},
		{ID: "Lookups", Type: model.Gauge, Value: utils.F64Ptr(float64(m.Lookups))},
		{ID: "MCacheInuse", Type: model.Gauge, Value: utils.F64Ptr(float64(m.MCacheInuse))},
		{ID: "MCacheSys", Type: model.Gauge, Value: utils.F64Ptr(float64(m.MCacheSys))},
		{ID: "MSpanInuse", Type: model.Gauge, Value: utils.F64Ptr(float64(m.MSpanInuse))},
		{ID: "MSpanSys", Type: model.Gauge, Value: utils.F64Ptr(float64(m.MSpanSys))},
		{ID: "Mallocs", Type: model.Gauge, Value: utils.F64Ptr(float64(m.Mallocs))},
		{ID: "NextGC", Type: model.Gauge, Value: utils.F64Ptr(float64(m.NextGC))},
		{ID: "NumForcedGC", Type: model.Gauge, Value: utils.F64Ptr(float64(m.NumForcedGC))},
		{ID: "NumGC", Type: model.Gauge, Value: utils.F64Ptr(float64(m.NumGC))},
		{ID: "OtherSys", Type: model.Gauge, Value: utils.F64Ptr(float64(m.OtherSys))},
		{ID: "PauseTotalNs", Type: model.Gauge, Value: utils.F64Ptr(float64(m.PauseTotalNs))},
		{ID: "StackInuse", Type: model.Gauge, Value: utils.F64Ptr(float64(m.StackInuse))},
		{ID: "StackSys", Type: model.Gauge, Value: utils.F64Ptr(float64(m.StackSys))},
		{ID: "Sys", Type: model.Gauge, Value: utils.F64Ptr(float64(m.Sys))},
		{ID: "TotalAlloc", Type: model.Gauge, Value: utils.F64Ptr(float64(m.TotalAlloc))},

		{ID: "PollCount", Type: model.Counter, Delta: utils.I64Ptr(pollCount)},
		{ID: "RandomValue", Type: model.Gauge, Value: utils.F64Ptr(rand.Float64())},
	}

	return res
}

func ResetPollCount() {
	pollCount = 0
}
