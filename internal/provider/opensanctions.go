package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/openclaw/sentinel-backend/internal/model"
)

// OpenSanctionsProvider fetches sanctions and PEP data from OpenSanctions
type OpenSanctionsProvider struct {
	name         string
	baseURL      string
	apiURL       string
	apiKey       string
	interval     time.Duration
	lastFetch    time.Time
	knownEntries map[string]bool
}




// NewOpenSanctionsProvider creates a new OpenSanctions provider
func NewOpenSanctionsProvider(apiKey string) *OpenSanctionsProvider {
	return &OpenSanctionsProvider{
		name:         "opensanctions",
		baseURL:      "https://data.opensanctions.org/datasets/latest/default/entities.ftm.json",
		apiURL:       "https://api.opensanctions.org/search/default",
		apiKey:       apiKey,
		interval:     24 * time.Hour, // Daily updates
		knownEntries: make(map[string]bool),
	}
}

// Name returns the provider identifier
func (p *OpenSanctionsProvider) Name() string {
	return "opensanctions"
}

// Enabled returns whether the provider is enabled
func (p *OpenSanctionsProvider) Enabled() bool {
	return true
}

// Interval returns the polling interval
func (p *OpenSanctionsProvider) Interval() time.Duration {
	return 24 * time.Hour // Sanctions data changes slowly
}

// Fetch retrieves sanctions data from OpenSanctions
func (p *OpenSanctionsProvider) Fetch(ctx context.Context) ([]*model.Event, error) {
	// Fetch bulk data (daily updates)
	events, err := p.fetchBulkData(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch bulk data: %w", err)
	}

	// Filter for new entries only
	newEvents := p.filterNewEntries(events)
	
	// Update known entries
	for _, event := range newEvents {
		p.knownEntries[event.SourceID] = true
	}

	p.lastFetch = time.Now()
	return newEvents, nil
}

// fetchBulkData fetches the complete dataset
func (p *OpenSanctionsProvider) fetchBulkData(ctx context.Context) ([]*model.Event, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "SENTINEL/1.0")
	
	// Add API key if available for higher rate limits
	if p.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("ApiKey %s", p.apiKey))
	}

	client := &http.Client{Timeout: 120 * time.Second} // Longer timeout for large dataset
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var data OpenSanctionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return p.convertToEvents(data), nil
}

// searchEntity searches for a specific entity by name
func (p *OpenSanctionsProvider) searchEntity(ctx context.Context, name string) ([]*model.Event, error) {
	searchURL := fmt.Sprintf("%s?q=%s", p.apiURL, url.QueryEscape(name))
	
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "SENTINEL/1.0")
	
	if p.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("ApiKey %s", p.apiKey))
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to search entity: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var searchResult OpenSanctionsSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&searchResult); err != nil {
		return nil, fmt.Errorf("failed to decode search result: %w", err)
	}

	return p.convertSearchResults(searchResult), nil
}

// convertToEvents converts OpenSanctions data to SENTINEL events
func (p *OpenSanctionsProvider) convertToEvents(data OpenSanctionsResponse) []*model.Event {
	events := make([]*model.Event, 0, len(data.Entities))

	for _, entity := range data.Entities {
		// Only process sanctions and PEPs
		if !p.isRelevantEntity(entity) {
			continue
		}

		event := &model.Event{
			ID:          fmt.Sprintf("opensanctions-%s", entity.ID),
			Title:       p.generateTitle(entity),
			Description: p.generateDescription(entity),
			Source:      p.name,
			SourceID:    entity.ID,
			OccurredAt:  time.Now(),
			Location:    p.extractLocation(entity),
			Precision:   model.PrecisionTextInferred,
			Magnitude:   p.calculateMagnitude(entity),
			Category:    "sanctions",
			Severity:    model.SeverityHigh, // Always high for sanctions/PEPs
			Metadata:    p.generateMetadata(entity),
			Badges:      p.generateBadges(entity),
		}

		events = append(events, event)
	}

	return events
}

// convertSearchResults converts search results to events
func (p *OpenSanctionsProvider) convertSearchResults(result OpenSanctionsSearchResult) []*model.Event {
	events := make([]*model.Event, 0, len(result.Results))

	for _, entity := range result.Results {
		if !p.isRelevantEntity(entity) {
			continue
		}

		event := &model.Event{
			ID:          fmt.Sprintf("opensanctions-search-%s", entity.ID),
			Title:       p.generateTitle(entity),
			Description: p.generateDescription(entity),
			Source:      p.name,
			SourceID:    entity.ID,
			OccurredAt:  time.Now(),
			Location:    p.extractLocation(entity),
			Precision:   model.PrecisionTextInferred,
			Magnitude:   p.calculateMagnitude(entity),
			Category:    "sanctions",
			Severity:    model.SeverityHigh,
			Metadata:    p.generateMetadata(entity),
			Badges:      p.generateBadges(entity),
		}

		events = append(events, event)
	}

	return events
}

// isRelevantEntity checks if entity is relevant for alerts
func (p *OpenSanctionsProvider) isRelevantEntity(entity OpenSanctionsEntity) bool {
	// Check if entity has sanctions or PEP designation
	for _, prop := range entity.Properties {
		if prop.Name == "topics" {
			for _, topic := range prop.Values {
				if topic == "sanction" || topic == "pep" || topic == "crime" {
					return true
				}
			}
		}
	}
	return false
}

// generateTitle generates event title
func (p *OpenSanctionsProvider) generateTitle(entity OpenSanctionsEntity) string {
	var name string
	var entityType string

	// Extract name
	for _, prop := range entity.Properties {
		if prop.Name == "name" && len(prop.Values) > 0 {
			name = prop.Values[0]
			break
		}
	}

	// Extract entity type
	for _, prop := range entity.Properties {
		if prop.Name == "schema" {
			entityType = strings.Title(prop.Values[0])
			break
		}
	}

	if name == "" {
		name = fmt.Sprintf("Entity %s", entity.ID)
	}

	// Extract sanction programs
	programs := p.extractPrograms(entity)
	if len(programs) > 0 {
		return fmt.Sprintf("%s: %s (%s)", entityType, name, strings.Join(programs, ", "))
	}

	return fmt.Sprintf("%s: %s", entityType, name)
}

// generateDescription generates event description
func (p *OpenSanctionsProvider) generateDescription(entity OpenSanctionsEntity) string {
	var desc strings.Builder
	
	desc.WriteString("Sanctions / PEP Alert\n")
	desc.WriteString("=====================\n\n")

	// Basic info
	for _, prop := range entity.Properties {
		switch prop.Name {
		case "name":
			desc.WriteString(fmt.Sprintf("Name: %s\n", strings.Join(prop.Values, "; ")))
		case "schema":
			desc.WriteString(fmt.Sprintf("Type: %s\n", strings.Title(prop.Values[0])))
		case "description":
			desc.WriteString(fmt.Sprintf("Description: %s\n", strings.Join(prop.Values, "; ")))
		case "country":
			desc.WriteString(fmt.Sprintf("Countries: %s\n", strings.Join(prop.Values, ", ")))
		}
	}

	// Sanction programs
	programs := p.extractPrograms(entity)
	if len(programs) > 0 {
		desc.WriteString(fmt.Sprintf("\nSanction Programs:\n"))
		for _, program := range programs {
			desc.WriteString(fmt.Sprintf("  • %s\n", program))
		}
	}

	// Data sources
	sources := p.extractSources(entity)
	if len(sources) > 0 {
		desc.WriteString(fmt.Sprintf("\nData Sources:\n"))
		for _, source := range sources {
			desc.WriteString(fmt.Sprintf("  • %s\n", source))
		}
	}

	// Dates
	for _, prop := range entity.Properties {
		if prop.Name == "modifiedAt" && len(prop.Values) > 0 {
			desc.WriteString(fmt.Sprintf("\nLast Updated: %s\n", prop.Values[0]))
		}
	}

	desc.WriteString("\nData Source: OpenSanctions (245+ sources)\n")
	desc.WriteString("Update Frequency: Daily\n")
	desc.WriteString("Coverage: UN, EU, UK, US, Switzerland, Australia sanctions + PEPs\n")

	return desc.String()
}

// extractLocation extracts location from entity
func (p *OpenSanctionsProvider) extractLocation(entity OpenSanctionsEntity) model.Location {
	// Try to extract country for approximate location
	for _, prop := range entity.Properties {
		if prop.Name == "country" && len(prop.Values) > 0 {
			// Use first country for approximate location
			country := prop.Values[0]
			// Return approximate coordinates for the country
			coords := p.getCountryCoordinates(country)
			if coords != nil {
				return model.Location{
					Type:        "Point",
					Coordinates: coords,
				}
			}
		}
	}

	// Default to world center if no location found
	return model.Location{
		Type:        "Point",
		Coordinates: []float64{0, 0},
	}
}

// getCountryCoordinates returns approximate coordinates for a country
func (p *OpenSanctionsProvider) getCountryCoordinates(country string) []float64 {
	// Country capital coordinates (approximate)
	countryCoords := map[string][]float64{
		"us":    {-95.7129, 37.0902},    // USA center
		"gb":    {-3.4359, 55.3781},     // UK center
		"ru":    {105.3188, 61.5240},    // Russia center
		"cn":    {104.1954, 35.8617},    // China center
		"ir":    {53.6880, 32.4279},     // Iran center
		"sy":    {38.9968, 34.8021},     // Syria center
		"kp":    {127.5101, 40.3399},    // North Korea
		"cu":    {-77.7812, 21.5218},    // Cuba
		"ve":    {-66.5897, 6.4238},     // Venezuela
		"by":    {27.9534, 53.7098},     // Belarus
	}

	country = strings.ToLower(country)
	if coords, ok := countryCoords[country]; ok {
		return coords
	}

	return nil
}

// calculateMagnitude calculates event magnitude
func (p *OpenSanctionsProvider) calculateMagnitude(entity OpenSanctionsEntity) float64 {
	magnitude := 3.0 // Base for sanctions

	// Increase for PEPs
	for _, prop := range entity.Properties {
		if prop.Name == "topics" {
			for _, topic := range prop.Values {
				if topic == "pep" {
					magnitude += 0.5
				}
				if topic == "crime" {
					magnitude += 0.8
				}
			}
		}
	}

	// Increase for multiple sanction programs
	programs := p.extractPrograms(entity)
	magnitude += float64(len(programs)) * 0.3

	// Cap magnitude
	if magnitude > 5.0 {
		magnitude = 5.0
	}

	return magnitude
}

// generateMetadata generates event metadata
func (p *OpenSanctionsProvider) generateMetadata(entity OpenSanctionsEntity) map[string]string {
	metadata := map[string]string{
		"entity_id":      entity.ID,
		"dataset":        "default",
		"data_sources":   strings.Join(p.extractSources(entity), "; "),
		"sanction_programs": strings.Join(p.extractPrograms(entity), "; "),
		"countries":      strings.Join(p.extractCountries(entity), "; "),
		"topics":         strings.Join(p.extractTopics(entity), "; "),
		"last_updated":   p.extractLastUpdated(entity),
		"data_provider":  "OpenSanctions",
		"coverage":       "245+ sources (UN, EU, UK, US, Switzerland, Australia)",
		"update_frequency": "Daily",
		"api_tier":       "Free (10k req/month)",
	}

	// Add entity properties
	for _, prop := range entity.Properties {
		if len(prop.Values) == 1 {
			metadata[prop.Name] = prop.Values[0]
		} else if len(prop.Values) > 1 {
			metadata[prop.Name] = strings.Join(prop.Values, "; ")
		}
	}

	return metadata
}

// generateBadges generates event badges
func (p *OpenSanctionsProvider) generateBadges(entity OpenSanctionsEntity) []model.Badge {
	badges := []model.Badge{
		{
			Type:      model.BadgeTypeSource,
			Label:     "OpenSanctions",
			Timestamp: time.Now().UTC(),
		},
		{
			Type:      model.BadgeTypePrecision,
			Label:     "Text Inferred",
			Timestamp: time.Now().UTC(),
		},
		{
			Type:      model.BadgeTypeFreshness,
			Label:     "Daily Update",
			Timestamp: time.Now().UTC(),
		},
	}

	// Add sanction program badges
	programs := p.extractPrograms(entity)
	for _, program := range programs {
		badges = append(badges, model.Badge{
			Type:      "sanction",
			Label:     program,
			Timestamp: time.Now().UTC(),
		})
	}

	// Add PEP badge
	for _, prop := range entity.Properties {
		if prop.Name == "topics" {
			for _, topic := range prop.Values {
				if topic == "pep" {
					badges = append(badges, model.Badge{
						Type:      "risk",
						Label:     "PEP",
						Timestamp: time.Now().UTC(),
					})
				}
			}
		}
	}

	// Add country flags
	countries := p.extractCountries(entity)
	for _, country := range countries {
		badges = append(badges, model.Badge{
			Type:      "country",
			Label:     strings.ToUpper(country),
			Timestamp: time.Now().UTC(),
		})
	}

	return badges
}

// extractPrograms extracts sanction programs
func (p *OpenSanctionsProvider) extractPrograms(entity OpenSanctionsEntity) []string {
	var programs []string
	for _, prop := range entity.Properties {
		if prop.Name == "program" {
			programs = append(programs, prop.Values...)
		}
	}
	return programs
}

// extractSources extracts data sources
func (p *OpenSanctionsProvider) extractSources(entity OpenSanctionsEntity) []string {
	var sources []string
	for _, prop := range entity.Properties {
		if prop.Name == "sourceUrl" {
			sources = append(sources, prop.Values...)
		}
	}
	return sources
}

// extractCountries extracts countries
func (p *OpenSanctionsProvider) extractCountries(entity OpenSanctionsEntity) []string {
	var countries []string
	for _, prop := range entity.Properties {
		if prop.Name == "country" {
			countries = append(countries, prop.Values...)
		}
	}
	return countries
}

// extractTopics extracts topics
func (p *OpenSanctionsProvider) extractTopics(entity OpenSanctionsEntity) []string {
	var topics []string
	for _, prop := range entity.Properties {
		if prop.Name == "topics" {
			topics = append(topics, prop.Values...)
		}
	}
	return topics
}

// extractLastUpdated extracts last updated timestamp
func (p *OpenSanctionsProvider) extractLastUpdated(entity OpenSanctionsEntity) string {
	for _, prop := range entity.Properties {
		if prop.Name == "modifiedAt" && len(prop.Values) > 0 {
			return prop.Values[0]
		}
	}
	return ""
}

// filterNewEntries filters out already known entries
func (p *OpenSanctionsProvider) filterNewEntries(events []*model.Event) []*model.Event {
	var newEvents []*model.Event
	for _, event := range events {
		if !p.knownEntries[event.SourceID] {
			newEvents = append(newEvents, event)
		}
	}
	return newEvents
}

// OpenSanctionsResponse represents the OpenSanctions bulk data response
type OpenSanctionsResponse struct {
	Entities []OpenSanctionsEntity `json:"entities"`
}

// OpenSanctionsSearchResult represents the OpenSanctions search response
type OpenSanctionsSearchResult struct {
	Results []OpenSanctionsEntity `json:"results"`
	Total   int                   `json:"total"`
}

// OpenSanctionsEntity represents a sanctions/PEP entity
type OpenSanctionsEntity struct {
	ID         string                     `json:"id"`
	Schema     string                     `json:"schema"`
	Properties []OpenSanctionsProperty    `json:"properties"`
	Datasets   []string                   `json:"datasets"`
	Referents  []string                   `json:"referents"`
	FirstSeen  string                     `json:"first_seen"`
	LastSeen   string                     `json:"last_seen"`
	Target     bool                       `json:"target"`
}

// OpenSanctionsProperty represents an entity property
type OpenSanctionsProperty struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
}
