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

// OpenWeatherMapProvider fetches severe weather alerts from OpenWeatherMap
// Tier 1: Free with API key (1000 calls/day)
// Category: weather
// Signup: https://openweathermap.org/api
type OpenWeatherMapProvider struct {
	client *http.Client
	config *Config
}

// NewOpenWeatherMapProvider creates a new OpenWeatherMapProvider
func NewOpenWeatherMapProvider(config *Config) *OpenWeatherMapProvider {
	return &OpenWeatherMapProvider{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		config: config,
	}
}

// Name returns the provider identifier
func (p *OpenWeatherMapProvider) Name() string {
	return "openweathermap"
}

// Enabled returns whether the provider is enabled (requires API key)
func (p *OpenWeatherMapProvider) Enabled() bool {
	if p.config == nil || p.config.APIKey == "" {
		return false
	}
	return p.config.Enabled
}

// Interval returns the polling interval
func (p *OpenWeatherMapProvider) Interval() time.Duration {
	if p.config != nil && p.config.PollInterval > 0 {
		return p.config.PollInterval
	}
	return 300 * time.Second
}

// owmOneCallResponse represents the OpenWeatherMap One Call API response
type owmOneCallResponse struct {
	Lat      float64     `json:"lat"`
	Lon      float64     `json:"lon"`
	Timezone string      `json:"timezone"`
	Alerts   []owmAlert  `json:"alerts"`
}

type owmAlert struct {
	SenderName  string   `json:"sender_name"`
	Event       string   `json:"event"`
	Start       int64    `json:"start"`
	End         int64    `json:"end"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

// Fetch retrieves severe weather alerts from OpenWeatherMap
func (p *OpenWeatherMapProvider) Fetch(ctx context.Context) ([]*model.Event, error) {
	// Query multiple locations for global weather alerts
	locations := []struct {
		name string
		lat  float64
		lon  float64
	}{
		{"New York", 40.7128, -74.0060},
		{"London", 51.5074, -0.1278},
		{"Tokyo", 35.6762, 139.6503},
		{"Sydney", -33.8688, 151.2093},
		{"Sao Paulo", -23.5505, -46.6333},
		{"Mumbai", 19.0760, 72.8777},
		{"Lagos", 6.5244, 3.3792},
		{"Moscow", 55.7558, 37.6173},
	}

	// If bounding box is set, use center point instead
	if len(p.config.BoundingBox) >= 4 {
		centerLat := (p.config.BoundingBox[1] + p.config.BoundingBox[3]) / 2
		centerLon := (p.config.BoundingBox[0] + p.config.BoundingBox[2]) / 2
		locations = []struct {
			name string
			lat  float64
			lon  float64
		}{
			{"Custom", centerLat, centerLon},
		}
	}

	var allEvents []*model.Event

	for _, loc := range locations {
		events, err := p.fetchAlerts(ctx, loc.name, loc.lat, loc.lon)
		if err != nil {
			continue // Skip failed locations
		}
		allEvents = append(allEvents, events...)
	}

	return allEvents, nil
}

func (p *OpenWeatherMapProvider) fetchAlerts(ctx context.Context, locName string, lat, lon float64) ([]*model.Event, error) {
	url := fmt.Sprintf("https://api.openweathermap.org/data/3.0/onecall?lat=%f&lon=%f&exclude=minutely,hourly,daily,current&appid=%s",
		lat, lon, p.config.APIKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create OWM request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch OWM data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OWM returned status %d: %s", resp.StatusCode, string(body))
	}

	var data owmOneCallResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode OWM response: %w", err)
	}

	var events []*model.Event

	for _, alert := range data.Alerts {
		start := time.Unix(alert.Start, 0).UTC()
		end := time.Unix(alert.End, 0).UTC()

		severity := p.determineSeverity(alert.Event, alert.Tags)

		event := &model.Event{
			Title:       fmt.Sprintf("Weather Alert: %s near %s", alert.Event, locName),
			Description: fmt.Sprintf("Weather Alert from %s\n\nEvent: %s\nStart: %s\nEnd: %s\nLocation: %s\n\n%s", alert.SenderName, alert.Event, start.Format(time.RFC3339), end.Format(time.RFC3339), locName, alert.Description),
			Source:      "openweathermap",
			SourceID:    fmt.Sprintf("owm_%s_%d", alert.Event, alert.Start),
			OccurredAt:  start,
			Location:    model.Point(lon, lat),
			Precision:   model.PrecisionApproximate,
			Category:    "weather",
			Severity:    severity,
			Metadata: map[string]string{
				"sender":    alert.SenderName,
				"event":     alert.Event,
				"start":     start.Format(time.RFC3339),
				"end":       end.Format(time.RFC3339),
				"location":  locName,
				"tier":      "1",
				"signup_url": "https://openweathermap.org/api",
			},
			Badges: []model.Badge{
				{Label: "OpenWeatherMap", Type: "source", Timestamp: start},
				{Label: "weather", Type: "category", Timestamp: start},
				{Label: alert.Event, Type: "alert_type", Timestamp: start},
			},
		}

		events = append(events, event)
	}

	return events, nil
}

func (p *OpenWeatherMapProvider) determineSeverity(event string, tags []string) model.Severity {
	for _, tag := range tags {
		switch tag {
		case "Extreme":
			return model.SeverityCritical
		case "Severe":
			return model.SeverityHigh
		case "Moderate":
			return model.SeverityMedium
		}
	}
	return model.SeverityMedium
}
