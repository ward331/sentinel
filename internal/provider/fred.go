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

// FREDProvider fetches Federal Reserve economic data (GDP, unemployment, inflation)
// Tier 1: Free with API key (120 requests/minute)
// Category: financial
// Signup: https://fred.stlouisfed.org/docs/api/api_key.html
type FREDProvider struct {
	client *http.Client
	config *Config
}

// NewFREDProvider creates a new FREDProvider
func NewFREDProvider(config *Config) *FREDProvider {
	return &FREDProvider{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		config: config,
	}
}

// Name returns the provider identifier
func (p *FREDProvider) Name() string {
	return "fred"
}

// Enabled returns whether the provider is enabled (requires API key)
func (p *FREDProvider) Enabled() bool {
	if p.config == nil || p.config.APIKey == "" {
		return false
	}
	return p.config.Enabled
}

// Interval returns the polling interval
func (p *FREDProvider) Interval() time.Duration {
	if p.config != nil && p.config.PollInterval > 0 {
		return p.config.PollInterval
	}
	return 3600 * time.Second
}

// fredSeriesResponse represents a FRED series observations response
type fredSeriesResponse struct {
	Observations []fredObservation `json:"observations"`
}

type fredObservation struct {
	Date  string `json:"date"`
	Value string `json:"value"`
}

// Fetch retrieves key economic indicators from FRED
func (p *FREDProvider) Fetch(ctx context.Context) ([]*model.Event, error) {
	// Key economic series to track
	series := []struct {
		id       string
		name     string
		unit     string
		category string
	}{
		{"GDP", "US GDP (Quarterly)", "Billions USD", "gdp"},
		{"UNRATE", "US Unemployment Rate", "%", "labor"},
		{"CPIAUCSL", "Consumer Price Index", "Index 1982-84=100", "inflation"},
		{"FEDFUNDS", "Federal Funds Rate", "%", "monetary_policy"},
		{"T10Y2Y", "10Y-2Y Treasury Spread", "%", "yield_curve"},
		{"DEXUSEU", "USD/EUR Exchange Rate", "USD per EUR", "forex"},
	}

	var allEvents []*model.Event

	for _, s := range series {
		event, err := p.fetchSeries(ctx, s.id, s.name, s.unit, s.category)
		if err != nil {
			fmt.Printf("Warning: FRED %s fetch failed: %v\n", s.id, err)
			continue
		}
		if event != nil {
			allEvents = append(allEvents, event)
		}
	}

	return allEvents, nil
}

func (p *FREDProvider) fetchSeries(ctx context.Context, seriesID, name, unit, category string) (*model.Event, error) {
	url := fmt.Sprintf("https://api.stlouisfed.org/fred/series/observations?series_id=%s&api_key=%s&file_type=json&sort_order=desc&limit=1",
		seriesID, p.config.APIKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create FRED request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch FRED data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("FRED returned status %d: %s", resp.StatusCode, string(body))
	}

	var data fredSeriesResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode FRED response: %w", err)
	}

	if len(data.Observations) == 0 {
		return nil, fmt.Errorf("no observations for %s", seriesID)
	}

	obs := data.Observations[0]
	if obs.Value == "." {
		return nil, fmt.Errorf("no data available for %s", seriesID)
	}

	value, _ := strconv.ParseFloat(obs.Value, 64)
	obsDate, err := time.Parse("2006-01-02", obs.Date)
	if err != nil {
		obsDate = time.Now().UTC()
	}

	severity := p.determineSeverity(seriesID, value)

	event := &model.Event{
		Title:       fmt.Sprintf("%s: %.2f %s", name, value, unit),
		Description: fmt.Sprintf("Federal Reserve Economic Data (FRED)\n\nSeries: %s\nIndicator: %s\nLatest Value: %.2f %s\nDate: %s\n\nSource: Federal Reserve Bank of St. Louis", seriesID, name, value, unit, obs.Date),
		Source:      "fred",
		SourceID:    fmt.Sprintf("fred_%s_%s", seriesID, obs.Date),
		OccurredAt:  obsDate,
		Location:    model.Point(-90.1994, 38.6270), // St. Louis Fed
		Precision:   model.PrecisionExact,
		Category:    "financial",
		Severity:    severity,
		Metadata: map[string]string{
			"series_id":    seriesID,
			"name":         name,
			"value":        obs.Value,
			"unit":         unit,
			"date":         obs.Date,
			"sub_category": category,
			"tier":         "1",
			"signup_url":   "https://fred.stlouisfed.org/docs/api/api_key.html",
		},
		Badges: []model.Badge{
			{Label: "FRED", Type: "source", Timestamp: obsDate},
			{Label: "financial", Type: "category", Timestamp: obsDate},
			{Label: category, Type: "indicator_type", Timestamp: obsDate},
		},
	}

	return event, nil
}

func (p *FREDProvider) determineSeverity(seriesID string, value float64) model.Severity {
	switch seriesID {
	case "UNRATE":
		if value > 8 {
			return model.SeverityCritical
		}
		if value > 6 {
			return model.SeverityHigh
		}
		if value > 5 {
			return model.SeverityMedium
		}
	case "FEDFUNDS":
		if value > 6 {
			return model.SeverityHigh
		}
		if value > 4 {
			return model.SeverityMedium
		}
	case "T10Y2Y":
		if value < -0.5 {
			return model.SeverityHigh // Deep inversion
		}
		if value < 0 {
			return model.SeverityMedium // Inverted
		}
	}
	return model.SeverityLow
}
