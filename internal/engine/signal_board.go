package engine

import (
	"context"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/openclaw/sentinel-backend/internal/storage"
)

// Domain represents a threat domain tracked on the signal board.
type Domain string

const (
	DomainMilitary  Domain = "military"
	DomainCyber     Domain = "cyber"
	DomainFinancial Domain = "financial"
	DomainNatural   Domain = "natural"
	DomainHealth    Domain = "health"
)

// SignalBoardEntry is a point-in-time snapshot of all domain threat levels.
type SignalBoardEntry struct {
	ID           int64     `json:"id"`
	Military     int       `json:"military"`  // 0-5
	Cyber        int       `json:"cyber"`     // 0-5
	Financial    int       `json:"financial"` // 0-5
	Natural      int       `json:"natural"`   // 0-5
	Health       int       `json:"health"`    // 0-5
	CalculatedAt time.Time `json:"calculated_at"`
}

// SignalBoard calculates domain threat levels from recent event data.
type SignalBoard struct {
	store   *storage.Storage
	enabled bool

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.Mutex
}

// NewSignalBoard creates a new signal board calculator.
func NewSignalBoard(enabled bool) *SignalBoard {
	return &SignalBoard{enabled: enabled}
}

// SetStorage sets the storage backend. Must be called before Start.
func (sb *SignalBoard) SetStorage(s *storage.Storage) {
	sb.store = s
}

// Start begins the periodic signal board calculation (every 60 seconds).
func (sb *SignalBoard) Start(ctx context.Context) {
	if !sb.enabled {
		log.Printf("[signal_board] disabled, not starting")
		return
	}
	sb.ctx, sb.cancel = context.WithCancel(ctx)
	sb.wg.Add(1)
	go func() {
		defer sb.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[signal_board] recovered from panic: %v", r)
			}
		}()

		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()

		log.Printf("[signal_board] calculator started")

		for {
			select {
			case <-sb.ctx.Done():
				log.Printf("[signal_board] calculator stopped")
				return
			case <-ticker.C:
				if _, err := sb.Calculate(); err != nil {
					log.Printf("[signal_board] calculation error: %v", err)
				}
			}
		}
	}()
}

// Stop halts the signal board calculator.
func (sb *SignalBoard) Stop() {
	if sb.cancel != nil {
		sb.cancel()
	}
	sb.wg.Wait()
}

// Calculate produces a new SignalBoardEntry based on current event data.
func (sb *SignalBoard) Calculate() (*SignalBoardEntry, error) {
	if !sb.enabled {
		return nil, nil
	}

	sb.mu.Lock()
	defer sb.mu.Unlock()

	if sb.store == nil {
		return &SignalBoardEntry{CalculatedAt: time.Now().UTC()}, nil
	}

	ctx := context.Background()
	now := time.Now().UTC()

	entry := &SignalBoardEntry{
		Military:     sb.calculateMilitary(ctx, now),
		Cyber:        sb.calculateCyber(ctx, now),
		Financial:    sb.calculateFinancial(ctx, now),
		Natural:      sb.calculateNatural(ctx, now),
		Health:       sb.calculateHealth(ctx, now),
		CalculatedAt: now,
	}

	// Store snapshot
	id, err := sb.store.InsertSignalBoardEntry(ctx, entry.Military, entry.Cyber, entry.Financial, entry.Natural, entry.Health)
	if err != nil {
		log.Printf("[signal_board] failed to store entry: %v", err)
	} else {
		entry.ID = id
	}

	log.Printf("[signal_board] MIL=%d CYB=%d FIN=%d NAT=%d HLT=%d",
		entry.Military, entry.Cyber, entry.Financial, entry.Natural, entry.Health)

	return entry, nil
}

// cap5 caps a value at 5.
func cap5(v int) int {
	if v > 5 {
		return 5
	}
	if v < 0 {
		return 0
	}
	return v
}

// calculateMilitary computes the military domain threat level.
func (sb *SignalBoard) calculateMilitary(ctx context.Context, now time.Time) int {
	level := 0
	since6h := now.Add(-6 * time.Hour)

	// Check conflict events with severity >= warning in last 6hr
	events, err := sb.store.GetEventsByCategoryAndTimeRange(ctx, "conflict", since6h, now)
	if err != nil {
		log.Printf("[signal_board] military query error: %v", err)
		return 0
	}

	hasCritical := false
	hasWarning := false
	for _, ev := range events {
		if ev.Severity == "critical" {
			hasCritical = true
		}
		if ev.Severity == "warning" || ev.Severity == "high" || ev.Severity == "critical" {
			hasWarning = true
		}
	}

	if hasWarning {
		level++
	}
	if hasCritical {
		level += 2
	}

	// Check for military squawk codes in aviation events
	aviationEvents, _ := sb.store.GetEventsByCategoryAndTimeRange(ctx, "aviation", since6h, now)
	for _, ev := range aviationEvents {
		if ev.Metadata != nil {
			squawk := ev.Metadata["squawk"]
			if squawk == "7700" || squawk == "7500" {
				level++
			}
		}
	}

	// Check missile alert events
	missileEvents, _ := sb.store.GetEventsByCategoryAndTimeRange(ctx, "missile_alert", since6h, now)
	if len(missileEvents) > 0 {
		level++
	}

	return cap5(level)
}

// calculateCyber computes the cyber domain threat level.
func (sb *SignalBoard) calculateCyber(ctx context.Context, now time.Time) int {
	level := 0
	since6h := now.Add(-6 * time.Hour)
	since24h := now.Add(-24 * time.Hour)

	// Check for CISA KEV new entries in last 24hr
	cisaEvents, _ := sb.store.GetEventsBySourceAndTimeRange(ctx, "cisa_kev", since24h, now)
	if len(cisaEvents) > 0 {
		level++
	}

	// Check OTX event count > 5 in last 6hr
	otxEvents, _ := sb.store.GetEventsBySourceAndTimeRange(ctx, "otx", since6h, now)
	if len(otxEvents) > 5 {
		level++
	}

	// Check cyber category events with severity >= warning
	cyberEvents, _ := sb.store.GetEventsByCategoryAndTimeRange(ctx, "cyber", since6h, now)
	hasCritical := false
	for _, ev := range cyberEvents {
		if ev.Severity == "warning" || ev.Severity == "high" || ev.Severity == "critical" {
			level++
			break
		}
	}
	for _, ev := range cyberEvents {
		if ev.Severity == "critical" {
			hasCritical = true
			break
		}
	}
	if hasCritical {
		level += 2
	}

	return cap5(level)
}

// calculateFinancial computes the financial domain threat level.
func (sb *SignalBoard) calculateFinancial(ctx context.Context, now time.Time) int {
	level := 0
	since24h := now.Add(-24 * time.Hour)

	// Check financial events
	events, _ := sb.store.GetEventsByCategoryAndTimeRange(ctx, "financial", since24h, now)

	for _, ev := range events {
		if ev.Metadata != nil {
			// VIX checks
			if vix, ok := ev.Metadata["vix"]; ok {
				if vixVal, err := parseFloat(vix); err == nil {
					if vixVal > 35 {
						level += 2
					} else if vixVal > 25 {
						level++
					}
				}
			}

			// Yield curve inversion
			if ev.Metadata["yield_curve"] == "inverted" {
				level++
			}

			// Crypto drop
			if drop, ok := ev.Metadata["crypto_drop_pct"]; ok {
				if dropVal, err := parseFloat(drop); err == nil && dropVal > 15 {
					level++
				}
			}
		}

		// Check for crash/flash in title
		titleLower := strings.ToLower(ev.Title)
		if strings.Contains(titleLower, "crash") || strings.Contains(titleLower, "flash") {
			level++
		}
	}

	return cap5(level)
}

// calculateNatural computes the natural domain threat level.
func (sb *SignalBoard) calculateNatural(ctx context.Context, now time.Time) int {
	level := 0
	since24h := now.Add(-24 * time.Hour)

	// Earthquake checks
	eqEvents, _ := sb.store.GetEventsByCategoryAndTimeRange(ctx, "earthquake", since24h, now)
	for _, ev := range eqEvents {
		if ev.Magnitude >= 7.0 {
			level += 2
		} else if ev.Magnitude >= 6.0 {
			level++
		}
	}

	// Tsunami events
	tsunamiEvents, _ := sb.store.GetEventsByCategoryAndTimeRange(ctx, "tsunami", since24h, now)
	if len(tsunamiEvents) > 0 {
		level++
	}

	// GDACS RED alerts
	gdacsEvents, _ := sb.store.GetEventsBySourceAndTimeRange(ctx, "gdacs", since24h, now)
	for _, ev := range gdacsEvents {
		if ev.Severity == "critical" || ev.Severity == "high" {
			level++
			break
		}
	}

	// Space weather: Kp index
	swpcEvents, _ := sb.store.GetEventsBySourceAndTimeRange(ctx, "swpc", since24h, now)
	for _, ev := range swpcEvents {
		if ev.Metadata != nil {
			if kp, ok := ev.Metadata["kp_index"]; ok {
				if kpVal, err := parseFloat(kp); err == nil && kpVal >= 7 {
					level++
					break
				}
			}
		}
	}

	return cap5(level)
}

// calculateHealth computes the health domain threat level.
func (sb *SignalBoard) calculateHealth(ctx context.Context, now time.Time) int {
	level := 0
	since7d := now.Add(-7 * 24 * time.Hour)

	// WHO events in last 7 days
	whoEvents, _ := sb.store.GetEventsBySourceAndTimeRange(ctx, "who", since7d, now)
	if len(whoEvents) > 0 {
		level++
	}

	hasCritical := false
	for _, ev := range whoEvents {
		if ev.Severity == "warning" || ev.Severity == "high" || ev.Severity == "critical" {
			level++
			break
		}
	}

	// Check all health events for critical + PHEIC
	healthEvents, _ := sb.store.GetEventsByCategoryAndTimeRange(ctx, "health", since7d, now)
	for _, ev := range healthEvents {
		if ev.Severity == "critical" {
			hasCritical = true
		}
		descLower := strings.ToLower(ev.Description)
		if strings.Contains(descLower, "pheic") {
			level++
			break
		}
	}
	if hasCritical {
		level += 2
	}

	return cap5(level)
}

// parseFloat is a helper that parses a float from string.
func parseFloat(s string) (float64, error) {
	return strconv.ParseFloat(strings.TrimSpace(s), 64)
}
