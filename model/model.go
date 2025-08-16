// Package model contains core data types for the project.
package model

// MetricType defines the type of a metric: gauge or counter.
type MetricType string

const (
	Gauge   MetricType = "gauge"   // Gauge represents a float64 metric.
	Counter MetricType = "counter" // Counter represents an int64 metric.
)

// Metric represents a single metric with its ID, type, and value.
type Metric struct {
	ID    string     `json:"id"`              // Metric name.
	Type  MetricType `json:"type"`            // Metric type: gauge or counter.
	Delta *int64     `json:"delta,omitempty"` // Value for counter metrics.
	Value *float64   `json:"value,omitempty"` // Value for gauge metrics.
}
