package qcdashboard

// Metric represents a single quality-control metric.
type Metric struct {
	Name   string
	Value  float64
	Min    float64
	Max    float64
	Unit   string
	Status string // "pass", "warn", "fail"
}
