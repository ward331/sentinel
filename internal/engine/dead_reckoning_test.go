package engine

import (
	"math"
	"testing"
	"time"
)

func TestHaversineDistance_DeadReckoning(t *testing.T) {
	tests := []struct {
		name                       string
		lat1, lon1, lat2, lon2     float64
		expectedKm                 float64
		tolerance                  float64
	}{
		{"same point", 40.0, -74.0, 40.0, -74.0, 0, 0.001},
		{"NYC to LA approx", 40.7128, -74.0060, 34.0522, -118.2437, 3944, 50},
		{"London to Paris", 51.5074, -0.1278, 48.8566, 2.3522, 344, 10},
		{"equator short hop", 0.0, 0.0, 0.0, 1.0, 111.2, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := HaversineDistance(tc.lat1, tc.lon1, tc.lat2, tc.lon2)
			if math.Abs(got-tc.expectedKm) > tc.tolerance {
				t.Errorf("HaversineDistance(%f,%f,%f,%f) = %.2f km, want ~%.2f km (tolerance %.1f)",
					tc.lat1, tc.lon1, tc.lat2, tc.lon2, got, tc.expectedKm, tc.tolerance)
			}
		})
	}
}

func TestProjectPosition_KnownHeading(t *testing.T) {
	// Starting from equator at prime meridian, heading due north (0 degrees), 111.2 km
	// Should end up approximately at 1 degree north
	lat, lon := ProjectPosition(0, 0, 0, 111.2)
	if math.Abs(lat-1.0) > 0.05 {
		t.Errorf("expected lat ~1.0 after projecting 111.2km north, got %f", lat)
	}
	if math.Abs(lon) > 0.05 {
		t.Errorf("expected lon ~0.0 after projecting north, got %f", lon)
	}
}

func TestProjectPosition_East(t *testing.T) {
	// From equator, heading east (90 degrees), 111.2 km
	lat, lon := ProjectPosition(0, 0, 90, 111.2)
	if math.Abs(lat) > 0.05 {
		t.Errorf("expected lat ~0.0 after projecting east on equator, got %f", lat)
	}
	if math.Abs(lon-1.0) > 0.05 {
		t.Errorf("expected lon ~1.0 after projecting 111.2km east on equator, got %f", lon)
	}
}

func TestDeadReckoning_ConfidenceDecay(t *testing.T) {
	engine := NewDeadReckoningEngine(30)

	entity := &TrackedEntity{
		ID:         "test-aircraft",
		EntityType: "aircraft",
		LastLat:    40.0,
		LastLon:    -74.0,
		Heading:    90,
		SpeedKnots: 400,
		LastSeenAt: time.Now().UTC().Add(-10 * time.Minute), // 10 min ago, stale > 5 min
		Confidence: 1.0,
	}

	result := engine.Project(entity)
	if result == nil {
		t.Fatal("expected non-nil projection")
	}

	// Signal lost for 10 - 5 = 5 minutes; confidence = 1.0 - (0.10 * 5) = 0.5
	expectedConfidence := 0.5
	if math.Abs(result.Confidence-expectedConfidence) > 0.05 {
		t.Errorf("expected confidence ~%.2f, got %.2f", expectedConfidence, result.Confidence)
	}
	if !result.Projected {
		t.Error("expected Projected=true")
	}
}

func TestDeadReckoning_30MinCutoff(t *testing.T) {
	engine := NewDeadReckoningEngine(30)

	entity := &TrackedEntity{
		ID:         "test-aircraft",
		EntityType: "aircraft",
		LastLat:    40.0,
		LastLon:    -74.0,
		Heading:    90,
		SpeedKnots: 400,
		LastSeenAt: time.Now().UTC().Add(-35 * time.Minute), // 35 min ago > 30 min max
		Confidence: 1.0,
	}

	result := engine.Project(entity)
	if result != nil {
		t.Errorf("expected nil projection for entity beyond 30-min cutoff, got %+v", result)
	}
}

func TestDeadReckoning_FreshEntityNotProjected(t *testing.T) {
	engine := NewDeadReckoningEngine(30)

	// Update entity with fresh data
	engine.UpdateEntity("jet-1", "aircraft", 40.0, -74.0, 90, 400)

	entities := engine.GetAllEntities()
	if len(entities) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(entities))
	}
	if entities[0].Projected {
		t.Error("fresh entity should not be projected")
	}
	if entities[0].Confidence != 1.0 {
		t.Errorf("fresh entity should have confidence 1.0, got %f", entities[0].Confidence)
	}
}

func TestDeadReckoning_ProjectAll_PrunesExpired(t *testing.T) {
	engine := NewDeadReckoningEngine(30)

	// Add an entity that is beyond maxAge
	engine.mu.Lock()
	engine.entities["expired-jet"] = &TrackedEntity{
		ID:         "expired-jet",
		EntityType: "aircraft",
		LastLat:    40.0,
		LastLon:    -74.0,
		Heading:    90,
		SpeedKnots: 400,
		LastSeenAt: time.Now().UTC().Add(-60 * time.Minute), // way past 30 min
		Confidence: 1.0,
	}
	engine.mu.Unlock()

	engine.ProjectAll()

	all := engine.GetAllEntities()
	for _, e := range all {
		if e.ID == "expired-jet" {
			t.Error("expired entity should have been pruned by ProjectAll")
		}
	}
}

func TestDeadReckoning_ProjectPosition_Distance(t *testing.T) {
	// Verify that an aircraft at 400 knots heading east for 1 hour
	// covers approximately 400 * 1.852 = 740.8 km
	startLat, startLon := 0.0, 0.0
	speedKnots := 400.0
	heading := 90.0 // east
	distanceKm := speedKnots * 1.852 * 1.0 // 1 hour

	newLat, newLon := ProjectPosition(startLat, startLon, heading, distanceKm)

	actualDist := HaversineDistance(startLat, startLon, newLat, newLon)
	if math.Abs(actualDist-distanceKm) > 5.0 {
		t.Errorf("projected distance %.2f km doesn't match expected ~%.2f km", actualDist, distanceKm)
	}
}

func TestNewDeadReckoningEngine_DefaultMaxAge(t *testing.T) {
	e := NewDeadReckoningEngine(0)
	if e.maxAgeMins != 30 {
		t.Errorf("expected default maxAgeMins=30 when 0 passed, got %d", e.maxAgeMins)
	}

	e = NewDeadReckoningEngine(-5)
	if e.maxAgeMins != 30 {
		t.Errorf("expected default maxAgeMins=30 when negative passed, got %d", e.maxAgeMins)
	}
}
