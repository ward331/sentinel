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

// GlobalForestWatchProvider fetches wildfire and deforestation alerts
type GlobalForestWatchProvider struct {
	name     string
	baseURL  string
	interval time.Duration
}

// NewGlobalForestWatchProvider creates a new GFW provider
func NewGlobalForestWatchProvider() *GlobalForestWatchProvider {
	return &GlobalForestWatchProvider{
		name:     "globalforestwatch",
		baseURL:  "https://data-api.globalforestwatch.org",
		interval: 1800 * time.Second, // 30 minutes
	}
}


// Fetch retrieves fire alerts from Global Forest Watch
func (p *GlobalForestWatchProvider) Fetch(ctx context.Context) ([]*model.Event, error) {
	// Fetch VIIRS fire alerts (NASA satellite data)
	events, err := p.fetchVIIRSFireAlerts(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch VIIRS fire alerts: %w", err)
	}

	return events, nil
}

// fetchVIIRSFireAlerts fetches NASA VIIRS fire alerts
func (p *GlobalForestWatchProvider) fetchVIIRSFireAlerts(ctx context.Context) ([]*model.Event, error) {
	url := fmt.Sprintf("%s/dataset/nasa_viirs_fire_alerts/latest/query", p.baseURL)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "SENTINEL/1.0")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var data GFWResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return p.convertToEvents(data), nil
}

// convertToEvents converts GFW data to SENTINEL events
func (p *GlobalForestWatchProvider) convertToEvents(data GFWResponse) []*model.Event {
	events := make([]*model.Event, 0, len(data.Data))

	for _, alert := range data.Data {
		// Calculate magnitude based on brightness and confidence
		magnitude := p.calculateMagnitude(alert)

		// Determine severity based on brightness and area
		severity := p.determineSeverity(alert)

		event := &model.Event{
			ID:          fmt.Sprintf("gfw-%s-%d", alert.ID, time.Now().Unix()),
			Title:       fmt.Sprintf("Wildfire Alert: %.4f, %.4f", alert.Latitude, alert.Longitude),
			Description: p.generateDescription(alert),
			Source:      p.name,
			SourceID:    alert.ID,
			OccurredAt:  time.Now(),
			Location: model.Location{
				Type:        "Point",
				Coordinates: []float64{alert.Longitude, alert.Latitude},
			},
			Precision: model.PrecisionExact,
			Magnitude: magnitude,
			Category:  "wildfire",
			Severity:  severity,
			Metadata: map[string]string{
				"alert_id":       alert.ID,
				"brightness":     fmt.Sprintf("%.1f", alert.Brightness),
				"confidence":     alert.Confidence,
				"acq_date":       alert.AcquisitionDate,
				"acq_time":       alert.AcquisitionTime,
				"satellite":      alert.Satellite,
				"instrument":     alert.Instrument,
				"version":        alert.Version,
				"bright_t31":     fmt.Sprintf("%.1f", alert.BrightT31),
				"frp":            fmt.Sprintf("%.1f", alert.FRP),
				"daynight":       alert.DayNight,
				"data_source":    "Global Forest Watch / NASA VIIRS",
				"update_frequency": "30 minutes",
			},
			Badges: []model.Badge{
				{
					Type:      model.BadgeTypeSource,
					Label:     "NASA VIIRS",
					Timestamp: time.Now().UTC(),
				},
				{
					Type:      model.BadgeTypePrecision,
					Label:     "Exact",
					Timestamp: time.Now().UTC(),
				},
				{
					Type:      model.BadgeTypeFreshness,
					Label:     "Near Real-time",
					Timestamp: time.Now().UTC(),
				},
			},
		}

		// Add high confidence badge
		if alert.Confidence == "high" {
			event.Badges = append(event.Badges, model.Badge{
				Type:      "confidence",
				Label:     "High Confidence",
				Timestamp: time.Now().UTC(),
			})
		}

		// Add large fire badge for high FRP
		if alert.FRP > 100 {
			event.Badges = append(event.Badges, model.Badge{
				Type:      "intensity",
				Label:     "Large Fire",
				Timestamp: time.Now().UTC(),
			})
		}

		events = append(events, event)
	}

	return events
}

// calculateMagnitude calculates event magnitude
func (p *GlobalForestWatchProvider) calculateMagnitude(alert GFWAlert) float64 {
	// Base magnitude for fire alerts
	magnitude := 2.5

	// Adjust based on brightness
	if alert.Brightness > 0 {
		magnitude += float64(alert.Brightness) / 100.0
	}

	// Adjust based on FRP (Fire Radiative Power)
	if alert.FRP > 0 {
		magnitude += float64(alert.FRP) / 50.0
	}

	// Adjust based on confidence
	switch alert.Confidence {
	case "high":
		magnitude += 0.5
	case "nominal":
		magnitude += 0.2
	}

	// Cap magnitude
	if magnitude > 5.0 {
		magnitude = 5.0
	}

	return magnitude
}

// determineSeverity determines event severity
func (p *GlobalForestWatchProvider) determineSeverity(alert GFWAlert) model.Severity {
	// High severity for large, bright fires
	if alert.Brightness > 350 && alert.FRP > 100 {
		return model.SeverityHigh
	}

	// Medium severity for moderate fires
	if alert.Brightness > 200 || alert.FRP > 50 {
		return model.SeverityMedium
	}

	// Low severity for small fires
	return model.SeverityLow
}

// generateDescription generates event description
func (p *GlobalForestWatchProvider) generateDescription(alert GFWAlert) string {
	return fmt.Sprintf(`Wildfire Detection Alert
=======================

Location: %.4f, %.4f
Date: %s
Time: %s

Satellite: %s (%s)
Instrument: %s

Detection Metrics:
- Brightness: %.1f K
- Fire Radiative Power: %.1f MW
- Confidence: %s
- Brightness T31: %.1f K
- Day/Night: %s

Data Source: NASA VIIRS via Global Forest Watch
Update: Near real-time satellite detection

Note: VIIRS (Visible Infrared Imaging Radiometer Suite) provides
higher resolution fire detection than MODIS, with better detection
of smaller fires and improved nighttime detection.`,
		alert.Latitude,
		alert.Longitude,
		alert.AcquisitionDate,
		alert.AcquisitionTime,
		alert.Satellite,
		alert.Version,
		alert.Instrument,
		alert.Brightness,
		alert.FRP,
		alert.Confidence,
		alert.BrightT31,
		alert.DayNight,
	)
}

// GFWResponse represents the Global Forest Watch API response
type GFWResponse struct {
	Data []GFWAlert `json:"data"`
}

// GFWAlert represents a single fire alert
type GFWAlert struct {
	ID              string  `json:"id"`
	Latitude        float64 `json:"latitude"`
	Longitude       float64 `json:"longitude"`
	Brightness      float64 `json:"brightness"`
	Confidence      string  `json:"confidence"`
	AcquisitionDate string  `json:"acq_date"`
	AcquisitionTime string  `json:"acq_time"`
	Satellite       string  `json:"satellite"`
	Instrument      string  `json:"instrument"`
	Version         string  `json:"version"`
	BrightT31       float64 `json:"bright_t31"`
	FRP             float64 `json:"frp"`
	DayNight        string  `json:"daynight"`
}