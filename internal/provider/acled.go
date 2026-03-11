package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/openclaw/sentinel-backend/internal/model"
)

// ACLEDProvider fetches conflict event data from ACLED (Armed Conflict Location & Event Data)
// Tier 1: Free with API key + email registration
// Category: conflict
// Signup: https://acleddata.com/register/
type ACLEDProvider struct {
	client *http.Client
	config *Config
}

// NewACLEDProvider creates a new ACLEDProvider
func NewACLEDProvider(config *Config) *ACLEDProvider {
	return &ACLEDProvider{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		config: config,
	}
}

// Name returns the provider identifier
func (p *ACLEDProvider) Name() string {
	return "acled"
}

// Enabled returns whether the provider is enabled (requires API key + email in Options)
func (p *ACLEDProvider) Enabled() bool {
	if p.config == nil || p.config.APIKey == "" {
		return false
	}
	if p.config.Options == nil || p.config.Options["email"] == "" {
		return false
	}
	return p.config.Enabled
}

// Interval returns the polling interval
func (p *ACLEDProvider) Interval() time.Duration {
	if p.config != nil && p.config.PollInterval > 0 {
		return p.config.PollInterval
	}
	return 3600 * time.Second
}

// acledResponse represents the ACLED API response
type acledResponse struct {
	Status int         `json:"status"`
	Data   []acledEvent `json:"data"`
}

type acledEvent struct {
	DataID        int    `json:"data_id"`
	EventDate     string `json:"event_date"`
	Year          int    `json:"year"`
	EventType     string `json:"event_type"`
	SubEventType  string `json:"sub_event_type"`
	Actor1        string `json:"actor1"`
	Actor2        string `json:"actor2"`
	Country       string `json:"country"`
	Region        string `json:"region"`
	Admin1        string `json:"admin1"`
	Admin2        string `json:"admin2"`
	Location      string `json:"location"`
	Latitude      string `json:"latitude"`
	Longitude     string `json:"longitude"`
	Fatalities    int    `json:"fatalities"`
	Notes         string `json:"notes"`
	Source        string `json:"source"`
	SourceScale   string `json:"source_scale"`
	InteractionID int    `json:"interaction"`
}

// Fetch retrieves conflict events from ACLED
func (p *ACLEDProvider) Fetch(ctx context.Context) ([]*model.Event, error) {
	email := p.config.Options["email"]

	// Fetch recent events (last 7 days)
	sevenDaysAgo := time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	url := fmt.Sprintf("https://api.acleddata.com/acled/read?key=%s&email=%s&event_date=%s|&event_date_where=>=&limit=200",
		p.config.APIKey, email, sevenDaysAgo)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create ACLED request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ACLED data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ACLED returned status %d: %s", resp.StatusCode, string(body))
	}

	var data acledResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode ACLED response: %w", err)
	}

	var events []*model.Event

	for _, ev := range data.Data {
		lat, err := strconv.ParseFloat(ev.Latitude, 64)
		if err != nil {
			continue
		}
		lon, err := strconv.ParseFloat(ev.Longitude, 64)
		if err != nil {
			continue
		}

		occurredAt, err := time.Parse("2006-01-02", ev.EventDate)
		if err != nil {
			occurredAt = time.Now().UTC()
		}

		severity := p.determineSeverity(ev.Fatalities, ev.EventType)

		title := fmt.Sprintf("%s in %s, %s", ev.EventType, ev.Location, ev.Country)
		if ev.Fatalities > 0 {
			title = fmt.Sprintf("%s (%d fatalities)", title, ev.Fatalities)
		}

		description := fmt.Sprintf("ACLED Conflict Event\n\nType: %s / %s\nActors: %s vs %s\nLocation: %s, %s, %s\nDate: %s\nFatalities: %d\n\nNotes: %s\n\nSource: %s",
			ev.EventType, ev.SubEventType, ev.Actor1, ev.Actor2,
			ev.Location, ev.Admin1, ev.Country, ev.EventDate, ev.Fatalities,
			ev.Notes, ev.Source)

		event := &model.Event{
			Title:       title,
			Description: description,
			Source:      "acled",
			SourceID:    fmt.Sprintf("acled_%d", ev.DataID),
			OccurredAt:  occurredAt,
			Location:    model.Point(lon, lat),
			Precision:   model.PrecisionApproximate,
			Category:    "conflict",
			Severity:    severity,
			Metadata: map[string]string{
				"data_id":        fmt.Sprintf("%d", ev.DataID),
				"event_type":     ev.EventType,
				"sub_event_type": ev.SubEventType,
				"actor1":         ev.Actor1,
				"actor2":         ev.Actor2,
				"country":        ev.Country,
				"region":         ev.Region,
				"fatalities":     fmt.Sprintf("%d", ev.Fatalities),
				"source":         ev.Source,
				"tier":           "1",
			},
			Badges: []model.Badge{
				{Label: "ACLED", Type: "source", Timestamp: occurredAt},
				{Label: "conflict", Type: "category", Timestamp: occurredAt},
			},
		}

		events = append(events, event)
	}

	return events, nil
}

// determineSeverity returns severity based on fatalities and event type
func (p *ACLEDProvider) determineSeverity(fatalities int, eventType string) model.Severity {
	if fatalities >= 50 {
		return model.SeverityCritical
	}
	if fatalities >= 10 {
		return model.SeverityHigh
	}
	if fatalities > 0 || eventType == "Battles" || eventType == "Explosions/Remote violence" {
		return model.SeverityMedium
	}
	return model.SeverityLow
}
