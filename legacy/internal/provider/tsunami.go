package provider

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/openclaw/sentinel-backend/internal/model"
)

// TsunamiProvider fetches tsunami alerts from Pacific Tsunami Warning Center
type TsunamiProvider struct {
	client *http.Client
	config *Config
}

// Name returns the provider name
func (p *TsunamiProvider) Name() string {
    return "tsunami"
}

// Interval returns the polling interval
func (p *TsunamiProvider) Interval() time.Duration {
    interval, _ := time.ParseDuration("1h")
    return interval
}

// Enabled returns whether the provider is enabled
func (p *TsunamiProvider) Enabled() bool {
    return p.config != nil && p.config.Enabled
}

// NewTsunamiProvider creates a new TsunamiProvider
func NewTsunamiProvider(config *Config) *TsunamiProvider {
	return &TsunamiProvider{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		config: config,
	}
}

// Fetch retrieves tsunami alerts from PTWC
func (p *TsunamiProvider) Fetch(ctx context.Context) ([]*model.Event, error) {
	// Use the NWS tsunami alerts Atom/RSS feed (PTWC old PHP endpoint no longer works)
	url := "https://www.tsunami.gov/events/xml/PAAQAtom.xml"
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("User-Agent", "SENTINEL/2.0 (https://github.com/ward331/sentinel)")
	
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch PTWC alerts: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("PTWC RSS returned status %d: %s", resp.StatusCode, string(body))
	}
	
	// Read the full body so we can try multiple parse formats
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Try parsing as RSS first
	var rss RSSFeed
	if err := xml.Unmarshal(body, &rss); err == nil && len(rss.Channel.Items) > 0 {
		return p.convertToEvents(rss)
	}

	// Try parsing as Atom feed (tsunami.gov uses Atom format)
	var atom TsunamiAtomFeed
	if err := xml.Unmarshal(body, &atom); err == nil && len(atom.Entries) > 0 {
		return p.convertAtomToEvents(atom)
	}

	// If both fail, return empty
	return []*model.Event{}, nil
}

// convertToEvents converts PTWC RSS feed to SENTINEL events
func (p *TsunamiProvider) convertToEvents(rss RSSFeed) ([]*model.Event, error) {
	var events []*model.Event
	
	for _, item := range rss.Channel.Items {
		// Skip non-tsunami items
		if !p.isTsunamiAlert(item) {
			continue
		}
		
		event := &model.Event{
			Title:       p.generateTitle(item),
			Description: p.generateDescription(item),
			Source:      "ptwc",
			SourceID:    p.extractSourceID(item),
			OccurredAt:  p.parsePubDate(item.PubDate),
			Location:    p.extractLocation(item),
			Precision:   model.PrecisionApproximate,
			Magnitude:   p.calculateMagnitude(item),
			Category:    "tsunami",
			Severity:    p.determineSeverity(item),
			Metadata:    p.generateMetadata(item),
			Badges:      p.generateBadges(item),
		}
		
		events = append(events, event)
	}
	
	return events, nil
}

// isTsunamiAlert checks if an RSS item is a tsunami alert
func (p *TsunamiProvider) isTsunamiAlert(item RSSItem) bool {
	title := strings.ToLower(item.Title)
	description := strings.ToLower(item.Description)
	
	// Check for tsunami-related keywords
	keywords := []string{"tsunami", "tidal wave", "seismic sea wave"}
	for _, keyword := range keywords {
		if strings.Contains(title, keyword) || strings.Contains(description, keyword) {
			return true
		}
	}
	
	return false
}

// generateTitle creates a title for the tsunami alert
func (p *TsunamiProvider) generateTitle(item RSSItem) string {
	// Clean up the title
	title := strings.TrimSpace(item.Title)
	title = strings.ReplaceAll(title, "PTWC - ", "")
	title = strings.ReplaceAll(title, "PTWC:", "")
	title = strings.ReplaceAll(title, "PTWC", "")
	title = strings.TrimSpace(title)
	
	return fmt.Sprintf("🌊 %s", title)
}

// generateDescription creates a description for the tsunami alert
func (p *TsunamiProvider) generateDescription(item RSSItem) string {
	// Clean up the description
	desc := strings.TrimSpace(item.Description)
	
	// Remove HTML tags and extra whitespace
	desc = strings.ReplaceAll(desc, "<br/>", "\n")
	desc = strings.ReplaceAll(desc, "<br>", "\n")
	desc = strings.ReplaceAll(desc, "<p>", "\n")
	desc = strings.ReplaceAll(desc, "</p>", "\n")
	
	// Remove any remaining HTML tags
	for strings.Contains(desc, "<") && strings.Contains(desc, ">") {
		start := strings.Index(desc, "<")
		end := strings.Index(desc, ">")
		if end > start {
			desc = desc[:start] + desc[end+1:]
		}
	}
	
	// Clean up whitespace
	desc = strings.ReplaceAll(desc, "\n\n", "\n")
	desc = strings.ReplaceAll(desc, "  ", " ")
	desc = strings.TrimSpace(desc)
	
	return desc
}

// extractSourceID extracts a unique source ID from the RSS item
func (p *TsunamiProvider) extractSourceID(item RSSItem) string {
	// Use GUID if available
	if item.GUID != "" {
		return fmt.Sprintf("ptwc_%s", item.GUID)
	}
	
	// Use link as fallback
	if item.Link != "" {
		// Extract last part of URL
		parts := strings.Split(item.Link, "/")
		if len(parts) > 0 {
			lastPart := parts[len(parts)-1]
			if lastPart != "" {
				return fmt.Sprintf("ptwc_%s", lastPart)
			}
		}
	}
	
	// Generate from title hash
	return fmt.Sprintf("ptwc_%d", time.Now().UnixNano())
}

// extractLocation extracts location from tsunami alert
func (p *TsunamiProvider) extractLocation(item RSSItem) model.GeoJSON {
	// Try to parse coordinates from description
	desc := strings.ToLower(item.Description)
	
	// Look for coordinate patterns
	coordPatterns := []string{
		"lat ", "latitude ", "lon ", "longitude ",
		"coordinates:", "location:",
	}
	
	for _, pattern := range coordPatterns {
		if idx := strings.Index(desc, pattern); idx != -1 {
			// Try to extract numbers after pattern
			substr := desc[idx+len(pattern):]
			if len(substr) > 20 {
				substr = substr[:20]
			}
			
			// Simple coordinate extraction (this is approximate)
			// In a real implementation, would use proper parsing
		}
	}
	
	// Default to Pacific Ocean center
	return model.GeoJSON{
		Type:        "Point",
		Coordinates: []float64{-160.0, 0.0}, // Central Pacific
	}
}

// calculateMagnitude calculates magnitude based on alert content
func (p *TsunamiProvider) calculateMagnitude(item RSSItem) float64 {
	desc := strings.ToLower(item.Description)
	title := strings.ToLower(item.Title)
	
	magnitude := 5.0 // Base magnitude for tsunami alerts
	
	// Check for magnitude indicators
	if strings.Contains(desc, "major") || strings.Contains(title, "major") {
		magnitude += 2.0
	}
	if strings.Contains(desc, "destructive") || strings.Contains(title, "destructive") {
		magnitude += 3.0
	}
	if strings.Contains(desc, "warning") {
		magnitude += 1.5
	}
	if strings.Contains(desc, "watch") {
		magnitude += 1.0
	}
	if strings.Contains(desc, "advisory") {
		magnitude += 0.5
	}
	
	// Check for earthquake magnitude references
	if strings.Contains(desc, "m ") || strings.Contains(desc, "magnitude ") {
		// Try to extract earthquake magnitude
		// Simple pattern matching
		for i := 90; i >= 10; i-- {
			magStr := fmt.Sprintf("m %d", i/10)
			if strings.Contains(desc, magStr) {
				magnitude += float64(i) / 10.0
				break
			}
		}
	}
	
	return magnitude
}

// determineSeverity determines the event severity
func (p *TsunamiProvider) determineSeverity(item RSSItem) model.Severity {
	desc := strings.ToLower(item.Description)
	title := strings.ToLower(item.Title)
	
	if strings.Contains(desc, "warning") || strings.Contains(title, "warning") {
		return model.SeverityCritical
	}
	if strings.Contains(desc, "watch") || strings.Contains(title, "watch") {
		return model.SeverityHigh
	}
	if strings.Contains(desc, "advisory") || strings.Contains(title, "advisory") {
		return model.SeverityMedium
	}
	if strings.Contains(desc, "information") || strings.Contains(title, "information") {
		return model.SeverityLow
	}
	
	return model.SeverityMedium
}

// generateMetadata creates metadata for the tsunami alert
func (p *TsunamiProvider) generateMetadata(item RSSItem) map[string]string {
	metadata := map[string]string{
		"source":    "PTWC",
		"timestamp": p.parsePubDate(item.PubDate).Format(time.RFC3339),
		"title":     item.Title,
		"link":      item.Link,
	}
	
	if item.GUID != "" {
		metadata["guid"] = item.GUID
	}
	
	if item.Category != "" {
		metadata["category"] = item.Category
	}
	
	// Extract key information from description
	desc := strings.ToLower(item.Description)
	
	// Check for affected areas
	areaKeywords := []string{"hawaii", "alaska", "japan", "chile", "indonesia", "philippines"}
	for _, area := range areaKeywords {
		if strings.Contains(desc, area) {
			metadata["affected_area"] = area
			break
		}
	}
	
	// Check for earthquake info
	if strings.Contains(desc, "earthquake") {
		metadata["trigger"] = "earthquake"
	}
	
	return metadata
}

// generateBadges creates badges for the tsunami alert
func (p *TsunamiProvider) generateBadges(item RSSItem) []model.Badge {
	badges := []model.Badge{
		{
			Label:     "PTWC",
			Type:      "source",
			Timestamp: p.parsePubDate(item.PubDate),
		},
		{
			Label:     "Tsunami",
			Type:      "hazard",
			Timestamp: p.parsePubDate(item.PubDate),
		},
	}
	
	// Add severity badge
	severity := p.determineSeverity(item)
	badges = append(badges, model.Badge{
		Label:     strings.Title(string(severity)),
		Type:      "severity",
		Timestamp: p.parsePubDate(item.PubDate),
	})
	
	// Add warning type badge
	desc := strings.ToLower(item.Description)
	if strings.Contains(desc, "warning") {
		badges = append(badges, model.Badge{
			Label:     "Warning",
			Type:      "alert_type",
			Timestamp: p.parsePubDate(item.PubDate),
		})
	} else if strings.Contains(desc, "watch") {
		badges = append(badges, model.Badge{
			Label:     "Watch",
			Type:      "alert_type",
			Timestamp: p.parsePubDate(item.PubDate),
		})
	} else if strings.Contains(desc, "advisory") {
		badges = append(badges, model.Badge{
			Label:     "Advisory",
			Type:      "alert_type",
			Timestamp: p.parsePubDate(item.PubDate),
		})
	}
	
	// Add Pacific badge (PTWC covers Pacific region)
	badges = append(badges, model.Badge{
		Label:     "Pacific",
		Type:      "region",
		Timestamp: p.parsePubDate(item.PubDate),
	})
	
	return badges
}

// TsunamiAtomFeed represents an Atom feed structure
type TsunamiAtomFeed struct {
	XMLName xml.Name           `xml:"feed"`
	Title   string             `xml:"title"`
	Entries []TsunamiAtomEntry `xml:"entry"`
}

// TsunamiAtomEntry represents an Atom feed entry
type TsunamiAtomEntry struct {
	Title   string `xml:"title"`
	ID      string `xml:"id"`
	Updated string `xml:"updated"`
	Summary string `xml:"summary"`
	Link    struct {
		Href string `xml:"href,attr"`
	} `xml:"link"`
}

// convertAtomToEvents converts Atom feed entries to SENTINEL events
func (p *TsunamiProvider) convertAtomToEvents(feed TsunamiAtomFeed) ([]*model.Event, error) {
	var events []*model.Event

	for _, entry := range feed.Entries {
		// Convert Atom entry to RSSItem to reuse existing logic
		item := RSSItem{
			Title:       entry.Title,
			Link:        entry.Link.Href,
			Description: entry.Summary,
			PubDate:     entry.Updated,
			GUID:        entry.ID,
		}

		// Only include tsunami-related entries
		if !p.isTsunamiAlert(item) {
			continue
		}

		event := &model.Event{
			Title:       p.generateTitle(item),
			Description: p.generateDescription(item),
			Source:      "ptwc",
			SourceID:    p.extractSourceID(item),
			OccurredAt:  p.parsePubDate(item.PubDate),
			Location:    p.extractLocation(item),
			Precision:   model.PrecisionApproximate,
			Magnitude:   p.calculateMagnitude(item),
			Category:    "tsunami",
			Severity:    p.determineSeverity(item),
			Metadata:    p.generateMetadata(item),
			Badges:      p.generateBadges(item),
		}

		events = append(events, event)
	}

	return events, nil
}

// parsePubDate parses RSS pubDate string
func (p *TsunamiProvider) parsePubDate(pubDate string) time.Time {
	if pubDate == "" {
		return time.Now().UTC()
	}
	
	// Try common RSS and Atom date formats
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		time.RFC1123,
		time.RFC1123Z,
		"Mon, 02 Jan 2006 15:04:05 MST",
		"Mon, 02 Jan 2006 15:04:05 -0700",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05-07:00",
		time.RFC822,
		time.RFC822Z,
	}

	for _, format := range formats {
		t, err := time.Parse(format, pubDate)
		if err == nil {
			return t.UTC()
		}
	}

	return time.Now().UTC()
}

// Using shared RSS types from common.go
