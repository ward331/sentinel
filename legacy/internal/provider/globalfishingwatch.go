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

// GlobalFishingWatchProvider fetches vessel activity data
type GlobalFishingWatchProvider struct {
	name     string
	baseURL  string
	apiKey   string
	interval time.Duration
}

// NewGlobalFishingWatchProvider creates a new GFW provider
func NewGlobalFishingWatchProvider(apiKey string) *GlobalFishingWatchProvider {
	return &GlobalFishingWatchProvider{
		name:     "globalfishingwatch",
		baseURL:  "https://gateway.api.globalfishingwatch.org/v3",
		apiKey:   apiKey,
		interval: 3600 * time.Second, // 1 hour
	}
}


// Fetch retrieves vessel events from Global Fishing Watch

// Name returns the provider identifier
func (p *GlobalFishingWatchProvider) Name() string {
	return "globalfishingwatch"
}


// Enabled returns whether the provider is enabled
func (p *GlobalFishingWatchProvider) Enabled() bool {
	return true
}


// Interval returns the polling interval
func (p *GlobalFishingWatchProvider) Interval() time.Duration {
	return p.interval
}

func (p *GlobalFishingWatchProvider) Fetch(ctx context.Context) ([]*model.Event, error) {
	// Fetch vessel events
	events, err := p.fetchVesselEvents(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch vessel events: %w", err)
	}

	return events, nil
}

// fetchVesselEvents fetches vessel activity events
func (p *GlobalFishingWatchProvider) fetchVesselEvents(ctx context.Context) ([]*model.Event, error) {
	url := fmt.Sprintf("%s/events", p.baseURL)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add API key if available
	if p.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "SENTINEL/1.0")

	// Add query parameters for recent events
	q := req.URL.Query()
	q.Add("limit", "100")
	q.Add("offset", "0")
	q.Add("sort", "-start")
	q.Add("include", "vessel")
	req.URL.RawQuery = q.Encode()

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

	var data GFWEventsResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return p.convertToEvents(data), nil
}

// convertToEvents converts GFW data to SENTINEL events
func (p *GlobalFishingWatchProvider) convertToEvents(data GFWEventsResponse) []*model.Event {
	events := make([]*model.Event, 0, len(data.Events))

	for _, event := range data.Events {
		// Skip events without location
		if event.Position == nil {
			continue
		}

		// Calculate magnitude based on event type and duration
		magnitude := p.calculateMagnitude(event)

		// Determine severity based on event type and vessel info
		severity := p.determineSeverity(event)

		sentinelEvent := &model.Event{
			ID:          fmt.Sprintf("gfw-%s-%d", event.ID, time.Now().Unix()),
			Title:       p.generateTitle(event),
			Description: p.generateDescription(event),
			Source:      p.name,
			SourceID:    event.ID,
			OccurredAt:  time.Now(),
			Location: model.Location{
				Type:        "Point",
				Coordinates: []float64{event.Position.Longitude, event.Position.Latitude},
			},
			Precision: model.PrecisionExact,
			Magnitude: magnitude,
			Category:  "maritime",
			Severity:  severity,
			Metadata:  p.generateMetadata(event),
			Badges: []model.Badge{
				{
					Type:      model.BadgeTypeSource,
					Label:     "Global Fishing Watch",
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

		// Add fishing activity badge
		if strings.Contains(strings.ToLower(event.Type), "fishing") {
			sentinelEvent.Badges = append(sentinelEvent.Badges, model.Badge{
				Type:      "activity",
				Label:     "Fishing",
				Timestamp: time.Now().UTC(),
			})
		}

		// Add dark vessel badge
		if event.Vessel != nil && event.Vessel.Flag == "UNK" {
			sentinelEvent.Badges = append(sentinelEvent.Badges, model.Badge{
				Type:      "risk",
				Label:     "Dark Vessel",
				Timestamp: time.Now().UTC(),
			})
			sentinelEvent.Severity = model.SeverityHigh
		}

		// Add transshipment badge
		if strings.Contains(strings.ToLower(event.Type), "transshipment") {
			sentinelEvent.Badges = append(sentinelEvent.Badges, model.Badge{
				Type:      "activity",
				Label:     "Transshipment",
				Timestamp: time.Now().UTC(),
			})
			sentinelEvent.Severity = model.SeverityHigh
		}

		events = append(events, sentinelEvent)
	}

	return events
}

// calculateMagnitude calculates event magnitude
func (p *GlobalFishingWatchProvider) calculateMagnitude(event GFWEvent) float64 {
	// Base magnitude for maritime events
	magnitude := 2.0

	// Adjust based on event type
	switch strings.ToLower(event.Type) {
	case "fishing":
		magnitude += 0.5
	case "transshipment":
		magnitude += 1.0
	case "loitering":
		magnitude += 0.3
	case "port_visit":
		magnitude += 0.2
	}

	// Adjust based on duration (longer events are more significant)
	if event.DurationHours > 0 {
		magnitude += float64(event.DurationHours) / 24.0
	}

	// Adjust for dark vessels
	if event.Vessel != nil && event.Vessel.Flag == "UNK" {
		magnitude += 1.0
	}

	// Cap magnitude
	if magnitude > 5.0 {
		magnitude = 5.0
	}

	return magnitude
}

// determineSeverity determines event severity
func (p *GlobalFishingWatchProvider) determineSeverity(event GFWEvent) model.Severity {
	// High severity for suspicious activities
	if event.Vessel != nil && event.Vessel.Flag == "UNK" {
		return model.SeverityHigh
	}

	if strings.Contains(strings.ToLower(event.Type), "transshipment") {
		return model.SeverityHigh
	}

	// Medium severity for fishing in protected areas
	if strings.Contains(strings.ToLower(event.Type), "fishing") {
		return model.SeverityMedium
	}

	// Low severity for normal maritime activities
	return model.SeverityLow
}

// generateTitle generates event title
func (p *GlobalFishingWatchProvider) generateTitle(event GFWEvent) string {
	title := fmt.Sprintf("Vessel Activity: %s", strings.Title(event.Type))

	if event.Vessel != nil && event.Vessel.Name != "" {
		title = fmt.Sprintf("%s - %s", event.Vessel.Name, title)
	}

	return title
}

// generateDescription generates event description
func (p *GlobalFishingWatchProvider) generateDescription(event GFWEvent) string {
	var desc strings.Builder
	
	desc.WriteString("Maritime Vessel Activity\n")
	desc.WriteString("=======================\n\n")
	
	desc.WriteString(fmt.Sprintf("Activity Type: %s\n", strings.Title(event.Type)))
	
	if event.Vessel != nil {
		desc.WriteString(fmt.Sprintf("Vessel: %s\n", event.Vessel.Name))
		desc.WriteString(fmt.Sprintf("MMSI: %s\n", event.Vessel.MMSI))
		desc.WriteString(fmt.Sprintf("Flag: %s\n", event.Vessel.Flag))
		desc.WriteString(fmt.Sprintf("Type: %s\n", event.Vessel.Type))
	}
	
	if event.Position != nil {
		desc.WriteString(fmt.Sprintf("Position: %.4f, %.4f\n", event.Position.Latitude, event.Position.Longitude))
	}
	
	desc.WriteString(fmt.Sprintf("Start: %s\n", event.Start))
	desc.WriteString(fmt.Sprintf("End: %s\n", event.End))
	desc.WriteString(fmt.Sprintf("Duration: %.1f hours\n", event.DurationHours))
	
	desc.WriteString("\nData Source: Global Fishing Watch\n")
	desc.WriteString("Update: Near real-time AIS tracking\n")
	
	// Add risk assessment
	if event.Vessel != nil && event.Vessel.Flag == "UNK" {
		desc.WriteString("\n⚠️  RISK ASSESSMENT: DARK VESSEL DETECTED\n")
		desc.WriteString("Vessel operating without proper identification.\n")
		desc.WriteString("Possible illegal, unreported, or unregulated (IUU) activity.\n")
	}
	
	if strings.Contains(strings.ToLower(event.Type), "transshipment") {
		desc.WriteString("\n⚠️  RISK ASSESSMENT: TRANSSHIPMENT DETECTED\n")
		desc.WriteString("At-sea transfer of catch between vessels.\n")
		desc.WriteString("Associated with IUU fishing and labor abuses.\n")
	}
	
	return desc.String()
}

// generateMetadata generates event metadata
func (p *GlobalFishingWatchProvider) generateMetadata(event GFWEvent) map[string]string {
	metadata := map[string]string{
		"event_id":        event.ID,
		"event_type":      event.Type,
		"start_time":      event.Start,
		"end_time":        event.End,
		"duration_hours":  fmt.Sprintf("%.1f", event.DurationHours),
		"data_source":     "Global Fishing Watch",
		"update_frequency": "1 hour",
	}

	if event.Position != nil {
		metadata["latitude"] = fmt.Sprintf("%.4f", event.Position.Latitude)
		metadata["longitude"] = fmt.Sprintf("%.4f", event.Position.Longitude)
	}

	if event.Vessel != nil {
		metadata["vessel_name"] = event.Vessel.Name
		metadata["vessel_mmsi"] = event.Vessel.MMSI
		metadata["vessel_flag"] = event.Vessel.Flag
		metadata["vessel_type"] = event.Vessel.Type
		metadata["vessel_length_m"] = fmt.Sprintf("%.1f", event.Vessel.Length)
		metadata["vessel_tonnage_gt"] = fmt.Sprintf("%.1f", event.Vessel.Tonnage)
	}

	return metadata
}

// GFWEventsResponse represents the Global Fishing Watch API response
type GFWEventsResponse struct {
	Events []GFWEvent `json:"events"`
	Total  int        `json:"total"`
}

// GFWEvent represents a vessel event
type GFWEvent struct {
	ID            string       `json:"id"`
	Type          string       `json:"type"`
	Start         string       `json:"start"`
	End           string       `json:"end"`
	DurationHours float64      `json:"duration_hours"`
	Position      *GFWPosition `json:"position"`
	Vessel        *GFWVessel   `json:"vessel"`
}

// GFWPosition represents a geographic position
type GFWPosition struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// GFWVessel represents a vessel
type GFWVessel struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	MMSI    string  `json:"mmsi"`
	Flag    string  `json:"flag"`
	Type    string  `json:"type"`
	Length  float64 `json:"length_m"`
	Tonnage float64 `json:"tonnage_gt"`
}