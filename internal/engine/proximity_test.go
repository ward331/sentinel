package engine

import (
	"sync"
	"testing"
	"time"

	"github.com/openclaw/sentinel-backend/internal/storage"
)

func TestProximityAlert_CheckEvent_WithinRadius(t *testing.T) {
	var alertFired bool
	var alertTitle, alertBody, alertSev string

	pa := NewProximityAlertRaw(40.7128, -74.0060, 100, func(title, body, severity string) {
		alertFired = true
		alertTitle = title
		alertBody = body
		alertSev = severity
	})

	// Event ~10km away from NYC (still within 100km radius)
	ev := storage.EventRow{
		ID:          "test-1",
		Title:       "Earthquake nearby",
		Description: "4.2 magnitude earthquake",
		Source:      "usgs",
		Lat:         40.80,
		Lon:         -73.95,
		Severity:    "high",
	}

	pa.CheckEvent(ev)

	if !alertFired {
		t.Fatal("expected proximity alert to fire for nearby event")
	}
	if alertTitle == "" {
		t.Error("expected non-empty alert title")
	}
	if alertSev != "high" {
		t.Errorf("expected severity 'high', got %q", alertSev)
	}
	_ = alertBody
}

func TestProximityAlert_CheckEvent_OutsideRadius(t *testing.T) {
	alertFired := false
	pa := NewProximityAlertRaw(40.7128, -74.0060, 50, func(_, _, _ string) {
		alertFired = true
	})

	// Event in London — well outside 50km radius
	ev := storage.EventRow{
		ID:       "test-2",
		Title:    "London event",
		Source:   "gdacs",
		Lat:      51.5074,
		Lon:      -0.1278,
		Severity: "critical",
	}

	pa.CheckEvent(ev)

	if alertFired {
		t.Error("expected no alert for event outside radius")
	}
}

func TestProximityAlert_SeverityFilter(t *testing.T) {
	alertFired := false
	pa := NewProximityAlertRaw(40.7128, -74.0060, 100, func(_, _, _ string) {
		alertFired = true
	})

	// Low severity event nearby — should NOT trigger
	ev := storage.EventRow{
		ID:       "test-3",
		Title:    "Minor event",
		Source:   "test",
		Lat:      40.72,
		Lon:      -74.00,
		Severity: "low",
	}

	pa.CheckEvent(ev)

	if alertFired {
		t.Error("expected no alert for low severity event")
	}
}

func TestProximityAlert_RateLimit(t *testing.T) {
	alertCount := 0
	pa := NewProximityAlertRaw(40.7128, -74.0060, 100, func(_, _, _ string) {
		alertCount++
	})

	ev := storage.EventRow{
		ID:       "test-4",
		Title:    "Repeated event",
		Source:   "usgs",
		Lat:      40.72,
		Lon:      -74.00,
		Severity: "high",
	}

	// First should fire
	pa.CheckEvent(ev)
	// Second from same source should be rate-limited
	ev.ID = "test-5"
	pa.CheckEvent(ev)

	if alertCount != 1 {
		t.Errorf("expected 1 alert (rate limited), got %d", alertCount)
	}

	// Different source should fire
	ev.ID = "test-6"
	ev.Source = "gdacs"
	pa.CheckEvent(ev)

	if alertCount != 2 {
		t.Errorf("expected 2 alerts (different source), got %d", alertCount)
	}
}

func TestProximityAlert_NotConfigured(t *testing.T) {
	alertFired := false
	pa := NewProximityAlertRaw(0, 0, 100, func(_, _, _ string) {
		alertFired = true
	})

	ev := storage.EventRow{
		ID:       "test-7",
		Title:    "Event",
		Source:   "test",
		Lat:      40.72,
		Lon:      -74.00,
		Severity: "critical",
	}

	pa.CheckEvent(ev)

	if alertFired {
		t.Error("expected no alert when home location is (0,0)")
	}
}

func TestProximityAlert_FilterNearby(t *testing.T) {
	pa := NewProximityAlertRaw(40.7128, -74.0060, 100, nil)

	events := []storage.EventRow{
		{ID: "near", Lat: 40.80, Lon: -73.95, Severity: "high"},
		{ID: "far", Lat: 51.50, Lon: -0.12, Severity: "high"},
		{ID: "close", Lat: 40.71, Lon: -74.01, Severity: "low"},
	}

	nearby := pa.FilterNearby(events)

	if len(nearby) != 2 {
		t.Errorf("expected 2 nearby events, got %d", len(nearby))
	}

	// Verify distance is populated
	for _, pe := range nearby {
		if pe.DistKm <= 0 && pe.Event.ID != "close" {
			t.Errorf("expected positive distance for event %s, got %.2f", pe.Event.ID, pe.DistKm)
		}
		if pe.Direction == "" {
			t.Errorf("expected non-empty direction for event %s", pe.Event.ID)
		}
	}
}

func TestProximityAlert_Cleanup(t *testing.T) {
	pa := NewProximityAlertRaw(40.7128, -74.0060, 100, func(_, _, _ string) {})

	// Manually inject an old rate-limit entry
	pa.mu.Lock()
	pa.lastAlerts["old-source"] = time.Now().Add(-31 * time.Minute)
	pa.lastAlerts["recent-source"] = time.Now().Add(-5 * time.Minute)
	pa.mu.Unlock()

	pa.Cleanup()

	pa.mu.Lock()
	defer pa.mu.Unlock()

	if _, exists := pa.lastAlerts["old-source"]; exists {
		t.Error("expected old-source to be cleaned up")
	}
	if _, exists := pa.lastAlerts["recent-source"]; !exists {
		t.Error("expected recent-source to still exist")
	}
}

func TestProximityAlert_UpdateLocation(t *testing.T) {
	pa := NewProximityAlertRaw(0, 0, 100, nil)

	if pa.Configured() {
		t.Error("expected not configured at (0,0)")
	}

	pa.UpdateLocation(51.5074, -0.1278, 300)

	if !pa.Configured() {
		t.Error("expected configured after update")
	}
	if pa.HomeLat != 51.5074 || pa.HomeLon != -0.1278 || pa.RadiusKm != 300 {
		t.Errorf("location not updated correctly: lat=%.4f lon=%.4f radius=%.0f",
			pa.HomeLat, pa.HomeLon, pa.RadiusKm)
	}
}

func TestProximityAlert_ConcurrentAccess(t *testing.T) {
	pa := NewProximityAlertRaw(40.7128, -74.0060, 200, func(_, _, _ string) {})

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ev := storage.EventRow{
				ID:       "concurrent-test",
				Title:    "Concurrent event",
				Source:   "test-source",
				Lat:      40.72,
				Lon:      -74.00,
				Severity: "high",
			}
			pa.CheckEvent(ev)
			pa.Cleanup()
		}(i)
	}
	wg.Wait()
}

func TestSeverityLevel(t *testing.T) {
	tests := []struct {
		sev  string
		want int
	}{
		{"low", 1},
		{"medium", 2},
		{"watch", 2},
		{"high", 3},
		{"warning", 3},
		{"critical", 4},
		{"alert", 4},
		{"unknown", 0},
		{"", 0},
	}

	for _, tt := range tests {
		got := severityLevel(tt.sev)
		if got != tt.want {
			t.Errorf("severityLevel(%q) = %d, want %d", tt.sev, got, tt.want)
		}
	}
}
