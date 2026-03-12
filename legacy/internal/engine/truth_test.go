package engine

import (
	"context"
	"testing"
	"time"

	"github.com/openclaw/sentinel-backend/internal/model"
	"github.com/openclaw/sentinel-backend/internal/storage"
)

// newTestModelEvent creates a model.Event for insertion.
func newTestModelEvent(source, category string, lat, lon float64, occurredAt time.Time) *model.Event {
	return &model.Event{
		Title:       "Test Event",
		Description: "A test event for unit testing",
		Source:      source,
		SourceID:    "",
		OccurredAt:  occurredAt,
		Location:    model.Point(lon, lat),
		Precision:   model.PrecisionExact,
		Magnitude:   5.0,
		Category:    category,
		Severity:    model.SeverityMedium,
	}
}

func TestTruthScore_SingleSource(t *testing.T) {
	s := newTestStore(t)

	now := time.Now().UTC()
	id := insertTestEvent(t, s, "usgs", "earthquake", 40.0, -74.0, now.Add(-10*time.Minute))

	calc := NewTruthScoreCalculator()
	calc.SetStorage(s)

	score, err := calc.Score(id)
	if err != nil {
		t.Fatalf("Score failed: %v", err)
	}
	if score != 1 {
		t.Errorf("expected truth score 1 for single source, got %d", score)
	}
}

func TestTruthScore_IncreaseWithConfirmingSources(t *testing.T) {
	s := newTestStore(t)

	now := time.Now().UTC()
	// Primary event
	primaryID := insertTestEvent(t, s, "usgs", "earthquake", 40.0, -74.0, now.Add(-10*time.Minute))
	// Confirming event from different source, same location, same category
	insertTestEvent(t, s, "gdacs", "earthquake", 40.01, -74.01, now.Add(-8*time.Minute))

	calc := NewTruthScoreCalculator()
	calc.SetStorage(s)

	score, err := calc.Score(primaryID)
	if err != nil {
		t.Fatalf("Score failed: %v", err)
	}
	if score < 2 {
		t.Errorf("expected truth score >= 2 with confirming source, got %d", score)
	}
}

func TestTruthScore_SameSourceNoConfirmation(t *testing.T) {
	s := newTestStore(t)

	now := time.Now().UTC()
	primaryID := insertTestEvent(t, s, "usgs", "earthquake", 40.0, -74.0, now.Add(-10*time.Minute))
	// Another USGS event nearby — same source should not count
	insertTestEvent(t, s, "usgs", "earthquake", 40.01, -74.01, now.Add(-8*time.Minute))

	calc := NewTruthScoreCalculator()
	calc.SetStorage(s)

	score, err := calc.Score(primaryID)
	if err != nil {
		t.Fatalf("Score failed: %v", err)
	}
	if score != 1 {
		t.Errorf("expected truth score 1 for same-source events, got %d", score)
	}
}

func TestTruthScore_100kmRadiusMatching(t *testing.T) {
	s := newTestStore(t)

	now := time.Now().UTC()
	primaryID := insertTestEvent(t, s, "usgs", "earthquake", 40.0, -74.0, now.Add(-10*time.Minute))
	// Event from different source, but >100km away (~200km)
	insertTestEvent(t, s, "gdacs", "earthquake", 41.8, -74.0, now.Add(-8*time.Minute))

	calc := NewTruthScoreCalculator()
	calc.SetStorage(s)

	score, err := calc.Score(primaryID)
	if err != nil {
		t.Fatalf("Score failed: %v", err)
	}
	if score != 1 {
		t.Errorf("expected truth score 1 for events >100km apart, got %d", score)
	}
}

func TestTruthScore_2HourTimeWindow(t *testing.T) {
	s := newTestStore(t)

	now := time.Now().UTC()
	primaryID := insertTestEvent(t, s, "usgs", "earthquake", 40.0, -74.0, now.Add(-10*time.Minute))
	// Event from different source, same location, but 3 hours ago — outside 2h window
	insertTestEvent(t, s, "gdacs", "earthquake", 40.01, -74.01, now.Add(-3*time.Hour))

	calc := NewTruthScoreCalculator()
	calc.SetStorage(s)

	score, err := calc.Score(primaryID)
	if err != nil {
		t.Fatalf("Score failed: %v", err)
	}
	// The similar events query uses "since" = primary.OccurredAt - 2h, so a 3h old event
	// should be out of range if primary occurred 10min ago (since = -2h10m).
	// But the similar event at -3h may still fall within that range. Let's verify the actual logic.
	// primary.OccurredAt = now - 10min, since = now - 10min - 2h = now - 2h10min
	// confirming at now - 3h is BEFORE since, so should NOT match.
	if score != 1 {
		t.Errorf("expected truth score 1 for event outside 2h window, got %d", score)
	}
}

func TestTruthScore_AuthoritativeSourceBoost(t *testing.T) {
	s := newTestStore(t)

	now := time.Now().UTC()
	primaryID := insertTestEvent(t, s, "gdelt", "earthquake", 40.0, -74.0, now.Add(-10*time.Minute))
	// USGS is authoritative
	insertTestEvent(t, s, "usgs", "earthquake", 40.01, -74.01, now.Add(-8*time.Minute))
	// GDACS is also authoritative
	insertTestEvent(t, s, "gdacs", "earthquake", 40.02, -74.02, now.Add(-6*time.Minute))
	// Non-authoritative source
	insertTestEvent(t, s, "reliefweb", "earthquake", 40.03, -74.03, now.Add(-4*time.Minute))

	calc := NewTruthScoreCalculator()
	calc.SetStorage(s)

	score, err := calc.Score(primaryID)
	if err != nil {
		t.Fatalf("Score failed: %v", err)
	}
	// 4 total sources (gdelt + usgs + gdacs + reliefweb), 3 authoritative confirms (usgs, gdacs, reliefweb)
	// Actually reliefweb IS authoritative per the source map. So authoritativeCount >= 2 → score 5
	if score < 4 {
		t.Errorf("expected truth score >= 4 with authoritative sources, got %d", score)
	}
}

func TestTruthScore_NilStore(t *testing.T) {
	calc := NewTruthScoreCalculator()
	// No storage set

	score, err := calc.Score("nonexistent")
	if err != nil {
		t.Fatalf("Score failed: %v", err)
	}
	if score != 1 {
		t.Errorf("expected truth score 1 with nil store, got %d", score)
	}
}

func TestScoreRecentEvents_NilStore(t *testing.T) {
	calc := NewTruthScoreCalculator()
	err := calc.ScoreRecentEvents()
	if err != nil {
		t.Fatalf("ScoreRecentEvents should not error with nil store, got: %v", err)
	}
}

// Ensure unused import doesn't cause compile error
var _ = (*storage.Storage)(nil)
var _ = context.Background
