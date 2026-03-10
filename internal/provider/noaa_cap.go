package provider

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/openclaw/sentinel-backend/internal/model"
)

// NOAACAPProvider fetches Common Alerting Protocol alerts from NOAA
type NOAACAPProvider struct {
	client *http.Client
	config *Config
}




// NewNOAACAPProvider creates a new NOAACAPProvider
func NewNOAACAPProvider(config *Config) *NOAACAPProvider {
	return &NOAACAPProvider{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		config: config,
	}
}

// Fetch retrieves CAP alerts from NOAA
func (p *NOAACAPProvider) Fetch(ctx context.Context) ([]*model.Event, error) {
	// NOAA CAP feed for US alerts
	url := "https://alerts.weather.gov/cap/us.php?x=0"
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create NOAA CAP request: %w", err)
	}
	
	req.Header.Set("User-Agent", "SENTINEL/2.0 (https://github.com/ward331/sentinel)")
	req.Header.Set("Accept", "application/cap+xml, application/xml, text/xml")
	
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch NOAA CAP feed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("NOAA CAP returned status %d: %s", resp.StatusCode, string(body))
	}
	
	// Parse CAP XML
	var capFeed CAPFeed
	decoder := xml.NewDecoder(resp.Body)
	if err := decoder.Decode(&capFeed); err != nil {
		return nil, fmt.Errorf("failed to parse NOAA CAP XML: %w", err)
	}
	
	return p.convertToEvents(capFeed)
}

// CAPFeed represents a CAP XML feed
type CAPFeed struct {
	XMLName xml.Name `xml:"feed"`
	Entries []CAPEntry `xml:"entry"`
}

// CAPEntry represents a CAP alert entry
type CAPEntry struct {
	ID      string `xml:"id"`
	Title   string `xml:"title"`
	Updated string `xml:"updated"`
	Summary string `xml:"summary"`
	Link    struct {
		Href string `xml:"href,attr"`
	} `xml:"link"`
	CapAlert struct {
		Identifier  string `xml:"identifier"`
		Sender      string `xml:"sender"`
		Sent        string `xml:"sent"`
		Status      string `xml:"status"`
		MsgType     string `xml:"msgType"`
		Scope       string `xml:"scope"`
		Category    string `xml:"category"`
		Event       string `xml:"event"`
		Urgency     string `xml:"urgency"`
		Severity    string `xml:"severity"`
		Certainty   string `xml:"certainty"`
		AreaDesc    string `xml:"areaDesc"`
		Polygon     string `xml:"polygon"`
		Geocode    []struct {
			ValueName string `xml:"valueName"`
			Value     string `xml:"value"`
		} `xml:"geocode"`
		Parameter []struct {
			ValueName string `xml:"valueName"`
			Value     string `xml:"value"`
		} `xml:"parameter"`
	} `xml:"alert"`
}

// convertToEvents converts CAP alerts to SENTINEL events
func (p *NOAACAPProvider) convertToEvents(feed CAPFeed) ([]*model.Event, error) {
	var events []*model.Event
	
	for _, entry := range feed.Entries {
		// Skip expired or test alerts
		if entry.CapAlert.Status == "Expired" || entry.CapAlert.MsgType == "Test" {
			continue
		}
		
		event := &model.Event{
			Title:       p.generateTitle(entry),
			Description: p.generateDescription(entry),
			Source:      "noaa_cap",
			SourceID:    entry.CapAlert.Identifier,
			OccurredAt:  p.parseTime(entry.CapAlert.Sent),
			Location:    p.extractLocation(entry),
			Precision:   model.PrecisionPolygonArea,
			Magnitude:   p.calculateMagnitude(entry),
			Category:    "weather",
			Severity:    p.determineSeverity(entry),
			Metadata:    p.generateMetadata(entry),
			Badges:      p.generateBadges(entry),
		}
		
		events = append(events, event)
	}
	
	return events, nil
}

// generateTitle creates a title for the CAP alert
func (p *NOAACAPProvider) generateTitle(entry CAPEntry) string {
	// Add emoji based on event type
	emoji := "⚠️"
	eventType := strings.ToLower(entry.CapAlert.Event)
	
	switch {
	case strings.Contains(eventType, "tornado"):
		emoji = "🌪️"
	case strings.Contains(eventType, "hurricane") || strings.Contains(eventType, "typhoon"):
		emoji = "🌀"
	case strings.Contains(eventType, "flood"):
		emoji = "🌊"
	case strings.Contains(eventType, "fire"):
		emoji = "🔥"
	case strings.Contains(eventType, "winter"):
		emoji = "❄️"
	case strings.Contains(eventType, "heat"):
		emoji = "☀️"
	case strings.Contains(eventType, "thunderstorm"):
		emoji = "⛈️"
	}
	
	return fmt.Sprintf("%s %s: %s", emoji, entry.CapAlert.Event, entry.CapAlert.AreaDesc)
}

// generateDescription creates a description for the CAP alert
func (p *NOAACAPProvider) generateDescription(entry CAPEntry) string {
	var builder strings.Builder
	
	builder.WriteString(fmt.Sprintf("%s\n\n", entry.Title))
	builder.WriteString(fmt.Sprintf("Event: %s\n", entry.CapAlert.Event))
	builder.WriteString(fmt.Sprintf("Area: %s\n", entry.CapAlert.AreaDesc))
	builder.WriteString(fmt.Sprintf("Urgency: %s\n", entry.CapAlert.Urgency))
	builder.WriteString(fmt.Sprintf("Severity: %s\n", entry.CapAlert.Severity))
	builder.WriteString(fmt.Sprintf("Certainty: %s\n", entry.CapAlert.Certainty))
	builder.WriteString(fmt.Sprintf("Status: %s\n", entry.CapAlert.Status))
	builder.WriteString(fmt.Sprintf("Issued by: %s\n", entry.CapAlert.Sender))
	builder.WriteString(fmt.Sprintf("Issued at: %s\n\n", p.formatTime(entry.CapAlert.Sent)))
	
	if entry.Summary != "" {
		builder.WriteString(fmt.Sprintf("Details:\n%s\n\n", entry.Summary))
	}
	
	// Add geocodes
	if len(entry.CapAlert.Geocode) > 0 {
		builder.WriteString("Affected areas:\n")
		for _, geocode := range entry.CapAlert.Geocode {
			builder.WriteString(fmt.Sprintf("• %s: %s\n", geocode.ValueName, geocode.Value))
		}
		builder.WriteString("\n")
	}
	
	builder.WriteString(fmt.Sprintf("Source: NOAA National Weather Service - CAP Alerts"))
	builder.WriteString(fmt.Sprintf("\nAlert ID: %s", entry.CapAlert.Identifier))
	
	return builder.String()
}

// extractLocation extracts location from CAP alert
func (p *NOAACAPProvider) extractLocation(entry CAPEntry) model.GeoJSON {
	// Try to parse polygon if available
	if entry.CapAlert.Polygon != "" {
		coords := strings.Split(entry.CapAlert.Polygon, " ")
		if len(coords) >= 6 { // Need at least 3 points (6 coordinates)
			var polygonCoords [][]float64
			
			for i := 0; i < len(coords); i += 2 {
				if i+1 < len(coords) {
					lon, err1 := strconv.ParseFloat(coords[i], 64)
					lat, err2 := strconv.ParseFloat(coords[i+1], 64)
					if err1 == nil && err2 == nil {
						polygonCoords = append(polygonCoords, []float64{lon, lat})
					}
				}
			}
			
			if len(polygonCoords) >= 3 {
				// Close the polygon
				polygonCoords = append(polygonCoords, polygonCoords[0])
				
				return model.GeoJSON{
					Type:        "Polygon",
					Coordinates: [][][]float64{polygonCoords},
				}
			}
		}
	}
	
	// Fallback to approximate center of US
	return model.GeoJSON{
		Type:        "Point",
		Coordinates: []float64{-95.7129, 37.0902},
	}
}

// calculateMagnitude calculates magnitude based on CAP alert severity
func (p *NOAACAPProvider) calculateMagnitude(entry CAPEntry) float64 {
	magnitude := 4.0 // Base for weather alerts
	
	// Adjust based on severity
	switch strings.ToLower(entry.CapAlert.Severity) {
	case "extreme":
		magnitude += 3.0
	case "severe":
		magnitude += 2.0
	case "moderate":
		magnitude += 1.0
	case "minor":
		magnitude += 0.5
	}
	
	// Adjust based on urgency
	switch strings.ToLower(entry.CapAlert.Urgency) {
	case "immediate":
		magnitude += 2.0
	case "expected":
		magnitude += 1.0
	case "future":
		magnitude += 0.5
	}
	
	// Adjust based on certainty
	switch strings.ToLower(entry.CapAlert.Certainty) {
	case "observed":
		magnitude += 1.5
	case "likely":
		magnitude += 1.0
	case "possible":
		magnitude += 0.5
	}
	
	// Adjust based on event type
	eventType := strings.ToLower(entry.CapAlert.Event)
	if strings.Contains(eventType, "tornado") || strings.Contains(eventType, "hurricane") {
		magnitude += 2.0
	} else if strings.Contains(eventType, "flood") || strings.Contains(eventType, "fire") {
		magnitude += 1.5
	} else if strings.Contains(eventType, "warning") {
		magnitude += 1.0
	} else if strings.Contains(eventType, "watch") {
		magnitude += 0.5
	}
	
	return magnitude
}

// determineSeverity determines event severity from CAP alert
func (p *NOAACAPProvider) determineSeverity(entry CAPEntry) model.Severity {
	switch strings.ToLower(entry.CapAlert.Severity) {
	case "extreme":
		return model.SeverityCritical
	case "severe":
		return model.SeverityHigh
	case "moderate":
		return model.SeverityMedium
	case "minor":
		return model.SeverityLow
	default:
		return model.SeverityMedium
	}
}

// generateMetadata creates metadata for the CAP alert
func (p *NOAACAPProvider) generateMetadata(entry CAPEntry) map[string]string {
	metadata := map[string]string{
		"source":      "NOAA NWS CAP",
		"identifier":  entry.CapAlert.Identifier,
		"sender":      entry.CapAlert.Sender,
		"sent":        entry.CapAlert.Sent,
		"status":      entry.CapAlert.Status,
		"msg_type":    entry.CapAlert.MsgType,
		"scope":       entry.CapAlert.Scope,
		"category":    entry.CapAlert.Category,
		"event":       entry.CapAlert.Event,
		"urgency":     entry.CapAlert.Urgency,
		"severity":    entry.CapAlert.Severity,
		"certainty":   entry.CapAlert.Certainty,
		"area_desc":   entry.CapAlert.AreaDesc,
		"timestamp":   p.parseTime(entry.CapAlert.Sent).Format(time.RFC3339),
	}
	
	if entry.CapAlert.Polygon != "" {
		metadata["polygon"] = entry.CapAlert.Polygon
	}
	
	// Add geocodes
	for _, geocode := range entry.CapAlert.Geocode {
		metadata[fmt.Sprintf("geocode_%s", geocode.ValueName)] = geocode.Value
	}
	
	// Add parameters
	for _, param := range entry.CapAlert.Parameter {
		metadata[fmt.Sprintf("param_%s", param.ValueName)] = param.Value
	}
	
	return metadata
}

// generateBadges creates badges for the CAP alert
func (p *NOAACAPProvider) generateBadges(entry CAPEntry) []model.Badge {
	timestamp := p.parseTime(entry.CapAlert.Sent)
	badges := []model.Badge{
		{
			Label:     "NOAA NWS",
			Type:      "source",
			Timestamp: timestamp,
		},
		{
			Label:     "Weather Alert",
			Type:      "category",
			Timestamp: timestamp,
		},
		{
			Label:     strings.Title(entry.CapAlert.Event),
			Type:      "event_type",
			Timestamp: timestamp,
		},
	}
	
	// Add severity badge
	severity := p.determineSeverity(entry)
	badges = append(badges, model.Badge{
		Label:     strings.Title(string(severity)),
		Type:      "severity",
		Timestamp: timestamp,
	})
	
	// Add urgency badge
	if entry.CapAlert.Urgency != "" {
		badges = append(badges, model.Badge{
			Label:     strings.Title(entry.CapAlert.Urgency),
			Type:      "urgency",
			Timestamp: timestamp,
		})
	}
	
	// Add certainty badge
	if entry.CapAlert.Certainty != "" {
		badges = append(badges, model.Badge{
			Label:     strings.Title(entry.CapAlert.Certainty),
			Type:      "certainty",
			Timestamp: timestamp,
		})
	}
	
	return badges
}

// parseTime parses CAP timestamp
func (p *NOAACAPProvider) parseTime(timeStr string) time.Time {
	if timeStr == "" {
		return time.Now().UTC()
	}
	
	// CAP uses ISO 8601 format: 2025-03-09T14:30:00-05:00
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		// Try without timezone
		t, err = time.Parse("2006-01-02T15:04:05", timeStr)
		if err != nil {
			return time.Now().UTC()
		}
	}
	
	return t.UTC()
}

// formatTime formats time for display
func (p *NOAACAPProvider) formatTime(timeStr string) string {
	t := p.parseTime(timeStr)
	return t.Format("January 2, 2006 15:04 MST")
}

