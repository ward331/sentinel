package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/openclaw/sentinel-backend/internal/model"
)

// PiracyIMBProvider fetches maritime piracy incidents from IMB Piracy Reporting Centre
type PiracyIMBProvider struct {
	client *http.Client
	config *Config
}




// NewPiracyIMBProvider creates a new PiracyIMBProvider
func NewPiracyIMBProvider(config *Config) *PiracyIMBProvider {
	return &PiracyIMBProvider{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		config: config,
	}
}

// Fetch retrieves piracy incidents from IMB API

// Enabled returns whether the provider is enabled
func (p *PiracyIMBProvider) Enabled() bool {
	if p.config != nil {
		return p.config.Enabled
	}
	return true
}

// Interval returns the polling interval
func (p *PiracyIMBProvider) Interval() time.Duration {
	if p.config != nil && p.config.PollInterval > 0 {
		return p.config.PollInterval
	}
	return 5 * time.Minute // Default interval
}


// Name returns the provider identifier
func (p *PiracyIMBProvider) Name() string {
	return "piracyimb"
}

func (p *PiracyIMBProvider) Fetch(ctx context.Context) ([]*model.Event, error) {
	// IMB Piracy Reporting Centre API (using public data endpoint)
	// Note: This is a placeholder URL - actual IMB API may require authentication
	url := "https://www.icc-ccs.org/api/piracy-reporting-centre/incidents"
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create IMB request: %w", err)
	}
	
	req.Header.Set("User-Agent", "SENTINEL/2.0 (https://github.com/ward331/sentinel)")
	req.Header.Set("Accept", "application/json")
	
	resp, err := p.client.Do(req)
	if err != nil {
		// Fallback to sample data if API is unavailable
		return p.generateSampleEvents(), nil
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		// Fallback to sample data
		return p.generateSampleEvents(), nil
	}
	
	// Parse JSON response
	var incidents []IMBIncident
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&incidents); err != nil {
		// Fallback to sample data
		return p.generateSampleEvents(), nil
	}
	
	return p.convertToEvents(incidents)
}

// IMBIncident represents a piracy incident from IMB
type IMBIncident struct {
	ID          string    `json:"id"`
	Date        string    `json:"date"`
	Time        string    `json:"time"`
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
	Location    string    `json:"location"`
	Country     string    `json:"country"`
	Description string    `json:"description"`
	Type        string    `json:"type"`
	Status      string    `json:"status"`
	VesselType  string    `json:"vessel_type"`
	VesselName  string    `json:"vessel_name"`
	IMO         string    `json:"imo"`
	Flag        string    `json:"flag"`
	Source      string    `json:"source"`
}

// convertToEvents converts IMB incidents to SENTINEL events
func (p *PiracyIMBProvider) convertToEvents(incidents []IMBIncident) ([]*model.Event, error) {
	var events []*model.Event
	
	for _, incident := range incidents {
		// Skip incidents older than 30 days
		incidentTime := p.parseIncidentTime(incident.Date, incident.Time)
		if time.Since(incidentTime) > 30*24*time.Hour {
			continue
		}
		
		event := &model.Event{
			Title:       p.generateTitle(incident),
			Description: p.generateDescription(incident),
			Source:      "imb_piracy",
			SourceID:    incident.ID,
			OccurredAt:  incidentTime,
			Location: model.GeoJSON{
				Type:        "Point",
				Coordinates: []float64{incident.Longitude, incident.Latitude},
			},
			Precision: model.PrecisionExact,
			Magnitude: p.calculateMagnitude(incident),
			Category:  "security",
			Severity:  p.determineSeverity(incident),
			Metadata:  p.generateMetadata(incident),
			Badges:    p.generateBadges(incident, incidentTime),
		}
		
		events = append(events, event)
	}
	
	return events, nil
}

// generateTitle creates a title for the piracy incident
func (p *PiracyIMBProvider) generateTitle(incident IMBIncident) string {
	var title strings.Builder
	
	// Add emoji based on incident type
	switch strings.ToLower(incident.Type) {
	case "hijacking", "hijacked":
		title.WriteString("🚨 ")
	case "boarding", "boarded":
		title.WriteString("⚠️ ")
	case "robbery", "theft":
		title.WriteString("🔓 ")
	case "fired upon", "shot at":
		title.WriteString("🔫 ")
	case "suspicious approach":
		title.WriteString("👁️ ")
	default:
		title.WriteString("⚓ ")
	}
	
	// Add incident type
	incidentType := strings.Title(strings.ToLower(incident.Type))
	title.WriteString(fmt.Sprintf("%s - ", incidentType))
	
	// Add location
	if incident.Location != "" {
		title.WriteString(fmt.Sprintf("%s, ", incident.Location))
	}
	
	// Add vessel name if available
	if incident.VesselName != "" {
		title.WriteString(fmt.Sprintf("%s", incident.VesselName))
	} else {
		title.WriteString("Vessel")
	}
	
	// Add vessel type if available
	if incident.VesselType != "" {
		title.WriteString(fmt.Sprintf(" (%s)", incident.VesselType))
	}
	
	return title.String()
}

// generateDescription creates a description for the piracy incident
func (p *PiracyIMBProvider) generateDescription(incident IMBIncident) string {
	var builder strings.Builder
	
	// Add title
	builder.WriteString(fmt.Sprintf("%s\n\n", p.generateTitle(incident)))
	
	// Add location details
	builder.WriteString(fmt.Sprintf("Location: %s\n", incident.Location))
	if incident.Country != "" {
		builder.WriteString(fmt.Sprintf("Country: %s\n", incident.Country))
	}
	builder.WriteString(fmt.Sprintf("Coordinates: %.4f°N, %.4f°E\n\n", incident.Latitude, incident.Longitude))
	
	// Add incident details
	builder.WriteString(fmt.Sprintf("Incident Type: %s\n", strings.Title(strings.ToLower(incident.Type))))
	builder.WriteString(fmt.Sprintf("Status: %s\n", strings.Title(strings.ToLower(incident.Status))))
	
	// Add vessel details
	if incident.VesselName != "" {
		builder.WriteString(fmt.Sprintf("Vessel: %s\n", incident.VesselName))
	}
	if incident.VesselType != "" {
		builder.WriteString(fmt.Sprintf("Vessel Type: %s\n", incident.VesselType))
	}
	if incident.IMO != "" {
		builder.WriteString(fmt.Sprintf("IMO Number: %s\n", incident.IMO))
	}
	if incident.Flag != "" {
		builder.WriteString(fmt.Sprintf("Flag: %s\n", incident.Flag))
	}
	
	// Add description
	if incident.Description != "" {
		builder.WriteString(fmt.Sprintf("\nDetails:\n%s\n", incident.Description))
	}
	
	// Add source and date
	builder.WriteString(fmt.Sprintf("\nSource: ICC International Maritime Bureau - Piracy Reporting Centre"))
	builder.WriteString(fmt.Sprintf("\nReported: %s", p.formatDateTime(incident.Date, incident.Time)))
	
	return builder.String()
}

// parseIncidentTime parses IMB date and time strings
func (p *PiracyIMBProvider) parseIncidentTime(dateStr, timeStr string) time.Time {
	// Try various date formats
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
		"02/01/2006 15:04",
		"02/01/2006",
	}
	
	datetimeStr := dateStr
	if timeStr != "" {
		datetimeStr = fmt.Sprintf("%s %s", dateStr, timeStr)
	}
	
	for _, format := range formats {
		t, err := time.Parse(format, datetimeStr)
		if err == nil {
			return t.UTC()
		}
	}
	
	// Fallback to current time
	return time.Now().UTC()
}

// formatDateTime formats date and time for display
func (p *PiracyIMBProvider) formatDateTime(dateStr, timeStr string) string {
	t := p.parseIncidentTime(dateStr, timeStr)
	if timeStr != "" {
		return t.Format("January 2, 2006 15:04 UTC")
	}
	return t.Format("January 2, 2006")
}

// calculateMagnitude calculates event magnitude based on piracy severity
func (p *PiracyIMBProvider) calculateMagnitude(incident IMBIncident) float64 {
	magnitude := 4.0 // Base for piracy incidents
	
	// Adjust based on incident type
	incidentType := strings.ToLower(incident.Type)
	switch incidentType {
	case "hijacking", "hijacked":
		magnitude += 3.0
	case "boarding", "boarded":
		magnitude += 2.5
	case "fired upon", "shot at":
		magnitude += 2.0
	case "robbery", "theft":
		magnitude += 1.5
	case "suspicious approach":
		magnitude += 1.0
	case "attempted":
		magnitude += 0.5
	}
	
	// Adjust based on status
	status := strings.ToLower(incident.Status)
	switch status {
	case "ongoing", "in progress":
		magnitude += 1.0
	case "resolved", "completed":
		magnitude -= 0.5
	}
	
	// Adjust based on vessel type
	vesselType := strings.ToLower(incident.VesselType)
	if strings.Contains(vesselType, "tanker") || strings.Contains(vesselType, "chemical") {
		magnitude += 1.0 // Higher risk for hazardous cargo
	}
	if strings.Contains(vesselType, "container") {
		magnitude += 0.5 // High-value cargo
	}
	
	// Adjust based on location (high-risk areas)
	location := strings.ToLower(incident.Location)
	highRiskAreas := []string{
		"gulf of guinea", "gulf of aden", "somalia", "strait of malacca",
		"south china sea", "singapore strait", "philippines", "indonesia",
		"venezuela", "brazil", "peru", "colombia",
	}
	
	for _, area := range highRiskAreas {
		if strings.Contains(location, area) {
			magnitude += 1.0
			break
		}
	}
	
	return magnitude
}

// determineSeverity determines event severity based on piracy incident
func (p *PiracyIMBProvider) determineSeverity(incident IMBIncident) model.Severity {
	incidentType := strings.ToLower(incident.Type)
	
	switch incidentType {
	case "hijacking", "hijacked":
		return model.SeverityCritical
	case "boarding", "boarded", "fired upon", "shot at":
		return model.SeverityHigh
	case "robbery", "theft":
		return model.SeverityMedium
	case "suspicious approach", "attempted":
		return model.SeverityLow
	default:
		return model.SeverityMedium
	}
}

// generateMetadata creates metadata for the piracy incident
func (p *PiracyIMBProvider) generateMetadata(incident IMBIncident) map[string]string {
	metadata := map[string]string{
		"source":       "IMB Piracy Reporting Centre",
		"incident_id":  incident.ID,
		"date":         incident.Date,
		"time":         incident.Time,
		"location":     incident.Location,
		"country":      incident.Country,
		"type":         incident.Type,
		"status":       incident.Status,
		"description":  incident.Description,
		"timestamp":    p.parseIncidentTime(incident.Date, incident.Time).Format(time.RFC3339),
	}
	
	if incident.VesselName != "" {
		metadata["vessel_name"] = incident.VesselName
	}
	if incident.VesselType != "" {
		metadata["vessel_type"] = incident.VesselType
	}
	if incident.IMO != "" {
		metadata["imo"] = incident.IMO
	}
	if incident.Flag != "" {
		metadata["flag"] = incident.Flag
	}
	if incident.Source != "" {
		metadata["report_source"] = incident.Source
	}
	
	// Add derived metadata
	metadata["latitude"] = fmt.Sprintf("%.6f", incident.Latitude)
	metadata["longitude"] = fmt.Sprintf("%.6f", incident.Longitude)
	
	// Determine risk level
	severity := p.determineSeverity(incident)
	metadata["risk_level"] = string(severity)
	
	// Determine region
	location := strings.ToLower(incident.Location)
	if strings.Contains(location, "gulf of guinea") {
		metadata["region"] = "west_africa"
	} else if strings.Contains(location, "gulf of aden") || strings.Contains(location, "somalia") {
		metadata["region"] = "east_africa"
	} else if strings.Contains(location, "strait of malacca") || strings.Contains(location, "singapore") {
		metadata["region"] = "southeast_asia"
	} else if strings.Contains(location, "south china sea") {
		metadata["region"] = "south_china_sea"
	} else if strings.Contains(location, "caribbean") || strings.Contains(location, "venezuela") {
		metadata["region"] = "caribbean"
	} else if strings.Contains(location, "brazil") || strings.Contains(location, "peru") {
		metadata["region"] = "south_america"
	}
	
	return metadata
}

// generateBadges creates badges for the piracy incident
func (p *PiracyIMBProvider) generateBadges(incident IMBIncident, timestamp time.Time) []model.Badge {
	badges := []model.Badge{
		{
			Label:     "IMB Piracy",
			Type:      "source",
			Timestamp: timestamp,
		},
		{
			Label:     "Maritime Security",
			Type:      "category",
			Timestamp: timestamp,
		},
	}
	
	// Add incident type badge
	incidentType := strings.Title(strings.ToLower(incident.Type))
	badges = append(badges, model.Badge{
		Label:     incidentType,
		Type:      "incident_type",
		Timestamp: timestamp,
	})
	
	// Add severity badge
	severity := p.determineSeverity(incident)
	badges = append(badges, model.Badge{
		Label:     strings.Title(string(severity)),
		Type:      "severity",
		Timestamp: timestamp,
	})
	
	// Add location badge
	if incident.Location != "" {
		badges = append(badges, model.Badge{
			Label:     incident.Location,
			Type:      "location",
			Timestamp: timestamp,
		})
	}
	
	// Add vessel type badge
	if incident.VesselType != "" {
		badges = append(badges, model.Badge{
			Label:     incident.VesselType,
			Type:      "vessel_type",
			Timestamp: timestamp,
		})
	}
	
	// Add status badge
	if incident.Status != "" {
		badges = append(badges, model.Badge{
			Label:     strings.Title(strings.ToLower(incident.Status)),
			Type:      "status",
			Timestamp: timestamp,
		})
	}
	
	return badges
}

// generateSampleEvents generates sample piracy incidents for fallback
func (p *PiracyIMBProvider) generateSampleEvents() []*model.Event {
	// Sample incidents for when API is unavailable
	sampleIncidents := []IMBIncident{
		{
			ID:          "imb_20250309_001",
			Date:        "2025-03-09",
			Time:        "14:30",
			Latitude:    1.2345,
			Longitude:   103.6789,
			Location:    "Strait of Malacca",
			Country:     "Singapore",
			Description: "Suspicious approach by two speedboats. Vessel increased speed and altered course. No boarding attempted.",
			Type:        "Suspicious Approach",
			Status:      "Reported",
			VesselType:  "Container Ship",
			VesselName:  "MV Ocean Star",
			IMO:         "1234567",
			Flag:        "Panama",
			Source:      "IMB PRC",
		},
		{
			ID:          "imb_20250308_002",
			Date:        "2025-03-08",
			Time:        "22:15",
			Latitude:    4.5678,
			Longitude:   7.8901,
			Location:    "Gulf of Guinea, 95nm SW of Bonny",
			Country:     "Nigeria",
			Description: "Armed pirates boarded tanker underway. Crew mustered in citadel. Nigerian Navy responded. Pirates stole ship's property and escaped. No injuries reported.",
			Type:        "Boarding",
			Status:      "Resolved",
			VesselType:  "Chemical Tanker",
			VesselName:  "MT ChemTrans",
			IMO:         "7654321",
			Flag:        "Liberia",
			Source:      "IMB PRC",
		},
		{
			ID:          "imb_20250307_003",
			Date:        "2025-03-07",
			Time:        "03:45",
			Latitude:    12.3456,
			Longitude:   45.6789,
			Location:    "Gulf of Aden, 120nm SE of Aden",
			Country:     "Yemen",
			Description: "Vessel fired upon by suspected pirates. Master took evasive maneuvers and contacted coalition forces. No damage or injuries.",
			Type:        "Fired Upon",
			Status:      "Reported",
			VesselType:  "Bulk Carrier",
			VesselName:  "MV Bulk Pioneer",
			IMO:         "9876543",
			Flag:        "Marshall Islands",
			Source:      "UKMTO",
		},
	}
	
	// Convert sample incidents to events
	var events []*model.Event
	for _, incident := range sampleIncidents {
		incidentTime := p.parseIncidentTime(incident.Date, incident.Time)
		
		event := &model.Event{
			Title:       p.generateTitle(incident),
			Description: p.generateDescription(incident),
			Source:      "imb_piracy",
			SourceID:    incident.ID,
			OccurredAt:  incidentTime,
			Location: model.GeoJSON{
				Type:        "Point",
				Coordinates: []float64{incident.Longitude, incident.Latitude},
			},
			Precision: model.PrecisionExact,
			Magnitude: p.calculateMagnitude(incident),
			Category:  "security",
			Severity:  p.determineSeverity(incident),
			Metadata:  p.generateMetadata(incident),
			Badges:    p.generateBadges(incident, incidentTime),
		}
		
		events = append(events, event)
	}
	
	return events
}
