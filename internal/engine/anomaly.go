package engine

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/openclaw/sentinel-backend/internal/storage"
)

// Anomaly represents a detected spike above the rolling baseline.
type Anomaly struct {
	ID           int64      `json:"id"`
	ProviderName string     `json:"provider_name"`
	Region       string     `json:"region"`
	ExpectedRate float64    `json:"expected_rate"`
	ActualRate   float64    `json:"actual_rate"`
	SpikeFactor  float64    `json:"spike_factor"`
	DetectedAt   time.Time  `json:"detected_at"`
	ResolvedAt   *time.Time `json:"resolved_at,omitempty"`
}

// AnomalyDetector maintains a rolling 24-hour baseline per provider per region
// and fires an alert when the event rate exceeds 3x the baseline.
type AnomalyDetector struct {
	store          *storage.Storage
	spikeThreshold float64 // default 3.0x
	resolveAt      float64 // auto-resolve below 2.0x
	windowHours    int     // default 24

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.Mutex
}

// NewAnomalyDetector creates a detector with default thresholds.
func NewAnomalyDetector() *AnomalyDetector {
	return &AnomalyDetector{
		spikeThreshold: 3.0,
		resolveAt:      2.0,
		windowHours:    24,
	}
}

// SetStorage sets the storage backend. Must be called before Start.
func (d *AnomalyDetector) SetStorage(s *storage.Storage) {
	d.store = s
}

// Start begins the periodic anomaly detection loop (every 5 minutes).
func (d *AnomalyDetector) Start(ctx context.Context) {
	d.ctx, d.cancel = context.WithCancel(ctx)
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[anomaly] recovered from panic: %v", r)
			}
		}()

		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		log.Printf("[anomaly] detector started (spike=%.1fx, resolve=%.1fx, window=%dh)",
			d.spikeThreshold, d.resolveAt, d.windowHours)

		for {
			select {
			case <-d.ctx.Done():
				log.Printf("[anomaly] detector stopped")
				return
			case <-ticker.C:
				if _, err := d.Detect(); err != nil {
					log.Printf("[anomaly] detection error: %v", err)
				}
			}
		}
	}()
}

// Stop halts the anomaly detector.
func (d *AnomalyDetector) Stop() {
	if d.cancel != nil {
		d.cancel()
	}
	d.wg.Wait()
}

// Detect checks for anomalous event rates.
func (d *AnomalyDetector) Detect() ([]Anomaly, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.store == nil {
		return nil, nil
	}

	ctx := context.Background()
	now := time.Now().UTC()

	// Get 24-hour baseline counts per source per region
	baselineStart := now.Add(-time.Duration(d.windowHours) * time.Hour)
	baselineCounts, err := d.store.GetEventCountBySourceAndHour(ctx, baselineStart, now)
	if err != nil {
		return nil, err
	}

	// Build baseline map: source:region -> total count over 24h
	type key struct{ source, region string }
	baselineMap := make(map[key]int)
	for _, c := range baselineCounts {
		k := key{c.Source, c.Region}
		baselineMap[k] += c.Count
	}

	// Get current hour's counts
	currentStart := now.Add(-1 * time.Hour)
	currentCounts, err := d.store.GetEventCountBySourceAndHour(ctx, currentStart, now)
	if err != nil {
		return nil, err
	}
	currentMap := make(map[key]int)
	for _, c := range currentCounts {
		k := key{c.Source, c.Region}
		currentMap[k] += c.Count
	}

	// Get active anomalies for auto-resolve check
	activeAnomalies, err := d.store.GetActiveAnomalies(ctx)
	if err != nil {
		log.Printf("[anomaly] warning: failed to get active anomalies: %v", err)
	}

	// Check for auto-resolve on active anomalies
	for _, active := range activeAnomalies {
		k := key{active.Provider, active.Region}
		currentRate := float64(currentMap[k])
		expectedRate := float64(baselineMap[k]) / float64(d.windowHours)
		if expectedRate <= 0 {
			expectedRate = 1
		}
		spikeFactor := currentRate / expectedRate
		if spikeFactor < d.resolveAt {
			if err := d.store.ResolveAnomaly(ctx, active.ID); err != nil {
				log.Printf("[anomaly] failed to resolve anomaly %d: %v", active.ID, err)
			} else {
				log.Printf("[anomaly] RESOLVED: %s/%s (spike factor now %.1fx)",
					active.Provider, active.Region, spikeFactor)
			}
		}
	}

	// Build set of active anomalies for dedup
	activeSet := make(map[key]bool)
	for _, a := range activeAnomalies {
		activeSet[key{a.Provider, a.Region}] = true
	}

	// Detect new anomalies
	var newAnomalies []Anomaly
	for k, currentCount := range currentMap {
		baselineTotal := baselineMap[k]
		expectedRate := float64(baselineTotal) / float64(d.windowHours)
		if expectedRate <= 0 {
			expectedRate = 1
		}
		currentRate := float64(currentCount)
		spikeFactor := currentRate / expectedRate

		if spikeFactor >= d.spikeThreshold {
			// Skip if already active
			if activeSet[k] {
				continue
			}

			id, err := d.store.InsertAnomaly(ctx, k.source, k.region, expectedRate, currentRate, spikeFactor)
			if err != nil {
				log.Printf("[anomaly] failed to store anomaly: %v", err)
				continue
			}

			anomaly := Anomaly{
				ID:           id,
				ProviderName: k.source,
				Region:       k.region,
				ExpectedRate: expectedRate,
				ActualRate:   currentRate,
				SpikeFactor:  spikeFactor,
				DetectedAt:   now,
			}
			newAnomalies = append(newAnomalies, anomaly)
			log.Printf("[anomaly] SPIKE: %s/%s — expected %.1f/hr, got %.0f (%.1fx)",
				k.source, k.region, expectedRate, currentRate, spikeFactor)
		}
	}

	return newAnomalies, nil
}
