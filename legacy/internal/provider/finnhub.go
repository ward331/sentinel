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

// FinnhubProvider fetches market news, earnings, and sentiment from Finnhub
// Tier 1: Free with API key (60 calls/minute)
// Category: financial
// Signup: https://finnhub.io/register
type FinnhubProvider struct {
	client *http.Client
	config *Config
}

// NewFinnhubProvider creates a new FinnhubProvider
func NewFinnhubProvider(config *Config) *FinnhubProvider {
	return &FinnhubProvider{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		config: config,
	}
}

// Name returns the provider identifier
func (p *FinnhubProvider) Name() string {
	return "finnhub"
}

// Enabled returns whether the provider is enabled (requires API key)
func (p *FinnhubProvider) Enabled() bool {
	if p.config == nil || p.config.APIKey == "" {
		return false
	}
	return p.config.Enabled
}

// Interval returns the polling interval
func (p *FinnhubProvider) Interval() time.Duration {
	if p.config != nil && p.config.PollInterval > 0 {
		return p.config.PollInterval
	}
	return 300 * time.Second
}

// finnhubNewsItem represents a Finnhub market news item
type finnhubNewsItem struct {
	Category string `json:"category"`
	Datetime int64  `json:"datetime"`
	Headline string `json:"headline"`
	ID       int64  `json:"id"`
	Image    string `json:"image"`
	Related  string `json:"related"`
	Source   string `json:"source"`
	Summary  string `json:"summary"`
	URL      string `json:"url"`
}

// Fetch retrieves market news and sentiment from Finnhub
func (p *FinnhubProvider) Fetch(ctx context.Context) ([]*model.Event, error) {
	var allEvents []*model.Event

	// Fetch general market news
	newsEvents, err := p.fetchMarketNews(ctx)
	if err != nil {
		fmt.Printf("Warning: Finnhub news fetch failed: %v\n", err)
	} else {
		allEvents = append(allEvents, newsEvents...)
	}

	return allEvents, nil
}

func (p *FinnhubProvider) fetchMarketNews(ctx context.Context) ([]*model.Event, error) {
	url := fmt.Sprintf("https://finnhub.io/api/v1/news?category=general&token=%s", p.config.APIKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Finnhub request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Finnhub data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Finnhub returned status %d: %s", resp.StatusCode, string(body))
	}

	var news []finnhubNewsItem
	if err := json.NewDecoder(resp.Body).Decode(&news); err != nil {
		return nil, fmt.Errorf("failed to decode Finnhub response: %w", err)
	}

	var events []*model.Event
	maxEvents := 50

	for _, item := range news {
		occurredAt := time.Unix(item.Datetime, 0).UTC()

		// Only include news from the last 24 hours
		if time.Since(occurredAt) > 24*time.Hour {
			continue
		}

		event := &model.Event{
			Title:       item.Headline,
			Description: fmt.Sprintf("Finnhub Market News\n\nSource: %s\nCategory: %s\nRelated: %s\n\n%s\n\nURL: %s", item.Source, item.Category, item.Related, item.Summary, item.URL),
			Source:      "finnhub",
			SourceID:    fmt.Sprintf("fh_%d", item.ID),
			OccurredAt:  occurredAt,
			Location:    model.Point(-74.0060, 40.7128), // NYC financial district
			Precision:   model.PrecisionApproximate,
			Category:    "financial",
			Severity:    model.SeverityLow,
			Metadata: map[string]string{
				"news_id":    fmt.Sprintf("%d", item.ID),
				"source":     item.Source,
				"category":   item.Category,
				"related":    item.Related,
				"url":        item.URL,
				"image":      item.Image,
				"tier":       "1",
				"signup_url": "https://finnhub.io/register",
			},
			Badges: []model.Badge{
				{Label: "Finnhub", Type: "source", Timestamp: occurredAt},
				{Label: "financial", Type: "category", Timestamp: occurredAt},
				{Label: item.Category, Type: "news_category", Timestamp: occurredAt},
			},
		}

		events = append(events, event)
		if len(events) >= maxEvents {
			break
		}
	}

	return events, nil
}
