package model

import (
	"context"
	"encoding/json"
	"time"
)

// Precision represents the location precision level
type Precision string

const (
	PrecisionExact        Precision = "exact"
	PrecisionPolygonArea  Precision = "polygon_area"
	PrecisionApproximate  Precision = "approximate"
	PrecisionTextInferred Precision = "text_inferred"
	PrecisionUnknown      Precision = "unknown"
)

// BadgeType represents the type of badge
type BadgeType string

const (
	BadgeTypeSource    BadgeType = "source"
	BadgeTypePrecision BadgeType = "precision"
	BadgeTypeFreshness BadgeType = "freshness"
	BadgeTypeFilter    BadgeType = "filter"
)

// Severity represents event severity level
type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// Location represents a geographic location using GeoJSON format
type Location struct {
	Type        string      `json:"type"` // "Point" or "Polygon"
	Coordinates interface{} `json:"coordinates"`
	BBox        []float64   `json:"bbox,omitempty"`
}

// GeoJSON is an alias for Location for backward compatibility
type GeoJSON = Location

// Point creates a Point location
func Point(lon, lat float64) Location {
	return Location{
		Type:        "Point",
		Coordinates: []float64{lon, lat},
		BBox:        []float64{lon, lat, lon, lat},
	}
}

// Badge represents a metadata badge for an event
type Badge struct {
	Label     string    `json:"label"`
	Type      BadgeType `json:"type"`
	Timestamp time.Time `json:"timestamp"`
}

// Event represents a global event (earthquake, wildfire, etc.)
type Event struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Source      string            `json:"source"`
	SourceID    string            `json:"source_id,omitempty"`
	OccurredAt  time.Time         `json:"occurred_at"`
	IngestedAt  time.Time         `json:"ingested_at"`
	Location    Location          `json:"location"`
	Precision   Precision         `json:"precision"`
	Magnitude   float64           `json:"magnitude,omitempty"`
	Category    string            `json:"category,omitempty"`
	Severity    Severity          `json:"severity,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Badges      []Badge           `json:"badges,omitempty"`
}

// EventInput represents event data for ingestion
type EventInput struct {
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Source      string            `json:"source"`
	SourceID    string            `json:"source_id,omitempty"`
	OccurredAt  time.Time         `json:"occurred_at"`
	Location    Location          `json:"location"`
	Precision   Precision         `json:"precision"`
	Magnitude   float64           `json:"magnitude,omitempty"`
	Category    string            `json:"category,omitempty"`
	Severity    Severity          `json:"severity,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// Provider defines the interface for event providers (USGS, GDACS, etc.)
type Provider interface {
	// Fetch retrieves events from the provider
	Fetch(ctx context.Context) ([]*Event, error)
	// Name returns the provider identifier
	Name() string
}

// MarshalJSON custom marshaling for Location to handle coordinates properly
func (l Location) MarshalJSON() ([]byte, error) {
	type Alias Location
	return json.Marshal(&struct {
		Coordinates json.RawMessage `json:"coordinates"`
		*Alias
	}{
		Coordinates: json.RawMessage(marshalCoordinates(l.Coordinates)),
		Alias:       (*Alias)(&l),
	})
}

func marshalCoordinates(coords interface{}) []byte {
	b, _ := json.Marshal(coords)
	return b
}