package engine

import (
	"time"
)

// CorrelationFlash groups events from 3+ sources in the same region within 60 minutes.
type CorrelationFlash struct {
	ID          int64     `json:"id"`
	RegionName  string    `json:"region_name"`
	Lat         float64   `json:"lat"`
	Lon         float64   `json:"lon"`
	RadiusKm    float64   `json:"radius_km"`
	EventCount  int       `json:"event_count"`
	SourceCount int       `json:"source_count"`
	StartedAt   time.Time `json:"started_at"`
	LastEventAt time.Time `json:"last_event_at"`
	Confirmed   bool      `json:"confirmed"`
	IncidentName string   `json:"incident_name,omitempty"`
	EventIDs    []string  `json:"event_ids,omitempty"`
}

// CorrelationEngine detects correlated events across multiple sources.
type CorrelationEngine struct {
	windowMinutes int
	minSources    int
	radiusKm      float64
}

// NewCorrelationEngine creates a new engine with default settings:
// 60-minute window, 3+ sources, 50km radius.
func NewCorrelationEngine() *CorrelationEngine {
	return &CorrelationEngine{
		windowMinutes: 60,
		minSources:    3,
		radiusKm:      50.0,
	}
}

// Evaluate checks recent events for correlations.
// Placeholder — full implementation in G3.
func (e *CorrelationEngine) Evaluate() ([]CorrelationFlash, error) {
	// TODO: query recent events, cluster by region, detect multi-source flashes
	return nil, nil
}
