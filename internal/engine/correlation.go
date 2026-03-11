package engine

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/openclaw/sentinel-backend/internal/storage"
)

// CorrelationFlash groups events from 3+ sources in the same region within 60 minutes.
type CorrelationFlash struct {
	ID           int64     `json:"id"`
	RegionName   string    `json:"region_name"`
	Lat          float64   `json:"lat"`
	Lon          float64   `json:"lon"`
	RadiusKm     float64   `json:"radius_km"`
	EventCount   int       `json:"event_count"`
	SourceCount  int       `json:"source_count"`
	StartedAt    time.Time `json:"started_at"`
	LastEventAt  time.Time `json:"last_event_at"`
	Confirmed    bool      `json:"confirmed"`
	IncidentName string    `json:"incident_name,omitempty"`
	EventIDs     []string  `json:"event_ids,omitempty"`
}

// CorrelationEngine detects correlated events across multiple sources.
type CorrelationEngine struct {
	store         *storage.Storage
	windowMinutes int
	minSources    int
	radiusKm      float64

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.Mutex
}

// NewCorrelationEngine creates a new engine with default settings:
// 60-minute window, 3+ sources, 150km radius.
func NewCorrelationEngine() *CorrelationEngine {
	return &CorrelationEngine{
		windowMinutes: 60,
		minSources:    3,
		radiusKm:      150.0,
	}
}

// SetStorage sets the storage backend. Must be called before Start.
func (e *CorrelationEngine) SetStorage(s *storage.Storage) {
	e.store = s
}

// Start begins the periodic correlation loop (every 60 seconds).
func (e *CorrelationEngine) Start(ctx context.Context) {
	e.ctx, e.cancel = context.WithCancel(ctx)
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[correlation] recovered from panic: %v", r)
			}
		}()

		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()

		log.Printf("[correlation] engine started (window=%dm, minSources=%d, radius=%.0fkm)",
			e.windowMinutes, e.minSources, e.radiusKm)

		for {
			select {
			case <-e.ctx.Done():
				log.Printf("[correlation] engine stopped")
				return
			case <-ticker.C:
				if _, err := e.Evaluate(); err != nil {
					log.Printf("[correlation] evaluate error: %v", err)
				}
			}
		}
	}()
}

// Stop halts the correlation engine.
func (e *CorrelationEngine) Stop() {
	if e.cancel != nil {
		e.cancel()
	}
	e.wg.Wait()
}

// Evaluate checks recent events for correlations using spatial clustering.
func (e *CorrelationEngine) Evaluate() ([]CorrelationFlash, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.store == nil {
		return nil, nil
	}

	ctx := context.Background()

	// Fetch events from the last windowMinutes
	events, err := e.store.GetRecentEvents(ctx, e.windowMinutes)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent events: %w", err)
	}

	if len(events) < e.minSources {
		return nil, nil
	}

	// Get existing correlations to avoid duplicates
	existing, err := e.store.GetRecentCorrelations(ctx, e.windowMinutes*2)
	if err != nil {
		log.Printf("[correlation] warning: failed to get existing correlations: %v", err)
	}

	// Build clusters using single-linkage clustering with radiusKm threshold
	clusters := e.clusterEvents(events)

	var flashes []CorrelationFlash
	for _, cluster := range clusters {
		// Count distinct sources
		sourceSet := make(map[string]bool)
		var lats, lons []float64
		var ids []string
		var earliest, latest time.Time

		for _, ev := range cluster {
			sourceSet[ev.Source] = true
			lats = append(lats, ev.Lat)
			lons = append(lons, ev.Lon)
			ids = append(ids, ev.ID)
			if earliest.IsZero() || ev.OccurredAt.Before(earliest) {
				earliest = ev.OccurredAt
			}
			if ev.OccurredAt.After(latest) {
				latest = ev.OccurredAt
			}
		}

		if len(sourceSet) < e.minSources {
			continue
		}

		centLat, centLon := Centroid(lats, lons)
		radius := MaxRadius(centLat, centLon, lats, lons)
		if radius < 1.0 {
			radius = 1.0
		}

		// Check for dedup: skip if an existing correlation overlaps significantly
		if e.isDuplicate(existing, centLat, centLon, ids) {
			continue
		}

		regionName := fmt.Sprintf("%.2f°N, %.2f°E", centLat, centLon)

		// Store to database
		id, err := e.store.InsertCorrelation(ctx, regionName, centLat, centLon, radius,
			len(cluster), len(sourceSet), earliest, latest, ids)
		if err != nil {
			log.Printf("[correlation] failed to store correlation: %v", err)
			continue
		}

		flash := CorrelationFlash{
			ID:          id,
			RegionName:  regionName,
			Lat:         centLat,
			Lon:         centLon,
			RadiusKm:    radius,
			EventCount:  len(cluster),
			SourceCount: len(sourceSet),
			StartedAt:   earliest,
			LastEventAt: latest,
			EventIDs:    ids,
		}
		flashes = append(flashes, flash)
		log.Printf("[correlation] FLASH: %d events from %d sources near %s (radius %.1fkm)",
			flash.EventCount, flash.SourceCount, flash.RegionName, flash.RadiusKm)
	}

	return flashes, nil
}

// clusterEvents groups events that are within radiusKm of each other.
// Uses a simple greedy clustering approach.
func (e *CorrelationEngine) clusterEvents(events []storage.EventRow) [][]storage.EventRow {
	n := len(events)
	assigned := make([]bool, n)
	var clusters [][]storage.EventRow

	for i := 0; i < n; i++ {
		if assigned[i] {
			continue
		}
		// Skip events without valid coordinates
		if events[i].Lat == 0 && events[i].Lon == 0 {
			continue
		}

		cluster := []storage.EventRow{events[i]}
		assigned[i] = true

		// Find all events within radiusKm of any event in the cluster
		changed := true
		for changed {
			changed = false
			for j := 0; j < n; j++ {
				if assigned[j] {
					continue
				}
				if events[j].Lat == 0 && events[j].Lon == 0 {
					continue
				}
				for _, member := range cluster {
					dist := HaversineDistance(member.Lat, member.Lon, events[j].Lat, events[j].Lon)
					if dist <= e.radiusKm {
						cluster = append(cluster, events[j])
						assigned[j] = true
						changed = true
						break
					}
				}
			}
		}

		clusters = append(clusters, cluster)
	}
	return clusters
}

// isDuplicate checks if a new correlation overlaps with an existing one.
func (e *CorrelationEngine) isDuplicate(existing []storage.CorrelationRow, lat, lon float64, eventIDs []string) bool {
	for _, ex := range existing {
		dist := HaversineDistance(ex.Lat, ex.Lon, lat, lon)
		if dist < e.radiusKm {
			// Check event overlap
			existingSet := make(map[string]bool)
			for _, id := range ex.EventIDs {
				existingSet[id] = true
			}
			overlap := 0
			for _, id := range eventIDs {
				if existingSet[id] {
					overlap++
				}
			}
			// If more than 50% of events overlap, consider duplicate
			if len(eventIDs) > 0 && float64(overlap)/float64(len(eventIDs)) > 0.5 {
				return true
			}
		}
	}
	return false
}
