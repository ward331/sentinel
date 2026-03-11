package engine

import (
	"time"
)

// Anomaly represents a detected spike above the rolling baseline.
type Anomaly struct {
	ID           int64      `json:"id"`
	ProviderName string     `json:"provider_name"`
	Region       string     `json:"region"`
	ExpectedRate float64    `json:"expected_rate"`
	ActualRate   float64    `json:"actual_rate"`
	SpikeFactor  float64    `json:"spike_factor"`
	DetectedAt   time.Time  `json:"detected_at"`
	ResolvedAt   *time.Time `json:"resolved_at,omitempty"`
}

// AnomalyDetector maintains a rolling 24-hour baseline per provider per region
// and fires an alert when the event rate exceeds 3x the baseline.
type AnomalyDetector struct {
	spikeThreshold float64 // default 3.0x
	windowHours    int     // default 24
}

// NewAnomalyDetector creates a detector with default thresholds.
func NewAnomalyDetector() *AnomalyDetector {
	return &AnomalyDetector{
		spikeThreshold: 3.0,
		windowHours:    24,
	}
}

// Detect checks for anomalous event rates.
// Placeholder — full implementation in G3.
func (d *AnomalyDetector) Detect() ([]Anomaly, error) {
	// TODO: query event counts per provider per region over windowHours,
	//       compare against rolling average, flag 3x+ spikes
	return nil, nil
}
