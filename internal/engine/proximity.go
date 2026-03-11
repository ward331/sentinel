package engine

import (
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"github.com/openclaw/sentinel-backend/internal/config"
	"github.com/openclaw/sentinel-backend/internal/storage"
)

// ProximityAlert monitors events near a user's configured home location
// and triggers notifications when significant events occur within the
// configured radius.
type ProximityAlert struct {
	HomeLat  float64
	HomeLon  float64
	RadiusKm float64

	// OnAlert is called when a proximity alert fires.
	// Parameters: title, body, severity.
	OnAlert func(title, body, severity string)

	mu         sync.Mutex
	lastAlerts map[string]time.Time // source -> last alert time (rate limiting)
}

const (
	// DefaultProximityRadiusKm is used when no radius is configured.
	DefaultProximityRadiusKm = 200.0
	// proximityRateLimitDuration prevents duplicate alerts from the same source.
	proximityRateLimitDuration = 15 * time.Minute
	// proximityCleanupThreshold is how old a rate-limit entry must be before cleanup.
	proximityCleanupThreshold = 30 * time.Minute
)

// NewProximityAlert creates a new proximity alert engine from config.
func NewProximityAlert(cfg config.LocationConfig, onAlert func(string, string, string)) *ProximityAlert {
	radius := cfg.RadiusKm
	if radius <= 0 {
		radius = DefaultProximityRadiusKm
	}
	return &ProximityAlert{
		HomeLat:    cfg.Lat,
		HomeLon:    cfg.Lon,
		RadiusKm:   radius,
		OnAlert:    onAlert,
		lastAlerts: make(map[string]time.Time),
	}
}

// NewProximityAlertRaw creates a proximity alert engine from explicit coordinates.
func NewProximityAlertRaw(lat, lon, radiusKm float64, onAlert func(string, string, string)) *ProximityAlert {
	if radiusKm <= 0 {
		radiusKm = DefaultProximityRadiusKm
	}
	return &ProximityAlert{
		HomeLat:    lat,
		HomeLon:    lon,
		RadiusKm:   radiusKm,
		OnAlert:    onAlert,
		lastAlerts: make(map[string]time.Time),
	}
}

// UpdateLocation updates the home location and radius at runtime.
func (p *ProximityAlert) UpdateLocation(lat, lon, radiusKm float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.HomeLat = lat
	p.HomeLon = lon
	if radiusKm > 0 {
		p.RadiusKm = radiusKm
	}
}

// Configured returns true when a valid home location has been set.
func (p *ProximityAlert) Configured() bool {
	return p.HomeLat != 0 || p.HomeLon != 0
}

// CheckEvent evaluates a single event against the proximity zone.
// If the event is within radius, has sufficient severity, and passes
// rate limiting, OnAlert is called.
func (p *ProximityAlert) CheckEvent(ev storage.EventRow) {
	if !p.Configured() {
		return
	}

	dist := HaversineDistance(p.HomeLat, p.HomeLon, ev.Lat, ev.Lon)
	if dist > p.RadiusKm {
		return
	}

	// Severity filter: must be >= "watch" (medium)
	if severityLevel(ev.Severity) < severityLevel("medium") {
		return
	}

	// Rate limit per source
	p.mu.Lock()
	last, exists := p.lastAlerts[ev.Source]
	if exists && time.Since(last) < proximityRateLimitDuration {
		p.mu.Unlock()
		return
	}
	p.lastAlerts[ev.Source] = time.Now()
	p.mu.Unlock()

	// Compute bearing for human-readable direction
	bearing := CalcBearing(p.HomeLat, p.HomeLon, ev.Lat, ev.Lon)
	direction := BearingToCompass(bearing)

	title := fmt.Sprintf("NEAR YOU: %s", ev.Title)
	body := fmt.Sprintf("%s\n%.0fkm %s of your location\nSeverity: %s\nSource: %s",
		ev.Description, dist, direction, ev.Severity, ev.Source)

	log.Printf("[proximity] Alert: %s (%.0fkm %s, source=%s)", ev.Title, dist, direction, ev.Source)

	if p.OnAlert != nil {
		p.OnAlert(title, body, ev.Severity)
	}
}

// Cleanup removes stale entries from the rate-limit map.
func (p *ProximityAlert) Cleanup() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for k, v := range p.lastAlerts {
		if time.Since(v) > proximityCleanupThreshold {
			delete(p.lastAlerts, k)
		}
	}
}

// FilterNearby returns events from the given slice that fall within the
// configured proximity radius, along with their computed distances.
func (p *ProximityAlert) FilterNearby(events []storage.EventRow) []ProximityEvent {
	if !p.Configured() {
		return nil
	}
	var result []ProximityEvent
	for _, ev := range events {
		dist := HaversineDistance(p.HomeLat, p.HomeLon, ev.Lat, ev.Lon)
		if dist <= p.RadiusKm {
			bearing := CalcBearing(p.HomeLat, p.HomeLon, ev.Lat, ev.Lon)
			result = append(result, ProximityEvent{
				Event:     ev,
				DistKm:    dist,
				Direction: BearingToCompass(bearing),
				Bearing:   bearing,
			})
		}
	}
	return result
}

// ProximityEvent wraps an event with distance and direction from home.
type ProximityEvent struct {
	Event     storage.EventRow `json:"event"`
	DistKm    float64          `json:"distance_km"`
	Direction string           `json:"direction"`
	Bearing   float64          `json:"bearing"`
}

// --- Severity helpers ---

// severityLevel maps severity strings to numeric levels for comparison.
func severityLevel(sev string) int {
	switch sev {
	case "low":
		return 1
	case "medium", "watch":
		return 2
	case "high", "warning":
		return 3
	case "critical", "alert":
		return 4
	default:
		return 0
	}
}

// --- Bearing / compass helpers ---

// CalcBearing returns the initial bearing in degrees from (lat1,lon1) to (lat2,lon2).
func CalcBearing(lat1, lon1, lat2, lon2 float64) float64 {
	lat1R := degToRad(lat1)
	lat2R := degToRad(lat2)
	dLon := degToRad(lon2 - lon1)

	y := math.Sin(dLon) * math.Cos(lat2R)
	x := math.Cos(lat1R)*math.Sin(lat2R) - math.Sin(lat1R)*math.Cos(lat2R)*math.Cos(dLon)

	bearing := radToDeg(math.Atan2(y, x))
	// Normalize to 0-360
	return math.Mod(bearing+360, 360)
}

// BearingToCompass converts a bearing in degrees to a compass direction string.
func BearingToCompass(bearing float64) string {
	directions := []string{"N", "NE", "E", "SE", "S", "SW", "W", "NW"}
	// Each sector is 45 degrees wide, offset by 22.5 so N spans 337.5..22.5
	idx := int(math.Round(bearing/45.0)) % 8
	return directions[idx]
}
