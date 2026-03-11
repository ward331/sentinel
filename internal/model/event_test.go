package model

import (
	"encoding/json"
	"testing"
	"time"
)

func TestEvent_JSONMarshalUnmarshal(t *testing.T) {
	event := Event{
		ID:          "test-123",
		Title:       "M 5.0 - Near Tokyo",
		Description: "A magnitude 5.0 earthquake occurred near Tokyo",
		Source:      "usgs",
		SourceID:    "usgs-abc",
		OccurredAt:  time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC),
		IngestedAt:  time.Date(2026, 3, 10, 12, 1, 0, 0, time.UTC),
		Location:    Point(139.7, 35.6),
		Precision:   PrecisionExact,
		Magnitude:   5.0,
		Category:    "earthquake",
		Severity:    SeverityHigh,
		Metadata: map[string]string{
			"usgs_id": "abc123",
			"depth":   "10.5",
		},
		Badges: []Badge{
			{Label: "usgs", Type: BadgeTypeSource, Timestamp: time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)},
		},
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var unmarshaled Event
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if unmarshaled.ID != event.ID {
		t.Errorf("ID mismatch: got %q, want %q", unmarshaled.ID, event.ID)
	}
	if unmarshaled.Title != event.Title {
		t.Errorf("Title mismatch: got %q, want %q", unmarshaled.Title, event.Title)
	}
	if unmarshaled.Source != event.Source {
		t.Errorf("Source mismatch: got %q, want %q", unmarshaled.Source, event.Source)
	}
	if unmarshaled.Magnitude != event.Magnitude {
		t.Errorf("Magnitude mismatch: got %f, want %f", unmarshaled.Magnitude, event.Magnitude)
	}
	if unmarshaled.Category != event.Category {
		t.Errorf("Category mismatch: got %q, want %q", unmarshaled.Category, event.Category)
	}
	if unmarshaled.Severity != event.Severity {
		t.Errorf("Severity mismatch: got %q, want %q", unmarshaled.Severity, event.Severity)
	}
	if len(unmarshaled.Metadata) != 2 {
		t.Errorf("expected 2 metadata entries, got %d", len(unmarshaled.Metadata))
	}
	if len(unmarshaled.Badges) != 1 {
		t.Errorf("expected 1 badge, got %d", len(unmarshaled.Badges))
	}
}

func TestEvent_OptionalFieldsOmitted(t *testing.T) {
	event := Event{
		ID:          "minimal-1",
		Title:       "Minimal Event",
		Description: "No optional fields",
		Source:      "test",
		OccurredAt:  time.Now().UTC(),
		IngestedAt:  time.Now().UTC(),
		Location:    Point(0, 0),
		Precision:   PrecisionUnknown,
		// No magnitude, category, severity, metadata, badges
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Check that omitempty fields are not present
	var raw map[string]interface{}
	json.Unmarshal(data, &raw)

	if _, exists := raw["source_id"]; exists {
		t.Error("source_id should be omitted when empty")
	}
	if _, exists := raw["metadata"]; exists {
		t.Error("metadata should be omitted when nil")
	}
	if _, exists := raw["badges"]; exists {
		t.Error("badges should be omitted when nil")
	}
}

func TestPoint(t *testing.T) {
	loc := Point(-74.0, 40.7)
	if loc.Type != "Point" {
		t.Errorf("expected Point type, got %s", loc.Type)
	}

	coords, ok := loc.Coordinates.([]float64)
	if !ok {
		t.Fatalf("expected []float64 coordinates, got %T", loc.Coordinates)
	}
	if len(coords) != 2 {
		t.Fatalf("expected 2 coordinates, got %d", len(coords))
	}
	if coords[0] != -74.0 {
		t.Errorf("expected lon -74.0, got %f", coords[0])
	}
	if coords[1] != 40.7 {
		t.Errorf("expected lat 40.7, got %f", coords[1])
	}

	if len(loc.BBox) != 4 {
		t.Errorf("expected 4-element bbox, got %d", len(loc.BBox))
	}
}

func TestSeverityConstants(t *testing.T) {
	if SeverityLow != "low" {
		t.Errorf("expected 'low', got %s", SeverityLow)
	}
	if SeverityMedium != "medium" {
		t.Errorf("expected 'medium', got %s", SeverityMedium)
	}
	if SeverityHigh != "high" {
		t.Errorf("expected 'high', got %s", SeverityHigh)
	}
	if SeverityCritical != "critical" {
		t.Errorf("expected 'critical', got %s", SeverityCritical)
	}
}

func TestPrecisionConstants(t *testing.T) {
	if PrecisionExact != "exact" {
		t.Errorf("expected 'exact', got %s", PrecisionExact)
	}
	if PrecisionUnknown != "unknown" {
		t.Errorf("expected 'unknown', got %s", PrecisionUnknown)
	}
}
