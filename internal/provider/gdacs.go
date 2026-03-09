package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/openclaw/sentinel-backend/internal/model"
)

// GDACSProvider implements the Provider interface for GDACS disaster alerts
type GDACSProvider struct {
	feedURL string
	client  *http.Client
}

// NewGDACSProvider creates a new GDACS provider
func NewGDACSProvider() *GDACSProvider {
	return &GDACSProvider{
		feedURL: "https://www.gdacs.org/gdacsapi/api/events/geteventlist/SEARCH",
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name returns the provider identifier
func (p *GDACSProvider) Name() string {
	return "gdacs"
}

// Fetch retrieves events from the GDACS feed
func (p *GDACSProvider) Fetch(ctx context.Context) ([]*model.Event, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", p.feedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch GDACS data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GDACS API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read GDACS response: %w", err)
	}

	var gdacsResponse GDACSResponse
	if err := json.Unmarshal(body, &gdacsResponse); err != nil {
		return nil, fmt.Errorf("failed to parse GDACS JSON: %w", err)
	}

	var events []*model.Event
	for _, feature := range gdacsResponse.Features {
		event, err := p.featureToEvent(feature)
		if err != nil {
			// Log error but continue processing other features
			fmt.Printf("Failed to convert GDACS feature to event: %v\n", err)
			continue
		}
		events = append(events, event)
	}

	return events, nil
}

// featureToEvent converts a GDACS GeoJSON feature to an Event
func (p *GDACSProvider) featureToEvent(feature GDACSFeature) (*model.Event, error) {
	props := feature.Properties

	// Parse event type and determine category
	eventType := strings.ToLower(props.EventType)
	var category string
	switch eventType {
	case "eq", "earthquake":
		category = "earthquake"
	case "tc", "tropical cyclone":
		category = "storm"
	case "fl", "flood":
		category = "flood"
	case "vo", "volcano":
		category = "volcano"
	case "dr", "drought":
		category = "drought"
	default:
		category = "disaster"
	}

	// Determine severity from alert level
	var severity model.Severity
	switch props.AlertLevel {
	case "Red":
		severity = model.SeverityCritical
	case "Orange":
		severity = model.SeverityHigh
	case "Yellow":
		severity = model.SeverityMedium
	case "Green":
		severity = model.SeverityLow
	default:
		severity = model.SeverityMedium
	}

	// Parse coordinates from geometry
	var coordinates []float64
	if feature.Geometry.Type == "Point" && len(feature.Geometry.Coordinates) >= 2 {
		lon := feature.Geometry.Coordinates[0]
		lat := feature.Geometry.Coordinates[1]
		coordinates = []float64{lon, lat}
	} else {
		return nil, fmt.Errorf("invalid geometry in GDACS feature")
	}

	// Parse time - GDACS uses format like "2025-12-11T00:00:00"
	occurredAt, err := time.Parse("2006-01-02T15:04:05", props.FromDate)
	if err != nil {
		// Try alternative format
		occurredAt, err = time.Parse(time.RFC3339, props.FromDate)
		if err != nil {
			// Fallback to current time
			occurredAt = time.Now().UTC()
		}
	}

	// Use provided description or generate one
	title := props.Name
	if title == "" {
		title = fmt.Sprintf("%s - %s", props.EventName, props.Country)
	}
	
	description := props.Description
	if description == "" {
		description = fmt.Sprintf("%s in %s. Alert level: %s.",
			props.EventType, props.Country, props.AlertLevel)
		
		// Add severity info if available
		if props.SeverityData != nil {
			description += fmt.Sprintf(" %s", props.SeverityData.SeverityText)
		}
	}

	// Handle optional fields
	magnitude := 0.0
	if props.Magnitude != nil {
		magnitude = *props.Magnitude
	}
	
	// Build metadata
	metadata := map[string]string{
		"gdacs_eventid":          strconv.Itoa(props.EventID),
		"gdacs_eventname":        props.EventName,
		"gdacs_country":          props.Country,
		"gdacs_alertlevel":       props.AlertLevel,
		"gdacs_alertscore":       strconv.Itoa(props.AlertScore),
		"gdacs_eventtype":        props.EventType,
		"gdacs_fromdate":         props.FromDate,
		"gdacs_todate":           props.ToDate,
		"gdacs_datemodified":     props.DateModified,
		"gdacs_description":      props.Description,
		"gdacs_episodeid":        strconv.Itoa(props.EpisodeID),
		"gdacs_episodealertlevel": props.EpisodeAlertLevel,
		"gdacs_episodealertscore": strconv.FormatFloat(props.EpisodeAlertScore, 'f', 1, 64),
		"gdacs_glide":            props.Glide,
		"gdacs_isocurrent":       props.IsCurrent,
		"gdacs_iso3":             props.ISO3,
		"gdacs_istemporary":      props.IsTemporary,
		"gdacs_name":             props.Name,
		"gdacs_polygonlabel":     props.PolygonLabel,
		"gdacs_source":           props.Source,
		"gdacs_sourceid":         props.SourceID,
		"gdacs_url_details":      props.URL.Details,
		"gdacs_url_geometry":     props.URL.Geometry,
		"gdacs_url_report":       props.URL.Report,
	}
	
	// Add optional fields
	if props.Population != nil {
		metadata["gdacs_population"] = *props.Population
	}
	if props.Vulnerability != nil {
		metadata["gdacs_vulnerability"] = *props.Vulnerability
	}
	if props.SeverityData != nil {
		metadata["gdacs_severity"] = strconv.FormatFloat(props.SeverityData.Severity, 'f', 0, 64)
		metadata["gdacs_severitytext"] = props.SeverityData.SeverityText
		metadata["gdacs_severityunit"] = props.SeverityData.SeverityUnit
	}
	
	// Add affected countries
	for i, country := range props.AffectedCountries {
		metadata[fmt.Sprintf("gdacs_country_%d_name", i)] = country.CountryName
		metadata[fmt.Sprintf("gdacs_country_%d_iso2", i)] = country.ISO2
		metadata[fmt.Sprintf("gdacs_country_%d_iso3", i)] = country.ISO3
	}

	// Create event
	event := &model.Event{
		ID:          uuid.New().String(),
		Title:       title,
		Description: description,
		Source:      "gdacs",
		SourceID:    strconv.Itoa(props.EventID),
		OccurredAt:  occurredAt,
		IngestedAt:  time.Now().UTC(),
		Location: model.Location{
			Type:        "Point",
			Coordinates: coordinates,
		},
		Precision: model.PrecisionApproximate, // GDACS locations are approximate
		Magnitude: magnitude,
		Category:  category,
		Severity:  severity,
		Metadata:  metadata,
		Badges: []model.Badge{
			{
				Label:     "gdacs",
				Type:      model.BadgeTypeSource,
				Timestamp: time.Now().UTC(),
			},
			{
				Label:     "approximate",
				Type:      model.BadgeTypePrecision,
				Timestamp: time.Now().UTC(),
			},
			{
				Label:     "just now",
				Type:      model.BadgeTypeFreshness,
				Timestamp: time.Now().UTC(),
			},
		},
	}

	return event, nil
}

// GDACSResponse represents the GDACS API response structure
type GDACSResponse struct {
	Type     string         `json:"type"`
	Features []GDACSFeature `json:"features"`
	BBox     []float64      `json:"bbox"`
}

// GDACSFeature represents a feature in GDACS response
type GDACSFeature struct {
	Type       string          `json:"type"`
	Properties GDACSProperties `json:"properties"`
	Geometry   GDACSGeometry   `json:"geometry"`
	BBox       []float64       `json:"bbox"`
}

// GDACSProperties represents properties in a GDACS feature
type GDACSProperties struct {
	EventID           int               `json:"eventid"`
	EventName         string            `json:"eventname"`
	EventType         string            `json:"eventtype"`
	AlertLevel        string            `json:"alertlevel"`
	AlertScore        int               `json:"alertscore"`
	Country           string            `json:"country"`
	FromDate          string            `json:"fromdate"`
	ToDate            string            `json:"todate"`
	DateModified      string            `json:"datemodified"`
	Description       string            `json:"description"`
	EpisodeID         int               `json:"episodeid"`
	EpisodeAlertLevel string            `json:"episodealertlevel"`
	EpisodeAlertScore float64           `json:"episodealertscore"`
	Glide             string            `json:"glide"`
	HTMLDescription   string            `json:"htmldescription"`
	Icon              string            `json:"icon"`
	IconOverall       string            `json:"iconoverall"`
	IsCurrent         string            `json:"iscurrent"`
	ISO3              string            `json:"iso3"`
	IsTemporary       string            `json:"istemporary"`
	Name              string            `json:"name"`
	PolygonLabel      string            `json:"polygonlabel"`
	Source            string            `json:"source"`
	SourceID          string            `json:"sourceid"`
	URL               GDACSURL          `json:"url"`
	// Optional fields that may not be present for all event types
	Magnitude         *float64          `json:"magnitude,omitempty"`
	Population        *string           `json:"population,omitempty"`
	Vulnerability     *string           `json:"vulnerability,omitempty"`
	SeverityData      *GDACSSeverityData `json:"severitydata,omitempty"`
	AffectedCountries []GDACSCountry    `json:"affectedcountries,omitempty"`
}

// GDACSGeometry represents geometry in a GDACS feature
type GDACSGeometry struct {
	Type        string    `json:"type"`
	Coordinates []float64 `json:"coordinates"`
}

// GDACSURL represents URL fields in GDACS properties
type GDACSURL struct {
	Details  string `json:"details"`
	Geometry string `json:"geometry"`
	Report   string `json:"report"`
}

// GDACSSeverityData represents severity data in GDACS properties
type GDACSSeverityData struct {
	Severity     float64 `json:"severity"`
	SeverityText string `json:"severitytext"`
	SeverityUnit string `json:"severityunit"`
}

// GDACSCountry represents affected country in GDACS properties
type GDACSCountry struct {
	CountryName string `json:"countryname"`
	ISO2        string `json:"iso2"`
	ISO3        string `json:"iso3"`
}