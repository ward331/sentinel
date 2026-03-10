package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/openclaw/sentinel-backend/internal/model"
)

// ReliefWebProvider fetches disaster reports from ReliefWeb API
type ReliefWebProvider struct {
	client *http.Client
	config *Config
}

// Name returns the provider name
func (p *ReliefWebProvider) Name() string {
    return "reliefweb"
}

// Interval returns the polling interval
func (p *ReliefWebProvider) Interval() time.Duration {
    interval, _ := time.ParseDuration("1h")
    return interval
}

// Enabled returns whether the provider is enabled
func (p *ReliefWebProvider) Enabled() bool {
    return p.config != nil && p.config.Enabled
}

// NewReliefWebProvider creates a new ReliefWebProvider
func NewReliefWebProvider(config *Config) *ReliefWebProvider {
	return &ReliefWebProvider{
		client: &http.Client{
			Timeout: 20 * time.Second,
		},
		config: config,
	}
}

// Fetch retrieves disaster reports from ReliefWeb API
func (p *ReliefWebProvider) Fetch(ctx context.Context) ([]*model.Event, error) {
	// ReliefWeb API endpoint for disasters
	url := "https://api.reliefweb.int/v1/reports?appname=sentinel&profile=full&limit=50&preset=latest&filter[field]=date.created&filter[value][from]=now-7d"
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("User-Agent", "SENTINEL/2.0 (https://github.com/ward331/sentinel)")
	req.Header.Set("Accept", "application/json")
	
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ReliefWeb data: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ReliefWeb API returned status %d: %s", resp.StatusCode, string(body))
	}
	
	var apiResponse ReliefWebResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("failed to decode API response: %w", err)
	}
	
	return p.convertToEvents(apiResponse)
}

// convertToEvents converts ReliefWeb API response to SENTINEL events
func (p *ReliefWebProvider) convertToEvents(response ReliefWebResponse) ([]*model.Event, error) {
	var events []*model.Event
	
	for _, report := range response.Data {
		// Skip non-disaster reports
		if !p.isDisasterReport(report) {
			continue
		}
		
		event := &model.Event{
			Title:       p.generateTitle(report),
			Description: p.generateDescription(report),
			Source:      "reliefweb",
			SourceID:    report.ID,
			OccurredAt:  p.parseDate(report.Fields.Date.Created),
			Location:    p.extractLocation(report),
			Precision:   model.PrecisionApproximate,
			Magnitude:   p.calculateMagnitude(report),
			Category:    p.determineCategory(report),
			Severity:    p.determineSeverity(report),
			Metadata:    p.generateMetadata(report),
			Badges:      p.generateBadges(report),
		}
		
		events = append(events, event)
	}
	
	return events, nil
}

// isDisasterReport checks if a report is about a disaster
func (p *ReliefWebProvider) isDisasterReport(report ReliefWebReport) bool {
	// Check disaster-related themes
	disasterThemes := []string{
		"Natural Disaster", "Earthquake", "Flood", "Storm", "Cyclone",
		"Typhoon", "Hurricane", "Drought", "Wildfire", "Volcano",
		"Tsunami", "Landslide", "Avalanche", "Extreme Temperature",
		"Complex Emergency", "Conflict", "Epidemic", "Pandemic",
	}
	
	for _, theme := range report.Fields.Theme {
		for _, disasterTheme := range disasterThemes {
			if strings.Contains(theme.Name, disasterTheme) {
				return true
			}
		}
	}
	
	// Check disaster-related primary types
	disasterTypes := []string{
		"Situation Report", "Flash Update", "Humanitarian Bulletin",
		"Disaster Management", "Emergency Response",
	}
	
	for _, disasterType := range disasterTypes {
		if strings.Contains(report.Fields.PrimaryType, disasterType) {
			return true
		}
	}
	
	return false
}

// generateTitle creates a title for the disaster report
func (p *ReliefWebProvider) generateTitle(report ReliefWebReport) string {
	title := report.Fields.Title
	
	// Add disaster type prefix
	disasterType := p.extractDisasterType(report)
	if disasterType != "" {
		emoji := p.getDisasterEmoji(disasterType)
		return fmt.Sprintf("%s %s: %s", emoji, disasterType, title)
	}
	
	return fmt.Sprintf("⚠️ %s", title)
}

// generateDescription creates a description for the disaster report
func (p *ReliefWebProvider) generateDescription(report ReliefWebReport) string {
	var builder strings.Builder
	
	// Add title
	builder.WriteString(fmt.Sprintf("%s\n\n", report.Fields.Title))
	
	// Add body if available
	if report.Fields.Body != "" {
		// Clean and truncate body
		body := p.cleanHTML(report.Fields.Body)
		if len(body) > 500 {
			body = body[:500] + "..."
		}
		builder.WriteString(fmt.Sprintf("%s\n\n", body))
	}
	
	// Add source information
	builder.WriteString("Source: ReliefWeb - United Nations Office for the Coordination of Humanitarian Affairs\n")
	
	// Add country information
	if len(report.Fields.Country) > 0 {
		countries := []string{}
		for _, country := range report.Fields.Country {
			countries = append(countries, country.Name)
		}
		builder.WriteString(fmt.Sprintf("Countries: %s\n", strings.Join(countries, ", ")))
	}
	
	// Add disaster type
	disasterType := p.extractDisasterType(report)
	if disasterType != "" {
		builder.WriteString(fmt.Sprintf("Disaster Type: %s\n", disasterType))
	}
	
	// Add date
	builder.WriteString(fmt.Sprintf("Published: %s", p.formatDate(report.Fields.Date.Created)))
	
	return builder.String()
}

// extractLocation extracts location from disaster report
func (p *ReliefWebProvider) extractLocation(report ReliefWebReport) model.GeoJSON {
	// Try to get coordinates from country centroids
	if len(report.Fields.Country) > 0 {
		country := report.Fields.Country[0]
		if country.Location != nil && len(country.Location.Coordinates) >= 2 {
			return model.GeoJSON{
				Type:        "Point",
				Coordinates: []float64{country.Location.Coordinates[0], country.Location.Coordinates[1]},
			}
		}
	}
	
	// Try to get coordinates from primary country
	if report.Fields.PrimaryCountry != nil && report.Fields.PrimaryCountry.Location != nil {
		if len(report.Fields.PrimaryCountry.Location.Coordinates) >= 2 {
			return model.GeoJSON{
				Type:        "Point",
				Coordinates: []float64{
					report.Fields.PrimaryCountry.Location.Coordinates[0],
					report.Fields.PrimaryCountry.Location.Coordinates[1],
				},
			}
		}
	}
	
	// Default to world center
	return model.GeoJSON{
		Type:        "Point",
		Coordinates: []float64{0.0, 0.0},
	}
}

// calculateMagnitude calculates magnitude based on disaster report
func (p *ReliefWebProvider) calculateMagnitude(report ReliefWebReport) float64 {
	magnitude := 5.0 // Base for disaster reports
	
	// Increase based on disaster type
	disasterType := p.extractDisasterType(report)
	switch strings.ToLower(disasterType) {
	case "earthquake":
		magnitude += 2.0
	case "flood", "cyclone", "typhoon", "hurricane":
		magnitude += 1.5
	case "conflict", "complex emergency":
		magnitude += 2.5
	case "epidemic", "pandemic":
		magnitude += 2.0
	case "drought":
		magnitude += 1.0
	}
	
	// Increase based on number of affected countries
	if len(report.Fields.Country) > 1 {
		magnitude += float64(len(report.Fields.Country)) * 0.5
	}
	
	// Check for emergency level indicators
	title := strings.ToLower(report.Fields.Title)
	body := strings.ToLower(report.Fields.Body)
	
	if strings.Contains(title, "major") || strings.Contains(body, "major") {
		magnitude += 1.5
	}
	if strings.Contains(title, "severe") || strings.Contains(body, "severe") {
		magnitude += 2.0
	}
	if strings.Contains(title, "catastrophic") || strings.Contains(body, "catastrophic") {
		magnitude += 3.0
	}
	if strings.Contains(title, "emergency") || strings.Contains(body, "emergency") {
		magnitude += 1.0
	}
	
	return magnitude
}

// determineCategory determines the event category
func (p *ReliefWebProvider) determineCategory(report ReliefWebReport) string {
	disasterType := p.extractDisasterType(report)
	
	switch strings.ToLower(disasterType) {
	case "earthquake":
		return "earthquake"
	case "flood":
		return "flood"
	case "cyclone", "typhoon", "hurricane", "storm":
		return "storm"
	case "drought":
		return "drought"
	case "wildfire":
		return "wildfire"
	case "volcano":
		return "volcanic"
	case "tsunami":
		return "tsunami"
	case "landslide", "avalanche":
		return "landslide"
	case "conflict":
		return "conflict"
	case "epidemic", "pandemic":
		return "health"
	case "complex emergency":
		return "complex"
	default:
		return "disaster"
	}
}

// determineSeverity determines the event severity
func (p *ReliefWebProvider) determineSeverity(report ReliefWebReport) string {
	// Check for emergency level in title/body
	title := strings.ToLower(report.Fields.Title)
	body := strings.ToLower(report.Fields.Body)
	
	if strings.Contains(title, "catastrophic") || strings.Contains(body, "catastrophic") {
		return model.SeverityCritical
	}
	if strings.Contains(title, "severe") || strings.Contains(body, "severe") {
		return model.SeverityHigh
	}
	if strings.Contains(title, "major") || strings.Contains(body, "major") {
		return model.SeverityHigh
	}
	if strings.Contains(title, "emergency") || strings.Contains(body, "emergency") {
		return model.SeverityMedium
	}
	if strings.Contains(title, "alert") || strings.Contains(body, "alert") {
		return model.SeverityMedium
	}
	
	return model.SeverityLow
}

// extractDisasterType extracts the main disaster type from the report
func (p *ReliefWebProvider) extractDisasterType(report ReliefWebReport) string {
	// Check themes first
	for _, theme := range report.Fields.Theme {
		if strings.Contains(theme.Name, "Natural Disaster") {
			// Look for specific disaster types in title
			title := strings.ToLower(report.Fields.Title)
			disasterTypes := []string{
				"earthquake", "flood", "storm", "cyclone", "typhoon",
				"hurricane", "drought", "wildfire", "volcano", "tsunami",
				"landslide", "avalanche", "conflict", "epidemic", "pandemic",
			}
			
			for _, disasterType := range disasterTypes {
				if strings.Contains(title, disasterType) {
					return strings.Title(disasterType)
				}
			}
			
			return "Natural Disaster"
		}
		if strings.Contains(theme.Name, "Complex Emergency") {
			return "Complex Emergency"
		}
		if strings.Contains(theme.Name, "Conflict") {
			return "Conflict"
		}
		if strings.Contains(theme.Name, "Epidemic") {
			return "Epidemic"
		}
	}
	
	// Check primary type
	if strings.Contains(report.Fields.PrimaryType, "Situation Report") {
		return "Situation Report"
	}
	if strings.Contains(report.Fields.PrimaryType, "Flash Update") {
		return "Flash Update"
	}
	
	return "Humanitarian Report"
}

// getDisasterEmoji returns an emoji for the disaster type
func (p *ReliefWebProvider) getDisasterEmoji(disasterType string) string {
	switch strings.ToLower(disasterType) {
	case "earthquake":
		return "🌍"
	case "flood":
		return "🌊"
	case "storm", "cyclone", "typhoon", "hurricane":
		return "🌀"
	case "drought":
		return "🏜️"
	case "wildfire":
		return "🔥"
	case "volcano":
		return "🌋"
	case "tsunami":
		return "🌊"
	case "landslide", "avalanche":
		return "⛰️"
	case "conflict":
		return "⚔️"
	case "epidemic", "pandemic":
		return "🦠"
	case "complex emergency":
		return "⚠️"
	default:
		return "⚠️"
	}
}

// generateMetadata creates metadata for the disaster report
func (p *ReliefWebProvider) generateMetadata(report ReliefWebReport) map[string]string {
	metadata := map[string]string{
		"id":           report.ID,
		"title":        report.Fields.Title,
		"source":       "ReliefWeb",
		"url":          report.Fields.URL,
		"date_created": report.Fields.Date.Created,
		"date_changed": report.Fields.Date.Changed,
		"primary_type": report.Fields.PrimaryType,
	}
	
	// Add body (truncated)
	if report.Fields.Body != "" {
		body := p.cleanHTML(report.Fields.Body)
		if len(body) > 1000 {
			body = body[:1000] + "..."
		}
		metadata["body"] = body
	}
	
	// Add countries
	if len(report.Fields.Country) > 0 {
		countries := []string{}
		for _, country := range report.Fields.Country {
			countries = append(countries, country.Name)
		}
		metadata["countries"] = strings.Join(countries, ", ")
	}
	
	// Add themes
	if len(report.Fields.Theme) > 0 {
		themes := []string{}
		for _, theme := range report.Fields.Theme {
			themes = append(themes, theme.Name)
		}
		metadata["themes"] = strings.Join(themes, ", ")
	}
	
	// Add disaster type
	disasterType := p.extractDisasterType(report)
	if disasterType != "" {
		metadata["disaster_type"] = disasterType
	}
	
	// Add source organization
	if len(report.Fields.Source) > 0 {
		sources := []string{}
		for _, source := range report.Fields.Source {
			sources = append(sources, source.Name)
		}
		metadata["sources"] = strings.Join(sources, ", ")
	}
	
	return metadata
}

// generateBadges creates badges for the disaster report
func (p *ReliefWebProvider) generateBadges(report ReliefWebReport) []model.Badge {
	timestamp := p.parseDate(report.Fields.Date.Created)
	badges := []model.Badge{
		{
			Label:     "ReliefWeb",
			Type:      "source",
			Timestamp: timestamp,
		},
		{
			Label:     "UN OCHA",
			Type:      "organization",
			Timestamp: timestamp,
		},
	}
	
	// Add disaster type badge
	disasterType := p.extractDisasterType(report)
	if disasterType != "" {
		badges = append(badges, model.Badge{
			Label:     disasterType,
			Type:      "disaster_type",
			Timestamp: timestamp,
		})
	}
	
	// Add severity badge
	severity := p.determineSeverity(report)
	badges = append(badges, model.Badge{
		Label:     strings.Title(severity),
		Type:      "severity",
		Timestamp: timestamp,
	})
	
	// Add country badge
	if len(report.Fields.Country) > 0 {
		country := report.Fields.Country[0].Name
		badges = append(badges, model.Badge{
			Label:     country,
			Type:      "country",
			Timestamp: timestamp,
		})
	}
	
	// Add report type badge
	if strings.Contains(report.Fields.PrimaryType, "Situation Report") {
		badges = append(badges, model.Badge{
			Label:     "SitRep",
			Type:      "report_type",
			Timestamp: timestamp,
		})
	} else if strings.Contains(report.Fields.PrimaryType, "Flash Update") {
		badges = append(badges, model.Badge{
			Label:     "Flash Update",
			Type:      "report_type",
			Timestamp: timestamp,
		})
	}
	
	return badges
}

//
