package provider

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/openclaw/sentinel-backend/internal/model"
)

// LiveUAMapProvider fetches conflict events from LiveUAMap RSS feed
type LiveUAMapProvider struct {
	name     string
	feedURL  string
	interval time.Duration
	config   *Config
}

// NewLiveUAMapProvider creates a new LiveUAMap provider
func NewLiveUAMapProvider(config *Config) *LiveUAMapProvider {
	return &LiveUAMapProvider{
		name:     "liveuamap",
		feedURL:  "https://liveuamap.com/rss",
		interval: 900 * time.Second, // 15 minutes
		config:   config,
	}
}




// Fetch retrieves conflict events from LiveUAMap RSS

// Name returns the provider identifier
func (p *LiveUAMapProvider) Name() string {
	return "liveuamap"
}


// Enabled returns whether the provider is enabled
func (p *LiveUAMapProvider) Enabled() bool {
	if p.config != nil {
		return p.config.Enabled
	}
	return true
}


// Interval returns the polling interval
func (p *LiveUAMapProvider) Interval() time.Duration {
	if p.config != nil && p.config.PollInterval > 0 {
		return p.config.PollInterval
	}
	return 5 * time.Minute // Default interval
}

func (p *LiveUAMapProvider) Fetch(ctx context.Context) ([]*model.Event, error) {
	events, err := p.fetchRSS(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch RSS feed: %w", err)
	}

	return events, nil
}

// fetchRSS fetches and parses the RSS feed
func (p *LiveUAMapProvider) fetchRSS(ctx context.Context) ([]*model.Event, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", p.feedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/rss+xml,application/xml")
	req.Header.Set("User-Agent", "SENTINEL/1.0")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch RSS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("RSS returned status %d: %s", resp.StatusCode, string(body))
	}

	var rss LiveUAMapRSSFeed
	if err := xml.NewDecoder(resp.Body).Decode(&rss); err != nil {
		return nil, fmt.Errorf("failed to decode RSS: %w", err)
	}

	return p.convertToEvents(rss), nil
}

// convertToEvents converts RSS items to SENTINEL events
func (p *LiveUAMapProvider) convertToEvents(rss LiveUAMapRSSFeed) []*model.Event {
	events := make([]*model.Event, 0, len(rss.Channel.Items))

	for _, item := range rss.Channel.Items {
		// Skip items without coordinates
		coords := p.extractCoordinates(item)
		if coords == nil {
			continue
		}

		event := &model.Event{
			ID:          fmt.Sprintf("liveuamap-%d", item.GUID),
			Title:       p.cleanTitle(item.Title),
			Description: p.generateDescription(item),
			Source:      p.name,
			SourceID:    fmt.Sprintf("%d", item.GUID),
			OccurredAt:  item.PubDate,
			Location: model.Location{
				Type:        "Point",
				Coordinates: coords,
			},
			Precision: model.PrecisionExact,
			Magnitude: p.calculateMagnitude(item),
			Category:  "conflict",
			Severity:  p.determineSeverity(item),
			Metadata:  p.generateMetadata(item),
			Badges:    p.generateBadges(item),
		}

		events = append(events, event)
	}

	return events
}

// extractCoordinates extracts coordinates from RSS item
func (p *LiveUAMapProvider) extractCoordinates(item LiveUAMapRSSItem) []float64 {
	// Try to extract from description first
	coordPattern := regexp.MustCompile(`(\-?\d+\.\d+)[,\s]+(\-?\d+\.\d+)`)
	
	// Check description
	if matches := coordPattern.FindStringSubmatch(item.Description); matches != nil {
		lat := p.parseFloat(matches[1])
		lon := p.parseFloat(matches[2])
		if lat != 0 || lon != 0 {
			return []float64{lon, lat}
		}
	}

	// Check title
	if matches := coordPattern.FindStringSubmatch(item.Title); matches != nil {
		lat := p.parseFloat(matches[1])
		lon := p.parseFloat(matches[2])
		if lat != 0 || lon != 0 {
			return []float64{lon, lat}
		}
	}

	// Try to extract from link (some LiveUAMap links contain coordinates)
	if strings.Contains(item.Link, "?ll=") {
		parts := strings.Split(item.Link, "?ll=")
		if len(parts) > 1 {
			coords := strings.Split(parts[1], "&")[0]
			coordParts := strings.Split(coords, ",")
			if len(coordParts) == 2 {
				lat := p.parseFloat(coordParts[0])
				lon := p.parseFloat(coordParts[1])
				if lat != 0 || lon != 0 {
					return []float64{lon, lat}
				}
			}
		}
	}

	return nil
}

// parseFloat safely parses a float string
func (p *LiveUAMapProvider) parseFloat(s string) float64 {
	var result float64
	fmt.Sscanf(s, "%f", &result)
	return result
}

// cleanTitle cleans and formats the title
func (p *LiveUAMapProvider) cleanTitle(title string) string {
	// Remove HTML tags
	title = regexp.MustCompile(`<[^>]*>`).ReplaceAllString(title, "")
	
	// Remove coordinates if present
	title = regexp.MustCompile(`\s*[\-\.\d]+[,\s]+[\-\.\d]+\s*`).ReplaceAllString(title, "")
	
	// Trim and clean
	title = strings.TrimSpace(title)
	title = strings.ReplaceAll(title, "&nbsp;", " ")
	title = strings.ReplaceAll(title, "&amp;", "&")
	
	// Capitalize first letter
	if len(title) > 0 {
		title = strings.ToUpper(title[:1]) + title[1:]
	}
	
	return title
}

// generateDescription generates event description
func (p *LiveUAMapProvider) generateDescription(item LiveUAMapRSSItem) string {
	var desc strings.Builder
	
	desc.WriteString("Conflict Event Report\n")
	desc.WriteString("=====================\n\n")
	
	// Clean description
	cleanDesc := p.cleanDescription(item.Description)
	desc.WriteString(cleanDesc)
	
	// Add source info
	desc.WriteString("\n\n---\n")
	desc.WriteString("Source: LiveUAMap\n")
	desc.WriteString("Type: Crowdsourced conflict reporting\n")
	desc.WriteString("Verification: Community-verified OSINT\n")
	desc.WriteString("Update: Real-time (15-minute polling)\n")
	
	// Add link
	if item.Link != "" {
		desc.WriteString(fmt.Sprintf("Original: %s\n", item.Link))
	}
	
	return desc.String()
}

// cleanDescription cleans HTML from description
func (p *LiveUAMapProvider) cleanDescription(desc string) string {
	// Remove HTML tags
	desc = regexp.MustCompile(`<[^>]*>`).ReplaceAllString(desc, "")
	
	// Replace HTML entities
	desc = strings.ReplaceAll(desc, "&nbsp;", " ")
	desc = strings.ReplaceAll(desc, "&amp;", "&")
	desc = strings.ReplaceAll(desc, "&lt;", "<")
	desc = strings.ReplaceAll(desc, "&gt;", ">")
	desc = strings.ReplaceAll(desc, "&quot;", "\"")
	desc = strings.ReplaceAll(desc, "&#39;", "'")
	
	// Clean up whitespace
	desc = strings.ReplaceAll(desc, "\n\n\n", "\n\n")
	desc = strings.TrimSpace(desc)
	
	return desc
}

// calculateMagnitude calculates event magnitude
func (p *LiveUAMapProvider) calculateMagnitude(item LiveUAMapRSSItem) float64 {
	magnitude := 2.5 // Base for conflict events
	
	// Adjust based on keywords
	text := strings.ToLower(item.Title + " " + item.Description)
	
	// High-impact keywords
	highImpactWords := []string{
		"strike", "attack", "missile", "drone", "artillery", "shelling",
		"casualties", "killed", "wounded", "destroyed", "damaged",
		"critical", "infrastructure", "power plant", "hospital", "school",
	}
	
	for _, word := range highImpactWords {
		if strings.Contains(text, word) {
			magnitude += 0.3
		}
	}
	
	// Very high-impact keywords
	veryHighImpactWords := []string{
		"mass casualty", "chemical", "nuclear", "biological",
		"war crime", "genocide", "ethnic cleansing", "mass grave",
		"cluster munition", "thermobaric", "vacuum bomb",
	}
	
	for _, word := range veryHighImpactWords {
		if strings.Contains(text, word) {
			magnitude += 0.8
		}
	}
	
	// Cap magnitude
	if magnitude > 5.0 {
		magnitude = 5.0
	}
	
	return magnitude
}

// determineSeverity determines event severity
func (p *LiveUAMapProvider) determineSeverity(item LiveUAMapRSSItem) model.Severity {
	text := strings.ToLower(item.Title + " " + item.Description)
	
	// Critical severity indicators
	criticalWords := []string{
		"chemical attack", "nuclear", "biological weapon",
		"mass grave", "genocide", "war crime", "ethnic cleansing",
		"hospital bombed", "school attacked", "children killed",
	}
	
	for _, word := range criticalWords {
		if strings.Contains(text, word) {
			return model.SeverityCritical
		}
	}
	
	// High severity indicators
	highWords := []string{
		"strike", "missile", "drone attack", "artillery",
		"casualties", "killed", "wounded", "destroyed",
		"infrastructure", "power plant", "bridge", "airport",
	}
	
	for _, word := range highWords {
		if strings.Contains(text, word) {
			return model.SeverityHigh
		}
	}
	
	// Medium severity indicators
	mediumWords := []string{
		"fighting", "clash", "skirmish", "exchange",
		"shelling", "mortar", "grenade", "small arms",
		"checkpoint", "border", "protest", "demonstration",
	}
	
	for _, word := range mediumWords {
		if strings.Contains(text, word) {
			return model.SeverityMedium
		}
	}
	
	return model.SeverityLow
}

// generateMetadata generates event metadata
func (p *LiveUAMapProvider) generateMetadata(item LiveUAMapRSSItem) map[string]string {
	metadata := map[string]string{
		"guid":           fmt.Sprintf("%d", item.GUID),
		"link":           item.Link,
		"author":         item.Author,
		"categories":     strings.Join(item.Categories, "; "),
		"comments":       item.Comments,
		"source":         "LiveUAMap",
		"data_type":      "crowdsourced_osint",
		"verification":   "community_verified",
		"update_frequency": "15 minutes",
		"coverage":       "Global conflict zones",
		"reliability":    "High (community-verified)",
	}
	
	// Extract location from description if available
	coords := p.extractCoordinates(item)
	if coords != nil {
		metadata["latitude"] = fmt.Sprintf("%.4f", coords[1])
		metadata["longitude"] = fmt.Sprintf("%.4f", coords[0])
	}
	
	// Extract country if mentioned
	country := p.extractCountry(item)
	if country != "" {
		metadata["country"] = country
	}
	
	return metadata
}

// extractCountry extracts country from item
func (p *LiveUAMapProvider) extractCountry(item LiveUAMapRSSItem) string {
	text := strings.ToLower(item.Title + " " + item.Description)
	
	countryPatterns := map[string][]string{
		"Ukraine":   {"ukraine", "kyiv", "kharkiv", "odesa", "donbas"},
		"Russia":    {"russia", "moscow", "st. petersburg", "rostov"},
		"Syria":     {"syria", "damascus", "aleppo", "idlib"},
		"Israel":    {"israel", "tel aviv", "jerusalem", "gaza"},
		"Iran":      {"iran", "tehran", "isfahan", "shiraz"},
		"Yemen":     {"yemen", "sanaa", "aden", "houthi"},
		"Afghanistan": {"afghanistan", "kabul", "taliban"},
		"Myanmar":   {"myanmar", "burma", "yangon", "naypyidaw"},
		"Sudan":     {"sudan", "khartoum", "darfur"},
		"Ethiopia":  {"ethiopia", "addis ababa", "tigray"},
	}
	
	for country, patterns := range countryPatterns {
		for _, pattern := range patterns {
			if strings.Contains(text, pattern) {
				return country
			}
		}
	}
	
	return ""
}

// generateBadges generates event badges
func (p *LiveUAMapProvider) generateBadges(item LiveUAMapRSSItem) []model.Badge {
	badges := []model.Badge{
		{
			Type:      model.BadgeTypeSource,
			Label:     "LiveUAMap",
			Timestamp: time.Now().UTC(),
		},
		{
			Type:      model.BadgeTypePrecision,
			Label:     "Exact",
			Timestamp: time.Now().UTC(),
		},
		{
			Type:      model.BadgeTypeFreshness,
			Label:     "Real-time",
			Timestamp: time.Now().UTC(),
		},
		{
			Type:      "verification",
			Label:     "Community OSINT",
			Timestamp: time.Now().UTC(),
		},
	}
	
	// Add conflict type badge
	text := strings.ToLower(item.Title + " " + item.Description)
	
	if strings.Contains(text, "drone") || strings.Contains(text, "uav") {
		badges = append(badges, model.Badge{
			Type:      "weapon",
			Label:     "Drone",
			Timestamp: time.Now().UTC(),
		})
	}
	
	if strings.Contains(text, "missile") {
		badges = append(badges, model.Badge{
			Type:      "weapon",
			Label:     "Missile",
			Timestamp: time.Now().UTC(),
		})
	}
	
	if strings.Contains(text, "artillery") || strings.Contains(text, "shelling") {
		badges = append(badges, model.Badge{
			Type:      "weapon",
			Label:     "Artillery",
			Timestamp: time.Now().UTC(),
		})
	}
	
	// Add country badge
	country := p.extractCountry(item)
	if country != "" {
		badges = append(badges, model.Badge{
			Type:      "country",
			Label:     country,
			Timestamp: time.Now().UTC(),
		})
	}
	
	return badges
}

// RSSFeed represents the LiveUAMap RSS feed structure
// LiveUAMapRSSFeed represents the LiveUAMap RSS feed structure
type LiveUAMapRSSFeed struct {
	XMLName xml.Name          `xml:"rss"`
	Channel LiveUAMapChannel  `xml:"channel"`
}

// LiveUAMapChannel represents an RSS channel for LiveUAMap
type LiveUAMapChannel struct {
	Title       string            `xml:"title"`
	Link        string            `xml:"link"`
	Description string            `xml:"description"`
	Language    string            `xml:"language"`
	PubDate     string            `xml:"pubDate"`
	LastBuildDate string          `xml:"lastBuildDate"`
	Items       []LiveUAMapRSSItem `xml:"item"`
}

// LiveUAMapRSSItem represents an RSS item for LiveUAMap
type LiveUAMapRSSItem struct {
	Title       string   `xml:"title"`
	Link        string   `xml:"link"`
	Description string   `xml:"description"`
	Author      string   `xml:"author"`
	Categories  []string `xml:"category"`
	Comments    string   `xml:"comments"`
	GUID        int      `xml:"guid"`
	PubDate     time.Time `xml:"pubDate"`
}
