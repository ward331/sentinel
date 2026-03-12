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

// AISStreamProvider fetches maritime AIS vessel data from aisstream.io
// Tier 1: Free with API key
// Category: maritime
// Signup: https://aisstream.io
type AISStreamProvider struct {
	client *http.Client
	config *Config
}

// NewAISStreamProvider creates a new AISStreamProvider
func NewAISStreamProvider(config *Config) *AISStreamProvider {
	return &AISStreamProvider{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		config: config,
	}
}

// Name returns the provider identifier
func (p *AISStreamProvider) Name() string {
	return "aisstream"
}

// Enabled returns whether the provider is enabled (requires API key)
func (p *AISStreamProvider) Enabled() bool {
	if p.config == nil || p.config.APIKey == "" {
		return false
	}
	return p.config.Enabled
}

// Interval returns the polling interval
func (p *AISStreamProvider) Interval() time.Duration {
	if p.config != nil && p.config.PollInterval > 0 {
		return p.config.PollInterval
	}
	return 60 * time.Second
}

// aisStreamResponse represents the AISStream REST API response
type aisStreamResponse struct {
	Data []aisStreamMessage `json:"data"`
}

type aisStreamMessage struct {
	MessageType string             `json:"MessageType"`
	MetaData    aisStreamMeta      `json:"MetaData"`
	Message     aisStreamAISReport `json:"Message"`
}

type aisStreamMeta struct {
	MMSI       int     `json:"MMSI"`
	ShipName   string  `json:"ShipName"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	TimeUtc    string  `json:"time_utc"`
}

type aisStreamAISReport struct {
	ShipType      int     `json:"ShipType"`
	SOG           float64 `json:"Sog"`
	COG           float64 `json:"Cog"`
	TrueHeading   int     `json:"TrueHeading"`
	NavigStatus   int     `json:"NavigationalStatus"`
	Destination   string  `json:"Destination"`
	CallSign      string  `json:"CallSign"`
}

// Fetch retrieves vessel positions from AISStream
func (p *AISStreamProvider) Fetch(ctx context.Context) ([]*model.Event, error) {
	// AISStream REST endpoint for latest vessel data
	url := "https://stream.aisstream.io/v0/stream"

	// Build request body for bounding box query
	bbox := p.config.BoundingBox
	if len(bbox) < 4 {
		// Default to global coverage sample
		bbox = []float64{-180, -90, 180, 90}
	}

	body := fmt.Sprintf(`{"APIKey":"%s","BoundingBoxes":[[%f,%f],[%f,%f]],"FiltersShipMMSI":[],"FilterMessageTypes":["PositionReport"]}`,
		p.config.APIKey, bbox[1], bbox[0], bbox[3], bbox[2])

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create AISStream request: %w", err)
	}

	// Use the REST snapshot endpoint instead of WebSocket
	snapshotURL := fmt.Sprintf("https://api.aisstream.io/v0/ships?apikey=%s&area=%f,%f,%f,%f&limit=200",
		p.config.APIKey, bbox[1], bbox[0], bbox[3], bbox[2])

	req, err = http.NewRequestWithContext(ctx, "GET", snapshotURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create AISStream snapshot request: %w", err)
	}

	_ = body // Used in WebSocket mode

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch AISStream data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("AISStream returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var data aisStreamResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode AISStream response: %w", err)
	}

	var events []*model.Event
	now := time.Now().UTC()

	for _, msg := range data.Data {
		if msg.MetaData.Latitude == 0 && msg.MetaData.Longitude == 0 {
			continue
		}

		shipName := msg.MetaData.ShipName
		if shipName == "" {
			shipName = fmt.Sprintf("MMSI-%d", msg.MetaData.MMSI)
		}

		shipType := classifyShipType(msg.Message.ShipType)

		event := &model.Event{
			Title:       fmt.Sprintf("Vessel %s (%s)", shipName, shipType),
			Description: fmt.Sprintf("Vessel %s (MMSI: %d, Type: %s) heading %.0f deg at %.1f kts, destination: %s", shipName, msg.MetaData.MMSI, shipType, msg.Message.COG, msg.Message.SOG, msg.Message.Destination),
			Source:      "aisstream",
			SourceID:    fmt.Sprintf("ais_%d_%d", msg.MetaData.MMSI, now.Unix()),
			OccurredAt:  now,
			Location:    model.Point(msg.MetaData.Longitude, msg.MetaData.Latitude),
			Precision:   model.PrecisionExact,
			Category:    "maritime",
			Severity:    model.SeverityLow,
			Metadata: map[string]string{
				"mmsi":        fmt.Sprintf("%d", msg.MetaData.MMSI),
				"ship_name":   shipName,
				"ship_type":   shipType,
				"sog":         fmt.Sprintf("%.1f", msg.Message.SOG),
				"cog":         fmt.Sprintf("%.0f", msg.Message.COG),
				"destination": msg.Message.Destination,
				"callsign":    msg.Message.CallSign,
				"tier":        "1",
			},
			Badges: []model.Badge{
				{Label: "AISStream", Type: "source", Timestamp: now},
				{Label: "maritime", Type: "category", Timestamp: now},
			},
		}

		events = append(events, event)
	}

	return events, nil
}

// classifyShipType returns a human-readable ship type from AIS type code
func classifyShipType(code int) string {
	switch {
	case code >= 70 && code <= 79:
		return "Cargo"
	case code >= 80 && code <= 89:
		return "Tanker"
	case code >= 60 && code <= 69:
		return "Passenger"
	case code >= 40 && code <= 49:
		return "High-Speed Craft"
	case code >= 30 && code <= 39:
		return "Fishing"
	case code >= 50 && code <= 59:
		return "Special Craft"
	case code == 0:
		return "Unknown"
	default:
		return fmt.Sprintf("Type-%d", code)
	}
}
