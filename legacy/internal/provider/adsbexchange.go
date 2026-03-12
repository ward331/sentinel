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

// ADSBExchangeProvider fetches aircraft position data from ADS-B Exchange via RapidAPI
// Tier 1: Free with API key (RapidAPI)
// Category: aviation
// Signup: https://rapidapi.com/adsbx/api/adsbexchange-com1
type ADSBExchangeProvider struct {
	client *http.Client
	config *Config
}

// NewADSBExchangeProvider creates a new ADSBExchangeProvider
func NewADSBExchangeProvider(config *Config) *ADSBExchangeProvider {
	return &ADSBExchangeProvider{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		config: config,
	}
}

// Name returns the provider identifier
func (p *ADSBExchangeProvider) Name() string {
	return "adsbexchange"
}

// Enabled returns whether the provider is enabled (requires API key)
func (p *ADSBExchangeProvider) Enabled() bool {
	if p.config == nil || p.config.APIKey == "" {
		return false
	}
	return p.config.Enabled
}

// Interval returns the polling interval
func (p *ADSBExchangeProvider) Interval() time.Duration {
	if p.config != nil && p.config.PollInterval > 0 {
		return p.config.PollInterval
	}
	return 30 * time.Second
}

// adsbExchangeResponse represents the ADS-B Exchange API response
type adsbExchangeResponse struct {
	AC []adsbExchangeAircraft `json:"ac"`
}

type adsbExchangeAircraft struct {
	Hex      string  `json:"hex"`
	Flight   string  `json:"flight"`
	Lat      float64 `json:"lat"`
	Lon      float64 `json:"lon"`
	AltBaro  int     `json:"alt_baro"`
	GS       float64 `json:"gs"`
	Track    float64 `json:"track"`
	Squawk   string  `json:"squawk"`
	Type     string  `json:"t"`
	DbFlags  int     `json:"dbFlags"`
	Category string  `json:"category"`
	Reg      string  `json:"r"`
}

// Fetch retrieves aircraft positions from ADS-B Exchange
func (p *ADSBExchangeProvider) Fetch(ctx context.Context) ([]*model.Event, error) {
	// Use the RapidAPI endpoint for military/interesting aircraft
	url := "https://adsbexchange-com1.p.rapidapi.com/v2/mil/"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create ADS-B Exchange request: %w", err)
	}

	req.Header.Set("X-RapidAPI-Key", p.config.APIKey)
	req.Header.Set("X-RapidAPI-Host", "adsbexchange-com1.p.rapidapi.com")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ADS-B Exchange data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ADS-B Exchange returned status %d: %s", resp.StatusCode, string(body))
	}

	var data adsbExchangeResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode ADS-B Exchange response: %w", err)
	}

	var events []*model.Event
	now := time.Now().UTC()
	maxEvents := 200

	for _, ac := range data.AC {
		if ac.Lat == 0 && ac.Lon == 0 {
			continue
		}

		callsign := ac.Flight
		if callsign == "" {
			callsign = ac.Hex
		}

		severity := model.SeverityLow
		if ac.DbFlags&1 != 0 { // Military flag
			severity = model.SeverityMedium
		}

		event := &model.Event{
			Title:       fmt.Sprintf("Aircraft %s (%s) at FL%d", callsign, ac.Type, ac.AltBaro/100),
			Description: fmt.Sprintf("Aircraft %s (ICAO: %s, Reg: %s, Type: %s) at altitude %d ft, speed %.0f kts, track %.0f deg", callsign, ac.Hex, ac.Reg, ac.Type, ac.AltBaro, ac.GS, ac.Track),
			Source:      "adsbexchange",
			SourceID:    fmt.Sprintf("adsbx_%s_%d", ac.Hex, now.Unix()),
			OccurredAt:  now,
			Location:    model.Point(ac.Lon, ac.Lat),
			Precision:   model.PrecisionExact,
			Category:    "aviation",
			Severity:    severity,
			Metadata: map[string]string{
				"icao":     ac.Hex,
				"callsign": callsign,
				"reg":      ac.Reg,
				"type":     ac.Type,
				"altitude": fmt.Sprintf("%d", ac.AltBaro),
				"speed":    fmt.Sprintf("%.0f", ac.GS),
				"track":    fmt.Sprintf("%.0f", ac.Track),
				"squawk":   ac.Squawk,
				"tier":     "1",
			},
			Badges: []model.Badge{
				{Label: "ADS-B Exchange", Type: "source", Timestamp: now},
				{Label: "aviation", Type: "category", Timestamp: now},
			},
		}

		events = append(events, event)
		if len(events) >= maxEvents {
			break
		}
	}

	return events, nil
}
