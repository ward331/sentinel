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

// ADSBOneProvider fetches aircraft data from ADSB.one as a fallback
type ADSBOneProvider struct {
	client *http.Client
	config *Config
}

// NewADSBOneProvider creates a new ADSBOneProvider
func NewADSBOneProvider(config *Config) *ADSBOneProvider {
	return &ADSBOneProvider{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		config: config,
	}
}

// Fetch retrieves aircraft data from ADSB.one
func (p *ADSBOneProvider) Fetch(ctx context.Context) ([]*model.Event, error) {
	// ADSB.one provides worldwide coverage
	url := "https://api.adsb.one/v2/aircraft"
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("User-Agent", "SENTINEL/2.0 (https://github.com/ward331/sentinel)")
	req.Header.Set("Accept", "application/json")
	
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ADSB.one data: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ADSB.one API returned status %d: %s", resp.StatusCode, string(body))
	}
	
	var apiResponse ADSBOneResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("failed to decode API response: %w", err)
	}
	
	return p.convertToEvents(apiResponse)
}

// convertToEvents converts ADSB.one API response to SENTINEL events
func (p *ADSBOneProvider) convertToEvents(response ADSBOneResponse) ([]*model.Event, error) {
	var events []*model.Event
	
	now := time.Now().UTC()
	
	for _, ac := range response.Aircraft {
		// Skip aircraft without position or stale data
		if ac.Lat == 0 && ac.Lon == 0 {
			continue
		}
		
		// Check if data is recent (within last 5 minutes)
		if ac.PositionTime > 0 {
			positionTime := time.Unix(int64(ac.PositionTime), 0)
			if now.Sub(positionTime) > 5*time.Minute {
				continue
			}
		}
		
		event := &model.Event{
			Title:       p.generateTitle(ac),
			Description: p.generateDescription(ac),
			Source:      "adsb_one",
			SourceID:    fmt.Sprintf("adsb_one_%s", ac.Hex),
			OccurredAt:  now,
			Location: model.GeoJSON{
				Type:        "Point",
				Coordinates: []float64{ac.Lon, ac.Lat},
			},
			Precision: model.PrecisionExact,
			Magnitude: p.calculateMagnitude(ac),
			Category:  p.determineCategory(ac),
			Severity:  model.SeverityLow,
			Metadata:  p.generateMetadata(ac, now),
			Badges:    p.generateBadges(ac, now),
		}
		
		events = append(events, event)
	}
	
	return events, nil
}

// generateTitle creates a title for the aircraft event
func (p *ADSBOneProvider) generateTitle(ac ADSBOneAircraft) string {
	if ac.Flight != "" && ac.Flight != "N/A" {
		return fmt.Sprintf("✈️ Flight %s", ac.Flight)
	}
	if ac.Registration != "" && ac.Registration != "N/A" {
		return fmt.Sprintf("✈️ Aircraft %s", ac.Registration)
	}
	return fmt.Sprintf("✈️ Aircraft %s", ac.Hex)
}

// generateDescription creates a description for the aircraft event
func (p *ADSBOneProvider) generateDescription(ac ADSBOneAircraft) string {
	desc := fmt.Sprintf("Aircraft %s", ac.Hex)
	
	if ac.Registration != "" && ac.Registration != "N/A" {
		desc += fmt.Sprintf(" (Registration: %s)", ac.Registration)
	}
	if ac.Type != "" && ac.Type != "N/A" {
		desc += fmt.Sprintf(" - Type: %s", ac.Type)
	}
	if ac.Altitude != 0 {
		desc += fmt.Sprintf(" - Altitude: %d ft", ac.Altitude)
	}
	if ac.Speed != 0 {
		desc += fmt.Sprintf(" - Speed: %d kts", ac.Speed)
	}
	if ac.Heading != 0 {
		desc += fmt.Sprintf(" - Heading: %d°", ac.Heading)
	}
	
	return desc
}

// calculateMagnitude calculates magnitude based on aircraft properties
func (p *ADSBOneProvider) calculateMagnitude(ac ADSBOneAircraft) float64 {
	magnitude := 1.0
	
	if ac.Altitude > 30000 {
		magnitude += 2.0
	} else if ac.Altitude > 10000 {
		magnitude += 1.0
	}
	
	if ac.Speed > 400 {
		magnitude += 2.0
	} else if ac.Speed > 200 {
		magnitude += 1.0
	}
	
	// Increase magnitude for military aircraft
	if ac.Category == "MIL" {
		magnitude += 1.5
	}
	
	return magnitude
}

// determineCategory determines the event category
func (p *ADSBOneProvider) determineCategory(ac ADSBOneAircraft) string {
	switch ac.Category {
	case "MIL":
		return "military"
	case "GOV":
		return "government"
	case "COM":
		return "commercial"
	case "PRV":
		return "private"
	default:
		return "aviation"
	}
}

// generateMetadata creates metadata for the aircraft event
func (p *ADSBOneProvider) generateMetadata(ac ADSBOneAircraft, timestamp time.Time) map[string]string {
	metadata := map[string]string{
		"hex":       ac.Hex,
		"timestamp": timestamp.Format(time.RFC3339),
	}
	
	if ac.Registration != "" && ac.Registration != "N/A" {
		metadata["registration"] = ac.Registration
	}
	if ac.Flight != "" && ac.Flight != "N/A" {
		metadata["flight"] = ac.Flight
	}
	if ac.Type != "" && ac.Type != "N/A" {
		metadata["type"] = ac.Type
	}
	if ac.Altitude != 0 {
		metadata["altitude_ft"] = fmt.Sprintf("%d", ac.Altitude)
	}
	if ac.Speed != 0 {
		metadata["speed_kts"] = fmt.Sprintf("%d", ac.Speed)
	}
	if ac.Heading != 0 {
		metadata["heading_deg"] = fmt.Sprintf("%d", ac.Heading)
	}
	if ac.VerticalRate != 0 {
		metadata["vertical_rate_fpm"] = fmt.Sprintf("%d", ac.VerticalRate)
	}
	if ac.Squawk != "" && ac.Squawk != "N/A" {
		metadata["squawk"] = ac.Squawk
	}
	if ac.Category != "" && ac.Category != "N/A" {
		metadata["category"] = ac.Category
	}
	if ac.PositionTime > 0 {
		positionTime := time.Unix(int64(ac.PositionTime), 0)
		metadata["position_time"] = positionTime.Format(time.RFC3339)
	}
	
	return metadata
}

// generateBadges creates badges for the aircraft event
func (p *ADSBOneProvider) generateBadges(ac ADSBOneAircraft, timestamp time.Time) []model.Badge {
	badges := []model.Badge{
		{
			Label:     "Real-time",
			Type:      "freshness",
			Timestamp: timestamp,
		},
		{
			Label:     "ADS-B",
			Type:      "source",
			Timestamp: timestamp,
		},
	}
	
	// Add altitude badge
	if ac.Altitude > 30000 {
		badges = append(badges, model.Badge{
			Label:     "High Altitude",
			Type:      "altitude",
			Timestamp: timestamp,
		})
	}
	
	// Add speed badge
	if ac.Speed > 400 {
		badges = append(badges, model.Badge{
			Label:     "High Speed",
			Type:      "speed",
			Timestamp: timestamp,
		})
	}
	
	// Add category badge
	if ac.Category != "" && ac.Category != "N/A" {
		badges = append(badges, model.Badge{
			Label:     ac.Category,
			Type:      "classification",
			Timestamp: timestamp,
		})
	}
	
	// Add data freshness badge
	if ac.PositionTime > 0 {
		positionTime := time.Unix(int64(ac.PositionTime), 0)
		age := timestamp.Sub(positionTime)
		
		if age < 30*time.Second {
			badges = append(badges, model.Badge{
				Label:     "Live",
				Type:      "freshness",
				Timestamp: timestamp,
			})
		} else if age < 2*time.Minute {
			badges = append(badges, model.Badge{
				Label:     "Recent",
				Type:      "freshness",
				Timestamp: timestamp,
			})
		}
	}
	
	return badges
}

// ADSBOneResponse represents the ADSB.one API response
type ADSBOneResponse struct {
	Now      float64           `json:"now"`
	Aircraft []ADSBOneAircraft `json:"aircraft"`
}

// ADSBOneAircraft represents an aircraft in the ADSB.one API
type ADSBOneAircraft struct {
	Hex         string  `json:"hex"`
	Registration string `json:"registration"`
	Type        string  `json:"type"`
	Flight      string  `json:"flight"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	Altitude    int     `json:"altitude"`
	Speed       int     `json:"speed"`
	Heading     int     `json:"heading"`
	VerticalRate int    `json:"vertical_rate"`
	Squawk      string  `json:"squawk"`
	Category    string  `json:"category"`
	PositionTime float64 `json:"position_time"`
}