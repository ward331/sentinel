package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/openclaw/sentinel-backend/internal/model"
)

// OpenMeteoProvider fetches weather alerts from Open-Meteo
type OpenMeteoProvider struct {
	name     string
	baseURL  string
	interval time.Duration
}

// Name returns the provider name
func (p *OpenMeteoProvider) Name() string {
    return "openmeteo"
}

// Interval returns the polling interval
func (p *OpenMeteoProvider) Interval() time.Duration {
    interval, _ := time.ParseDuration("5m")
    return interval
}

// Enabled returns whether the provider is enabled
func (p *OpenMeteoProvider) Enabled() bool {
    return p.config != nil && p.config.Enabled
}

// NewOpenMeteoProvider creates a new Open-Meteo provider
func NewOpenMeteoProvider() *OpenMeteoProvider {
	return &OpenMeteoProvider{
		name:     "openmeteo",
		baseURL:  "https://api.open-meteo.com/v1",
		interval: 15 * time.Minute,
	}
}

// Name returns the provider name
func (p *OpenMeteoProvider) Name() string {
	return p.name
}

// Interval returns the polling interval
func (p *OpenMeteoProvider) Interval() time.Duration {
	return p.interval
}

// Fetch retrieves weather alerts
func (p *OpenMeteoProvider) Fetch(ctx context.Context) ([]*model.Event, error) {
	// Open-Meteo doesn't have a dedicated alerts API, but we can use severe weather endpoints
	// For now, we'll fetch severe weather warnings for major regions
	regions := []struct {
		name string
		lat  float64
		lon  float64
	}{
		{"North America", 40.0, -100.0},
		{"Europe", 50.0, 10.0},
		{"Asia", 35.0, 105.0},
		{"South America", -15.0, -60.0},
		{"Africa", 0.0, 20.0},
		{"Australia", -25.0, 135.0},
	}

	var allEvents []*model.Event
	for _, region := range regions {
		events, err := p.fetchRegionAlerts(ctx, region.name, region.lat, region.lon)
		if err != nil {
			// Log error but continue with other regions
			continue
		}
		allEvents = append(allEvents, events...)
	}

	return allEvents, nil
}

// fetchRegionAlerts fetches alerts for a specific region
func (p *OpenMeteoProvider) fetchRegionAlerts(ctx context.Context, regionName string, lat, lon float64) ([]*model.Event, error) {
	// Build URL for severe weather data
	u, err := url.Parse(fmt.Sprintf("%s/severe-weather", p.baseURL))
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	q := u.Query()
	q.Set("latitude", fmt.Sprintf("%.4f", lat))
	q.Set("longitude", fmt.Sprintf("%.4f", lon))
	q.Set("timezone", "UTC")
	q.Set("past_days", "1")
	q.Set("forecast_days", "2")
	q.Set("models", "icon_seamless")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "SENTINEL/2.0 (https://github.com/openclaw/sentinel-backend)")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Open-Meteo data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Open-Meteo API returned status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return p.parseWeatherData(data, regionName, lat, lon)
}

// OpenMeteoResponse represents the Open-Meteo API response
type OpenMeteoResponse struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Timezone  string  `json:"timezone"`
	Hourly    struct {
		Time          []string  `json:"time"`
		Temperature2m []float64 `json:"temperature_2m"`
		Precipitation []float64 `json:"precipitation"`
		WeatherCode   []int     `json:"weather_code"`
		WindSpeed10m  []float64 `json:"wind_speed_10m"`
		WindGusts10m  []float64 `json:"wind_gusts_10m"`
	} `json:"hourly"`
	SevereWeather struct {
		Time        []string `json:"time"`
		WarningType []string `json:"warning_type"`
		Severity    []string `json:"severity"`
		Description []string `json:"description"`
	} `json:"severe_weather,omitempty"`
}

// parseWeatherData parses Open-Meteo weather data
func (p *OpenMeteoProvider) parseWeatherData(data []byte, regionName string, lat, lon float64) ([]*model.Event, error) {
	var response OpenMeteoResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse Open-Meteo JSON: %w", err)
	}

	var events []*model.Event
	
	// Check for severe weather warnings
	if len(response.SevereWeather.Time) > 0 {
		for i, warningTime := range response.SevereWeather.Time {
			if i >= len(response.SevereWeather.WarningType) || 
			   i >= len(response.SevereWeather.Severity) ||
			   i >= len(response.SevereWeather.Description) {
				continue
			}

			warningType := response.SevereWeather.WarningType[i]
			severity := response.SevereWeather.Severity[i]
			description := response.SevereWeather.Description[i]

			event := p.createWeatherEvent(warningTime, warningType, severity, description, regionName, lat, lon)
			if event != nil {
				events = append(events, event)
			}
		}
	}

	// Check for extreme weather conditions in hourly data
	if len(response.Hourly.Time) > 0 {
		for i, hourTime := range response.Hourly.Time {
			// Check for extreme temperatures
			if i < len(response.Hourly.Temperature2m) {
				temp := response.Hourly.Temperature2m[i]
				if temp > 40.0 || temp < -20.0 { // Extreme heat or cold
					event := p.createExtremeTempEvent(hourTime, temp, regionName, lat, lon)
					if event != nil {
						events = append(events, event)
					}
				}
			}

			// Check for heavy precipitation
			if i < len(response.Hourly.Precipitation) {
				precip := response.Hourly.Precipitation[i]
				if precip > 20.0 { // Heavy rain (>20mm/hour)
					event := p.createHeavyRainEvent(hourTime, precip, regionName, lat, lon)
					if event != nil {
						events = append(events, event)
					}
				}
			}

			// Check for strong winds
			if i < len(response.Hourly.WindGusts10m) {
				windGust := response.Hourly.WindGusts10m[i]
				if windGust > 25.0 { // Strong wind gusts (>25 m/s)
					event := p.createStrongWindEvent(hourTime, windGust, regionName, lat, lon)
					if event != nil {
						events = append(events, event)
					}
				}
			}
		}
	}

	return events, nil
}

// createWeatherEvent creates a severe weather warning event
func (p *OpenMeteoProvider) createWeatherEvent(warningTime, warningType, severity, description, regionName string, lat, lon float64) *model.Event {
	// Parse timestamp
	eventTime, err := time.Parse(time.RFC3339, warningTime)
	if err != nil {
		eventTime = time.Now()
	}

	// Determine SENTINEL severity
	sentinelSeverity := p.determineSeverity(severity)
	magnitude := p.calculateMagnitude(warningType, severity)

	// Clean description
	cleanDesc := strings.TrimSpace(description)
	if cleanDesc == "" {
		cleanDesc = fmt.Sprintf("%s warning for %s", warningType, regionName)
	}

	// Generate title
	title := fmt.Sprintf("%s: %s", strings.Title(warningType), regionName)

	// Generate metadata
	metadata := p.generateWeatherMetadata(warningType, severity, regionName)

	// Generate badges
	badges := p.generateWeatherBadges(warningType, severity)

	return &model.Event{
		ID:          fmt.Sprintf("openmeteo-%s-%s-%d", warningType, regionName, eventTime.Unix()),
		Title:       title,
		Description: cleanDesc,
		Source:      "openmeteo",
		SourceID:    fmt.Sprintf("%s-%s", warningType, warningTime),
		OccurredAt:  eventTime,
		IngestedAt:  time.Now(),
		Location:    model.Point(lon, lat),
		Precision:   model.PrecisionApproximate,
		Magnitude:   magnitude,
		Category:    "weather",
		Severity:    sentinelSeverity,
		Metadata:    metadata,
		Badges:      badges,
	}
}

// createExtremeTempEvent creates extreme temperature event
func (p *OpenMeteoProvider) createExtremeTempEvent(hourTime string, temp float64, regionName string, lat, lon float64) *model.Event {
	eventTime, err := time.Parse(time.RFC3339, hourTime)
	if err != nil {
		eventTime = time.Now()
	}

	condition := "extreme heat"
	severity := model.SeverityMedium
	if temp < -20.0 {
		condition = "extreme cold"
		severity = model.SeverityHigh
	}

	title := fmt.Sprintf("%s in %s", strings.Title(condition), regionName)
	description := fmt.Sprintf("Temperature: %.1f°C", temp)

	return &model.Event{
		ID:          fmt.Sprintf("openmeteo-temp-%s-%d", regionName, eventTime.Unix()),
		Title:       title,
		Description: description,
		Source:      "openmeteo",
		SourceID:    fmt.Sprintf("temp-%s", hourTime),
		OccurredAt:  eventTime,
		IngestedAt:  time.Now(),
		Location:    model.Point(lon, lat),
		Precision:   model.PrecisionApproximate,
		Magnitude:   4.0,
		Category:    "weather",
		Severity:    severity,
		Metadata: map[string]string{
			"temperature_c": fmt.Sprintf("%.1f", temp),
			"condition":     condition,
			"region":        regionName,
			"source":        "Open-Meteo",
			"data_type":     "extreme_temperature",
		},
		Badges: []model.Badge{
			{Type: model.BadgeTypeSource, Label: "Open-Meteo", Timestamp: time.Now().UTC()},
			{Type: model.BadgeTypePrecision, Label: "Approximate", Timestamp: time.Now().UTC()},
			{Type: "condition", Label: condition, Timestamp: time.Now().UTC()},
		},
	}
}

// createHeavyRainEvent creates heavy rain event
func (p *OpenMeteoProvider) createHeavyRainEvent(hourTime string, precip float64, regionName string, lat, lon float64) *model.Event {
	eventTime, err := time.Parse(time.RFC3339, hourTime)
	if err != nil {
		eventTime = time.Now()
	}

	title := fmt.Sprintf("Heavy Rain in %s", regionName)
	description := fmt.Sprintf("Precipitation: %.1f mm/hour", precip)

	return &model.Event{
		ID:          fmt.Sprintf("openmeteo-rain-%s-%d", regionName, eventTime.Unix()),
		Title:       title,
		Description: description,
		Source:      "openmeteo",
		SourceID:    fmt.Sprintf("rain-%s", hourTime),
		OccurredAt:  eventTime,
		IngestedAt:  time.Now(),
		Location:    model.Point(lon, lat),
		Precision:   model.PrecisionApproximate,
		Magnitude:   3.5,
		Category:    "weather",
		Severity:    model.SeverityMedium,
		Metadata: map[string]string{
			"precipitation_mm_h": fmt.Sprintf("%.1f", precip),
			"condition":         "heavy_rain",
			"region":            regionName,
			"source":            "Open-Meteo",
			"data_type":         "heavy_precipitation",
		},
		Badges: []model.Badge{
			{Type: model.BadgeTypeSource, Label: "Open-Meteo", Timestamp: time.Now().UTC()},
			{Type: model.BadgeTypePrecision, Label: "Approximate", Timestamp: time.Now().UTC()},
			{Type: "condition", Label: "Heavy Rain", Timestamp: time.Now().UTC()},
		},
	}
}

// createStrongWindEvent creates strong wind event
func (p *OpenMeteoProvider) createStrongWindEvent(hourTime string, windGust float64, regionName string, lat, lon float64) *model.Event {
	eventTime, err := time.Parse(time.RFC3339, hourTime)
	if err != nil {
		eventTime = time.Now()
	}

	title := fmt.Sprintf("Strong Winds in %s", regionName)
	description := fmt.Sprintf("Wind gusts: %.1f m/s", windGust)

	return &model.Event{
		ID:          fmt.Sprintf("openmeteo-wind-%s-%d", regionName, eventTime.Unix()),
		Title:       title,
		Description: description,
		Source:      "openmeteo",
		SourceID:    fmt.Sprintf("wind-%s", hourTime),
		OccurredAt:  eventTime,
		IngestedAt:  time.Now(),
		Location:    model.Point(lon, lat),
		Precision:   model.PrecisionApproximate,
		Magnitude:   3.8,
		Category:    "weather",
		Severity:    model.SeverityMedium,
		Metadata: map[string]string{
			"wind_gust_m_s": fmt.Sprintf("%.1f", windGust),
			"condition":     "strong_winds",
			"region":        regionName,
			"source":        "Open-Meteo",
			"data_type":     "strong_winds",
		},
		Badges: []model.Badge{
			{Type: model.BadgeTypeSource, Label: "Open-Meteo", Timestamp: time.Now().UTC()},
			{Type: model.BadgeTypePrecision, Label: "Approximate", Timestamp: time.Now().UTC()},
			{Type: "condition", Label: "Strong Winds", Timestamp: time.Now().UTC()},
		},
	}
}

// determineSeverity converts Open-Meteo severity to SENTINEL severity
func (p *OpenMeteoProvider) determineSeverity(severity string) model.Severity {
	severityMap := map[string]model.Severity{
		"extreme":  model.SeverityCritical,
		"severe":   model.SeverityHigh,
		"moderate": model.SeverityMedium,
		"minor":    model.SeverityLow,
		"unknown":  model.SeverityLow,
	}

	if sev, ok := severityMap[strings.ToLower(severity)]; ok {
		return sev
	}
	return model.SeverityMedium
}

// calculateMagnitude calculates event magnitude
func (p *OpenMeteoProvider) calculateMagnitude(warningType, severity string) float64 {
	magnitude := 3.0

	// Add warning type factor
	switch strings.ToLower(warningType) {
	case "tornado", "hurricane", "typhoon":
		magnitude += 2.0
	case "thunderstorm", "blizzard", "flood":
		magnitude += 1.5
	case "heat", "cold", "wind", "rain":
		magnitude += 1.0
	}

	// Add severity factor
	switch strings.ToLower(severity) {
	case "extreme":
		magnitude += 2.0
	case "severe":
		magnitude += 1.5
	case "moderate":
		magnitude += 1.0
	case "minor":
		magnitude += 0.5
	}

	return magnitude
}

// generateWeatherMetadata generates weather event metadata
func (p *OpenMeteoProvider) generateWeatherMetadata(warningType, severity, regionName string) map[string]string {
	return map[string]string{
		"warning_type":   warningType,
		"severity":       severity,
		"region":         regionName,
		"source":         "Open-Meteo",
		"data_type":      "weather_warning",
		"update_frequency": "15 minutes",
		"coverage":       "Global",
		"api_provider":   "Open-Meteo",
	}
}

// generateWeatherBadges generates weather event badges
func (p *OpenMeteoProvider) generateWeatherBadges(warningType, severity string) []model.Badge {
	badges := []model.Badge{
		{
			Type:      model.BadgeTypeSource,
			Label:     "Open-Meteo",
			Timestamp: time.Now().UTC(),
		},
		{
			Type:      model.BadgeTypePrecision,
			Label:     "Approximate",
			Timestamp: time.Now().UTC(),
		},
		{
			Type:      model.BadgeTypeFreshness,
			Label:     "15-minute updates",
			Timestamp: time.Now().UTC(),
		},
	}

	// Add warning type badge
	if warningType != "" {
		badges = append(badges, model.Badge{
			Type:      "warning_type",
			Label:     strings.Title(warningType),
			Timestamp: time.Now().UTC(),
		})
	}

	// Add severity badge
	if severity != "" {
		badges = append(badges, model.Badge{
			Type:      "severity",
			Label:     strings.Title(severity),
			Timestamp: time.Now().UTC(),
		})
	}

	return badges
}
