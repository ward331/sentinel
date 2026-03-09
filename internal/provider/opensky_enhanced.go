package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/openclaw/sentinel-backend/internal/model"
	"github.com/openclaw/sentinel-backend/internal/providers/aircraft"
)

// OpenSkyEnhancedProvider fetches flight data with aircraft identification
type OpenSkyEnhancedProvider struct {
	name       string
	baseURL    string
	interval   time.Duration
	db         *aircraft.Database
}

// NewOpenSkyEnhancedProvider creates a new enhanced OpenSky provider
func NewOpenSkyEnhancedProvider(db *aircraft.Database) *OpenSkyEnhancedProvider {
	return &OpenSkyEnhancedProvider{
		name:     "opensky",
		baseURL:  "https://opensky-network.org/api",
		interval: 60 * time.Second,
		db:       db,
	}
}

// Name returns the provider name
func (p *OpenSkyEnhancedProvider) Name() string {
	return p.name
}

// Fetch retrieves flight data with aircraft identification
func (p *OpenSkyEnhancedProvider) Fetch(ctx context.Context) ([]*model.Event, error) {
	// Ensure aircraft database is loaded
	if !p.db.IsLoaded() {
		if err := p.db.Load(); err != nil {
			return nil, fmt.Errorf("failed to load aircraft database: %w", err)
		}
	}

	// Fetch states data
	states, err := p.fetchStatesData(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch states data: %w", err)
	}

	// Convert to events with aircraft identification
	events := make([]*model.Event, 0, len(states))
	for _, state := range states {
		event := p.stateToEvent(state)
		if event != nil {
			events = append(events, event)
		}
	}

	return events, nil
}

// stateToEvent converts a flight state to an event with aircraft identification
func (p *OpenSkyEnhancedProvider) stateToEvent(state *OpenSkyState) *model.Event {
	if state == nil || !state.OnGround {
		return nil
	}

	// Look up aircraft information
	aircraftInfo := p.db.LookupWithFallback(state.ICAO24, state.Callsign)

	// Determine category and severity
	category, severity := p.determineCategoryAndSeverity(aircraftInfo)

	// Create event
	event := &model.Event{
		ID:          fmt.Sprintf("opensky-%s-%d", state.ICAO24, time.Now().Unix()),
		Title:       aircraftInfo["display_name"].(string),
		Description: p.generateDescription(state, aircraftInfo),
		Source:      "opensky",
		SourceID:    state.ICAO24,
		OccurredAt:  time.Now(),
		Location: model.Location{
			Type:        "Point",
			Coordinates: []float64{state.Longitude, state.Latitude},
		},
		Precision: model.PrecisionExact,
		Magnitude: p.calculateMagnitude(state),
		Category:  category,
		Severity:  severity,
		Metadata: map[string]interface{}{
			"icao24":        state.ICAO24,
			"callsign":      state.Callsign,
			"altitude":      state.Altitude,
			"velocity":      state.Velocity,
			"heading":       state.Heading,
			"vertical_rate": state.VerticalRate,
			"on_ground":     state.OnGround,
			"aircraft_info": aircraftInfo,
			"data_source":   "OpenSky Network",
			"update_frequency": "real-time",
		},
		Badges: []model.Badge{
			{
				Type:  "source",
				Label: "ADS-B Flight Data",
				Color: "#FFD700", // Gold
			},
			{
				Type:  "precision",
				Label: "Exact",
				Color: "#006400", // Dark green
			},
			{
				Type:  "freshness",
				Label: "Real-time",
				Color: "#1E90FF", // Dodger blue
			},
		},
	}

	// Add military badge if aircraft is military
	if military, ok := aircraftInfo["military"].(bool); ok && military {
		event.Badges = append(event.Badges, model.Badge{
			Type:  "military",
			Label: "Military",
			Color: "#DC143C", // Crimson
		})
		event.Severity = model.SeverityHigh // Military aircraft get higher severity
	}

	return event
}

// determineCategoryAndSeverity determines event category and severity
func (p *OpenSkyEnhancedProvider) determineCategoryAndSeverity(aircraftInfo map[string]interface{}) (string, model.Severity) {
	// Check if military
	if military, ok := aircraftInfo["military"].(bool); ok && military {
		return "military", model.SeverityHigh
	}

	// Check aircraft type
	typeCode, _ := aircraftInfo["type_code"].(string)
	if typeCode != "" {
		// Commercial airliners
		if strings.HasPrefix(typeCode, "A3") || strings.HasPrefix(typeCode, "B7") ||
		   strings.HasPrefix(typeCode, "B3") || strings.HasPrefix(typeCode, "B7") {
			return "flight", model.SeverityLow
		}
		// General aviation
		if strings.HasPrefix(typeCode, "C") {
			return "flight", model.SeverityLow
		}
		// Helicopters
		if strings.HasPrefix(typeCode, "H") {
			return "flight", model.SeverityLow
		}
	}

	return "flight", model.SeverityLow
}

// calculateMagnitude calculates event magnitude
func (p *OpenSkyEnhancedProvider) calculateMagnitude(state *OpenSkyState) float64 {
	// Base magnitude for flight events
	magnitude := 2.0

	// Adjust based on altitude (higher altitude = higher magnitude)
	if state.Altitude > 0 {
		magnitude += float64(state.Altitude) / 10000.0
	}

	// Adjust based on velocity (faster = higher magnitude)
	if state.Velocity > 0 {
		magnitude += float64(state.Velocity) / 500.0
	}

	// Cap magnitude
	if magnitude > 5.0 {
		magnitude = 5.0
	}

	return magnitude
}

// generateDescription generates event description
func (p *OpenSkyEnhancedProvider) generateDescription(state *OpenSkyState, aircraftInfo map[string]interface{}) string {
	var desc strings.Builder
	
	desc.WriteString("Flight Tracking Data\n")
	desc.WriteString("====================\n\n")
	
	// Aircraft information
	if identified, ok := aircraftInfo["identified"].(bool); ok && identified {
		desc.WriteString("Aircraft: ")
		if aircraft, ok := aircraftInfo["aircraft"].(string); ok && aircraft != "" {
			desc.WriteString(aircraft)
		}
		if registration, ok := aircraftInfo["registration"].(string); ok && registration != "" {
			desc.WriteString(fmt.Sprintf(" (%s)", registration))
		}
		desc.WriteString("\n")
		
		if owner, ok := aircraftInfo["owner"].(string); ok && owner != "" {
			desc.WriteString(fmt.Sprintf("Owner: %s\n", owner))
		}
		if typeCode, ok := aircraftInfo["type_code"].(string); ok && typeCode != "" {
			desc.WriteString(fmt.Sprintf("Type: %s\n", typeCode))
		}
	} else {
		desc.WriteString(fmt.Sprintf("Aircraft: Unknown (%s)\n", state.ICAO24))
	}
	
	// Flight information
	if state.Callsign != "" {
		desc.WriteString(fmt.Sprintf("Callsign: %s\n", state.Callsign))
	}
	
	// Position and movement
	desc.WriteString(fmt.Sprintf("Position: %.4f, %.4f\n", state.Latitude, state.Longitude))
	if state.Altitude > 0 {
		desc.WriteString(fmt.Sprintf("Altitude: %d m\n", state.Altitude))
	}
	if state.Velocity > 0 {
		desc.WriteString(fmt.Sprintf("Speed: %.1f m/s\n", state.Velocity))
	}
	if state.Heading >= 0 {
		desc.WriteString(fmt.Sprintf("Heading: %d°\n", state.Heading))
	}
	
	// Military flag
	if military, ok := aircraftInfo["military"].(bool); ok && military {
		desc.WriteString("\n⚠️  MILITARY AIRCRAFT\n")
	}
	
	return desc.String()
}

// fetchStatesData fetches states data from OpenSky API
func (p *OpenSkyEnhancedProvider) fetchStatesData(ctx context.Context) ([]*OpenSkyState, error) {
	// For now, return empty slice as placeholder
	// In production, this would call the OpenSky API
	return []*OpenSkyState{}, nil
}

// OpenSkyState represents a flight state from OpenSky
type OpenSkyState struct {
	ICAO24       string  `json:"icao24"`
	Callsign     string  `json:"callsign"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	Altitude     float64 `json:"altitude"`
	Velocity     float64 `json:"velocity"`
	Heading      float64 `json:"heading"`
	VerticalRate float64 `json:"vertical_rate"`
	OnGround     bool    `json:"on_ground"`
}

// InitializeAircraftDatabase initializes and returns the aircraft database
func InitializeAircraftDatabase() *aircraft.Database {
	db := aircraft.NewDatabase()
	
	// Load database
	if err := db.Load(); err != nil {
		fmt.Printf("Warning: Failed to load aircraft database: %v\n", err)
		fmt.Println("Aircraft identification will be limited")
	} else {
		fmt.Printf("Aircraft database loaded: %d aircraft\n", db.Count())
		
		// Start auto-refresh
		db.AutoRefresh()
	}
	
	return db
}