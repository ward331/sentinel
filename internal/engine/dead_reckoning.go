package engine

import (
	"math"
	"time"
)

// TrackedEntity represents an aircraft or vessel whose position is projected
// when the signal is lost.
type TrackedEntity struct {
	ID           string    `json:"id"`
	EntityType   string    `json:"entity_type"` // "aircraft" | "vessel"
	LastLat      float64   `json:"last_lat"`
	LastLon      float64   `json:"last_lon"`
	Heading      float64   `json:"heading"`      // degrees
	SpeedKnots   float64   `json:"speed_knots"`
	LastSeenAt   time.Time `json:"last_seen_at"`
	ProjectedLat float64   `json:"projected_lat"`
	ProjectedLon float64   `json:"projected_lon"`
}

// DeadReckoningEngine projects asset positions forward from the last known
// position, heading, and speed when the signal is lost.
type DeadReckoningEngine struct {
	maxAgeMins int
}

// NewDeadReckoningEngine creates a new engine.
// maxAgeMins is how many minutes of dead-reckoning to allow before giving up.
func NewDeadReckoningEngine(maxAgeMins int) *DeadReckoningEngine {
	if maxAgeMins <= 0 {
		maxAgeMins = 30
	}
	return &DeadReckoningEngine{maxAgeMins: maxAgeMins}
}

// Project calculates the projected position for an entity.
func (e *DeadReckoningEngine) Project(entity *TrackedEntity) *TrackedEntity {
	elapsed := time.Since(entity.LastSeenAt)
	if elapsed.Minutes() > float64(e.maxAgeMins) {
		return nil // too old to project
	}

	// Convert knots to km/h (1 knot = 1.852 km/h)
	speedKmH := entity.SpeedKnots * 1.852
	distanceKm := speedKmH * elapsed.Hours()

	// Simple great-circle projection
	headingRad := entity.Heading * math.Pi / 180.0
	latRad := entity.LastLat * math.Pi / 180.0
	lonRad := entity.LastLon * math.Pi / 180.0
	earthRadiusKm := 6371.0

	angularDist := distanceKm / earthRadiusKm

	newLat := math.Asin(math.Sin(latRad)*math.Cos(angularDist) +
		math.Cos(latRad)*math.Sin(angularDist)*math.Cos(headingRad))
	newLon := lonRad + math.Atan2(
		math.Sin(headingRad)*math.Sin(angularDist)*math.Cos(latRad),
		math.Cos(angularDist)-math.Sin(latRad)*math.Sin(newLat),
	)

	result := *entity
	result.ProjectedLat = newLat * 180.0 / math.Pi
	result.ProjectedLon = newLon * 180.0 / math.Pi
	return &result
}
