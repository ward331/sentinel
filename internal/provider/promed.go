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

// ProMEDProvider fetches emerging disease reports from ProMED-mail RSS
type ProMEDProvider struct {
	client *http.Client
	config *Config
}




// NewProMEDProvider creates a new ProMEDProvider
func NewProMEDProvider(config *Config) *ProMEDProvider {
	return &ProMEDProvider{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		config: config,
	}
}

// Fetch retrieves emerging disease reports from ProMED RSS feed

// Enabled returns whether the provider is enabled
func (p *ProMEDProvider) Enabled() bool {
	if p.config != nil {
		return p.config.Enabled
	}
	return true
}

// Interval returns the polling interval
func (p *ProMEDProvider) Interval() time.Duration {
	if p.config != nil && p.config.PollInterval > 0 {
		return p.config.PollInterval
	}
	return 5 * time.Minute // Default interval
}


// Name returns the provider identifier
func (p *ProMEDProvider) Name() string {
	return "promed"
}

func (p *ProMEDProvider) Fetch(ctx context.Context) ([]*model.Event, error) {
	// ProMED-mail RSS feed for emerging infectious diseases
	// Try multiple known ProMED feed URLs since they change periodically
	urls := []string{
		"https://promedmail.org/feed/",
		"https://promedmail.org/promed-posts/feed",
		"https://promedmail.org/promed-posts/feed/",
	}

	var lastErr error
	for _, url := range urls {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			lastErr = fmt.Errorf("failed to create ProMED request: %w", err)
			continue
		}

		req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; SENTINEL/2.0)")
		req.Header.Set("Accept", "application/rss+xml, application/xml, text/xml, */*")

		resp, err := p.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to fetch ProMED RSS from %s: %w", url, err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			lastErr = fmt.Errorf("ProMED RSS returned status %d from %s: %s", resp.StatusCode, url, string(body))
			continue
		}

		// Parse RSS feed
		var rss RSSFeed
		decoder := xml.NewDecoder(resp.Body)
		if err := decoder.Decode(&rss); err != nil {
			resp.Body.Close()
			lastErr = fmt.Errorf("failed to parse ProMED RSS from %s: %w", url, err)
			continue
		}
		resp.Body.Close()

		events, err := p.convertToEvents(rss)
		if err != nil {
			lastErr = err
			continue
		}
		return events, nil
	}

	// All URLs failed — return empty rather than crashing
	if lastErr != nil {
		return []*model.Event{}, nil
	}
	return []*model.Event{}, nil
}

// convertToEvents converts ProMED RSS feed to SENTINEL events
func (p *ProMEDProvider) convertToEvents(rss RSSFeed) ([]*model.Event, error) {
	var events []*model.Event
	
	for _, item := range rss.Channel.Items {
		// Skip non-disease items
		if !p.isEmergingDisease(item) {
			continue
		}
		
		event := &model.Event{
			Title:       p.generateTitle(item),
			Description: p.generateDescription(item),
			Source:      "promed",
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

// isEmergingDisease checks if an RSS item is about emerging diseases
func (p *ProMEDProvider) isEmergingDisease(item RSSItem) bool {
	title := strings.ToLower(item.Title)
	description := strings.ToLower(item.Description)
	
	// ProMED specific keywords for emerging diseases
	diseaseKeywords := []string{
		// Infectious diseases
		"virus", "bacterial", "infection", "outbreak", "cases",
		"epidemic", "pandemic", "contagious", "transmission",
		// Specific diseases
		"influenza", "flu", "covid", "sars", "mers", "ebola",
		"cholera", "malaria", "dengue", "zika", "yellow fever",
		"plague", "meningitis", "hepatitis", "tuberculosis",
		"hiv", "aids", "polio", "typhoid", "leptospirosis",
		"lassa", "marburg", "monkeypox", "mpox", "nipah",
		"rift valley", "chikungunya", "west nile", "anthrax",
		"brucellosis", "q fever", "tularemia", "salmonella",
		// Emerging threats
		"emerging", "novel", "new strain", "mutated", "variant",
		"zoonotic", "animal-to-human", "spillover",
		// Public health terms
		"public health", "health alert", "disease surveillance",
		"case report", "laboratory confirmed", "suspected cases",
	}
	
	for _, keyword := range diseaseKeywords {
		if strings.Contains(title, keyword) || strings.Contains(description, keyword) {
			return true
		}
	}
	
	// Check for ProMED-specific patterns
	if strings.Contains(title, "promed") || strings.Contains(title, "promedmail") {
		return true
	}
	
	return false
}

// generateTitle creates a title for the emerging disease event
func (p *ProMEDProvider) generateTitle(item RSSItem) string {
	title := strings.TrimSpace(item.Title)
	
	// Clean up common prefixes
	title = strings.ReplaceAll(title, "ProMED-mail post: ", "")
	title = strings.ReplaceAll(title, "ProMED-mail: ", "")
	title = strings.ReplaceAll(title, "ProMED: ", "")
	
	// Extract disease name if possible
	diseaseName := p.extractDiseaseName(item)
	if diseaseName != "" {
		return fmt.Sprintf("🦠 %s: %s", diseaseName, title)
	}
	
	return fmt.Sprintf("🦠 %s", title)
}

// generateDescription creates a description for the emerging disease event
func (p *ProMEDProvider) generateDescription(item RSSItem) string {
	var builder strings.Builder
	
	// Add title
	builder.WriteString(fmt.Sprintf("%s\n\n", item.Title))
	
	// Add cleaned description
	desc := p.cleanHTML(item.Description)
	if desc != "" {
		// Truncate if too long
		if len(desc) > 600 {
			desc = desc[:600] + "..."
		}
		builder.WriteString(fmt.Sprintf("%s\n\n", desc))
	}
	
	// Add disease information
	diseaseName := p.extractDiseaseName(item)
	if diseaseName != "" {
		builder.WriteString(fmt.Sprintf("Disease: %s\n", diseaseName))
	}
	
	// Add location information
	location := p.extractLocationText(item)
	if location != "" {
		builder.WriteString(fmt.Sprintf("Location: %s\n", location))
	}
	
	// Add case information if available
	cases := p.extractCaseCount(item)
	if cases > 0 {
		builder.WriteString(fmt.Sprintf("Reported cases: %d\n", cases))
	}
	
	// Add source and date
	builder.WriteString(fmt.Sprintf("\nSource: ProMED-mail - International Society for Infectious Diseases"))
	builder.WriteString(fmt.Sprintf("\nPublished: %s", p.formatDate(item.PubDate)))
	
	return builder.String()
}

// extractSourceID extracts a unique source ID from the RSS item
func (p *ProMEDProvider) extractSourceID(item RSSItem) string {
	if item.GUID != "" {
		return fmt.Sprintf("promed_%s", item.GUID)
	}
	
	// Use link as fallback
	if item.Link != "" {
		// Extract post ID from URL
		re := regexp.MustCompile(`p=(\d+)`)
		matches := re.FindStringSubmatch(item.Link)
		if len(matches) > 1 {
			return fmt.Sprintf("promed_%s", matches[1])
		}
	}
	
	// Generate from title hash
	titleHash := fmt.Sprintf("%d", len(item.Title))
	return fmt.Sprintf("promed_%s_%d", titleHash, time.Now().Unix())
}

// extractLocation extracts location from disease report
func (p *ProMEDProvider) extractLocation(item RSSItem) model.GeoJSON {
	// Try to extract location from title/description
	locationText := p.extractLocationText(item)
	
	if locationText != "" {
		// Try to parse as coordinates
		coordPattern := regexp.MustCompile(`(\d+\.\d+)[°\s]*[NS]?\s*[,;]\s*(\d+\.\d+)[°\s]*[EW]?`)
		matches := coordPattern.FindStringSubmatch(locationText)
		
		if len(matches) >= 3 {
			lat, err1 := strconv.ParseFloat(matches[1], 64)
			lon, err2 := strconv.ParseFloat(matches[2], 64)
			
			if err1 == nil && err2 == nil {
				// Check hemisphere indicators
				if strings.Contains(strings.ToUpper(locationText), "S") && !strings.Contains(strings.ToUpper(locationText), "N") {
					lat = -lat
				}
				if strings.Contains(strings.ToUpper(locationText), "W") && !strings.Contains(strings.ToUpper(locationText), "E") {
					lon = -lon
				}
				
				return model.GeoJSON{
					Type:        "Point",
					Coordinates: []float64{lon, lat},
				}
			}
		}
		
		// Try to extract country name
		countries := p.extractCountries(item)
		if len(countries) > 0 {
			// Use approximate country centroid
			country := strings.ToLower(countries[0])
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
			
			if coords, ok := countryCentroids[country]; ok {
				return model.GeoJSON{
					Type:        "Point",
					Coordinates: []float64{coords[0], coords[1]},
				}
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
func (p *ProMEDProvider) extractDiseaseName(item RSSItem) string {
	title := strings.ToLower(item.Title)
	description := strings.ToLower(item.Description)
	
	// Common disease names in ProMED
	diseases := []string{
		"influenza", "flu", "covid-19", "covid", "sars", "mers",
		"ebola", "cholera", "malaria", "dengue", "zika",
		"yellow fever", "plague", "meningitis", "hepatitis",
		"tuberculosis", "hiv", "aids", "polio", "typhoid",
		"leptospirosis", "lassa fever", "marburg", "monkeypox",
		"mpox", "nipah", "rift valley fever", "chikungunya",
		"west nile", "anthrax", "brucellosis", "q fever",
		"tularemia", "salmonella", "shigella", "campylobacter",
		"legionella", "mycoplasma", "rickettsia", "leishmania",
		"trypanosoma", "schistosoma", "ascaris", "taenia",
	}
	
	for _, disease := range diseases {
		if strings.Contains(title, disease) || strings.Contains(description, disease) {
			return strings.Title(disease)
		}
	}
	
	// Look for virus/bacteria names
	virusPattern := regexp.MustCompile(`([A-Z][a-z]+)\s+(virus|bacteria|infection)`)
	matches := virusPattern.FindStringSubmatch(title + " " + description)
	if len(matches) >= 2 {
		return strings.Title(matches[1] + " " + matches[2])
	}
	
	return ""
}

// extractLocationText extracts location description from report
func (p *ProMEDProvider) extractLocationText(item RSSItem) string {
	title := strings.ToLower(item.Title)
	description := strings.ToLower(item.Description)
	text := title + " " + description
	
	// Look for location patterns
	locationPatterns := []string{
		"in ", "at ", "location: ", "area: ", "region: ",
		"country: ", "state: ", "province: ", "city: ",
		"district: ", "county: ", "village: ",
	}
	
	for _, pattern := range locationPatterns {
		if idx := strings.Index(text, pattern); idx != -1 {
			locPart := text[idx+len(pattern):]
			if len(locPart) > 100 {
				locPart = locPart[:100]
			}
			// Clean up
			locPart = p.cleanHTML(locPart)
			locPart = strings.TrimSpace(locPart)
			// Take until next punctuation or keyword
			delimiters := []string{".", ";", ",", "-", " - ", "\n", " cases", " outbreak"}
			for _, delim := range delimiters {
				if delimIdx := strings.Index(locPart, delim); delimIdx != -1 {
					locPart = locPart[:delimIdx]
				}
			}
			return strings.Title(strings.TrimSpace(locPart))
		}
	}
	
	return ""
}

// extractCountries extracts affected countries from the report
func (p *ProMEDProvider) extractCountries(item RSSItem) []string {
	text := strings.ToLower(item.Title + " " + item.Description)
	
	// Common country names
	countries := []string{
		"china", "india", "united states", "usa", "brazil", "nigeria",
		"indonesia", "pakistan", "bangladesh", "russia", "mexico",
		"japan", "ethiopia", "philippines", "egypt", "vietnam",
		"congo", "iran", "turkey", "germany", "thailand", "france",
		"united kingdom", "uk", "italy", "south africa", "tanzania",
		"kenya", "colombia", "spain", "argentina", "algeria",
		"sudan", "ukraine", "iraq", "afghanistan", "poland",
		"canada", "morocco", "saudi arabia", "uzbekistan",
		"peru", "malaysia", "venezuela", "ghana", "yemen",
		"nepal", "mozambique", "madagascar", "cameroon",
		"côte d'ivoire", "north korea", "south korea", "australia",
		"new zealand", "chile", "ecuador", "bolivia", "paraguay",
		"uruguay", "guatemala", "honduras", "el salvador",
		"nicaragua", "costa rica", "panama", "cuba", "haiti",
		"dominican republic", "jamaica", "puerto rico",
	}
	
	var foundCountries []string
	for _, country := range countries {
		if strings.Contains(text, country) {
			// Standardize country name
			standardName := strings.Title(country)
			if country == "usa" || country == "united states" {
				standardName = "United States"
			} else if country == "uk" || country == "united kingdom" {
				standardName = "United Kingdom"
			}
			foundCountries = append(foundCountries, standardName)
		}
	}
	
	return foundCountries
}

// extractCaseCount extracts reported case count from the report
func (p *ProMEDProvider) extractCaseCount(item RSSItem) int {
	text := strings.ToLower(item.Title + " " + item.Description)
	
	// Look for case number patterns
	casePatterns := []*regexp.Regexp{
		regexp.MustCompile(`(\d+)\s*(cases|infections)`),
		regexp.MustCompile(`(\d+)\s*reported`),
		regexp.MustCompile(`(\d+)\s*confirmed`),
		regexp.MustCompile(`(\d+)\s*suspected`),
		regexp.MustCompile(`case count:\s*(\d+)`),
		regexp.MustCompile(`total:\s*(\d+)`),
	}
	
	for _, pattern := range casePatterns {
		matches := pattern.FindStringSubmatch(text)
		if len(matches) >= 2 {
			if cases, err := strconv.Atoi(matches[1]); err == nil {
				return cases
			}
		}
	}
	
	return 0
}

// calculateMagnitude calculates magnitude based on outbreak severity
func (p *ProMEDProvider) calculateMagnitude(item RSSItem) float64 {
	text := strings.ToLower(item.Title + " " + item.Description)
	
	magnitude := 4.0 // Base for disease reports
	
	// Increase based on outbreak type
	if strings.Contains(text, "pandemic") {
		magnitude += 3.0
	}
	if strings.Contains(text, "epidemic") {
		magnitude += 2.0
	}
	if strings.Contains(text, "outbreak") {
		magnitude += 1.0
	}
	if strings.Contains(text, "cluster") {
		magnitude += 0.5
	}
	
	// Increase based on severity indicators
	if strings.Contains(text, "severe") || strings.Contains(text, "critical") {
		magnitude += 1.5
	}
	if strings.Contains(text, "deadly") || strings.Contains(text, "fatal") {
		magnitude += 2.0
	}
	if strings.Contains(text, "death") || strings.Contains(text, "fatality") {
		magnitude += 1.0
	}
	if strings.Contains(text, "hospital") || strings.Contains(text, "icu") {
		magnitude += 0.5
	}
	
	// Increase based on novelty
	if strings.Contains(text, "novel") || strings.Contains(text, "new strain") {
		magnitude += 2.0
	}
	if strings.Contains(text, "emerging") || strings.Contains(text, "emergence") {
		magnitude += 1.5
	}
	if strings.Contains(text, "mutated") || strings.Contains(text, "variant") {
		magnitude += 1.0
	}
	
	// Increase based on transmission
	if strings.Contains(text, "human-to-human") || strings.Contains(text, "person-to-person") {
		magnitude += 1.5
	}
	if strings.Contains(text, "zoonotic") || strings.Contains(text, "animal-to-human") {
		magnitude += 1.0
	}
	if strings.Contains(text, "vector-borne") {
		magnitude += 0.5
	}
	
	// Increase based on case count
	cases := p.extractCaseCount(item)
	if cases > 1000 {
		magnitude += 2.0
	} else if cases > 100 {
		magnitude += 1.0
	} else if cases > 10 {
		magnitude += 0.5
	} else if cases > 0 {
		magnitude += 0.2
	}
	
	// Increase based on geographic spread
	countries := p.extractCountries(item)
	if len(countries) > 3 {
		magnitude += 2.0
	} else if len(countries) > 1 {
		magnitude += 1.0
	} else if len(countries) == 1 {
		magnitude += 0.5
	}
	
	return magnitude
}

// determineSeverity determines the event severity
func (p *ProMEDProvider) determineSeverity(item RSSItem) model.Severity {
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
	if strings.Contains(text, "cluster") || strings.Contains(text, "cases") {
		return model.SeverityLow
	}
	
	return model.SeverityMedium
}

// generateMetadata creates metadata for the emerging disease event
func (p *ProMEDProvider) generateMetadata(item RSSItem) map[string]string {
	metadata := map[string]string{
		"source":      "ProMED-mail",
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
	
	// Add location information
	locationText := p.extractLocationText(item)
	if locationText != "" {
		metadata["location"] = locationText
	}
	
	// Add country information
	countries := p.extractCountries(item)
	if len(countries) > 0 {
		metadata["countries"] = strings.Join(countries, ", ")
		metadata["country_count"] = fmt.Sprintf("%d", len(countries))
	}
	
	// Add case information
	cases := p.extractCaseCount(item)
	if cases > 0 {
		metadata["case_count"] = fmt.Sprintf("%d", cases)
	}
	
	// Add description (cleaned and truncated)
	desc := p.cleanHTML(item.Description)
	if desc != "" {
		if len(desc) > 1000 {
			desc = desc[:1000] + "..."
		}
		metadata["description"] = desc
	}
	
	// Extract key information from text
	text := strings.ToLower(item.Title + " " + item.Description)
	
	// Check for transmission patterns
	if strings.Contains(text, "human-to-human") {
		metadata["transmission"] = "human_to_human"
	} else if strings.Contains(text, "zoonotic") || strings.Contains(text, "animal-to-human") {
		metadata["transmission"] = "zoonotic"
	} else if strings.Contains(text, "vector-borne") {
		metadata["transmission"] = "vector_borne"
	}
	
	// Check for outbreak type
	if strings.Contains(text, "novel") || strings.Contains(text, "new strain") {
		metadata["novelty"] = "novel"
	}
	if strings.Contains(text, "emerging") {
		metadata["emerging"] = "yes"
	}
	if strings.Contains(text, "mutated") || strings.Contains(text, "variant") {
		metadata["mutation"] = "yes"
	}
	
	// Check for laboratory confirmation
	if strings.Contains(text, "laboratory confirmed") || strings.Contains(text, "lab confirmed") {
		metadata["lab_confirmed"] = "yes"
	}
	if strings.Contains(text, "suspected") {
		metadata["case_status"] = "suspected"
	} else if strings.Contains(text, "confirmed") {
		metadata["case_status"] = "confirmed"
	}
	
	return metadata
}

// generateBadges creates badges for the emerging disease event
func (p *ProMEDProvider) generateBadges(item RSSItem) []model.Badge {
	timestamp := p.parsePubDate(item.PubDate)
	badges := []model.Badge{
		{
			Label:     "ProMED",
			Type:      "source",
			Timestamp: timestamp,
		},
		{
			Label:     "Emerging Disease",
			Type:      "health",
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
	} else if strings.Contains(text, "cluster") {
		badges = append(badges, model.Badge{
			Label:     "Cluster",
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
	
	// Add transmission badge
	if strings.Contains(text, "human-to-human") {
		badges = append(badges, model.Badge{
			Label:     "Human Transmission",
			Type:      "transmission",
			Timestamp: timestamp,
		})
	} else if strings.Contains(text, "zoonotic") {
		badges = append(badges, model.Badge{
			Label:     "Zoonotic",
			Type:      "transmission",
			Timestamp: timestamp,
		})
	}
	
	// Add novelty badge if applicable
	if strings.Contains(text, "novel") || strings.Contains(text, "new strain") {
		badges = append(badges, model.Badge{
			Label:     "Novel",
			Type:      "novelty",
			Timestamp: timestamp,
		})
	}
	
	return badges
}

// cleanHTML removes HTML tags from text
func (p *ProMEDProvider) cleanHTML(text string) string {
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
func (p *ProMEDProvider) parsePubDate(pubDate string) time.Time {
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
func (p *ProMEDProvider) formatDate(dateStr string) string {
	t := p.parsePubDate(dateStr)
	return t.Format("January 2, 2006 15:04 UTC")
}
