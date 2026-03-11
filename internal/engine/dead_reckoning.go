package engine

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/openclaw/sentinel-backend/internal/storage"
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
	Confidence   float64   `json:"confidence"` // 0.0 - 1.0
	Projected    bool      `json:"projected"`
}

// DeadReckoningEngine projects asset positions forward from the last known
// position, heading, and speed when the signal is lost.
type DeadReckoningEngine struct {
	store       *storage.Storage
	maxAgeMins  int
	staleMins   int     // minutes without update before projecting (5)
	fadePerMin  float64 // confidence decrease per minute (0.10 = 10%)

	entities map[string]*TrackedEntity // id -> entity
	mu       sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewDeadReckoningEngine creates a new engine.
// maxAgeMins is how many minutes of dead-reckoning to allow before giving up.
func NewDeadReckoningEngine(maxAgeMins int) *DeadReckoningEngine {
	if maxAgeMins <= 0 {
		maxAgeMins = 30
	}
	return &DeadReckoningEngine{
		maxAgeMins: maxAgeMins,
		staleMins:  5,
		fadePerMin: 0.10,
		entities:   make(map[string]*TrackedEntity),
	}
}

// SetStorage sets the storage backend. Must be called before Start.
func (e *DeadReckoningEngine) SetStorage(s *storage.Storage) {
	e.store = s
}

// Start begins the periodic projection loop (every 30 seconds).
func (e *DeadReckoningEngine) Start(ctx context.Context) {
	e.ctx, e.cancel = context.WithCancel(ctx)
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[dead_reckoning] recovered from panic: %v", r)
			}
		}()

		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		log.Printf("[dead_reckoning] engine started (maxAge=%dm, stale=%dm, fade=%.0f%%/min)",
			e.maxAgeMins, e.staleMins, e.fadePerMin*100)

		for {
			select {
			case <-e.ctx.Done():
				log.Printf("[dead_reckoning] engine stopped")
				return
			case <-ticker.C:
				e.ProjectAll()
			}
		}
	}()
}

// Stop halts the dead reckoning engine.
func (e *DeadReckoningEngine) Stop() {
	if e.cancel != nil {
		e.cancel()
	}
	e.wg.Wait()
}

// UpdateEntity registers or updates a tracked entity with fresh position data.
func (e *DeadReckoningEngine) UpdateEntity(id, entityType string, lat, lon, heading, speedKnots float64) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.entities[id] = &TrackedEntity{
		ID:         id,
		EntityType: entityType,
		LastLat:    lat,
		LastLon:    lon,
		Heading:    heading,
		SpeedKnots: speedKnots,
		LastSeenAt: time.Now().UTC(),
		Confidence: 1.0,
		Projected:  false,
	}
}

// ProjectAll projects all stale entities and prunes expired ones.
func (e *DeadReckoningEngine) ProjectAll() {
	e.mu.Lock()
	defer e.mu.Unlock()

	now := time.Now().UTC()
	toDelete := make([]string, 0)

	for id, entity := range e.entities {
		elapsed := now.Sub(entity.LastSeenAt)
		elapsedMins := elapsed.Minutes()

		// Prune entities older than maxAgeMins
		if elapsedMins > float64(e.maxAgeMins) {
			toDelete = append(toDelete, id)
			continue
		}

		// Only project if signal lost for staleMins+
		if elapsedMins < float64(e.staleMins) {
			entity.Projected = false
			entity.ProjectedLat = entity.LastLat
			entity.ProjectedLon = entity.LastLon
			entity.Confidence = 1.0
			continue
		}

		// Project forward
		projected := e.Project(entity)
		if projected != nil {
			e.entities[id] = projected
		}
	}

	for _, id := range toDelete {
		delete(e.entities, id)
	}
}

// Project calculates the projected position for an entity.
func (e *DeadReckoningEngine) Project(entity *TrackedEntity) *TrackedEntity {
	elapsed := time.Since(entity.LastSeenAt)
	elapsedMins := elapsed.Minutes()

	if elapsedMins > float64(e.maxAgeMins) {
		return nil // too old to project
	}

	// Convert knots to km/h (1 knot = 1.852 km/h)
	speedKmH := entity.SpeedKnots * 1.852
	distanceKm := speedKmH * elapsed.Hours()

	// Great-circle projection
	newLat, newLon := ProjectPosition(entity.LastLat, entity.LastLon, entity.Heading, distanceKm)

	// Confidence fades 10% per minute of signal loss beyond staleMins
	signalLostMins := elapsedMins - float64(e.staleMins)
	if signalLostMins < 0 {
		signalLostMins = 0
	}
	confidence := 1.0 - (e.fadePerMin * signalLostMins)
	if confidence < 0 {
		confidence = 0
	}

	result := *entity
	result.ProjectedLat = newLat
	result.ProjectedLon = newLon
	result.Confidence = confidence
	result.Projected = true
	return &result
}

// GetProjectedEntities returns all currently projected entities.
func (e *DeadReckoningEngine) GetProjectedEntities() []TrackedEntity {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var projected []TrackedEntity
	for _, entity := range e.entities {
		if entity.Projected {
			projected = append(projected, *entity)
		}
	}
	return projected
}

// GetAllEntities returns all tracked entities.
func (e *DeadReckoningEngine) GetAllEntities() []TrackedEntity {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make([]TrackedEntity, 0, len(e.entities))
	for _, entity := range e.entities {
		result = append(result, *entity)
	}
	return result
}
