package provider

import (
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

// WHOProvider fetches disease outbreak news from World Health Organization
type WHOProvider struct {
	client *http.Client
	config *Config
}

// Name returns the provider name
func (p *WHOProvider) Name() string {
    return "who"
}

// Interval returns the polling interval
func (p *WHOProvider) Interval() time.Duration {
    interval, _ := time.ParseDuration("1h")
    return interval
}

// Enabled returns whether the provider is enabled
func (p *WHOProvider) Enabled() bool {
    return p.config != nil && p.config.Enabled
}

// NewWHOProvider creates a new WHOProvider
func NewWHOProvider(config *Config) *WHOProvider {
	return &WHOProvider{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		config: config,
	}
}

// Fetch retrieves disease outbreak news from WHO RSS feed
func (p *WHOProvider) Fetch(ctx context.Context) ([]*model.Event, error) {
	// WHO Disease Outbreak News RSS feed
	url := "https://www.who.int/feeds/entity/csr/don/en/rss.xml"
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create WHO request: %w", err)
	}
	
	req.Header.Set("User-Agent", "SENTINEL/2.0 (https://github.com/ward331/sentinel)")
	req.Header.Set("Accept", "application/rss+xml, application/xml, text/xml")
	
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch WHO RSS: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("WHO RSS returned status %d: %s", resp.StatusCode, string(body))
	}
	
	// Parse RSS feed
	var rss RSSFeed
	decoder := xml.NewDecoder(resp.Body)
	if err := decoder.Decode(&rss); err != nil {
		return nil, fmt.Errorf("failed to parse WHO RSS: %w", err)
	}
	
	return p.convertToEvents(rss)
}

// convertToEvents converts WHO RSS feed to SENTINEL events
func (p *WHOProvider) convertToEvents(rss RSSFeed) ([]*model.Event, error) {
	var events []*model.Event
	
	for _, item := range rss.Channel.Items {
		// Skip non-outbreak items
		if !p.isDiseaseOutbreak(item) {
			continue
		}
		
		event := &model.Event{
			Title:       p.generateTitle(item),
			Description: p.generateDescription(item),
			Source:      "who_don",
			SourceID:    p.extractSourceID(item),
			OccurredAt:  p.parsePubDate(item.PubDate),
			Location:    p.extractLocation(item),
			Precision:   model.PrecisionApproximate,
			Magnitude:   p.calculateMagnitude(item),
			Category:    "health",
			Severity:    p.determineSeverity(item),
			Metadata:    p.generateMetadata(item),
			Badges:      p.generateBadges(item),
		}
		
		events = append(events, event)
	}
	
	return events, nil
}

// isDiseaseOutbreak checks if an RSS item is about disease outbreaks
func (p *WHOProvider) isDiseaseOutbreak(item RSSItem) bool {
	title := strings.ToLower(item.Title)
	description := strings.ToLower(item.Description)
	
	// WHO Disease Outbreak News specific keywords
	outbreakKeywords := []string{
		"outbreak", "disease", "epidemic", "pandemic", "virus", "bacteria",
		"infection", "contagious", "transmission", "cases", "deaths",
		"cholera", "ebola", "covid", "influenza", "measles", "malaria",
		"dengue", "zika", "yellow fever", "plague", "meningitis",
		"hepatitis", "tuberculosis", "hiv", "aids",
	}
	
	for _, keyword := range outbreakKeywords {
		if strings.Contains(title, keyword) || strings.Contains(description, keyword) {
			return true
		}
	}
	
	// Check for WHO-specific patterns
	if strings.Contains(title, "who") && (strings.Contains(title, "update") || strings.Contains(title, "alert")) {
		return true
	}
	
	return false
}

// generateTitle creates a title for the disease outbreak event
func (p *WHOProvider) generateTitle(item RSSItem) string {
	title := strings.TrimSpace(item.Title)
	
	// Clean up common prefixes
	title = strings.ReplaceAll(title, "Disease outbreak news: ", "")
	title = strings.ReplaceAll(title, "WHO Disease Outbreak News: ", "")
	title = strings.ReplaceAll(title, "WHO: ", "")
	
	// Extract disease name if possible
	diseaseName := p.extractDiseaseName(item)
	if diseaseName != "" {
		return fmt.Sprintf("🦠 %s: %s", diseaseName, title)
	}
	
	return fmt.Sprintf("🦠 %s", title)
}

// generateDescription creates a description for the disease outbreak event
func (p *WHOProvider) generateDescription(item RSSItem) string {
	var builder strings.Builder
	
	// Add title
	builder.WriteString(fmt.Sprintf("%s\n\n", item.Title))
	
	// Add cleaned description
	desc := p.cleanHTML(item.Description)
	if desc != "" {
		// Truncate if too long
		if len(desc) > 500 {
			desc = desc[:500] + "..."
		}
		builder.WriteString(fmt.Sprintf("%s\n\n", desc))
	}
	
	// Add disease information
	diseaseName := p.extractDiseaseName(item)
	if diseaseName != "" {
		builder.WriteString(fmt.Sprintf("Disease: %s\n", diseaseName))
	}
	
	// Add country information
	countries := p.extractCountries(item)
	if len(countries) > 0 {
		builder.WriteString(fmt.Sprintf("Affected countries: %s\n", strings.Join(countries, ", ")))
	}
	
	// Add source and date
	builder.WriteString(fmt.Sprintf("\nSource: World Health Organization - Disease Outbreak News"))
	builder.WriteString(fmt.Sprintf("\nPublished: %s", p.formatDate(item.PubDate)))
	
	return builder.String()
}

// extractSourceID extracts a unique source ID from the RSS item
func (p *WHOProvider) extractSourceID(item RSSItem) string {
	if item.GUID != "" {
		return fmt.Sprintf("who_%s", item.GUID)
	}
	
	// Use link as fallback
	if item.Link != "" {
		// Extract article ID from URL
		re := regexp.MustCompile(`/(\d+)/`)
		matches := re.FindStringSubmatch(item.Link)
		if len(matches) > 1 {
			return fmt.Sprintf("who_%s", matches[1])
		}
	}
	
	// Generate from title hash
	titleHash := fmt.Sprintf("%d", len(item.Title))
	return fmt.Sprintf("who_%s_%d", titleHash, time.Now().Unix())
}

// extractLocation extracts location from disease outbreak report
func (p *WHOProvider) extractLocation(item RSSItem) model.GeoJSON {
	// Try to extract country from title/description
	countries := p.extractCountries(item)
	
	if len(countries) > 0 {
		// Use first country's approximate location
		// In a real implementation, would geocode country names
		country := countries[0]
		
		// Approximate country centroids (simplified)
		countryCentroids := map[string][]float64{
			"china":       {104.1954, 35.8617},
			"india":       {78.9629, 20.5937},
			"united states": {-95.7129, 37.0902},
			"brazil":      {-51.9253, -14.2350},
			"nigeria":     {8.6753, 9.0820},
			"indonesia":   {113.9213, -0.7893},
			"pakistan":    {69.3451, 30.3753},
			"bangladesh":  {90.3563, 23.6850},
			"russia":      {105.3188, 61.5240},
			"mexico":      {-102.5528, 23.6345},
			"japan":       {138.2529, 36.2048},
			"ethiopia":    {40.4897, 9.1450},
			"philippines": {121.7740, 12.8797},
			"egypt":       {30.8025, 26.8206},
			"vietnam":     {108.2772, 14.0583},
			"congo":       {21.7587, -4.0383},
			"iran":        {53.6880, 32.4279},
			"turkey":      {35.2433, 38.9637},
			"germany":     {10.4515, 51.1657},
			"thailand":    {100.9925, 15.8700},
		}
		
		if coords, ok := countryCentroids[strings.ToLower(country)]; ok {
			return model.GeoJSON{
				Type:        "Point",
				Coordinates: []float64{coords[0], coords[1]},
			}
		}
	}
	
	// Default to world center
	return model.GeoJSON{
		Type:        "Point",
		Coordinates: []float64{0.0, 0.0},
	}
}

// extractDiseaseName extracts disease name from the report
func (p *WHOProvider) extractDiseaseName(item RSSItem) string {
	title := strings.ToLower(item.Title)
	description := strings.ToLower(item.Description)
	
	// Common disease names
	diseases := []string{
		"cholera", "ebola", "covid-19", "covid", "influenza", "flu",
		"measles", "malaria", "dengue", "zika", "yellow fever",
		"plague", "meningitis", "hepatitis", "tuberculosis", "hiv",
		"aids", "polio", "typhoid", "leptospirosis", "lassa fever",
		"marburg", "monkeypox", "mpox", "nipah", "rift valley fever",
		"chikungunya", "west nile", "anthrax", "brucellosis",
	}
	
	for _, disease := range diseases {
		if strings.Contains(title, disease) || strings.Contains(description, disease) {
			return strings.Title(disease)
		}
	}
	
	// Look for disease patterns
	diseasePatterns := []string{
		"virus", "bacterial", "infection", "outbreak of",
	}
	
	for _, pattern := range diseasePatterns {
		if idx := strings.Index(title, pattern); idx != -1 {
			// Extract word before pattern
			words := strings.Fields(title[:idx])
			if len(words) > 0 {
				return strings.Title(words[len(words)-1] + " " + pattern)
			}
		}
	}
	
	return ""
}

// extractCountries extracts affected countries from the report
func (p *WHOProvider) extractCountries(item RSSItem) []string {
	title := strings.ToLower(item.Title)
	description := strings.ToLower(item.Description)
	text := title + " " + description
	
	// Common country names (partial list)
	countries := []string{
		"china", "india", "united states", "brazil", "nigeria",
		"indonesia", "pakistan", "bangladesh", "russia", "mexico",
		"japan", "ethiopia", "philippines", "egypt", "vietnam",
		"congo", "iran", "turkey", "germany", "thailand", "france",
		"united kingdom", "italy", "south africa", "tanzania",
		"kenya", "colombia", "spain", "argentina", "algeria",
		"sudan", "ukraine", "iraq", "afghanistan", "poland",
		"canada", "morocco", "saudi arabia", "uzbekistan",
		"peru", "malaysia", "venezuela", "ghana", "yemen",
		"nepal", "mozambique", "madagascar", "cameroon",
		"côte d'ivoire", "north korea", "australia",
	}
	
	var foundCountries []string
	for _, country := range countries {
		if strings.Contains(text, country) {
			foundCountries = append(foundCountries, strings.Title(country))
		}
	}
	
	return foundCountries
}

// calculateMagnitude calculates magnitude based on outbreak severity
func (p *WHOProvider) calculateMagnitude(item RSSItem) float64 {
	title := strings.ToLower(item.Title)
	description := strings.ToLower(item.Description)
	text := title + " " + description
	
	magnitude := 4.0 // Base for disease outbreaks
	
	// Increase based on keywords
	if strings.Contains(text, "pandemic") {
		magnitude += 3.0
	}
	if strings.Contains(text, "epidemic") {
		magnitude += 2.0
	}
	if strings.Contains(text, "outbreak") {
		magnitude += 1.0
	}
	
	// Increase based on severity indicators
	if strings.Contains(text, "severe") || strings.Contains(text, "critical") {
		magnitude += 1.5
	}
	if strings.Contains(text, "deadly") || strings.Contains(text, "fatal") {
		magnitude += 2.0
	}
	if strings.Contains(text, "emergency") {
		magnitude += 1.0
	}
	
	// Increase based on scale indicators
	if strings.Contains(text, "global") || strings.Contains(text, "worldwide") {
		magnitude += 2.0
	}
	if strings.Contains(text, "regional") || strings.Contains(text, "multiple countries") {
		magnitude += 1.5
	}
	
	// Check for case/death numbers
	casePattern := regexp.MustCompile(`(\d+)\s*(cases|infections)`)
	deathPattern := regexp.MustCompile(`(\d+)\s*(deaths|fatalities)`)
	
	if matches := casePattern.FindStringSubmatch(text); len(matches) >= 2 {
		if cases, err := strconv.Atoi(matches[1]); err == nil {
			if cases > 1000 {
				magnitude += 2.0
			} else if cases > 100 {
				magnitude += 1.0
			} else if cases > 10 {
				magnitude += 0.5
			}
		}
	}
	
	if matches := deathPattern.FindStringSubmatch(text); len(matches) >= 2 {
		if deaths, err := strconv.Atoi(matches[1]); err == nil {
			if deaths > 100 {
				magnitude += 2.5
			} else if deaths > 10 {
				magnitude += 1.5
			} else if deaths > 1 {
				magnitude += 1.0
			}
		}
	}
	
	return magnitude
}

// determineSeverity determines the event severity
func (p *WHOProvider) determineSeverity(item RSSItem) string {
	text := strings.ToLower(item.Title + " " + item.Description)
	
	if strings.Contains(text, "pandemic") {
		return model.SeverityCritical
	}
	if strings.Contains(text, "epidemic") {
		return model.SeverityHigh
	}
	if strings.Contains(text, "outbreak") {
		return model.SeverityMedium
	}
	if strings.Contains(text, "cases") || strings.Contains(text, "infection") {
		return model.SeverityLow
	}
	
	return model.SeverityMedium
}

// generateMetadata creates metadata for the disease outbreak event
func (p *WHOProvider) generateMetadata(item RSSItem) map[string]string {
	metadata := map[string]string{
		"source":      "WHO Disease Outbreak News",
		"title":       item.Title,
		"link":        item.Link,
		"pub_date":    item.PubDate,
		"timestamp":   p.parsePubDate(item.PubDate).Format(time.RFC3339),
	}
	
	if item.GUID != "" {
		metadata["guid"] = item.GUID
	}
	
	// Add disease information
	diseaseName := p.extractDiseaseName(item)
	if diseaseName != "" {
		metadata["disease"] = diseaseName
	}
	
	// Add country information
	countries := p.extractCountries(item)
	if len(countries) > 0 {
		metadata["countries"] = strings.Join(countries, ", ")
		metadata["country_count"] = fmt.Sprintf("%d", len(countries))
	}
	
	// Add description (cleaned and truncated)
	desc := p.cleanHTML(item.Description)
	if desc != "" {
		if len(desc) > 1000 {
			desc = desc[:1000] + "..."
		}
		metadata["description"] = desc
	}
	
	// Extract key information
	text := strings.ToLower(item.Title + " " + item.Description)
	
	// Check for emergency declarations
	if strings.Contains(text, "public health emergency") {
		metadata["emergency_type"] = "public_health_emergency"
	}
	if strings.Contains(text, "international concern") {
		metadata["emergency_level"] = "international_concern"
	}
	
	// Check for transmission patterns
	if strings.Contains(text, "human-to-human") {
		metadata["transmission"] = "human_to_human"
	}
	if strings.Contains(text, "animal-to-human") || strings.Contains(text, "zoonotic") {
		metadata["transmission"] = "zoonotic"
	}
	if strings.Contains(text, "vector-borne") {
		metadata["transmission"] = "vector_borne"
	}
	
	// Check for vaccine/treatment availability
	if strings.Contains(text, "vaccine") {
		metadata["vaccine_available"] = "yes"
	}
	if strings.Contains(text, "treatment") || strings.Contains(text, "therapy") {
		metadata["treatment_available"] = "yes"
	}
	
	return metadata
}

// generateBadges creates badges for the disease outbreak event
func (p *WHOProvider) generateBadges(item RSSItem) []model.Badge {
	timestamp := p.parsePubDate(item.PubDate)
	badges := []model.Badge{
		{
			Label:     "WHO",
			Type:      "source",
			Timestamp: timestamp,
		},
		{
			Label:     "Disease Outbreak",
			Type:      "health",
			Timestamp: timestamp,
		},
	}
	
	// Add severity badge
	severity := p.determineSeverity(item)
	badges = append(badges, model.Badge{
		Label:     strings.Title(severity),
		Type:      "severity",
		Timestamp: timestamp,
	})
	
	// Add disease badge
	diseaseName := p.extractDiseaseName(item)
	if diseaseName != "" {
		badges = append(badges, model.Badge{
			Label:     diseaseName,
			Type:      "disease",
			Timestamp: timestamp,
		})
	}
	
	// Add outbreak type badge
	text := strings.ToLower(item.Title + " " + item.Description)
	if strings.Contains(text, "pandemic") {
		badges = append(badges, model.Badge{
			Label:     "Pandemic",
			Type:      "scale",
			Timestamp: timestamp,
		})
	} else if strings.Contains(text, "epidemic") {
		badges = append(badges, model.Badge{
			Label:     "Epidemic",
			Type:      "scale",
			Timestamp: timestamp,
		})
	} else if strings.Contains(text, "outbreak") {
		badges = append(badges, model.Badge{
			Label:     "Outbreak",
			Type:      "scale",
			Timestamp: timestamp,
		})
	}
	
	// Add country badge
	countries := p.extractCountries(item)
	if len(countries) > 0 {
		badges = append(badges, model.Badge{
			Label:     countries[0],
			Type:      "country",
			Timestamp: timestamp,
		})
	}
	
	// Add emergency badge if applicable
	if strings.Contains(text, "emergency") {
		badges = append(badges, model.Badge{
			Label:     "Emergency",
			Type:      "alert_type",
			Timestamp: timestamp,
		})
	}
	
	return badges
}

// cleanHTML removes HTML tags from text
func (p *WHOProvider) cleanHTML(text string) string {
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
func (p *WHOProvider) parsePubDate(pubDate string) time.Time {
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

// formatDate formats a date string for display
func (p *WHOProvider) formatDate(dateStr string) string {
	t := p.parsePubDate(dateStr)
	return t.Format("January 2, 2006 15:04 UTC")
}
