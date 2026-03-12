package engine

import (
	"math"
	"testing"
)

func TestHaversineDistance(t *testing.T) {
	tests := []struct {
		name     string
		lat1     float64
		lon1     float64
		lat2     float64
		lon2     float64
		wantKm   float64
		tolerance float64
	}{
		{
			name:      "same point",
			lat1:      40.7128, lon1: -74.0060,
			lat2:      40.7128, lon2: -74.0060,
			wantKm:    0,
			tolerance: 0.001,
		},
		{
			name:      "New York to London",
			lat1:      40.7128, lon1: -74.0060,
			lat2:      51.5074, lon2: -0.1278,
			wantKm:    5570,
			tolerance: 20, // ~0.4% tolerance
		},
		{
			name:      "North Pole to South Pole",
			lat1:      90, lon1: 0,
			lat2:      -90, lon2: 0,
			wantKm:    20015,
			tolerance: 20,
		},
		{
			name:      "Short distance — ~1km",
			lat1:      51.5000, lon1: -0.1000,
			lat2:      51.5090, lon2: -0.1000,
			wantKm:    1.0,
			tolerance: 0.05,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HaversineDistance(tt.lat1, tt.lon1, tt.lat2, tt.lon2)
			if math.Abs(got-tt.wantKm) > tt.tolerance {
				t.Errorf("HaversineDistance(%v,%v -> %v,%v) = %.2f km, want ~%.2f km (tolerance %.2f)",
					tt.lat1, tt.lon1, tt.lat2, tt.lon2, got, tt.wantKm, tt.tolerance)
			}
		})
	}
}

func TestCalcBearing(t *testing.T) {
	tests := []struct {
		name    string
		lat1    float64
		lon1    float64
		lat2    float64
		lon2    float64
		wantMin float64
		wantMax float64
	}{
		{
			name:    "due north",
			lat1:    0, lon1: 0,
			lat2:    10, lon2: 0,
			wantMin: 0, wantMax: 1,
		},
		{
			name:    "due east",
			lat1:    0, lon1: 0,
			lat2:    0, lon2: 10,
			wantMin: 89, wantMax: 91,
		},
		{
			name:    "due south",
			lat1:    10, lon1: 0,
			lat2:    0, lon2: 0,
			wantMin: 179, wantMax: 181,
		},
		{
			name:    "due west",
			lat1:    0, lon1: 10,
			lat2:    0, lon2: 0,
			wantMin: 269, wantMax: 271,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalcBearing(tt.lat1, tt.lon1, tt.lat2, tt.lon2)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("CalcBearing(%v,%v -> %v,%v) = %.2f, want between %.0f and %.0f",
					tt.lat1, tt.lon1, tt.lat2, tt.lon2, got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestBearingToCompass(t *testing.T) {
	tests := []struct {
		bearing float64
		want    string
	}{
		{0, "N"},
		{22.5, "NE"},
		{45, "NE"},
		{90, "E"},
		{135, "SE"},
		{180, "S"},
		{225, "SW"},
		{270, "W"},
		{315, "NW"},
		{350, "N"},
		{359, "N"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := BearingToCompass(tt.bearing)
			if got != tt.want {
				t.Errorf("BearingToCompass(%.1f) = %q, want %q", tt.bearing, got, tt.want)
			}
		})
	}
}

func TestCentroid(t *testing.T) {
	// Simple test: centroid of a single point should be itself
	lat, lon := Centroid([]float64{40.0}, []float64{-74.0})
	if math.Abs(lat-40.0) > 0.01 || math.Abs(lon-(-74.0)) > 0.01 {
		t.Errorf("Centroid of single point: got (%.4f, %.4f), want (40.0, -74.0)", lat, lon)
	}

	// Empty input
	lat, lon = Centroid(nil, nil)
	if lat != 0 || lon != 0 {
		t.Errorf("Centroid of empty: got (%.4f, %.4f), want (0, 0)", lat, lon)
	}
}

func TestMaxRadius(t *testing.T) {
	r := MaxRadius(0, 0, []float64{1, -1, 0}, []float64{0, 0, 1})
	// All points are ~111km from origin
	if r < 100 || r > 120 {
		t.Errorf("MaxRadius: got %.2f, want ~111 km", r)
	}
}
