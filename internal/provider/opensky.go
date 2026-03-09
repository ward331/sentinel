package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/openclaw/sentinel-backend/internal/model"
)

// OpenSkyProvider fetches flight data from OpenSky Network
type OpenSkyProvider struct {
	name     string
	baseURL  string
	username string
	password string
	client   *http.Client
}

// OpenSkyState represents aircraft state from OpenSky API
type OpenSkyState struct {
	Icao24       string  `json:"icao24"`
	Callsign     string  `json:"callsign,omitempty"`
	OriginCountry string  `json:"origin_country"`
	TimePosition int64   `json:"time_position,omitempty"`
	LastContact  int64   `json:"last_contact"`
	Longitude    float64 `json:"longitude,omitempty"`
	Latitude     float64 `json:"latitude,omitempty"`
	BaroAltitude float64 `json:"baro_altitude,omitempty"`
	OnGround     bool    `json:"on_ground"`
	Velocity     float64 `json:"velocity,omitempty"`
	TrueTrack    float64 `json:"true_track,omitempty"`
	VerticalRate float64 `json:"vertical_rate,omitempty"`
	Sensors      []int   `json:"sensors,omitempty"`
	GeoAltitude  float64 `json:"geo_altitude,omitempty"`
	Squawk       string  `json:"squawk,omitempty"`
	Spi          bool    `json:"spi"`
	PositionSource int   `json:"position_source"`
}

// OpenSkyResponse represents the response from OpenSky API
type OpenSkyResponse struct {
	Time   int64          `json:"time"`
	States [][]interface{} `json:"states"`
}

// NewOpenSkyProvider creates a new OpenSky provider
func NewOpenSkyProvider(username, password string) *OpenSkyProvider {
	return &OpenSkyProvider{
		name:     "opensky",
		baseURL:  "https://opensky-network.org/api",
		username: username,
		password: password,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name returns the provider name
func (p *OpenSkyProvider) Name() string {
	return p.name
}

// Fetch fetches current flight data from OpenSky
func (p *OpenSkyProvider) Fetch(ctx context.Context) ([]*model.Event, error) {
	// Build request
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/states/all", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication if provided
	if p.username != "" && p.password != "" {
		req.SetBasicAuth(p.username, p.password)
	}

	// Execute request
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from OpenSky: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenSky API returned status %d", resp.StatusCode)
	}

	// Parse response
	var apiResp OpenSkyResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode OpenSky response: %w", err)
	}

	// Convert to events
	return p.convertToEvents(apiResp)
}

// convertToEvents converts OpenSky states to SENTINEL events
func (p *OpenSkyProvider) convertToEvents(resp OpenSkyResponse) ([]*model.Event, error) {
	var events []*model.Event
	maxFlights := 100 // Limit to 100 flights to avoid overwhelming the system
	flightCount := 0

	for _, stateArray := range resp.States {
		// Limit number of flights
		if flightCount >= maxFlights {
			break
		}
		
		if len(stateArray) < 17 {
			continue // Skip invalid states
		}

		// Parse state from array (OpenSky returns arrays, not objects)
		state, err := p.parseStateArray(stateArray)
		if err != nil {
			continue // Skip states that can't be parsed
		}

		// Skip if no position data
		if state.Latitude == 0 && state.Longitude == 0 {
			continue
		}

		// Create event
		event := &model.Event{
			ID:          uuid.New().String(),
			Title:       p.generateTitle(state),
			Description: p.generateDescription(state),
			Source:      p.name,
			SourceID:    state.Icao24,
			OccurredAt:  time.Unix(state.LastContact, 0).UTC(),
			IngestedAt:  time.Now().UTC(),
			Location: model.Location{
				Type:        "Point",
				Coordinates: []float64{state.Longitude, state.Latitude},
			},
			Precision: model.PrecisionExact,
			Magnitude: p.calculateMagnitude(state),
			Category:  "flight",
			Severity:  model.Severity(p.determineSeverity(state)),
			Metadata: map[string]string{
				"icao24":         state.Icao24,
				"callsign":       state.Callsign,
				"origin_country": state.OriginCountry,
				"altitude":       fmt.Sprintf("%.0f", state.GeoAltitude),
				"velocity":       fmt.Sprintf("%.0f", state.Velocity),
				"on_ground":      fmt.Sprintf("%v", state.OnGround),
				"vertical_rate":  fmt.Sprintf("%.0f", state.VerticalRate),
				"true_track":     fmt.Sprintf("%.0f", state.TrueTrack),
				"squawk":         state.Squawk,
			},
		}

		events = append(events, event)
		flightCount++
	}

	log.Printf("OpenSky: Returning %d flights (limited from %d total)", len(events), len(resp.States))
	return events, nil
}

// parseStateArray parses an OpenSky state array into a struct
func (p *OpenSkyProvider) parseStateArray(arr []interface{}) (*OpenSkyState, error) {
	state := &OpenSkyState{}

	// Parse fields based on OpenSky API documentation
	// https://opensky-network.org/apidoc/rest.html#all-state-vectors
	if v, ok := arr[0].(string); ok {
		state.Icao24 = v
	}
	if v, ok := arr[1].(string); ok && v != "" {
		state.Callsign = v
	}
	if v, ok := arr[2].(string); ok {
		state.OriginCountry = v
	}
	if v, ok := arr[3].(float64); ok && v > 0 {
		state.TimePosition = int64(v)
	}
	if v, ok := arr[4].(float64); ok {
		state.LastContact = int64(v)
	}
	if v, ok := arr[5].(float64); ok {
		state.Longitude = v
	}
	if v, ok := arr[6].(float64); ok {
		state.Latitude = v
	}
	if v, ok := arr[7].(float64); ok {
		state.BaroAltitude = v
	}
	if v, ok := arr[8].(bool); ok {
		state.OnGround = v
	}
	if v, ok := arr[9].(float64); ok {
		state.Velocity = v
	}
	if v, ok := arr[10].(float64); ok {
		state.TrueTrack = v
	}
	if v, ok := arr[11].(float64); ok {
		state.VerticalRate = v
	}
	if v, ok := arr[12].([]interface{}); ok {
		sensors := make([]int, len(v))
		for i, sensor := range v {
			if s, ok := sensor.(float64); ok {
				sensors[i] = int(s)
			}
		}
		state.Sensors = sensors
	}
	if v, ok := arr[13].(float64); ok {
		state.GeoAltitude = v
	}
	if v, ok := arr[14].(string); ok {
		state.Squawk = v
	}
	if v, ok := arr[15].(bool); ok {
		state.Spi = v
	}
	if v, ok := arr[16].(float64); ok {
		state.PositionSource = int(v)
	}

	return state, nil
}

// generateTitle generates a title for the flight event
func (p *OpenSkyProvider) generateTitle(state *OpenSkyState) string {
	if state.Callsign != "" {
		return fmt.Sprintf("Flight %s", state.Callsign)
	}
	return fmt.Sprintf("Aircraft %s", state.Icao24)
}

// generateDescription generates a description for the flight event
func (p *OpenSkyProvider) generateDescription(state *OpenSkyState) string {
	desc := fmt.Sprintf("Aircraft from %s", state.OriginCountry)
	
	if state.Callsign != "" {
		desc += fmt.Sprintf(" (Callsign: %s)", state.Callsign)
	}
	
	if state.GeoAltitude > 0 {
		desc += fmt.Sprintf(" at %.0f meters", state.GeoAltitude)
	}
	
	if state.Velocity > 0 {
		desc += fmt.Sprintf(", speed %.0f m/s", state.Velocity)
	}
	
	if state.OnGround {
		desc += " (on ground)"
	} else {
		desc += " (in flight)"
	}
	
	return desc
}

// calculateMagnitude calculates a magnitude value for the flight
func (p *OpenSkyProvider) calculateMagnitude(state *OpenSkyState) float64 {
	// Use altitude as magnitude (higher altitude = higher magnitude)
	if state.GeoAltitude > 0 {
		return state.GeoAltitude / 1000.0 // Convert to kilometers
	}
	return 0.0
}

// determineSeverity determines the severity of the flight event
func (p *OpenSkyProvider) determineSeverity(state *OpenSkyState) string {
	// Simple severity based on altitude and speed
	if state.OnGround {
		return "low"
	}
	
	if state.GeoAltitude > 10000 || state.Velocity > 200 {
		return "high"
	}
	
	return "medium"
}