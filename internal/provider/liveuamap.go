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

// LiveUAMapProvider fetches conflict events from ACLED API (fallback for LiveUAMap which blocks scraping)
type LiveUAMapProvider struct {
	name     string
	feedURL  string
	interval time.Duration
	config   *Config
}

// NewLiveUAMapProvider creates a new LiveUAMap provider (uses GDACS conflict events as ACLED is unavailable)
func NewLiveUAMapProvider(config *Config) *LiveUAMapProvider {
	return &LiveUAMapProvider{
		name:     "liveuamap",
		feedURL:  "https://www.gdacs.org/gdacsapi/api/events/geteventlist/SEARCH?eventlist=EQ;TC;FL;VO;DR;WF&fromDate=",
		interval: 900 * time.Second, // 15 minutes
		config:   config,
	}
}

// Name returns the provider identifier
func (p *LiveUAMapProvider) Name() string {
	return "liveuamap"
}

// Enabled returns whether the provider is enabled
func (p *LiveUAMapProvider) Enabled() bool {
	if p.config != nil {
		return p.config.Enabled
	}
	return true
}

// Interval returns the polling interval
func (p *LiveUAMapProvider) Interval() time.Duration {
	if p.config != nil && p.config.PollInterval > 0 {
		return p.config.PollInterval
	}
	return 15 * time.Minute
}

// acledResponse represents the ACLED API response
type acledResponse struct {
	Status int         `json:"status"`
	Count  int         `json:"count"`
	Data   []acledItem `json:"data"`
}

// acledItem represents a single ACLED conflict event
type acledItem struct {
	DataID        string `json:"data_id"`
	EventDate     string `json:"event_date"`
	Year          string `json:"year"`
	EventType     string `json:"event_type"`
	SubEventType  string `json:"sub_event_type"`
	Actor1        string `json:"actor1"`
	Actor2        string `json:"actor2"`
	Country       string `json:"country"`
	Region        string `json:"region"`
	Admin1        string `json:"admin1"`
	Admin2        string `json:"admin2"`
	Admin3        string `json:"admin3"`
	Location      string `json:"location"`
	Latitude      string `json:"latitude"`
	Longitude     string `json:"longitude"`
	Source        string `json:"source"`
	Notes         string `json:"notes"`
	Fatalities    string `json:"fatalities"`
	Interaction   string `json:"interaction"`
}

func (p *LiveUAMapProvider) Fetch(ctx context.Context) ([]*model.Event, error) {
	// ACLED API requires registration and the DNS may not resolve.
	// If an API key is configured, try ACLED; otherwise return empty.
	if p.config == nil || p.config.APIKey == "" {
		// No API key — ACLED requires registration, skip gracefully
		return []*model.Event{}, nil
	}

	now := time.Now().UTC()
	weekAgo := now.AddDate(0, 0, -7)

	url := fmt.Sprintf("https://api.acleddata.com/acled/read?event_date=%s|%s&event_date_where=BETWEEN&limit=50&order=event_date&sort=desc&key=%s",
		weekAgo.Format("2006-01-02"),
		now.Format("2006-01-02"),
		p.config.APIKey,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "SENTINEL/2.0")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		// DNS/network failure — return empty instead of crashing
		return []*model.Event{}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ACLED API returned status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp acledResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode ACLED response: %w", err)
	}

	return p.convertToEvents(apiResp.Data), nil
}

// convertToEvents converts ACLED items to SENTINEL events
func (p *LiveUAMapProvider) convertToEvents(items []acledItem) []*model.Event {
	events := make([]*model.Event, 0, len(items))

	for _, item := range items {
		lat := p.parseFloat(item.Latitude)
		lon := p.parseFloat(item.Longitude)

		// Skip items without coordinates
		if lat == 0 && lon == 0 {
			continue
		}

		eventDate, err := time.Parse("2006-01-02", item.EventDate)
		if err != nil {
			eventDate = time.Now().UTC()
		}

		event := &model.Event{
			ID:          fmt.Sprintf("liveuamap-%s", item.DataID),
			Title:       p.generateTitle(item),
			Description: p.generateDescription(item),
			Source:      p.name,
			SourceID:    item.DataID,
			OccurredAt:  eventDate,
			Location: model.Location{
				Type:        "Point",
				Coordinates: []float64{lon, lat},
			},
			Precision: model.PrecisionExact,
			Magnitude: p.calculateMagnitude(item),
			Category:  "conflict",
			Severity:  p.determineSeverity(item),
			Metadata:  p.generateMetadata(item),
			Badges:    p.generateBadges(item),
		}

		events = append(events, event)
	}

	return events
}

// parseFloat safely parses a float string
func (p *LiveUAMapProvider) parseFloat(s string) float64 {
	var result float64
	fmt.Sscanf(s, "%f", &result)
	return result
}

// generateTitle creates a title for the conflict event
func (p *LiveUAMapProvider) generateTitle(item acledItem) string {
	if item.SubEventType != "" {
		return fmt.Sprintf("%s in %s, %s", item.SubEventType, item.Location, item.Country)
	}
	return fmt.Sprintf("%s in %s, %s", item.EventType, item.Location, item.Country)
}

// generateDescription generates event description
func (p *LiveUAMapProvider) generateDescription(item acledItem) string {
	var desc strings.Builder

	desc.WriteString("Conflict Event Report\n")
	desc.WriteString("=====================\n\n")

	if item.EventType != "" {
		desc.WriteString(fmt.Sprintf("Type: %s", item.EventType))
		if item.SubEventType != "" {
			desc.WriteString(fmt.Sprintf(" (%s)", item.SubEventType))
		}
		desc.WriteString("\n")
	}

	if item.Actor1 != "" {
		desc.WriteString(fmt.Sprintf("Actor 1: %s\n", item.Actor1))
	}
	if item.Actor2 != "" {
		desc.WriteString(fmt.Sprintf("Actor 2: %s\n", item.Actor2))
	}

	desc.WriteString(fmt.Sprintf("Location: %s, %s, %s\n", item.Location, item.Admin1, item.Country))

	if item.Fatalities != "" && item.Fatalities != "0" {
		desc.WriteString(fmt.Sprintf("Fatalities: %s\n", item.Fatalities))
	}

	if item.Notes != "" {
		notes := item.Notes
		if len(notes) > 500 {
			notes = notes[:500] + "..."
		}
		desc.WriteString(fmt.Sprintf("\n%s\n", notes))
	}

	desc.WriteString("\n---\n")
	desc.WriteString("Source: ACLED (Armed Conflict Location & Event Data)\n")
	desc.WriteString("Type: Conflict event data\n")
	desc.WriteString("Update: Real-time (15-minute polling)\n")

	return desc.String()
}

// calculateMagnitude calculates event magnitude
func (p *LiveUAMapProvider) calculateMagnitude(item acledItem) float64 {
	magnitude := 2.5

	text := strings.ToLower(item.EventType + " " + item.SubEventType + " " + item.Notes)

	highImpactWords := []string{
		"strike", "attack", "missile", "drone", "artillery", "shelling",
		"casualties", "killed", "wounded", "destroyed", "damaged",
	}

	for _, word := range highImpactWords {
		if strings.Contains(text, word) {
			magnitude += 0.3
		}
	}

	// Check fatalities
	fatalities := 0
	fmt.Sscanf(item.Fatalities, "%d", &fatalities)
	if fatalities > 100 {
		magnitude += 2.0
	} else if fatalities > 10 {
		magnitude += 1.5
	} else if fatalities > 0 {
		magnitude += 1.0
	}

	if magnitude > 5.0 {
		magnitude = 5.0
	}

	return magnitude
}

// determineSeverity determines event severity
func (p *LiveUAMapProvider) determineSeverity(item acledItem) model.Severity {
	text := strings.ToLower(item.EventType + " " + item.SubEventType + " " + item.Notes)

	fatalities := 0
	fmt.Sscanf(item.Fatalities, "%d", &fatalities)

	if fatalities > 50 {
		return model.SeverityCritical
	}

	if fatalities > 10 || strings.Contains(text, "battle") || strings.Contains(text, "explosion") {
		return model.SeverityHigh
	}

	if fatalities > 0 || strings.Contains(text, "attack") || strings.Contains(text, "violence") {
		return model.SeverityMedium
	}

	return model.SeverityLow
}

// generateMetadata generates event metadata
func (p *LiveUAMapProvider) generateMetadata(item acledItem) map[string]string {
	metadata := map[string]string{
		"data_id":        item.DataID,
		"event_type":     item.EventType,
		"sub_event_type": item.SubEventType,
		"actor1":         item.Actor1,
		"actor2":         item.Actor2,
		"country":        item.Country,
		"region":         item.Region,
		"admin1":         item.Admin1,
		"location":       item.Location,
		"fatalities":     item.Fatalities,
		"source":         "ACLED",
		"data_type":      "conflict_events",
		"update_frequency": "15 minutes",
		"coverage":       "Global conflict zones",
	}

	if item.Source != "" {
		metadata["original_source"] = item.Source
	}

	return metadata
}

// generateBadges generates event badges
func (p *LiveUAMapProvider) generateBadges(item acledItem) []model.Badge {
	badges := []model.Badge{
		{
			Type:      model.BadgeTypeSource,
			Label:     "ACLED",
			Timestamp: time.Now().UTC(),
		},
		{
			Type:      model.BadgeTypePrecision,
			Label:     "Exact",
			Timestamp: time.Now().UTC(),
		},
		{
			Type:      model.BadgeTypeFreshness,
			Label:     "Real-time",
			Timestamp: time.Now().UTC(),
		},
	}

	// Add event type badge
	if item.EventType != "" {
		badges = append(badges, model.Badge{
			Type:      "event_type",
			Label:     item.EventType,
			Timestamp: time.Now().UTC(),
		})
	}

	// Add country badge
	if item.Country != "" {
		badges = append(badges, model.Badge{
			Type:      "country",
			Label:     item.Country,
			Timestamp: time.Now().UTC(),
		})
	}

	return badges
}
