package model

// CorrelationFlash represents a geographic cluster of events from multiple sources.
type CorrelationFlash struct {
	ID           int64        `json:"id"`
	RegionName   string       `json:"region_name"`
	Lat          float64      `json:"lat"`
	Lon          float64      `json:"lon"`
	RadiusKm     float64      `json:"radius_km"`
	EventCount   int          `json:"event_count"`
	SourceCount  int          `json:"source_count"`
	StartedAt    string       `json:"started_at"`
	LastEventAt  string       `json:"last_event_at"`
	Confirmed    bool         `json:"confirmed"`
	IncidentName string       `json:"incident_name,omitempty"`
	Events       []EventBrief `json:"events"`
}

// EventBrief is a lightweight event summary used inside correlation flashes.
type EventBrief struct {
	Source     string `json:"source"`
	Title      string `json:"title"`
	OccurredAt string `json:"occurred_at"`
}
