package storage

import (
	"context"
	"testing"
	"time"

	"github.com/openclaw/sentinel-backend/internal/model"
)

// testStore creates an in-memory storage with V3 migration applied.
func testStore(t *testing.T) *Storage {
	t.Helper()
	s, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to create in-memory storage: %v", err)
	}
	if err := RunV3Migration(s.DB()); err != nil {
		t.Fatalf("failed to run V3 migration: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func makeEvent(source, category string, lat, lon float64, occurredAt time.Time) *model.Event {
	return &model.Event{
		Title:       "Test " + category,
		Description: "A test event for " + category,
		Source:      source,
		OccurredAt:  occurredAt,
		Location:    model.Point(lon, lat),
		Precision:   model.PrecisionExact,
		Magnitude:   5.0,
		Category:    category,
		Severity:    model.SeverityMedium,
	}
}

func TestStorage_InsertAndRetrieveEvent(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	ev := makeEvent("usgs", "earthquake", 40.0, -74.0, now)
	if err := s.StoreEvent(ctx, ev); err != nil {
		t.Fatalf("StoreEvent failed: %v", err)
	}
	if ev.ID == "" {
		t.Fatal("expected event ID to be set after insert")
	}

	// Retrieve by ID
	got, err := s.GetEvent(ctx, ev.ID)
	if err != nil {
		t.Fatalf("GetEvent failed: %v", err)
	}
	if got.Title != ev.Title {
		t.Errorf("title mismatch: got %q, want %q", got.Title, ev.Title)
	}
	if got.Source != "usgs" {
		t.Errorf("source mismatch: got %q, want %q", got.Source, "usgs")
	}
}

func TestStorage_FTS5Search(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	ev1 := makeEvent("usgs", "earthquake", 40.0, -74.0, now)
	ev1.Title = "Major earthquake in California"
	ev1.Description = "A 7.0 magnitude earthquake struck near Los Angeles"
	s.StoreEvent(ctx, ev1)

	ev2 := makeEvent("gdacs", "flood", 35.0, 139.0, now)
	ev2.Title = "Flooding in Tokyo"
	ev2.Description = "Heavy rainfall causes flooding in metropolitan area"
	s.StoreEvent(ctx, ev2)

	// Search for "earthquake" via FTS
	events, total, err := s.ListEvents(ctx, ListFilter{
		Query: "earthquake",
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("FTS search failed: %v", err)
	}
	if total == 0 || len(events) == 0 {
		t.Fatal("expected at least 1 result for 'earthquake' search")
	}
	if events[0].Source != "usgs" {
		t.Errorf("expected usgs event, got %s", events[0].Source)
	}
}

func TestStorage_BBoxSpatialQuery(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	// Event in NYC area
	ev1 := makeEvent("usgs", "earthquake", 40.7, -74.0, now)
	s.StoreEvent(ctx, ev1)

	// Event in Tokyo
	ev2 := makeEvent("usgs", "earthquake", 35.6, 139.7, now)
	s.StoreEvent(ctx, ev2)

	// BBox around NYC: [min_lon, min_lat, max_lon, max_lat]
	events, total, err := s.ListEvents(ctx, ListFilter{
		BBox:  []float64{-75.0, 39.0, -73.0, 42.0},
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("BBox query failed: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 event in NYC bbox, got %d", total)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].ID != ev1.ID {
		t.Errorf("expected NYC event ID %s, got %s", ev1.ID, events[0].ID)
	}
}

func TestStorage_TimeRangeQuery(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	ev1 := makeEvent("usgs", "earthquake", 40.0, -74.0, now.Add(-2*time.Hour))
	s.StoreEvent(ctx, ev1)

	ev2 := makeEvent("usgs", "earthquake", 40.0, -74.0, now.Add(-10*time.Minute))
	s.StoreEvent(ctx, ev2)

	// Query only last hour
	events, total, err := s.ListEvents(ctx, ListFilter{
		StartTime: now.Add(-1 * time.Hour),
		EndTime:   now,
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("time range query failed: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 event in last hour, got %d", total)
	}
	_ = events
}

func TestStorage_CorrelationInsertAndRetrieve(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	eventIDs := []string{"ev1", "ev2", "ev3"}
	id, err := s.InsertCorrelation(ctx, "40.00N, 74.00W", 40.0, -74.0, 10.0,
		3, 3, now.Add(-10*time.Minute), now, eventIDs)
	if err != nil {
		t.Fatalf("InsertCorrelation failed: %v", err)
	}
	if id == 0 {
		t.Error("expected non-zero correlation ID")
	}

	// Retrieve
	rows, err := s.GetRecentCorrelations(ctx, 60)
	if err != nil {
		t.Fatalf("GetRecentCorrelations failed: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 correlation, got %d", len(rows))
	}
	if rows[0].RegionName != "40.00N, 74.00W" {
		t.Errorf("region mismatch: %s", rows[0].RegionName)
	}
	if len(rows[0].EventIDs) != 3 {
		t.Errorf("expected 3 event IDs, got %d", len(rows[0].EventIDs))
	}
}

func TestStorage_SignalBoardInsertAndRetrieve(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	id, err := s.InsertSignalBoardEntry(ctx, 2, 1, 3, 4, 0)
	if err != nil {
		t.Fatalf("InsertSignalBoardEntry failed: %v", err)
	}
	if id == 0 {
		t.Error("expected non-zero signal board ID")
	}

	row, err := s.GetLatestSignalBoard(ctx)
	if err != nil {
		t.Fatalf("GetLatestSignalBoard failed: %v", err)
	}
	if row == nil {
		t.Fatal("expected non-nil signal board row")
	}
	if row.Military != 2 || row.Cyber != 1 || row.Financial != 3 || row.Natural != 4 || row.Health != 0 {
		t.Errorf("signal board mismatch: MIL=%d CYB=%d FIN=%d NAT=%d HLT=%d",
			row.Military, row.Cyber, row.Financial, row.Natural, row.Health)
	}
}

func TestStorage_AnomalyInsertAndResolve(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	id, err := s.InsertAnomaly(ctx, "usgs", "US", 5.0, 20.0, 4.0)
	if err != nil {
		t.Fatalf("InsertAnomaly failed: %v", err)
	}

	// Check active anomalies
	active, err := s.GetActiveAnomalies(ctx)
	if err != nil {
		t.Fatalf("GetActiveAnomalies failed: %v", err)
	}
	if len(active) != 1 {
		t.Fatalf("expected 1 active anomaly, got %d", len(active))
	}
	if active[0].Provider != "usgs" {
		t.Errorf("provider mismatch: %s", active[0].Provider)
	}

	// Resolve it
	if err := s.ResolveAnomaly(ctx, id); err != nil {
		t.Fatalf("ResolveAnomaly failed: %v", err)
	}

	// Should no longer be active
	active, err = s.GetActiveAnomalies(ctx)
	if err != nil {
		t.Fatalf("GetActiveAnomalies failed: %v", err)
	}
	if len(active) != 0 {
		t.Errorf("expected 0 active anomalies after resolve, got %d", len(active))
	}
}

func TestStorage_NewsItemInsertAndDedup(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	// Insert a news item
	_, err := s.DB().ExecContext(ctx, `
		INSERT INTO news_items (title, url, description, source_name, source_category, pub_date)
		VALUES (?, ?, ?, ?, ?, ?)
	`, "Breaking News", "https://example.com/breaking", "Something happened", "example", "news", time.Now().UTC())
	if err != nil {
		t.Fatalf("insert news item failed: %v", err)
	}

	// Attempt duplicate — url is UNIQUE
	_, err = s.DB().ExecContext(ctx, `
		INSERT INTO news_items (title, url, description, source_name, source_category, pub_date)
		VALUES (?, ?, ?, ?, ?, ?)
	`, "Breaking News Dupe", "https://example.com/breaking", "Same URL", "example", "news", time.Now().UTC())
	if err == nil {
		t.Error("expected UNIQUE constraint error for duplicate news URL")
	}
}

func TestStorage_GetRecentEvents(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	// Insert recent event
	ev := makeEvent("usgs", "earthquake", 40.0, -74.0, now.Add(-5*time.Minute))
	s.StoreEvent(ctx, ev)

	// Insert old event
	old := makeEvent("usgs", "earthquake", 40.0, -74.0, now.Add(-120*time.Minute))
	s.StoreEvent(ctx, old)

	rows, err := s.GetRecentEvents(ctx, 60)
	if err != nil {
		t.Fatalf("GetRecentEvents failed: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("expected 1 recent event (within 60 min), got %d", len(rows))
	}
}

func TestStorage_TruthConfirmation(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	err := s.InsertTruthConfirmation(ctx, "event-1", "gdacs", "event-2")
	if err != nil {
		t.Fatalf("InsertTruthConfirmation failed: %v", err)
	}

	count, err := s.GetTruthConfirmationCount(ctx, "event-1")
	if err != nil {
		t.Fatalf("GetTruthConfirmationCount failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 confirmation, got %d", count)
	}
}

func TestStorage_ListEvents_CategoryFilter(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	s.StoreEvent(ctx, makeEvent("usgs", "earthquake", 40.0, -74.0, now))
	s.StoreEvent(ctx, makeEvent("gdacs", "flood", 35.0, 139.0, now))

	events, total, err := s.ListEvents(ctx, ListFilter{
		Category: "earthquake",
		Limit:    10,
	})
	if err != nil {
		t.Fatalf("ListEvents with category filter failed: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 earthquake event, got %d", total)
	}
	if len(events) == 1 && events[0].Category != "earthquake" {
		t.Errorf("expected category earthquake, got %s", events[0].Category)
	}
}
