package engine

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/openclaw/sentinel-backend/internal/storage"
)

// TruthConfirmation records a cross-source confirmation of an event.
type TruthConfirmation struct {
	ID                int64     `json:"id"`
	PrimaryEventID    string    `json:"primary_event_id"`
	ConfirmingSource  string    `json:"confirming_source"`
	ConfirmingEventID string    `json:"confirming_event_id"`
	ConfirmedAt       time.Time `json:"confirmed_at"`
}

// TruthScoreCalculator computes a 1-5 truth score based on cross-source confirmation.
//
//	1 = single source only
//	2 = two independent sources
//	3 = three+ sources agree
//	4 = confirmed by an official/authoritative source
//	5 = confirmed by multiple authoritative sources
type TruthScoreCalculator struct {
	store    *storage.Storage
	radiusKm float64       // max distance for "similar" events
	window   time.Duration // max time gap for "similar" events

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.Mutex
}

// authoritativeSources are sources considered official/authoritative.
var authoritativeSources = map[string]bool{
	"usgs":         true,
	"gdacs":        true,
	"noaa_cap":     true,
	"noaa_nws":     true,
	"tsunami":      true,
	"who":          true,
	"swpc":         true,
	"nasa_firms":   true,
	"volcano":      true,
	"reliefweb":    true,
}

// NewTruthScoreCalculator creates a new calculator.
func NewTruthScoreCalculator() *TruthScoreCalculator {
	return &TruthScoreCalculator{
		radiusKm: 100.0,
		window:   2 * time.Hour,
	}
}

// SetStorage sets the storage backend. Must be called before Start.
func (t *TruthScoreCalculator) SetStorage(s *storage.Storage) {
	t.store = s
}

// Start begins the periodic truth scoring loop (every 2 minutes).
func (t *TruthScoreCalculator) Start(ctx context.Context) {
	t.ctx, t.cancel = context.WithCancel(ctx)
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[truth] recovered from panic: %v", r)
			}
		}()

		ticker := time.NewTicker(2 * time.Minute)
		defer ticker.Stop()

		log.Printf("[truth] score calculator started (radius=%.0fkm, window=%v)", t.radiusKm, t.window)

		for {
			select {
			case <-t.ctx.Done():
				log.Printf("[truth] score calculator stopped")
				return
			case <-ticker.C:
				if err := t.ScoreRecentEvents(); err != nil {
					log.Printf("[truth] scoring error: %v", err)
				}
			}
		}
	}()
}

// Stop halts the truth score calculator.
func (t *TruthScoreCalculator) Stop() {
	if t.cancel != nil {
		t.cancel()
	}
	t.wg.Wait()
}

// ScoreRecentEvents scores all events from the last 2 hours.
func (t *TruthScoreCalculator) ScoreRecentEvents() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.store == nil {
		return nil
	}

	ctx := context.Background()
	now := time.Now().UTC()
	since := now.Add(-t.window)

	events, err := t.store.GetEventsByTimeRange(ctx, since, now)
	if err != nil {
		return err
	}

	for _, event := range events {
		score, err := t.Score(event.ID)
		if err != nil {
			log.Printf("[truth] error scoring event %s: %v", event.ID, err)
			continue
		}
		if score > 1 {
			if err := t.store.UpdateTruthScore(ctx, event.ID, score); err != nil {
				log.Printf("[truth] error updating score for %s: %v", event.ID, err)
			}
		}
	}

	return nil
}

// Score returns the truth score for a given event.
func (t *TruthScoreCalculator) Score(eventID string) (int, error) {
	if t.store == nil {
		return 1, nil
	}

	ctx := context.Background()

	// Get the primary event
	primary, err := t.store.GetEventRowByID(ctx, eventID)
	if err != nil || primary == nil {
		return 1, err
	}

	if primary.Category == "" {
		return 1, nil
	}

	// Find similar events: same category, within 100km, within 2 hours
	since := primary.OccurredAt.Add(-t.window)
	similar, err := t.store.GetSimilarEvents(ctx, eventID, primary.Category,
		primary.Lat, primary.Lon, t.radiusKm, since)
	if err != nil {
		return 1, err
	}

	// Filter by distance and collect unique sources
	sourceSet := map[string]string{} // source -> confirming event ID
	authoritativeCount := 0

	for _, ev := range similar {
		if ev.Source == primary.Source {
			continue // Same source, not independent
		}
		dist := HaversineDistance(primary.Lat, primary.Lon, ev.Lat, ev.Lon)
		if dist > t.radiusKm {
			continue
		}
		if _, exists := sourceSet[ev.Source]; !exists {
			sourceSet[ev.Source] = ev.ID

			// Record confirmation
			_ = t.store.InsertTruthConfirmation(ctx, eventID, ev.Source, ev.ID)

			if authoritativeSources[ev.Source] {
				authoritativeCount++
			}
		}
	}

	totalSources := len(sourceSet) + 1 // +1 for the primary source

	// Calculate score
	score := 1
	if totalSources >= 2 {
		score = 2
	}
	if totalSources >= 3 {
		score = 3
	}
	if authoritativeCount >= 1 {
		score = 4
	}
	if authoritativeCount >= 2 {
		score = 5
	}

	return score, nil
}
