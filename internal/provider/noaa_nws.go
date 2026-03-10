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
)

// NOAANWSProvider fetches weather alerts from NOAA National Weather Service
type NOAANWSProvider struct {
	client *http.Client
	config *Config
}

// Name returns the provider name
func (p *NOAANWSProvider) Name() string {
    return "noaanws"
}

// Interval returns the polling interval
func (p *NOAANWSProvider) Interval() time.Duration {
    interval, _ := time.ParseDuration("1m")
    return interval
}

// Enabled returns whether the provider is enabled
func (p *NOAANWSProvider) Enabled() bool {
    return p.config != nil && p.config.Enabled
}

// NewNOAANWSProvider creates a new NOAANWSProvider
func NewNOAANWSProvider(config *Config) *NOAANWSProvider {
	return &NOAANWSProvider{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		config: config,
	}
}

// Fetch retrieves active weather alerts from NOAA NWS
func (p *NOAANWSProvider) Fetch(ctx context.Context) ([]*model.Event, error) {
	url := "https://api.weather.gov/alerts/active"
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// NOAA NWS requires User-Agent header
	req.Header.Set("User-Agent", "SENTINEL/2.0 (https://github.com/ward331/sentinel)")
	req.Header.Set("Accept", "application/geo+json")
	
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch NOAA NWS alerts: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("NOAA NWS API returned status %d: %s", resp.StatusCode, string(body))
	}
	
	var apiResponse NOAANWSResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("failed to decode API response: %w", err)
	}
	
	return p.convertToEvents(apiResponse)
}

// convertToEvents converts NOAA NWS API response to SENTINEL events
func (p *NOAANWSProvider) convertToEvents(response NOAANWSResponse) ([]*model.Event, error) {
	var events []*model.Event
	
	for _, feature := range response.Features {
		alert := feature.Properties
		
		// Skip expired or test alerts
		if alert.Status == "Expired" || strings.Contains(strings.ToLower(alert.Event), "test") {
			continue
		}
		
		event := &model.Event{
			Title:       p.generateTitle(alert),
			Description: p.generateDescription(alert),
			Source:      "noaa_nws",
			SourceID:    alert.ID,
			OccurredAt:  p.parseTime(alert.Sent),
			Location:    p.extractLocation(feature),
			Precision:   model.PrecisionExact,
			Magnitude:   p.calculateMagnitude(alert),
			Category:    p.determineCategory(alert),
			Severity:    p.determineSeverity(alert),
			Metadata:    p.generateMetadata(alert),
			Badges:      p.generateBadges(alert),
		}
		
		events = append(events, event)
	}
	
	return events, nil
}

// generateTitle creates a title for the weather alert
func (p *NOAANWSProvider) generateTitle(alert NOAANWSAlert) string {
	severity := strings.ToUpper(alert.Severity)
	return fmt.Sprintf("⚠️ %s %s - %s", severity, alert.Event, alert.AreaDesc)
}

// generateDescription creates a description for the weather alert
func (p *NOAANWSProvider) generateDescription(alert NOAANWSAlert) string {
	var desc strings.Builder
	
	desc.WriteString(fmt.Sprintf("%s issued by %s.\n\n", alert.Event, alert.SenderName))
	
	if alert.Headline != "" {
		desc.WriteString(fmt.Sprintf("Headline: %s\n", alert.Headline))
	}
	
	if alert.Description != "" {
		// Clean up description text
		cleanDesc := strings.ReplaceAll(alert.Description, "*", "")
		cleanDesc = strings.ReplaceAll(cleanDesc, "\n\n", "\n")
		desc.WriteString(fmt.Sprintf("Description: %s\n", cleanDesc))
	}
	
	if alert.Instruction != "" {
		desc.WriteString(fmt.Sprintf("Instructions: %s\n", alert.Instruction))
	}
	
	desc.WriteString(fmt.Sprintf("\nEffective: %s to %s", 
		p.formatTime(alert.Effective),
		p.formatTime(alert.Expires)))
	
	return desc.String()
}

// extractLocation extracts GeoJSON from the feature
func (p *NOAANWSProvider) extractLocation(feature NOAANWSFeature) model.GeoJSON {
	// Use the geometry from the feature if available
	if feature.Geometry.Type != "" && len(feature.Geometry.Coordinates) > 0 {
		return model.GeoJSON{
			Type:        feature.Geometry.Type,
			Coordinates: feature.Geometry.Coordinates,
		}
	}
	
	// Fallback to point based on area description
	return model.GeoJSON{
		Type:        "Point",
		Coordinates: []float64{-98.5795, 39.8283}, // Center of US
	}
}

// calculateMagnitude calculates magnitude based on alert severity
func (p *NOAANWSProvider) calculateMagnitude(alert NOAANWSAlert) float64 {
	switch strings.ToUpper(alert.Severity) {
	case "Extreme":
		return 9.0
	case "Severe":
		return 7.5
	case "Moderate":
		return 6.0
	case "Minor":
		return 4.5
	default:
		return 3.0
	}
}

// determineCategory determines the event category
func (p *NOAANWSProvider) determineCategory(alert NOAANWSAlert) string {
	event := strings.ToLower(alert.Event)
	
	switch {
	case strings.Contains(event, "tornado"):
		return "tornado"
	case strings.Contains(event, "hurricane") || strings.Contains(event, "typhoon"):
		return "hurricane"
	case strings.Contains(event, "flood"):
		return "flood"
	case strings.Contains(event, "fire"):
		return "wildfire"
	case strings.Contains(event, "winter") || strings.Contains(event, "snow") || strings.Contains(event, "ice"):
		return "winter_storm"
	case strings.Contains(event, "thunderstorm") || strings.Contains(event, "lightning"):
		return "thunderstorm"
	case strings.Contains(event, "heat"):
		return "heat_wave"
	case strings.Contains(event, "wind"):
		return "wind"
	default:
		return "weather"
	}
}

// determineSeverity determines the event severity
func (p *NOAANWSProvider) determineSeverity(alert NOAANWSAlert) string {
	switch strings.ToUpper(alert.Severity) {
	case "Extreme":
		return model.SeverityCritical
	case "Severe":
		return model.SeverityHigh
	case "Moderate":
		return model.SeverityMedium
	case "Minor":
		return model.SeverityLow
	default:
		return model.SeverityInfo
	}
}

// generateMetadata creates metadata for the weather alert
func (p *NOAANWSProvider) generateMetadata(alert NOAANWSAlert) map[string]string {
	metadata := map[string]string{
		"id":           alert.ID,
		"event":        alert.Event,
		"severity":     alert.Severity,
		"urgency":      alert.Urgency,
		"certainty":    alert.Certainty,
		"area_desc":    alert.AreaDesc,
		"sender_name":  alert.SenderName,
		"sent":         alert.Sent,
		"effective":    alert.Effective,
		"expires":      alert.Expires,
		"status":       alert.Status,
	}
	
	if alert.Headline != "" {
		metadata["headline"] = alert.Headline
	}
	if alert.Description != "" {
		metadata["description"] = alert.Description
	}
	if alert.Instruction != "" {
		metadata["instruction"] = alert.Instruction
	}
	
	// Add affected zones
	for i, zone := range alert.AffectedZones {
		metadata[fmt.Sprintf("zone_%d", i)] = zone
	}
	
	return metadata
}

// generateBadges creates badges for the weather alert
func (p *NOAANWSProvider) generateBadges(alert NOAANWSAlert) []model.Badge {
	badges := []model.Badge{
		{
			Label:     "NOAA NWS",
			Type:      "source",
			Timestamp: p.parseTime(alert.Sent),
		},
		{
			Label:     strings.ToUpper(alert.Severity),
			Type:      "severity",
			Timestamp: p.parseTime(alert.Sent),
		},
		{
			Label:     alert.Certainty,
			Type:      "certainty",
			Timestamp: p.parseTime(alert.Sent),
		},
	}
	
	// Add urgency badge
	if alert.Urgency != "" {
		badges = append(badges, model.Badge{
			Label:     alert.Urgency,
			Type:      "urgency",
			Timestamp: p.parseTime(alert.Sent),
		})
	}
	
	// Add polygon badge if geometry is available
	if alert.Geometry != "" {
		badges = append(badges, model.Badge{
			Label:     "Polygon Area",
			Type:      "precision",
			Timestamp: p.parseTime(alert.Sent),
		})
	}
	
	return badges
}

// parseTime parses ISO 8601 time string
func (p *NOAANWSProvider) parseTime(timeStr string) time.Time {
	if timeStr == "" {
		return time.Now().UTC()
	}
	
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		// Try other common formats
		t, err = time.Parse("2006-01-02T15:04:05-07:00", timeStr)
		if err != nil {
			return time.Now().UTC()
		}
	}
	
	return t.UTC()
}

// formatTime formats time for display
func (p *NOAANWSProvider) formatTime(timeStr string) string {
	t := p.parseTime(timeStr)
	return t.Format("Jan 2, 2006 15:04 MST")
}

// NOAANWSResponse represents the NOAA NWS API response
type NOAANWSResponse struct {
	Type     string           `json:"type"`
	Features []NOAANWSFeature `json:"features"`
}

// NOAANWSFeature represents a feature in the NOAA NWS API response
type NOAANWSFeature struct {
	Type       string          `json:"type"`
	Properties NOAANWSAlert    `json:"properties"`
	Geometry   NOAANWSGeometry `json:"geometry"`
}

// NOAANWSAlert represents a weather alert
type NOAANWSAlert struct {
	ID            string   `json:"id"`
	Event         string   `json:"event"`
	Severity      string   `json:"severity"`
	Urgency       string   `json:"urgency"`
	Certainty     string   `json:"certainty"`
	Headline      string   `json:"headline"`
	Description   string   `json:"description"`
	Instruction   string   `json:"instruction"`
	AreaDesc      string   `json:"areaDesc"`
	SenderName    string   `json:"senderName"`
	Sent          string   `json:"sent"`
	Effective     string   `json:"effective"`
	Expires       string   `json:"expires"`
	Status        string   `json:"status"`
	AffectedZones []string `json:"affectedZones"`
	Geometry      string   `json:"geometry"`
}

// NOAANWSGeometry represents the geometry of a weather alert
type NOAANWSGeometry struct {
	Type        string        `json:"type"`
	Coordinates []interface{} `json:"coordinates"`
}
