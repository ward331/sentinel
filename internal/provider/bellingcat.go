package provider

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/openclaw/sentinel-backend/internal/model"
)

// BellingcatProvider fetches Bellingcat OSINT investigations RSS feed
type BellingcatProvider struct {
	name     string
	feedURL  string
	interval time.Duration
}

// NewBellingcatProvider creates a new Bellingcat RSS provider
func NewBellingcatProvider() *BellingcatProvider {
	return &BellingcatProvider{
		name:     "bellingcat",
		feedURL:  "https://www.bellingcat.com/feed/",
		interval: 3600 * time.Second,
	}
}

// Name returns the provider identifier
func (p *BellingcatProvider) Name() string {
	return p.name
}

// Enabled returns whether the provider is enabled
func (p *BellingcatProvider) Enabled() bool {
	return true
}

// Interval returns the polling interval
func (p *BellingcatProvider) Interval() time.Duration {
	return p.interval
}

// Fetch retrieves Bellingcat investigations from RSS
func (p *BellingcatProvider) Fetch(ctx context.Context) ([]*model.Event, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", p.feedURL, nil)
	if err != nil {
		return []*model.Event{}, nil
	}

	req.Header.Set("User-Agent", "SENTINEL/3.0")
	req.Header.Set("Accept", "application/rss+xml, application/xml, text/xml")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return []*model.Event{}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []*model.Event{}, nil
	}

	var rss RSSFeed
	if err := xml.NewDecoder(resp.Body).Decode(&rss); err != nil {
		return []*model.Event{}, nil
	}

	var events []*model.Event
	for _, item := range rss.Channel.Items {
		pubDate := parseRSSDate(item.PubDate)

		// Only include items from the last 7 days
		if time.Since(pubDate) > 7*24*time.Hour {
			continue
		}

		sourceID := item.GUID
		if sourceID == "" {
			sourceID = item.Link
		}

		severity := p.determineSeverity(item)
		category := p.determineCategory(item)

		desc := cleanHTMLTags(item.Description)
		if len(desc) > 600 {
			desc = desc[:600] + "..."
		}

		event := &model.Event{
			Title:       fmt.Sprintf("Bellingcat: %s", item.Title),
			Description: desc,
			Source:      p.name,
			SourceID:    fmt.Sprintf("bellingcat_%s", sourceID),
			OccurredAt:  pubDate,
			Location:    model.Point(0, 0), // OSINT — location varies per article
			Precision:   model.PrecisionUnknown,
			Category:    category,
			Severity:    severity,
			Metadata: map[string]string{
				"link":     item.Link,
				"category": item.Category,
				"source":   "Bellingcat",
				"type":     "osint_investigation",
			},
		}
		events = append(events, event)
	}

	return events, nil
}

func (p *BellingcatProvider) determineSeverity(item RSSItem) model.Severity {
	text := strings.ToLower(item.Title + " " + item.Description)
	switch {
	case strings.Contains(text, "war crime") || strings.Contains(text, "chemical weapon") || strings.Contains(text, "mass grave"):
		return model.SeverityCritical
	case strings.Contains(text, "attack") || strings.Contains(text, "strike") || strings.Contains(text, "missile") || strings.Contains(text, "bombing"):
		return model.SeverityHigh
	case strings.Contains(text, "disinformation") || strings.Contains(text, "investigation") || strings.Contains(text, "satellite"):
		return model.SeverityMedium
	default:
		return model.SeverityLow
	}
}

func (p *BellingcatProvider) determineCategory(item RSSItem) string {
	text := strings.ToLower(item.Title + " " + item.Category)
	switch {
	case strings.Contains(text, "ukraine") || strings.Contains(text, "russia") || strings.Contains(text, "conflict") || strings.Contains(text, "syria"):
		return "conflict"
	case strings.Contains(text, "technology") || strings.Contains(text, "tool") || strings.Contains(text, "guide"):
		return "osint"
	default:
		return "osint"
	}
}
