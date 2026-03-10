package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/openclaw/sentinel-backend/internal/model"
)

// USGSProvider implements the Provider interface for USGS earthquake data
type USGSProvider struct {
	feedURL string
	client  *http.Client
}

// Name returns the provider name
func (p *USGSProvider) Name() string {
    return "usgs"
}

// Interval returns the polling interval
func (p *USGSProvider) Interval() time.Duration {
    interval, _ := time.ParseDuration("1m")
    return interval
}

// Enabled returns whether the provider is enabled
func (p *USGSProvider) Enabled() bool {
    return p.config != nil && p.config.Enabled
}

// NewUSGSProvider creates a new USGS provider
func NewUSGSProvider() *USGSProvider {
	return &USGSProvider{
		feedURL: "https://earthquake.usgs.gov/earthquakes/feed/v1.0/summary/all_hour.geojson",
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name returns the provider identifier
func (p *USGSProvider) Name() string {
	return "usgs"
}

// Fetch retrieves events from the USGS feed
func (p *USGSProvider) Fetch(ctx context.Context) ([]*model.Event, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", p.feedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch USGS data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("USGS API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read USGS response: %w", err)
	}

	var usgsResponse USGSGeoJSON
	if err := json.Unmarshal(body, &usgsResponse); err != nil {
		return nil, fmt.Errorf("failed to parse USGS GeoJSON: %w", err)
	}

	var events []*model.Event
	for _, feature := range usgsResponse.Features {
		event, err := p.featureToEvent(feature)
		if err != nil {
			// Log error but continue processing other features
			fmt.Printf("Failed to convert USGS feature to event: %v\n", err)
			continue
		}
		events = append(events, event)
	}

	return events, nil
}

// featureToEvent converts a USGS GeoJSON feature to an Event
func (p *USGSProvider) featureToEvent(feature USGSFeature) (*model.Event, error) {
	// Parse properties
	props := feature.Properties

	// Parse magnitude (already a float64)
	magnitude := props.Mag

	// Parse time (milliseconds since epoch)
	occurredAt := time.Unix(props.Time/1000, 0)

	// Generate title
	title := fmt.Sprintf("M %.1f - %s", magnitude, props.Place)

	// Parse coordinates (GeoJSON uses [lon, lat, depth])
	var coordinates []float64
	if len(feature.Geometry.Coordinates) >= 2 {
		lon := feature.Geometry.Coordinates[0]
		lat := feature.Geometry.Coordinates[1]
		coordinates = []float64{lon, lat}
	} else {
		return nil, fmt.Errorf("invalid coordinates in feature")
	}

	// Determine severity based on magnitude
	var severity model.Severity
	switch {
	case magnitude >= 6.0:
		severity = model.SeverityCritical
	case magnitude >= 5.0:
		severity = model.SeverityHigh
	case magnitude >= 4.0:
		severity = model.SeverityMedium
	default:
		severity = model.SeverityLow
	}

	// Generate description
	description := fmt.Sprintf("A magnitude %.1f earthquake occurred %s. Depth: %.1f km.", 
		magnitude, props.Place, props.Depth)

	// Create event
	event := &model.Event{
		ID:          uuid.New().String(),
		Title:       title,
		Description: description,
		Source:      "usgs",
		SourceID:    feature.ID,
		OccurredAt:  occurredAt,
		IngestedAt:  time.Now().UTC(),
		Location: model.Location{
			Type:        "Point",
			Coordinates: coordinates,
		},
		Precision: model.PrecisionExact,
		Magnitude: magnitude,
		Category:  "earthquake",
		Severity:  severity,
		Metadata: map[string]string{
			"usgs_id":      feature.ID,
			"usgs_url":     props.URL,
			"usgs_detail":  props.Detail,
			"usgs_status":  props.Status,
			"usgs_tsunami": strconv.Itoa(props.Tsunami),
			"usgs_sig":     strconv.Itoa(props.Sig),
			"usgs_net":     props.Net,
			"usgs_code":    props.Code,
			"usgs_ids":     props.IDS,
			"usgs_sources": props.Sources,
			"usgs_types":   props.Types,
			"usgs_nst":     strconv.Itoa(props.NST),
			"usgs_dmin":    strconv.FormatFloat(props.Dmin, 'f', -1, 64),
			"usgs_rms":     strconv.FormatFloat(props.RMS, 'f', -1, 64),
			"usgs_gap":     strconv.FormatFloat(props.Gap, 'f', -1, 64),
			"usgs_magType": props.MagType,
			"usgs_type":    props.Type,
		},
		Badges: []model.Badge{
			{
				Label:     "usgs",
				Type:      model.BadgeTypeSource,
				Timestamp: time.Now().UTC(),
			},
			{
				Label:     "exact",
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

// USGSGeoJSON represents the USGS GeoJSON response structure
type USGSGeoJSON struct {
	Type     string        `json:"type"`
	Metadata USGSMetadata  `json:"metadata"`
	Features []USGSFeature `json:"features"`
}

// USGSMetadata represents metadata in USGS GeoJSON
type USGSMetadata struct {
	Generated int64  `json:"generated"`
	URL       string `json:"url"`
	Title     string `json:"title"`
	Status    int    `json:"status"`
	API       string `json:"api"`
	Count     int    `json:"count"`
}

// USGSFeature represents a feature in USGS GeoJSON
type USGSFeature struct {
	Type       string         `json:"type"`
	Properties USGSProperties `json:"properties"`
	Geometry   USGSGeometry   `json:"geometry"`
	ID         string         `json:"id"`
}

// USGSProperties represents properties in a USGS feature
type USGSProperties struct {
	Mag     float64 `json:"mag"`
	Place   string  `json:"place"`
	Time    int64   `json:"time"`
	Updated int64   `json:"updated"`
	TZ      int     `json:"tz"`
	URL     string  `json:"url"`
	Detail  string  `json:"detail"`
	Felt    int     `json:"felt"`
	CDI     float64 `json:"cdi"`
	MMI     float64 `json:"mmi"`
	Alert   string  `json:"alert"`
	Status  string  `json:"status"`
	Tsunami int     `json:"tsunami"`
	Sig     int     `json:"sig"`
	Net     string  `json:"net"`
	Code    string  `json:"code"`
	IDS     string  `json:"ids"`
	Sources string  `json:"sources"`
	Types   string  `json:"types"`
	NST     int     `json:"nst"`
	Dmin    float64 `json:"dmin"`
	RMS     float64 `json:"rms"`
	Gap     float64 `json:"gap"`
	MagType string  `json:"magType"`
	Type    string  `json:"type"`
	Title   string  `json:"title"`
	Depth   float64 `json:"depth"`
}

// USGSGeometry represents geometry in a USGS feature
type USGSGeometry struct {
	Type        string      `json:"type"`
	Coordinates []float64   `json:"coordinates"`
}
