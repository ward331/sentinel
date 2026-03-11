package provider

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/openclaw/sentinel-backend/internal/model"
)

// VolcanoProvider fetches volcanic activity reports from Volcano Discovery
type VolcanoProvider struct {
	client *http.Client
	config *Config
}

// Name returns the provider name
func (p *VolcanoProvider) Name() string {
    return "volcano"
}

// Interval returns the polling interval
func (p *VolcanoProvider) Interval() time.Duration {
    interval, _ := time.ParseDuration("1h")
    return interval
}

// Enabled returns whether the provider is enabled
func (p *VolcanoProvider) Enabled() bool {
    return p.config != nil && p.config.Enabled
}

// NewVolcanoProvider creates a new VolcanoProvider
func NewVolcanoProvider(config *Config) *VolcanoProvider {
	return &VolcanoProvider{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		config: config,
	}
}

// Fetch retrieves volcanic activity reports from Volcano Discovery RSS
func (p *VolcanoProvider) Fetch(ctx context.Context) ([]*model.Event, error) {
	// Use the Smithsonian Global Volcanism Program weekly reports RSS feed
	url := "https://volcano.si.edu/news/WeeklyVolcanoRSS.xml"
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("User-Agent", "SENTINEL/2.0 (https://github.com/ward331/sentinel)")
	
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Volcano Discovery RSS: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Volcano Discovery RSS returned status %d: %s", resp.StatusCode, string(body))
	}

	// Read entire body first so we can handle encoding issues
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read RSS body: %w", err)
	}

	// Strip invalid UTF-8 and fix encoding declaration
	bodyStr := strings.ToValidUTF8(string(body), "")
	bodyStr = strings.Replace(bodyStr, `encoding="ISO-8859-1"`, `encoding="UTF-8"`, 1)
	bodyStr = strings.Replace(bodyStr, `encoding="iso-8859-1"`, `encoding="UTF-8"`, 1)
	bodyStr = strings.Replace(bodyStr, `encoding="latin1"`, `encoding="UTF-8"`, 1)

	// Parse RSS feed from the sanitized body
	var rss RSSFeed
	decoder := xml.NewDecoder(bytes.NewReader([]byte(bodyStr)))
	decoder.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		return input, nil
	}
	decoder.Strict = false
	if err := decoder.Decode(&rss); err != nil {
		return nil, fmt.Errorf("failed to parse RSS feed: %w", err)
	}
	
	return p.convertToEvents(rss)
}

// convertToEvents converts Volcano Discovery RSS feed to SENTINEL events
func (p *VolcanoProvider) convertToEvents(rss RSSFeed) ([]*model.Event, error) {
	var events []*model.Event
	
	for _, item := range rss.Channel.Items {
		// Check if this is volcanic activity (not just earthquakes)
		if !p.isVolcanicActivity(item) {
			continue
		}
		
		event := &model.Event{
			Title:       p.generateTitle(item),
			Description: p.generateDescription(item),
			Source:      "volcano_discovery",
			SourceID:    p.extractSourceID(item),
			OccurredAt:  p.parsePubDate(item.PubDate),
			Location:    p.extractLocation(item),
			Precision:   model.PrecisionExact,
			Magnitude:   p.extractMagnitude(item),
			Category:    "volcanic",
			Severity:    p.determineSeverity(item),
			Metadata:    p.generateMetadata(item),
			Badges:      p.generateBadges(item),
		}
		
		events = append(events, event)
	}
	
	return events, nil
}

// isVolcanicActivity checks if an RSS item describes volcanic activity
func (p *VolcanoProvider) isVolcanicActivity(item RSSItem) bool {
	title := strings.ToLower(item.Title)
	description := strings.ToLower(item.Description)
	
	// Check for volcanic activity keywords
	volcanoKeywords := []string{
		"volcano", "volcanic", "eruption", "ash", "lava",
		"fumarole", "crater", "magma", "pyroclastic",
		"volcan", // Common in volcano names
	}
	
	for _, keyword := range volcanoKeywords {
		if strings.Contains(title, keyword) || strings.Contains(description, keyword) {
			return true
		}
	}
	
	// Check for specific volcano names in title
	volcanoNames := []string{
		"etna", "vesuvius", "kilauea", "mauna loa", "popocatepetl",
		"fuego", "sakurajima", "merapi", "sinabung", "taal",
		"yellowstone", "st. helens", "rainier", "fujisan",
	}
	
	for _, name := range volcanoNames {
		if strings.Contains(title, name) {
			return true
		}
	}
	
	return false
}

// generateTitle creates a title for the volcanic activity event
func (p *VolcanoProvider) generateTitle(item RSSItem) string {
	title := strings.TrimSpace(item.Title)
	
	// Clean up common prefixes
	title = strings.ReplaceAll(title, "VolcanoDiscovery.com: ", "")
	title = strings.ReplaceAll(title, "VolcanoDiscovery: ", "")
	title = strings.ReplaceAll(title, "Volcano Discovery: ", "")
	
	// Extract volcano name if possible
	volcanoName := p.extractVolcanoName(item)
	if volcanoName != "" {
		return fmt.Sprintf("🌋 %s - %s", volcanoName, title)
	}
	
	return fmt.Sprintf("🌋 %s", title)
}

// generateDescription creates a description for the volcanic activity event
func (p *VolcanoProvider) generateDescription(item RSSItem) string {
	desc := strings.TrimSpace(item.Description)
	
	// Clean HTML tags
	desc = p.cleanHTML(desc)
	
	// Extract key information
	var builder strings.Builder
	
	// Add volcano name if found
	volcanoName := p.extractVolcanoName(item)
	if volcanoName != "" {
		builder.WriteString(fmt.Sprintf("Volcano: %s\n\n", volcanoName))
	}
	
	// Add location if available
	location := p.extractLocationText(item)
	if location != "" {
		builder.WriteString(fmt.Sprintf("Location: %s\n\n", location))
	}
	
	// Add the cleaned description
	builder.WriteString(desc)
	
	return builder.String()
}

// extractSourceID extracts a unique source ID from the RSS item
func (p *VolcanoProvider) extractSourceID(item RSSItem) string {
	if item.GUID != "" {
		return fmt.Sprintf("volcano_%s", item.GUID)
	}
	
	// Use link as fallback
	if item.Link != "" {
		// Extract article ID from URL
		re := regexp.MustCompile(`/(\d+)/`)
		matches := re.FindStringSubmatch(item.Link)
		if len(matches) > 1 {
			return fmt.Sprintf("volcano_%s", matches[1])
		}
	}
	
	// Generate from title and timestamp
	timestamp := p.parsePubDate(item.PubDate).Unix()
	return fmt.Sprintf("volcano_%d_%d", timestamp, len(item.Title))
}

// extractLocation extracts coordinates from volcanic activity report
func (p *VolcanoProvider) extractLocation(item RSSItem) model.GeoJSON {
	// Try to extract coordinates from description
	desc := strings.ToLower(item.Description)
	
	// Look for coordinate patterns
	latLonPattern := regexp.MustCompile(`(\d+\.\d+)[°\s]*[ns]?\s*[,;]\s*(\d+\.\d+)[°\s]*[ew]?`)
	matches := latLonPattern.FindStringSubmatch(desc)
	
	if len(matches) >= 3 {
		lat, err1 := strconv.ParseFloat(matches[1], 64)
		lon, err2 := strconv.ParseFloat(matches[2], 64)
		
		if err1 == nil && err2 == nil {
			// Check hemisphere indicators
			if strings.Contains(strings.ToLower(desc), "s") && !strings.Contains(strings.ToLower(desc), "n") {
				lat = -lat
			}
			if strings.Contains(strings.ToLower(desc), "w") && !strings.Contains(strings.ToLower(desc), "e") {
				lon = -lon
			}
			
			return model.GeoJSON{
				Type:        "Point",
				Coordinates: []float64{lon, lat},
			}
		}
	}
	
	// Look for simpler patterns
	coordPattern := regexp.MustCompile(`(\d+\.\d+)\s*,\s*(\d+\.\d+)`)
	matches = coordPattern.FindStringSubmatch(desc)
	
	if len(matches) >= 3 {
		lat, err1 := strconv.ParseFloat(matches[1], 64)
		lon, err2 := strconv.ParseFloat(matches[2], 64)
		
		if err1 == nil && err2 == nil {
			return model.GeoJSON{
				Type:        "Point",
				Coordinates: []float64{lon, lat},
			}
		}
	}
	
	// Default location (Pacific Ring of Fire)
	return model.GeoJSON{
		Type:        "Point",
		Coordinates: []float64{120.0, 0.0}, // Southeast Asia region
	}
}

// extractMagnitude extracts magnitude from volcanic activity report
func (p *VolcanoProvider) extractMagnitude(item RSSItem) float64 {
	title := strings.ToLower(item.Title)
	description := strings.ToLower(item.Description)
	
	// Look for magnitude patterns
	magPattern := regexp.MustCompile(`magnitude\s*([\d\.]+)|m\s*([\d\.]+)`)
	matches := magPattern.FindStringSubmatch(title + " " + description)
	
	if len(matches) >= 2 {
		for i := 1; i < len(matches); i++ {
			if matches[i] != "" {
				mag, err := strconv.ParseFloat(matches[i], 64)
				if err == nil && mag > 0 {
					return mag
				}
			}
		}
	}
	
	// Estimate based on keywords
	magnitude := 3.0 // Base for volcanic activity
	
	// Check for intensity indicators
	text := title + " " + description
	if strings.Contains(text, "major") || strings.Contains(text, "large") {
		magnitude += 2.0
	}
	if strings.Contains(text, "explosive") {
		magnitude += 2.5
	}
	if strings.Contains(text, "ash plume") {
		magnitude += 1.5
	}
	if strings.Contains(text, "lava flow") {
		magnitude += 1.0
	}
	if strings.Contains(text, "alert level") {
		magnitude += 0.5
	}
	
	return magnitude
}

// determineSeverity determines the event severity
func (p *VolcanoProvider) determineSeverity(item RSSItem) model.Severity {
	text := strings.ToLower(item.Title + " " + item.Description)
	
	if strings.Contains(text, "explosive") || strings.Contains(text, "major eruption") {
		return model.SeverityCritical
	}
	if strings.Contains(text, "eruption") || strings.Contains(text, "ash emission") {
		return model.SeverityHigh
	}
	if strings.Contains(text, "increased activity") || strings.Contains(text, "seismic") {
		return model.SeverityMedium
	}
	if strings.Contains(text, "alert") || strings.Contains(text, "monitoring") {
		return model.SeverityLow
	}
	
	return model.SeverityMedium
}

// extractVolcanoName extracts volcano name from the report
func (p *VolcanoProvider) extractVolcanoName(item RSSItem) string {
	title := item.Title
	
	// Common volcano name patterns
	volcanoPatterns := []string{
		"Mount ", "Mt. ", "Volcano ", "Volcan ", "Caldera ",
	}
	
	for _, pattern := range volcanoPatterns {
		if idx := strings.Index(title, pattern); idx != -1 {
			// Extract name after pattern
			namePart := title[idx+len(pattern):]
			// Take first word or phrase
			if spaceIdx := strings.Index(namePart, " "); spaceIdx != -1 {
				return pattern + namePart[:spaceIdx]
			}
			return pattern + namePart
		}
	}
	
	// Look for known volcano names
	knownVolcanoes := []string{
		"Etna", "Vesuvius", "Kilauea", "Mauna Loa", "Popocatepetl",
		"Fuego", "Sakurajima", "Merapi", "Sinabung", "Taal",
		"Yellowstone", "St. Helens", "Rainier", "Fujisan",
	}
	
	for _, volcano := range knownVolcanoes {
		if strings.Contains(title, volcano) {
			return volcano
		}
	}
	
	return ""
}

// extractLocationText extracts location description
func (p *VolcanoProvider) extractLocationText(item RSSItem) string {
	desc := item.Description
	
	// Look for location patterns
	locationPatterns := []string{
		"Location:", "located in", "near", "region of",
	}
	
	for _, pattern := range locationPatterns {
		if idx := strings.Index(strings.ToLower(desc), strings.ToLower(pattern)); idx != -1 {
			locPart := desc[idx+len(pattern):]
			if len(locPart) > 100 {
				locPart = locPart[:100]
			}
			// Clean up
			locPart = p.cleanHTML(locPart)
			locPart = strings.TrimSpace(locPart)
			if dotIdx := strings.Index(locPart, "."); dotIdx != -1 {
				locPart = locPart[:dotIdx]
			}
			return locPart
		}
	}
	
	return ""
}

// generateMetadata creates metadata for the volcanic activity event
func (p *VolcanoProvider) generateMetadata(item RSSItem) map[string]string {
	metadata := map[string]string{
		"source":    "Volcano Discovery",
		"timestamp": p.parsePubDate(item.PubDate).Format(time.RFC3339),
		"title":     item.Title,
		"link":      item.Link,
	}
	
	if item.GUID != "" {
		metadata["guid"] = item.GUID
	}
	
	// Extract volcano name
	volcanoName := p.extractVolcanoName(item)
	if volcanoName != "" {
		metadata["volcano_name"] = volcanoName
	}
	
	// Extract location text
	locationText := p.extractLocationText(item)
	if locationText != "" {
		metadata["location_text"] = locationText
	}
	
	// Extract country/region
	desc := strings.ToLower(item.Description)
	countries := []string{
		"italy", "iceland", "japan", "indonesia", "philippines",
		"mexico", "guatemala", "usa", "united states", "chile",
		"peru", "ecuador", "new zealand", "papua new guinea",
	}
	
	for _, country := range countries {
		if strings.Contains(desc, country) {
			metadata["country"] = country
			break
		}
	}
	
	// Extract activity type
	activityTypes := []string{
		"eruption", "explosion", "ash emission", "lava flow",
		"seismic activity", "fumarolic activity", "thermal anomaly",
	}
	
	for _, activity := range activityTypes {
		if strings.Contains(desc, activity) {
			metadata["activity_type"] = activity
			break
		}
	}
	
	return metadata
}

// generateBadges creates badges for the volcanic activity event
func (p *VolcanoProvider) generateBadges(item RSSItem) []model.Badge {
	timestamp := p.parsePubDate(item.PubDate)
	badges := []model.Badge{
		{
			Label:     "Volcano Discovery",
			Type:      "source",
			Timestamp: timestamp,
		},
		{
			Label:     "Volcanic",
			Type:      "hazard",
			Timestamp: timestamp,
		},
	}
	
	// Add severity badge
	severity := p.determineSeverity(item)
	badges = append(badges, model.Badge{
		Label:     strings.Title(string(severity)),
		Type:      "severity",
		Timestamp: timestamp,
	})
	
	// Add volcano name badge if available
	volcanoName := p.extractVolcanoName(item)
	if volcanoName != "" {
		badges = append(badges, model.Badge{
			Label:     volcanoName,
			Type:      "volcano",
			Timestamp: timestamp,
		})
	}
	
	// Add activity type badge
	desc := strings.ToLower(item.Description)
	if strings.Contains(desc, "eruption") {
		badges = append(badges, model.Badge{
			Label:     "Eruption",
			Type:      "activity",
			Timestamp: timestamp,
		})
	} else if strings.Contains(desc, "ash") {
		badges = append(badges, model.Badge{
			Label:     "Ash Emission",
			Type:      "activity",
			Timestamp: timestamp,
		})
	} else if strings.Contains(desc, "seismic") {
		badges = append(badges, model.Badge{
			Label:     "Seismic",
			Type:      "activity",
			Timestamp: timestamp,
		})
	}
	
	// Add region badge based on location
	locationText := p.extractLocationText(item)
	if strings.Contains(strings.ToLower(locationText), "pacific") {
		badges = append(badges, model.Badge{
			Label:     "Pacific Ring",
			Type:      "region",
			Timestamp: timestamp,
		})
	}
	
	return badges
}

// cleanHTML removes HTML tags from text
func (p *VolcanoProvider) cleanHTML(text string) string {
	// Simple HTML tag removal
	text = strings.ReplaceAll(text, "<br/>", "\n")
	text = strings.ReplaceAll(text, "<br>", "\n")
	text = strings.ReplaceAll(text, "<p>", "\n")
	text = strings.ReplaceAll(text, "</p>", "\n")
	text = strings.ReplaceAll(text, "<strong>", "")
	text = strings.ReplaceAll(text, "</strong>", "")
	text = strings.ReplaceAll(text, "<em>", "")
	text = strings.ReplaceAll(text, "</em>", "")
	text = strings.ReplaceAll(text, "<b>", "")
	text = strings.ReplaceAll(text, "</b>", "")
	text = strings.ReplaceAll(text, "<i>", "")
	text = strings.ReplaceAll(text, "</i>", "")
	
	// Remove any remaining HTML tags
	for strings.Contains(text, "<") && strings.Contains(text, ">") {
		start := strings.Index(text, "<")
		end := strings.Index(text, ">")
		if end > start {
			text = text[:start] + text[end+1:]
		} else {
			break
		}
	}
	
	// Clean up whitespace
	text = strings.ReplaceAll(text, "\n\n", "\n")
	text = strings.ReplaceAll(text, "  ", " ")
	text = strings.TrimSpace(text)
	
	return text
}

// parsePubDate parses RSS pubDate string
func (p *VolcanoProvider) parsePubDate(pubDate string) time.Time {
	if pubDate == "" {
		return time.Now().UTC()
	}
	
	// Try common RSS date formats
	formats := []string{
		time.RFC1123,
		time.RFC1123Z,
		"Mon, 02 Jan 2006 15:04:05 MST",
		"Mon, 02 Jan 2006 15:04:05 -0700",
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
