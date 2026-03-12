package model

// Anomaly represents a detected ingestion-rate anomaly for a provider/region.
type Anomaly struct {
	ID           int64   `json:"id"`
	ProviderName string  `json:"provider_name"`
	Region       string  `json:"region"`
	ExpectedRate float64 `json:"expected_rate"`
	ActualRate   float64 `json:"actual_rate"`
	SpikeFactor  float64 `json:"spike_factor"`
	DetectedAt   string  `json:"detected_at"`
	ResolvedAt   string  `json:"resolved_at,omitempty"`
}
