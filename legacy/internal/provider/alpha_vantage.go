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

// AlphaVantageProvider fetches stock quotes, forex, and commodities from Alpha Vantage
// Tier 1: Free with API key (25 requests/day on free tier)
// Category: financial
// Signup: https://www.alphavantage.co/support/#api-key
type AlphaVantageProvider struct {
	client *http.Client
	config *Config
}

// NewAlphaVantageProvider creates a new AlphaVantageProvider
func NewAlphaVantageProvider(config *Config) *AlphaVantageProvider {
	return &AlphaVantageProvider{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		config: config,
	}
}

// Name returns the provider identifier
func (p *AlphaVantageProvider) Name() string {
	return "alpha_vantage"
}

// Enabled returns whether the provider is enabled (requires API key)
func (p *AlphaVantageProvider) Enabled() bool {
	if p.config == nil || p.config.APIKey == "" {
		return false
	}
	return p.config.Enabled
}

// Interval returns the polling interval
func (p *AlphaVantageProvider) Interval() time.Duration {
	if p.config != nil && p.config.PollInterval > 0 {
		return p.config.PollInterval
	}
	return 300 * time.Second
}

// avGlobalQuote represents the Alpha Vantage Global Quote response
type avGlobalQuote struct {
	GlobalQuote struct {
		Symbol           string `json:"01. symbol"`
		Open             string `json:"02. open"`
		High             string `json:"03. high"`
		Low              string `json:"04. low"`
		Price            string `json:"05. price"`
		Volume           string `json:"06. volume"`
		LatestTradingDay string `json:"07. latest trading day"`
		PreviousClose    string `json:"08. previous close"`
		Change           string `json:"09. change"`
		ChangePercent    string `json:"10. change percent"`
	} `json:"Global Quote"`
}

// Fetch retrieves financial data from Alpha Vantage
func (p *AlphaVantageProvider) Fetch(ctx context.Context) ([]*model.Event, error) {
	// Query a few key market indicators
	symbols := []struct {
		symbol string
		name   string
		lon    float64
		lat    float64
	}{
		{"SPY", "S&P 500 ETF", -74.0060, 40.7128},    // NYSE
		{"DIA", "Dow Jones ETF", -74.0060, 40.7128},   // NYSE
		{"GLD", "Gold ETF", -74.0060, 40.7128},        // NYSE
		{"USO", "US Oil Fund", -74.0060, 40.7128},     // NYSE
	}

	var allEvents []*model.Event

	for _, sym := range symbols {
		event, err := p.fetchQuote(ctx, sym.symbol, sym.name, sym.lon, sym.lat)
		if err != nil {
			fmt.Printf("Warning: Alpha Vantage %s fetch failed: %v\n", sym.symbol, err)
			continue
		}
		if event != nil {
			allEvents = append(allEvents, event)
		}
	}

	return allEvents, nil
}

func (p *AlphaVantageProvider) fetchQuote(ctx context.Context, symbol, name string, lon, lat float64) (*model.Event, error) {
	url := fmt.Sprintf("https://www.alphavantage.co/query?function=GLOBAL_QUOTE&symbol=%s&apikey=%s",
		symbol, p.config.APIKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Alpha Vantage request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Alpha Vantage data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Alpha Vantage returned status %d: %s", resp.StatusCode, string(body))
	}

	var data avGlobalQuote
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode Alpha Vantage response: %w", err)
	}

	q := data.GlobalQuote
	if q.Symbol == "" {
		return nil, fmt.Errorf("empty response for %s (may be rate limited)", symbol)
	}

	price, _ := strconv.ParseFloat(q.Price, 64)
	change, _ := strconv.ParseFloat(q.Change, 64)

	tradingDay, err := time.Parse("2006-01-02", q.LatestTradingDay)
	if err != nil {
		tradingDay = time.Now().UTC()
	}

	direction := "unchanged"
	if change > 0 {
		direction = "up"
	} else if change < 0 {
		direction = "down"
	}

	severity := model.SeverityLow
	changeAbs := change
	if changeAbs < 0 {
		changeAbs = -changeAbs
	}
	pctAbs := changeAbs / price * 100
	if pctAbs > 5 {
		severity = model.SeverityCritical
	} else if pctAbs > 3 {
		severity = model.SeverityHigh
	} else if pctAbs > 1 {
		severity = model.SeverityMedium
	}

	event := &model.Event{
		Title:       fmt.Sprintf("%s (%s): $%.2f %s %s", name, symbol, price, direction, q.ChangePercent),
		Description: fmt.Sprintf("Alpha Vantage Market Data\n\nSymbol: %s (%s)\nPrice: $%s\nChange: %s (%s)\nOpen: $%s\nHigh: $%s\nLow: $%s\nVolume: %s\nPrev Close: $%s\nTrading Day: %s", symbol, name, q.Price, q.Change, q.ChangePercent, q.Open, q.High, q.Low, q.Volume, q.PreviousClose, q.LatestTradingDay),
		Source:      "alpha_vantage",
		SourceID:    fmt.Sprintf("av_%s_%s", symbol, q.LatestTradingDay),
		OccurredAt:  tradingDay,
		Location:    model.Point(lon, lat),
		Precision:   model.PrecisionExact,
		Category:    "financial",
		Severity:    severity,
		Metadata: map[string]string{
			"symbol":         symbol,
			"name":           name,
			"price":          q.Price,
			"change":         q.Change,
			"change_percent": q.ChangePercent,
			"volume":         q.Volume,
			"trading_day":    q.LatestTradingDay,
			"tier":           "1",
			"signup_url":     "https://www.alphavantage.co/support/#api-key",
		},
		Badges: []model.Badge{
			{Label: "Alpha Vantage", Type: "source", Timestamp: tradingDay},
			{Label: "financial", Type: "category", Timestamp: tradingDay},
			{Label: symbol, Type: "symbol", Timestamp: tradingDay},
		},
	}

	return event, nil
}
