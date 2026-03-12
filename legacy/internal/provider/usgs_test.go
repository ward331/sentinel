package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUSGSProvider_Name(t *testing.T) {
	p := NewUSGSProvider(&Config{Enabled: true})
	if p.Name() != "usgs" {
		t.Errorf("expected name 'usgs', got %q", p.Name())
	}
}

func TestUSGSProvider_Enabled(t *testing.T) {
	p := NewUSGSProvider(&Config{Enabled: true})
	if !p.Enabled() {
		t.Error("expected enabled=true")
	}

	p2 := NewUSGSProvider(&Config{Enabled: false})
	if p2.Enabled() {
		t.Error("expected enabled=false")
	}
}

func TestUSGSProvider_ParseGeoJSON(t *testing.T) {
	// Serve a canned USGS GeoJSON response
	cannedResponse := USGSGeoJSON{
		Type: "FeatureCollection",
		Metadata: USGSMetadata{
			Generated: 1700000000000,
			URL:       "https://earthquake.usgs.gov/test",
			Title:     "Test Feed",
			Status:    200,
			Count:     2,
		},
		Features: []USGSFeature{
			{
				Type: "Feature",
				ID:   "us2023abc1",
				Properties: USGSProperties{
					Mag:     6.2,
					Place:   "50km NW of Tokyo, Japan",
					Time:    1700000000000,
					URL:     "https://earthquake.usgs.gov/earthquakes/eventpage/us2023abc1",
					Detail:  "https://earthquake.usgs.gov/fdsnws/event/1/query?eventid=us2023abc1",
					Status:  "reviewed",
					Tsunami: 0,
					Sig:     591,
					Net:     "us",
					Code:    "abc1",
					MagType: "mww",
					Type:    "earthquake",
					Depth:   15.3,
				},
				Geometry: USGSGeometry{
					Type:        "Point",
					Coordinates: []float64{139.7, 35.6, 15.3},
				},
			},
			{
				Type: "Feature",
				ID:   "us2023abc2",
				Properties: USGSProperties{
					Mag:   3.1,
					Place: "10km S of Los Angeles, CA",
					Time:  1700000100000,
					Net:   "us",
					Code:  "abc2",
					Type:  "earthquake",
					Depth: 8.0,
				},
				Geometry: USGSGeometry{
					Type:        "Point",
					Coordinates: []float64{-118.2, 34.0, 8.0},
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cannedResponse)
	}))
	defer server.Close()

	p := NewUSGSProvider(&Config{Enabled: true})
	p.feedURL = server.URL

	events, err := p.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	// Verify first event
	ev := events[0]
	if ev.Source != "usgs" {
		t.Errorf("expected source 'usgs', got %q", ev.Source)
	}
	if ev.SourceID != "us2023abc1" {
		t.Errorf("expected source_id 'us2023abc1', got %q", ev.SourceID)
	}
	if ev.Magnitude != 6.2 {
		t.Errorf("expected magnitude 6.2, got %f", ev.Magnitude)
	}
	if ev.Category != "earthquake" {
		t.Errorf("expected category 'earthquake', got %q", ev.Category)
	}
	if ev.Severity != "critical" {
		t.Errorf("expected severity 'critical' for M6.2, got %q", ev.Severity)
	}

	// Verify second event severity
	ev2 := events[1]
	if ev2.Severity != "low" {
		t.Errorf("expected severity 'low' for M3.1, got %q", ev2.Severity)
	}
}

func TestUSGSProvider_InvalidCoordinates(t *testing.T) {
	cannedResponse := USGSGeoJSON{
		Type: "FeatureCollection",
		Features: []USGSFeature{
			{
				Type: "Feature",
				ID:   "bad",
				Properties: USGSProperties{
					Mag:   1.0,
					Place: "Unknown",
					Time:  1700000000000,
				},
				Geometry: USGSGeometry{
					Type:        "Point",
					Coordinates: []float64{}, // invalid: empty
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(cannedResponse)
	}))
	defer server.Close()

	p := NewUSGSProvider(&Config{Enabled: true})
	p.feedURL = server.URL

	events, err := p.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	// Invalid coordinates should be skipped, not crash
	if len(events) != 0 {
		t.Errorf("expected 0 events for invalid coordinates, got %d", len(events))
	}
}

func TestUSGSProvider_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	p := NewUSGSProvider(&Config{Enabled: true})
	p.feedURL = server.URL

	_, err := p.Fetch(context.Background())
	if err == nil {
		t.Error("expected error for 500 response")
	}
}
