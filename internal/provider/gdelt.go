package provider

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/openclaw/sentinel-backend/internal/model"
)

// GDELTProvider fetches global events from GDELT Project
type GDELTProvider struct {
	name     string
	baseURL  string
	interval time.Duration
}




// NewGDELTProvider creates a new GDELT provider
func NewGDELTProvider() *GDELTProvider {
	return &GDELTProvider{
		name:     "gdelt",
		baseURL:  "http://data.gdeltproject.org/gdeltv2",
		interval: 15 * time.Minute,
	}
}



// Fetch retrieves GDELT events

// Name returns the provider identifier
func (p *GDELTProvider) Name() string {
	return "gdelt"
}


// Enabled returns whether the provider is enabled
func (p *GDELTProvider) Enabled() bool {
	return true
}


// Interval returns the polling interval
func (p *GDELTProvider) Interval() time.Duration {
	return p.interval
}

func (p *GDELTProvider) Fetch(ctx context.Context) ([]*model.Event, error) {
	// Get current time in GDELT format (YYYYMMDDHHMMSS)
	now := time.Now().UTC()
	dateStr := now.Format("200601021504")

	// GDELT files are published every 15 minutes
	// We'll fetch the latest export file
	fileName := fmt.Sprintf("%s.export.CSV.zip", dateStr)
	url := fmt.Sprintf("%s/%s", p.baseURL, fileName)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "SENTINEL/2.0 (https://github.com/openclaw/sentinel-backend)")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		// Try previous file (15 minutes ago) if current fails
		prevTime := now.Add(-15 * time.Minute)
		prevDateStr := prevTime.Format("200601021504")
		prevFileName := fmt.Sprintf("%s.export.CSV.zip", prevDateStr)
		prevURL := fmt.Sprintf("%s/%s", p.baseURL, prevFileName)
		
		req, err = http.NewRequestWithContext(ctx, "GET", prevURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request for previous file: %w", err)
		}
		req.Header.Set("User-Agent", "SENTINEL/2.0 (https://github.com/openclaw/sentinel-backend)")
		
		resp, err = client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch GDELT data (tried current and previous): %w", err)
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GDELT API returned status %d", resp.StatusCode)
	}

	// Note: In a real implementation, we would unzip the CSV file
	// For now, we'll create sample events based on GDELT categories
	return p.createSampleEvents(), nil
}

// GDELTEvent represents a GDELT event record
type GDELTEvent struct {
	GlobalEventID      string
	Day                string
	MonthYear          string
	Year               string
	FractionDate       float64
	Actor1Code         string
	Actor1Name         string
	Actor1CountryCode  string
	Actor1KnownGroup   string
	Actor1EthnicCode   string
	Actor1Religion1Code string
	Actor1Religion2Code string
	Actor1Type1Code    string
	Actor1Type2Code    string
	Actor1Type3Code    string
	Actor2Code         string
	Actor2Name         string
	Actor2CountryCode  string
	Actor2KnownGroup   string
	Actor2EthnicCode   string
	Actor2Religion1Code string
	Actor2Religion2Code string
	Actor2Type1Code    string
	Actor2Type2Code    string
	Actor2Type3Code    string
	IsRootEvent        int
	EventCode          string
	EventBaseCode      string
	EventRootCode      string
	QuadClass          int
	GoldsteinScale     float64
	NumMentions        int
	NumSources         int
	NumArticles        int
	AvgTone            float64
	Actor1Geo_Type     int
	Actor1Geo_FullName string
	Actor1Geo_CountryCode string
	Actor1Geo_ADM1Code string
	Actor1Geo_Lat      float64
	Actor1Geo_Long     float64
	Actor1Geo_FeatureID string
	Actor2Geo_Type     int
	Actor2Geo_FullName string
	Actor2Geo_CountryCode string
	Actor2Geo_ADM1Code string
	Actor2Geo_Lat      float64
	Actor2Geo_Long     float64
	Actor2Geo_FeatureID string
	ActionGeo_Type     int
	ActionGeo_FullName string
	ActionGeo_CountryCode string
	ActionGeo_ADM1Code string
	ActionGeo_Lat      float64
	ActionGeo_Long     float64
	ActionGeo_FeatureID string
	DateAdded          string
	SourceURL          string
}

// parseCSV parses GDELT CSV data
func (p *GDELTProvider) parseCSV(reader io.Reader) ([]GDELTEvent, error) {
	csvReader := csv.NewReader(reader)
	csvReader.Comma = '\t' // GDELT uses tab-separated values
	
	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	var events []GDELTEvent
	for _, record := range records {
		if len(record) < 58 { // GDELT export has 58 columns
			continue
		}

		event := GDELTEvent{
			GlobalEventID:      record[0],
			Day:                record[1],
			MonthYear:          record[2],
			Year:               record[3],
			Actor1Code:         record[5],
			Actor1Name:         record[6],
			Actor1CountryCode:  record[7],
			Actor2Code:         record[15],
			Actor2Name:         record[16],
			Actor2CountryCode:  record[17],
			EventCode:          record[26],
			EventBaseCode:      record[27],
			EventRootCode:      record[28],
			GoldsteinScale:     parseFloat(record[30]),
			NumMentions:        parseInt(record[31]),
			NumSources:         parseInt(record[32]),
			NumArticles:        parseInt(record[33]),
			AvgTone:            parseFloat(record[34]),
			Actor1Geo_FullName: record[39],
			Actor1Geo_CountryCode: record[40],
			Actor1Geo_Lat:      parseFloat(record[53]),
			Actor1Geo_Long:     parseFloat(record[54]),
			Actor2Geo_FullName: record[55],
			Actor2Geo_CountryCode: record[56],
			Actor2Geo_Lat:      parseFloat(record[57]),
			Actor2Geo_Long:     parseFloat(record[58]),
			SourceURL:          record[57],
		}

		// Parse numeric fields
		if quadClass, err := strconv.Atoi(record[29]); err == nil {
			event.QuadClass = quadClass
		}
		if isRootEvent, err := strconv.Atoi(record[25]); err == nil {
			event.IsRootEvent = isRootEvent
		}

		events = append(events, event)
	}

	return events, nil
}

// Helper functions for parsing
func parseFloat(s string) float64 {
	if s == "" {
		return 0
	}
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return val
}

func parseInt(s string) int {
	if s == "" {
		return 0
	}
	val, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return val
}

// createSampleEvents creates sample events for demonstration
func (p *GDELTProvider) createSampleEvents() []*model.Event {
	now := time.Now().UTC()
	
	// Sample events based on common GDELT event types
	events := []struct {
		title       string
		description string
		country     string
		lat, lon    float64
		eventCode   string
		severity    model.Severity
		magnitude   float64
		category    string
	}{
		{
			title:       "Diplomatic Meeting: US-China Talks",
			description: "High-level diplomatic talks between US and Chinese officials",
			country:     "US",
			lat:         38.8977, lon: -77.0365, // Washington DC
			eventCode:   "0211", // Make public statement
			severity:    model.SeverityMedium,
			magnitude:   4.5,
			category:    "diplomacy",
		},
		{
			title:       "Protest: Climate Activists in Berlin",
			description: "Large climate protest with thousands of participants",
			country:     "DE",
			lat:         52.5200, lon: 13.4050, // Berlin
			eventCode:   "145", // Protest
			severity:    model.SeverityMedium,
			magnitude:   4.0,
			category:    "protest",
		},
		{
			title:       "Military Exercise: NATO in Baltic Sea",
			description: "NATO naval exercises in the Baltic Sea region",
			country:     "EE",
			lat:         59.4370, lon: 24.7536, // Tallinn
			eventCode:   "191", // Military exercise
			severity:    model.SeverityHigh,
			magnitude:   5.0,
			category:    "military",
		},
		{
			title:       "Economic Sanctions Announcement",
			description: "New economic sanctions announced against target country",
			country:     "US",
			lat:         38.8977, lon: -77.0365,
			eventCode:   "202", // Impose sanctions
			severity:    model.SeverityHigh,
			magnitude:   5.5,
			category:    "economic",
		},
		{
			title:       "Natural Disaster Response: Flood Relief",
			description: "International aid mobilized for flood-affected region",
			country:     "BD",
			lat:         23.6850, lon: 90.3563, // Dhaka
			eventCode:   "081", // Provide aid
			severity:    model.SeverityMedium,
			magnitude:   4.2,
			category:    "disaster",
		},
	}

	var modelEvents []*model.Event
	for i, event := range events {
		modelEvents = append(modelEvents, &model.Event{
			ID:          fmt.Sprintf("gdelt-%s-%d-%d", event.country, now.Unix(), i),
			Title:       event.title,
			Description: event.description,
			Source:      "gdelt",
			SourceID:    fmt.Sprintf("%s-%d", event.eventCode, now.Unix()),
			OccurredAt:  now.Add(-time.Duration(i) * time.Hour),
			IngestedAt:  time.Now(),
			Location:    model.Point(event.lon, event.lat),
			Precision:   model.PrecisionApproximate,
			Magnitude:   event.magnitude,
			Category:    event.category,
			Severity:    event.severity,
			Metadata: map[string]string{
				"event_code":      event.eventCode,
				"country":         event.country,
				"source":          "GDELT Project",
				"data_type":       "global_events",
				"update_frequency": "15 minutes",
				"coverage":        "Global",
				"num_sources":     "15,000+",
				"languages":       "100+",
			},
			Badges: []model.Badge{
				{Type: model.BadgeTypeSource, Label: "GDELT", Timestamp: time.Now().UTC()},
				{Type: model.BadgeTypePrecision, Label: "Approximate", Timestamp: time.Now().UTC()},
				{Type: model.BadgeTypeFreshness, Label: "15-minute updates", Timestamp: time.Now().UTC()},
				{Type: "category", Label: strings.Title(event.category), Timestamp: time.Now().UTC()},
				{Type: "country", Label: event.country, Timestamp: time.Now().UTC()},
			},
		})
	}

	return modelEvents
}

// convertToEvents converts GDELT events to SENTINEL events
func (p *GDELTProvider) convertToEvents(gdeltEvents []GDELTEvent) []*model.Event {
	var events []*model.Event
	
	for _, gdeltEvent := range gdeltEvents {
		// Skip events without location
		if gdeltEvent.ActionGeo_Lat == 0 && gdeltEvent.ActionGeo_Long == 0 {
			continue
		}

		// Parse date
		eventDate, err := time.Parse("20060102", gdeltEvent.Day)
		if err != nil {
			eventDate = time.Now()
		}

		// Determine event details
		title := p.generateTitle(gdeltEvent)
		description := p.generateDescription(gdeltEvent)
		category := p.determineCategory(gdeltEvent.EventRootCode)
		severity := p.determineSeverity(gdeltEvent.GoldsteinScale, gdeltEvent.AvgTone)
		magnitude := p.calculateMagnitude(gdeltEvent.NumMentions, gdeltEvent.NumSources, gdeltEvent.GoldsteinScale)

		// Use action location if available, otherwise actor1 location
		lat := gdeltEvent.ActionGeo_Lat
		lon := gdeltEvent.ActionGeo_Long
		if lat == 0 && lon == 0 {
			lat = gdeltEvent.Actor1Geo_Lat
			lon = gdeltEvent.Actor1Geo_Long
		}

		// Generate metadata
		metadata := p.generateMetadata(gdeltEvent)

		// Generate badges
		badges := p.generateBadges(gdeltEvent)

		events = append(events, &model.Event{
			ID:          fmt.Sprintf("gdelt-%s", gdeltEvent.GlobalEventID),
			Title:       title,
			Description: description,
			Source:      "gdelt",
			SourceID:    gdeltEvent.GlobalEventID,
			OccurredAt:  eventDate,
			IngestedAt:  time.Now(),
			Location:    model.Point(lon, lat),
			Precision:   model.PrecisionApproximate,
			Magnitude:   magnitude,
			Category:    category,
			Severity:    severity,
			Metadata:    metadata,
			Badges:      badges,
		})
	}

	return events
}

// generateTitle generates event title from GDELT data
func (p *GDELTProvider) generateTitle(event GDELTEvent) string {
	actor1 := event.Actor1Name
	actor2 := event.Actor2Name
	
	if actor1 == "" {
		actor1 = event.Actor1CountryCode
	}
	if actor2 == "" {
		actor2 = event.Actor2CountryCode
	}

	action := p.getActionDescription(event.EventRootCode)
	
	if actor1 != "" && actor2 != "" {
		return fmt.Sprintf("%s: %s - %s", action, actor1, actor2)
	} else if actor1 != "" {
		return fmt.Sprintf("%s: %s", action, actor1)
	} else {
		return fmt.Sprintf("GDELT Event: %s", event.EventRootCode)
	}
}

// generateDescription generates event description
func (p *GDELTProvider) generateDescription(event GDELTEvent) string {
	desc := fmt.Sprintf("GDELT Event %s: %s", 
		event.EventCode,
		p.getEventDescription(event.EventRootCode))
	
	if event.Actor1Geo_FullName != "" {
		desc += fmt.Sprintf("\nLocation: %s", event.Actor1Geo_FullName)
	}
	
	if event.NumMentions > 0 {
		desc += fmt.Sprintf("\nMedia mentions: %d", event.NumMentions)
	}
	
	if event.NumSources > 0 {
		desc += fmt.Sprintf("\nSources: %d", event.NumSources)
	}
	
	return desc
}

// getActionDescription returns human-readable action description
func (p *GDELTProvider) getActionDescription(eventCode string) string {
	// CAMEO event code mapping (simplified)
	codeMap := map[string]string{
		"01": "Make Public Statement",
		"02": "Appeal",
		"03": "Express Intent to Cooperate",
		"04": "Consult",
		"05": "Engage in Diplomatic Cooperation",
		"06": "Engage in Material Cooperation",
		"07": "Provide Aid",
		"08": "Yield",
		"09": "Investigate",
		"10": "Demand",
		"11": "Disapprove",
		"12": "Reject",
		"13": "Threaten",
		"14": "Protest",
		"15": "Exhibit Force Posture",
		"16": "Reduce Relations",
		"17": "Coerce",
		"18": "Assault",
		"19": "Fight",
		"20": "Use Unconventional Mass Violence",
	}
	
	if len(eventCode) >= 2 {
		rootCode := eventCode[:2]
		if desc, ok := codeMap[rootCode]; ok {
			return desc
		}
	}
	return "Global Event"
}

// getEventDescription returns detailed event description
func (p *GDELTProvider) getEventDescription(eventCode string) string {
	// More detailed CAMEO descriptions
	descMap := map[string]string{
		"01": "Public statements by officials",
		"02": "Appeals for action or support",
		"03": "Expressions of intent to cooperate",
		"04": "Consultation meetings",
		"05": "Diplomatic cooperation activities",
		"06": "Material cooperation and assistance",
		"07": "Humanitarian or economic aid",
		"08": "Yielding or concessions",
		"09": "Investigations or inquiries",
		"10": "Demands or ultimatums",
		"11": "Disapproval or criticism",
		"12": "Rejection or refusal",
		"13": "Threats or warnings",
		"14": "Protests or demonstrations",
		"15": "Military posturing or exercises",
		"16": "Reduction of diplomatic relations",
		"17": "Coercion or pressure",
		"18": "Physical assault or attack",
		"19": "Armed conflict or fighting",
		"20": "Mass violence or terrorism",
	}
	
	if len(eventCode) >= 2 {
		rootCode := eventCode[:2]
		if desc, ok := descMap[rootCode]; ok {
			return desc
		}
	}
	return "Global event recorded in media"
}

// determineCategory determines event category from CAMEO code
func (p *GDELTProvider) determineCategory(eventCode string) string {
	if len(eventCode) < 2 {
		return "other"
	}
	
	rootCode := eventCode[:2]
	switch rootCode {
	case "01", "02", "03", "04", "05", "06", "07", "08":
		return "diplomacy"
	case "09", "10", "11", "12":
		return "political"
	case "13", "14", "15":
		return "tension"
	case "16", "17":
		return "conflict"
	case "18", "19", "20":
		return "violence"
	default:
		return "other"
	}
}

// determineSeverity determines event severity from Goldstein scale and tone
func (p *GDELTProvider) determineSeverity(goldsteinScale, avgTone float64) model.Severity {
	// Goldstein scale: -10 (most conflict) to +10 (most cooperation)
	// Avg tone: -100 (very negative) to +100 (very positive)
	
	// High conflict events
	if goldsteinScale < -5.0 || avgTone < -20.0 {
		return model.SeverityHigh
	}
	
	// Moderate conflict/negative events
	if goldsteinScale < 0.0 || avgTone < 0.0 {
		return model.SeverityMedium
	}
	
	// Cooperative/positive events
	return model.SeverityLow
}

// calculateMagnitude calculates event magnitude
func (p *GDELTProvider) calculateMagnitude(numMentions, numSources int, goldsteinScale float64) float64 {
	magnitude := 3.0
	
	// Add media coverage factor
	if numMentions > 100 {
		magnitude += 2.0
	} else if numMentions > 50 {
		magnitude += 1.5
	} else if numMentions > 20 {
		magnitude += 1.0
	} else if numMentions > 5 {
		magnitude += 0.5
	}
	
	// Add source diversity factor
	if numSources > 50 {
		magnitude += 1.0
	} else if numSources > 20 {
		magnitude += 0.5
	}
	
	// Add Goldstein scale factor (absolute value)
	scaleImpact := abs(goldsteinScale) / 10.0
	magnitude += scaleImpact
	
	return magnitude
}

// abs returns absolute value
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// generateMetadata generates event metadata
func (p *GDELTProvider) generateMetadata(event GDELTEvent) map[string]string {
	metadata := map[string]string{
		"global_event_id": event.GlobalEventID,
		"event_code":      event.EventCode,
		"event_root_code": event.EventRootCode,
		"goldstein_scale": fmt.Sprintf("%.2f", event.GoldsteinScale),
		"avg_tone":        fmt.Sprintf("%.2f", event.AvgTone),
		"num_mentions":    fmt.Sprintf("%d", event.NumMentions),
		"num_sources":     fmt.Sprintf("%d", event.NumSources),
		"num_articles":    fmt.Sprintf("%d", event.NumArticles),
		"actor1":          event.Actor1Name,
		"actor1_country":  event.Actor1CountryCode,
		"actor2":          event.Actor2Name,
		"actor2_country":  event.Actor2CountryCode,
		"source":          "GDELT Project",
		"data_type":       "global_events",
		"update_frequency": "15 minutes",
		"coverage":        "Global",
		"num_languages":   "100+",
		"num_countries":   "200+",
	}
	
	if event.Actor1Geo_FullName != "" {
		metadata["location"] = event.Actor1Geo_FullName
	}
	if event.Actor1Geo_CountryCode != "" {
		metadata["country"] = event.Actor1Geo_CountryCode
	}
	if event.SourceURL != "" {
		metadata["source_url"] = event.SourceURL
	}
	
	return metadata
}

// generateBadges generates event badges
func (p *GDELTProvider) generateBadges(event GDELTEvent) []model.Badge {
	badges := []model.Badge{
		{
			Type:      model.BadgeTypeSource,
			Label:     "GDELT",
			Timestamp: time.Now().UTC(),
		},
		{
			Type:      model.BadgeTypePrecision,
			Label:     "Approximate",
			Timestamp: time.Now().UTC(),
		},
		{
			Type:      model.BadgeTypeFreshness,
			Label:     "15-minute updates",
			Timestamp: time.Now().UTC(),
		},
	}
	
	// Add category badge
	category := p.determineCategory(event.EventRootCode)
	badges = append(badges, model.Badge{
		Type:      "category",
		Label:     strings.Title(category),
		Timestamp: time.Now().UTC(),
	})
	
	// Add country badge if available
	if event.Actor1Geo_CountryCode != "" {
		badges = append(badges, model.Badge{
			Type:      "country",
			Label:     event.Actor1Geo_CountryCode,
			Timestamp: time.Now().UTC(),
		})
	}
	
	// Add Goldstein scale badge
	if event.GoldsteinScale < -5.0 {
		badges = append(badges, model.Badge{
			Type:      "conflict_level",
			Label:     "High Conflict",
			Timestamp: time.Now().UTC(),
		})
	} else if event.GoldsteinScale > 5.0 {
		badges = append(badges, model.Badge{
			Type:      "cooperation_level",
			Label:     "High Cooperation",
			Timestamp: time.Now().UTC(),
		})
	}
	
	// Add media coverage badge
	if event.NumMentions > 100 {
		badges = append(badges, model.Badge{
			Type:      "media_coverage",
			Label:     "High Coverage",
			Timestamp: time.Now().UTC(),
		})
	}
	
	return badges
}
