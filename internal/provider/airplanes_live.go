package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/openclaw/sentinel-backend/internal/model"
)

// AirplanesLiveProvider fetches real-time aircraft data from airplanes.live
type AirplanesLiveProvider struct {
	client *http.Client
	config *Config
}




// NewAirplanesLiveProvider creates a new AirplanesLiveProvider
func NewAirplanesLiveProvider(config *Config) *AirplanesLiveProvider {
	return &AirplanesLiveProvider{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		config: config,
	}
}

// Fetch retrieves aircraft data from airplanes.live
// Enabled returns whether the provider is enabled
func (p *AirplanesLiveProvider) Enabled() bool {
	if p.config != nil {
		return p.config.Enabled
	}
	return true
}

// Interval returns the polling interval
func (p *AirplanesLiveProvider) Interval() time.Duration {
	if p.config != nil && p.config.PollInterval > 0 {
		return p.config.PollInterval
	}
	return 5 * time.Minute // Default interval
}

// Name returns the provider identifier
func (p *AirplanesLiveProvider) Name() string {
	return "airplanes_live"
}

func (p *AirplanesLiveProvider) Fetch(ctx context.Context) ([]*model.Event, error) {
	// Get bounding box from config or use worldwide
	bbox := p.getBoundingBox()
	
	// Build URL for airplanes.live API
	url := fmt.Sprintf("https://api.airplanes.live/v2/point/%f/%f/%f", bbox.CenterLat, bbox.CenterLon, bbox.RadiusKm)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("User-Agent", "SENTINEL/2.0 (https://github.com/ward331/sentinel)")
	
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch aircraft data: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("airplanes.live API returned status %d: %s", resp.StatusCode, string(body))
	}
	
	var apiResponse AirplanesLiveResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("failed to decode API response: %w", err)
	}
	
	return p.convertToEvents(apiResponse)
}

// getBoundingBox returns the bounding box for aircraft queries
func (p *AirplanesLiveProvider) getBoundingBox() BoundingBox {
	// If location is configured in config, use it
	// Location is stored as string "lat,lon" in config
	if p.config != nil && p.config.Location != "" {
		// Parse location string
		var lat, lon float64
		if _, err := fmt.Sscanf(p.config.Location, "%f,%f", &lat, &lon); err == nil && lat != 0 && lon != 0 {
			return BoundingBox{
				CenterLat: lat,
				CenterLon: lon,
				RadiusKm:  500, // 500km radius around configured location
			}
		}
	}
	
	// Default to worldwide coverage
	return BoundingBox{
		CenterLat: 0,
		CenterLon: 0,
		RadiusKm:  20000, // Earth's radius in km
	}
}

// convertToEvents converts airplanes.live API response to SENTINEL events
func (p *AirplanesLiveProvider) convertToEvents(response AirplanesLiveResponse) ([]*model.Event, error) {
	var events []*model.Event
	
	for _, ac := range response.Aircraft {
		// Skip aircraft without position
		if ac.Lat == 0 && ac.Lon == 0 {
			continue
		}
		
		event := &model.Event{
			Title:       p.generateTitle(ac),
			Description: p.generateDescription(ac),
			Source:      "airplanes_live",
			SourceID:    fmt.Sprintf("airplanes_live_%s", ac.Hex),
			OccurredAt:  time.Now().UTC(),
			Location: model.GeoJSON{
				Type:        "Point",
				Coordinates: []float64{ac.Lon, ac.Lat},
			},
			Precision: model.PrecisionExact,
			Magnitude: p.calculateMagnitude(ac),
			Category:  p.determineCategory(ac),
			Severity:  model.SeverityLow,
			Metadata:  p.generateMetadata(ac),
			Badges:    p.generateBadges(ac),
		}
		
		events = append(events, event)
	}
	
	return events, nil
}

// generateTitle creates a title for the aircraft event
func (p *AirplanesLiveProvider) generateTitle(ac Aircraft) string {
	if ac.Flight != "" && ac.Flight != "N/A" {
		return fmt.Sprintf("✈️ Flight %s", ac.Flight)
	}
	if ac.Reg != "" && ac.Reg != "N/A" {
		return fmt.Sprintf("✈️ Aircraft %s", ac.Reg)
	}
	return fmt.Sprintf("✈️ Aircraft %s", ac.Hex)
}

// generateDescription creates a description for the aircraft event
func (p *AirplanesLiveProvider) generateDescription(ac Aircraft) string {
	desc := fmt.Sprintf("Aircraft %s", ac.Hex)
	
	if ac.Reg != "" && ac.Reg != "N/A" {
		desc += fmt.Sprintf(" (Registration: %s)", ac.Reg)
	}
	if ac.Type != "" && ac.Type != "N/A" {
		desc += fmt.Sprintf(" - Type: %s", ac.Type)
	}
	if ac.AltBaro != 0 {
		desc += fmt.Sprintf(" - Altitude: %d ft", ac.AltBaro)
	}
	if ac.Speed != 0 {
		desc += fmt.Sprintf(" - Speed: %d kts", ac.Speed)
	}
	if ac.Track != 0 {
		desc += fmt.Sprintf(" - Heading: %d°", ac.Track)
	}
	
	return desc
}

// calculateMagnitude calculates magnitude based on aircraft properties
func (p *AirplanesLiveProvider) calculateMagnitude(ac Aircraft) float64 {
	// Base magnitude on altitude and speed
	magnitude := 1.0
	
	if ac.AltBaro > 30000 {
		magnitude += 2.0 // High altitude
	} else if ac.AltBaro > 10000 {
		magnitude += 1.0
	}
	
	if ac.Speed > 400 {
		magnitude += 2.0 // High speed
	} else if ac.Speed > 200 {
		magnitude += 1.0
	}
	
	return magnitude
}

// determineCategory determines the event category
func (p *AirplanesLiveProvider) determineCategory(ac Aircraft) string {
	// Check if it's a military aircraft
	if ac.Type != "" && (ac.Type[0] == 'F' || // Fighter
		ac.Type[0] == 'B' || // Bomber
		ac.Type[0] == 'C' || // Cargo (military)
		ac.Type[0] == 'A') { // Attack
		return "military"
	}
	
	// Check for unusual patterns
	if ac.AltBaro < 1000 && ac.Speed > 300 {
		return "suspicious" // Low altitude, high speed
	}
	
	return "aviation"
}

// generateMetadata creates metadata for the aircraft event
func (p *AirplanesLiveProvider) generateMetadata(ac Aircraft) map[string]string {
	metadata := map[string]string{
		"hex":       ac.Hex,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	
	if ac.Reg != "" && ac.Reg != "N/A" {
		metadata["registration"] = ac.Reg
	}
	if ac.Flight != "" && ac.Flight != "N/A" {
		metadata["flight"] = ac.Flight
	}
	if ac.Type != "" && ac.Type != "N/A" {
		metadata["type"] = ac.Type
	}
	if ac.AltBaro != 0 {
		metadata["altitude_ft"] = fmt.Sprintf("%d", ac.AltBaro)
	}
	if ac.Speed != 0 {
		metadata["speed_kts"] = fmt.Sprintf("%d", ac.Speed)
	}
	if ac.Track != 0 {
		metadata["heading_deg"] = fmt.Sprintf("%d", ac.Track)
	}
	if ac.VertRate != 0 {
		metadata["vertical_rate_fpm"] = fmt.Sprintf("%d", ac.VertRate)
	}
	if ac.Squawk != "" && ac.Squawk != "N/A" {
		metadata["squawk"] = ac.Squawk
	}
	
	return metadata
}

// generateBadges creates badges for the aircraft event
func (p *AirplanesLiveProvider) generateBadges(ac Aircraft) []model.Badge {
	badges := []model.Badge{
		{
			Label:     "Real-time",
			Type:      "freshness",
			Timestamp: time.Now().UTC(),
		},
		{
			Label:     "ADS-B",
			Type:      "source",
			Timestamp: time.Now().UTC(),
		},
	}
	
	// Add altitude badge
	if ac.AltBaro > 30000 {
		badges = append(badges, model.Badge{
			Label:     "High Altitude",
			Type:      "altitude",
			Timestamp: time.Now().UTC(),
		})
	}
	
	// Add speed badge
	if ac.Speed > 400 {
		badges = append(badges, model.Badge{
			Label:     "High Speed",
			Type:      "speed",
			Timestamp: time.Now().UTC(),
		})
	}
	
	// Add military badge if applicable
	if p.determineCategory(ac) == "military" {
		badges = append(badges, model.Badge{
			Label:     "Military",
			Type:      "classification",
			Timestamp: time.Now().UTC(),
		})
	}
	
	return badges
}

// BoundingBox represents a geographic bounding box
type BoundingBox struct {
	CenterLat float64
	CenterLon float64
	RadiusKm  float64
}

// AirplanesLiveResponse represents the airplanes.live API response
type AirplanesLiveResponse struct {
	Now      float64    `json:"now"`
	Aircraft []Aircraft `json:"aircraft"`
}

// Aircraft represents an aircraft in the airplanes.live API
type Aircraft struct {
	Hex     string  `json:"hex"`
	Reg     string  `json:"reg"`
	Type    string  `json:"type"`
	Flight  string  `json:"flight"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
	AltBaro int     `json:"alt_baro"`
	Speed   int     `json:"speed"`
	Track   int     `json:"track"`
	VertRate int    `json:"vert_rate"`
	Squawk  string  `json:"squawk"`
	Category string `json:"category"`
}
