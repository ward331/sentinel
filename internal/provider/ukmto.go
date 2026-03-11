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

// UKMTOProvider fetches UKMTO (United Kingdom Maritime Trade Operations) maritime security warnings
type UKMTOProvider struct {
	name     string
	feedURL  string
	interval time.Duration
}

// NewUKMTOProvider creates a new UKMTO provider
func NewUKMTOProvider() *UKMTOProvider {
	return &UKMTOProvider{
		name:     "ukmto",
		feedURL:  "https://www.ukmto.org/indian-ocean/rss",
		interval: 1800 * time.Second,
	}
}

// Name returns the provider identifier
func (p *UKMTOProvider) Name() string {
	return p.name
}

// Enabled returns whether the provider is enabled
func (p *UKMTOProvider) Enabled() bool {
	return true
}

// Interval returns the polling interval
func (p *UKMTOProvider) Interval() time.Duration {
	return p.interval
}

// Fetch retrieves maritime security warnings from UKMTO RSS
func (p *UKMTOProvider) Fetch(ctx context.Context) ([]*model.Event, error) {
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
		severity := p.determineSeverity(item)
		pubDate := parseRSSDate(item.PubDate)

		sourceID := item.GUID
		if sourceID == "" {
			sourceID = item.Link
		}

		event := &model.Event{
			Title:       fmt.Sprintf("UKMTO: %s", item.Title),
			Description: cleanHTMLTags(item.Description),
			Source:      p.name,
			SourceID:    fmt.Sprintf("ukmto_%s", sourceID),
			OccurredAt:  pubDate,
			Location:    model.Point(57.0, 13.0), // Indian Ocean / Gulf of Aden area
			Precision:   model.PrecisionApproximate,
			Category:    "security",
			Severity:    severity,
			Metadata: map[string]string{
				"source": "UKMTO (United Kingdom Maritime Trade Operations)",
				"link":   item.Link,
				"type":   "maritime_security",
			},
		}
		events = append(events, event)
	}

	return events, nil
}

func (p *UKMTOProvider) determineSeverity(item RSSItem) model.Severity {
	text := strings.ToLower(item.Title + " " + item.Description)
	switch {
	case strings.Contains(text, "attack") || strings.Contains(text, "hijack") || strings.Contains(text, "missile"):
		return model.SeverityCritical
	case strings.Contains(text, "suspicious") || strings.Contains(text, "approach") || strings.Contains(text, "fired"):
		return model.SeverityHigh
	case strings.Contains(text, "warning") || strings.Contains(text, "advisory"):
		return model.SeverityMedium
	default:
		return model.SeverityLow
	}
}

// parseRSSDate parses common RSS date formats
func parseRSSDate(dateStr string) time.Time {
	if dateStr == "" {
		return time.Now().UTC()
	}
	formats := []string{
		time.RFC1123,
		time.RFC1123Z,
		"Mon, 02 Jan 2006 15:04:05 MST",
		"Mon, 02 Jan 2006 15:04:05 -0700",
		time.RFC822,
		time.RFC822Z,
		"2006-01-02T15:04:05Z",
		time.RFC3339,
	}
	for _, f := range formats {
		if t, err := time.Parse(f, dateStr); err == nil {
			return t.UTC()
		}
	}
	return time.Now().UTC()
}

// cleanHTMLTags strips HTML tags from text
func cleanHTMLTags(text string) string {
	result := text
	for strings.Contains(result, "<") && strings.Contains(result, ">") {
		start := strings.Index(result, "<")
		end := strings.Index(result, ">")
		if end > start {
			result = result[:start] + result[end+1:]
		} else {
			break
		}
	}
	return strings.TrimSpace(result)
}
