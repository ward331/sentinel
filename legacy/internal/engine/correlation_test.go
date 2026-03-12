package engine

import (
	"context"
	"testing"
	"time"

	"github.com/openclaw/sentinel-backend/internal/storage"
)

// newTestStore creates an in-memory storage for testing.
func newTestStore(t *testing.T) *storage.Storage {
	t.Helper()
	s, err := storage.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create in-memory storage: %v", err)
	}
	if err := storage.RunV3Migration(s.DB()); err != nil {
		t.Fatalf("failed to run V3 migration: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// insertTestEvent is a helper that inserts an event into storage and returns the ID.
func insertTestEvent(t *testing.T, s *storage.Storage, source, category string, lat, lon float64, occurredAt time.Time) string {
	t.Helper()
	ctx := context.Background()
	event := newTestModelEvent(source, category, lat, lon, occurredAt)
	if err := s.StoreEvent(ctx, event); err != nil {
		t.Fatalf("failed to store event: %v", err)
	}
	return event.ID
}

func TestCorrelation_ThreeSourcesTrigger(t *testing.T) {
	s := newTestStore(t)
	now := time.Now().UTC()

	// Insert 3 events from different sources within 150km
	// All near (40.0, -74.0) — within ~10km of each other
	insertTestEvent(t, s, "usgs", "earthquake", 40.0, -74.0, now.Add(-10*time.Minute))
	insertTestEvent(t, s, "gdacs", "earthquake", 40.05, -74.05, now.Add(-8*time.Minute))
	insertTestEvent(t, s, "noaa_cap", "earthquake", 40.02, -74.02, now.Add(-5*time.Minute))

	engine := NewCorrelationEngine()
	engine.SetStorage(s)

	flashes, err := engine.Evaluate()
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if len(flashes) == 0 {
		t.Fatal("expected at least one correlation flash from 3 distinct sources, got 0")
	}
	if flashes[0].SourceCount < 3 {
		t.Errorf("expected SourceCount >= 3, got %d", flashes[0].SourceCount)
	}
}

func TestCorrelation_SameSourceNotIndependent(t *testing.T) {
	s := newTestStore(t)
	now := time.Now().UTC()

	// 3 events from SAME source — should NOT correlate
	insertTestEvent(t, s, "usgs", "earthquake", 40.0, -74.0, now.Add(-10*time.Minute))
	insertTestEvent(t, s, "usgs", "earthquake", 40.01, -74.01, now.Add(-8*time.Minute))
	insertTestEvent(t, s, "usgs", "earthquake", 40.02, -74.02, now.Add(-5*time.Minute))

	engine := NewCorrelationEngine()
	engine.SetStorage(s)

	flashes, err := engine.Evaluate()
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if len(flashes) != 0 {
		t.Errorf("expected no correlation flash from same-source events, got %d", len(flashes))
	}
}

func TestCorrelation_FarApartNoCorrelation(t *testing.T) {
	s := newTestStore(t)
	now := time.Now().UTC()

	// 3 events from different sources, but >150km apart
	insertTestEvent(t, s, "usgs", "earthquake", 40.0, -74.0, now.Add(-10*time.Minute))
	insertTestEvent(t, s, "gdacs", "earthquake", 50.0, -74.0, now.Add(-8*time.Minute))   // ~1100km away
	insertTestEvent(t, s, "noaa_cap", "earthquake", 60.0, -74.0, now.Add(-5*time.Minute)) // ~2200km away

	engine := NewCorrelationEngine()
	engine.SetStorage(s)

	flashes, err := engine.Evaluate()
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if len(flashes) != 0 {
		t.Errorf("expected no correlation flash for events >150km apart, got %d", len(flashes))
	}
}

func TestCorrelation_OldEventsNoCorrelation(t *testing.T) {
	s := newTestStore(t)
	now := time.Now().UTC()

	// 3 events from different sources, but >60min apart from "now"
	insertTestEvent(t, s, "usgs", "earthquake", 40.0, -74.0, now.Add(-120*time.Minute))
	insertTestEvent(t, s, "gdacs", "earthquake", 40.01, -74.01, now.Add(-110*time.Minute))
	insertTestEvent(t, s, "noaa_cap", "earthquake", 40.02, -74.02, now.Add(-100*time.Minute))

	engine := NewCorrelationEngine()
	engine.SetStorage(s)

	flashes, err := engine.Evaluate()
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	// Events older than 60 min should not be fetched by GetRecentEvents
	if len(flashes) != 0 {
		t.Errorf("expected no correlation for events older than window, got %d", len(flashes))
	}
}

func TestCorrelation_Deduplication(t *testing.T) {
	s := newTestStore(t)
	now := time.Now().UTC()

	// Insert 3 events from different sources
	insertTestEvent(t, s, "usgs", "earthquake", 40.0, -74.0, now.Add(-10*time.Minute))
	insertTestEvent(t, s, "gdacs", "earthquake", 40.05, -74.05, now.Add(-8*time.Minute))
	insertTestEvent(t, s, "noaa_cap", "earthquake", 40.02, -74.02, now.Add(-5*time.Minute))

	engine := NewCorrelationEngine()
	engine.SetStorage(s)

	// First evaluation should produce a flash
	flashes1, err := engine.Evaluate()
	if err != nil {
		t.Fatalf("first Evaluate failed: %v", err)
	}
	if len(flashes1) == 0 {
		t.Fatal("expected a correlation flash on first evaluation")
	}

	// Second evaluation on the same data should be deduplicated
	flashes2, err := engine.Evaluate()
	if err != nil {
		t.Fatalf("second Evaluate failed: %v", err)
	}
	if len(flashes2) != 0 {
		t.Errorf("expected deduplication to suppress second correlation, got %d", len(flashes2))
	}
}
