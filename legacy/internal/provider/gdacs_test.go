package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGDACSProvider_Name(t *testing.T) {
	p := NewGDACSProvider(&Config{Enabled: true})
	if p.Name() != "gdacs" {
		t.Errorf("expected name 'gdacs', got %q", p.Name())
	}
}

func TestGDACSProvider_ParseGeoJSON(t *testing.T) {
	mag := 7.1
	cannedResponse := GDACSResponse{
		Type: "FeatureCollection",
		Features: []GDACSFeature{
			{
				Type: "Feature",
				Properties: GDACSProperties{
					EventID:    12345,
					EventName:  "Test Earthquake",
					EventType:  "EQ",
					AlertLevel: "Red",
					AlertScore: 3,
					Country:    "Japan",
					FromDate:   "2026-03-10T08:30:00",
					ToDate:     "2026-03-10T10:00:00",
					Magnitude:  &mag,
					Name:       "M7.1 Earthquake Japan",
					SeverityData: &GDACSSeverityData{
						Severity:     7.1,
						SeverityText: "Magnitude 7.1",
						SeverityUnit: "M",
					},
					AffectedCountries: []GDACSCountry{
						{CountryName: "Japan", ISO2: "JP", ISO3: "JPN"},
					},
				},
				Geometry: GDACSGeometry{
					Type:        "Point",
					Coordinates: []float64{139.7, 35.6},
				},
			},
			{
				Type: "Feature",
				Properties: GDACSProperties{
					EventID:    12346,
					EventName:  "Tropical Cyclone",
					EventType:  "TC",
					AlertLevel: "Orange",
					AlertScore: 2,
					Country:    "Philippines",
					FromDate:   "2026-03-09T00:00:00",
					Name:       "TC Warning Philippines",
				},
				Geometry: GDACSGeometry{
					Type:        "Point",
					Coordinates: []float64{121.0, 14.6},
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cannedResponse)
	}))
	defer server.Close()

	p := NewGDACSProvider(&Config{Enabled: true})
	p.feedURL = server.URL

	events, err := p.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	// First event: earthquake, Red alert -> critical
	eq := events[0]
	if eq.Source != "gdacs" {
		t.Errorf("expected source 'gdacs', got %q", eq.Source)
	}
	if eq.Category != "earthquake" {
		t.Errorf("expected category 'earthquake' for EQ type, got %q", eq.Category)
	}
	if eq.Severity != "critical" {
		t.Errorf("expected severity 'critical' for Red alert, got %q", eq.Severity)
	}
	if eq.Magnitude != 7.1 {
		t.Errorf("expected magnitude 7.1, got %f", eq.Magnitude)
	}

	// Second event: tropical cyclone, Orange alert -> high
	tc := events[1]
	if tc.Category != "storm" {
		t.Errorf("expected category 'storm' for TC type, got %q", tc.Category)
	}
	if tc.Severity != "high" {
		t.Errorf("expected severity 'high' for Orange alert, got %q", tc.Severity)
	}
}

func TestGDACSProvider_EventTypeMapping(t *testing.T) {
	tests := []struct {
		eventType string
		category  string
	}{
		{"EQ", "earthquake"},
		{"TC", "storm"},
		{"FL", "flood"},
		{"VO", "volcano"},
		{"DR", "drought"},
		{"XX", "disaster"}, // unknown type
	}

	for _, tc := range tests {
		t.Run(tc.eventType, func(t *testing.T) {
			response := GDACSResponse{
				Type: "FeatureCollection",
				Features: []GDACSFeature{{
					Type: "Feature",
					Properties: GDACSProperties{
						EventID:    1,
						EventType:  tc.eventType,
						AlertLevel: "Green",
						Country:    "Test",
						FromDate:   "2026-01-01T00:00:00",
						Name:       "Test Event",
					},
					Geometry: GDACSGeometry{
						Type:        "Point",
						Coordinates: []float64{0, 0},
					},
				}},
			}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			p := NewGDACSProvider(&Config{Enabled: true})
			p.feedURL = server.URL

			events, err := p.Fetch(context.Background())
			if err != nil {
				t.Fatalf("Fetch failed: %v", err)
			}
			if len(events) != 1 {
				t.Fatalf("expected 1 event, got %d", len(events))
			}
			if events[0].Category != tc.category {
				t.Errorf("expected category %q for type %q, got %q", tc.category, tc.eventType, events[0].Category)
			}
		})
	}
}

func TestGDACSProvider_AlertLevelSeverity(t *testing.T) {
	tests := []struct {
		alertLevel string
		severity   string
	}{
		{"Red", "critical"},
		{"Orange", "high"},
		{"Yellow", "medium"},
		{"Green", "low"},
		{"Unknown", "medium"}, // default
	}

	for _, tc := range tests {
		t.Run(tc.alertLevel, func(t *testing.T) {
			response := GDACSResponse{
				Type: "FeatureCollection",
				Features: []GDACSFeature{{
					Type: "Feature",
					Properties: GDACSProperties{
						EventID:    1,
						EventType:  "EQ",
						AlertLevel: tc.alertLevel,
						Country:    "Test",
						FromDate:   "2026-01-01T00:00:00",
						Name:       "Test",
					},
					Geometry: GDACSGeometry{Type: "Point", Coordinates: []float64{0, 0}},
				}},
			}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			p := NewGDACSProvider(&Config{Enabled: true})
			p.feedURL = server.URL

			events, _ := p.Fetch(context.Background())
			if len(events) != 1 {
				t.Fatalf("expected 1 event, got %d", len(events))
			}
			if string(events[0].Severity) != tc.severity {
				t.Errorf("expected severity %q for %q alert, got %q", tc.severity, tc.alertLevel, events[0].Severity)
			}
		})
	}
}

func TestGDACSProvider_InvalidGeometry(t *testing.T) {
	response := GDACSResponse{
		Type: "FeatureCollection",
		Features: []GDACSFeature{{
			Type: "Feature",
			Properties: GDACSProperties{
				EventID:    1,
				EventType:  "EQ",
				AlertLevel: "Green",
				FromDate:   "2026-01-01T00:00:00",
				Name:       "Test",
			},
			Geometry: GDACSGeometry{Type: "Polygon", Coordinates: []float64{}},
		}},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	p := NewGDACSProvider(&Config{Enabled: true})
	p.feedURL = server.URL

	events, err := p.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch should not error for invalid features: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events for invalid geometry, got %d", len(events))
	}
}
