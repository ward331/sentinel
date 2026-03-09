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

// NOAACAPProvider fetches NOAA Common Alerting Protocol alerts
type NOAACAPProvider struct {
	name     string
	baseURL  string
	interval time.Duration
}

// NewNOAACAPProvider creates a new NOAA CAP provider
func NewNOAACAPProvider() *NOAACAPProvider {
	return &NOAACAPProvider{
		name:     "noaa_cap",
		baseURL:  "https://alerts.weather.gov/cap/us.php?x=0",
		interval: 5 * time.Minute,
	}
}

// Name returns the provider name
func (p *NOAACAPProvider) Name() string {
	return p.name
}

// Interval returns the polling interval
func (p *NOAACAPProvider) Interval() time.Duration {
	return p.interval
}

// Fetch retrieves NOAA CAP alerts
func (p *NOAACAPProvider) Fetch(ctx context.Context) ([]*model.Event, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "SENTINEL/2.0 (https://github.com/openclaw/sentinel-backend)")
	req.Header.Set("Accept", "application/cap+xml")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch NOAA CAP data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("NOAA CAP API returned status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return p.parseCAP(data)
}

// CAP structures
type CAPFeed struct {
	XMLName xml.Name `xml:"feed"`
	Entries []CAPEntry `xml:"entry"`
}

type CAPEntry struct {
	ID        string    `xml:"id"`
	Title     string    `xml:"title"`
	Updated   string    `xml:"updated"`
	Published string    `xml:"published"`
	Summary   string    `xml:"summary"`
	Link      CAPLink   `xml:"link"`
	Category  []string  `xml:"category"`
	CapAlert  CAPAlert  `xml:"alert"`
}

type CAPLink struct {
	Href string `xml:"href,attr"`
}

type CAPAlert struct {
	Identifier  string    `xml:"identifier"`
	Sender      string    `xml:"sender"`
	Sent        string    `xml:"sent"`
	Status      string    `xml:"status"`
	MsgType     string    `xml:"msgType"`
	Source      string    `xml:"source"`
	Scope       string    `xml:"scope"`
	Restriction string    `xml:"restriction"`
	Addresses   string    `xml:"addresses"`
	Code        []string  `xml:"code"`
	Note        string    `xml:"note"`
	References  string    `xml:"references"`
	Incidents   string    `xml:"incidents"`
	Info        []CAPInfo `xml:"info"`
}

type CAPInfo struct {
	Language     string        `xml:"language"`
	Category     []string      `xml:"category"`
	Event        string        `xml:"event"`
	ResponseType []string      `xml:"responseType"`
	Urgency      string        `xml:"urgency"`
	Severity     string        `xml:"severity"`
	Certainty    string        `xml:"certainty"`
	EventCode    []CAPEventCode `xml:"eventCode"`
	Effective    string        `xml:"effective"`
	Onset        string        `xml:"onset"`
	Expires      string        `xml:"expires"`
	SenderName   string        `xml:"senderName"`
	Headline     string        `xml:"headline"`
	Description  string        `xml:"description"`
	Instruction  string        `xml:"instruction"`
	Web          string        `xml:"web"`
	Contact      string        `xml:"contact"`
	Parameter    []CAPParameter `xml:"parameter"`
	Area         []CAPArea     `xml:"area"`
}

type CAPEventCode struct {
	ValueName string `xml:"valueName,attr"`
	Value     string `xml:"value,attr"`
}

type CAPParameter struct {
	ValueName string `xml:"valueName,attr"`
	Value     string `xml:"value,attr"`
}

type CAPArea struct {
	AreaDesc string   `xml:"areaDesc"`
	Polygon  []string `xml:"polygon"`
	Circle   []string `xml:"circle"`
	Geocode  []CAPGeocode `xml:"geocode"`
}

type CAPGeocode struct {
	ValueName string `xml:"valueName,attr"`
	Value     string `xml:"value,attr"`
}

// parseCAP parses CAP XML data
func (p *NOAACAPProvider) parseCAP(data []byte) ([]*model.Event, error) {
	var feed CAPFeed
	if err := xml.Unmarshal(data, &feed); err != nil {
		return nil, fmt.Errorf("failed to parse CAP XML: %w", err)
	}

	var events []*model.Event
	for _, entry := range feed.Entries {
		if len(entry.CapAlert.Info) == 0 {
			continue
		}

		info := entry.CapAlert.Info[0]
		event := p.convertToEvent(entry, info)
		if event != nil {
			events = append(events, event)
		}
	}

	return events, nil
}

// convertToEvent converts CAP alert to SENTINEL event
func (p *NOAACAPProvider) convertToEvent(entry CAPEntry, info CAPInfo) *model.Event {
	// Parse timestamps
	sentTime, err := time.Parse(time.RFC3339, entry.CapAlert.Sent)
	if err != nil {
		sentTime = time.Now()
	}

	effectiveTime := sentTime
	if info.Effective != "" {
		if t, err := time.Parse(time.RFC3339, info.Effective); err == nil {
			effectiveTime = t
		}
	}

	// Determine severity from CAP fields
	severity := p.determineSeverity(info.Severity, info.Urgency, info.Certainty)
	magnitude := p.calculateMagnitude(info.Severity, info.Urgency, info.Certainty)

	// Extract location from area description
	location := model.Point(-98.5795, 39.8283) // Default to US center
	if len(info.Area) > 0 && info.Area[0].AreaDesc != "" {
		// In a real implementation, we would geocode the area description
		// For now, use default US location
	}

	// Clean description
	description := strings.TrimSpace(info.Description)
	if description == "" {
		description = info.Headline
	}
	if description == "" {
		description = entry.Title
	}

	// Generate metadata
	metadata := p.generateMetadata(entry, info)

	// Generate badges
	badges := p.generateBadges(info)

	return &model.Event{
		ID:          fmt.Sprintf("noaa-cap-%s", entry.CapAlert.Identifier),
		Title:       info.Headline,
		Description: description,
		Source:      "noaa_cap",
		SourceID:    entry.CapAlert.Identifier,
		OccurredAt:  effectiveTime,
		IngestedAt:  time.Now(),
		Location:    location,
		Precision:   model.PrecisionTextInferred,
		Magnitude:   magnitude,
		Category:    p.determineCategory(info.Category),
		Severity:    severity,
		Metadata:    metadata,
		Badges:      badges,
	}
}

// determineSeverity converts CAP severity to SENTINEL severity
func (p *NOAACAPProvider) determineSeverity(capSeverity, urgency, certainty string) model.Severity {
	// CAP severity: Extreme, Severe, Moderate, Minor, Unknown
	// Urgency: Immediate, Expected, Future, Past, Unknown
	// Certainty: Observed, Likely, Possible, Unlikely, Unknown

	severityMap := map[string]model.Severity{
		"Extreme":  model.SeverityCritical,
		"Severe":   model.SeverityHigh,
		"Moderate": model.SeverityMedium,
		"Minor":    model.SeverityLow,
		"Unknown":  model.SeverityLow,
	}

	if sev, ok := severityMap[capSeverity]; ok {
		return sev
	}
	return model.SeverityMedium
}

// calculateMagnitude calculates event magnitude
func (p *NOAACAPProvider) calculateMagnitude(severity, urgency, certainty string) float64 {
	magnitude := 3.0

	// Add severity factor
	switch severity {
	case "Extreme":
		magnitude += 2.0
	case "Severe":
		magnitude += 1.5
	case "Moderate":
		magnitude += 1.0
	case "Minor":
		magnitude += 0.5
	}

	// Add urgency factor
	switch urgency {
	case "Immediate":
		magnitude += 1.0
	case "Expected":
		magnitude += 0.5
	}

	// Add certainty factor
	switch certainty {
	case "Observed":
		magnitude += 1.0
	case "Likely":
		magnitude += 0.5
	}

	return magnitude
}

// determineCategory determines event category
func (p *NOAACAPProvider) determineCategory(categories []string) string {
	if len(categories) == 0 {
		return "weather"
	}

	// CAP categories: Met, Geo, Safety, Security, Rescue, Fire, Health, Env, Transport, Infra, CBRNE, Other
	categoryMap := map[string]string{
		"Met":      "weather",
		"Geo":      "geological",
		"Safety":   "safety",
		"Security": "security",
		"Rescue":   "rescue",
		"Fire":     "fire",
		"Health":   "health",
		"Env":      "environment",
		"Transport": "transport",
		"Infra":    "infrastructure",
		"CBRNE":    "hazardous",
		"Other":    "other",
	}

	for _, cat := range categories {
		if mapped, ok := categoryMap[cat]; ok {
			return mapped
		}
	}

	return "weather"
}

// generateMetadata generates event metadata
func (p *NOAACAPProvider) generateMetadata(entry CAPEntry, info CAPInfo) map[string]string {
	metadata := map[string]string{
		"alert_id":      entry.CapAlert.Identifier,
		"sender":        entry.CapAlert.Sender,
		"status":        entry.CapAlert.Status,
		"msg_type":      entry.CapAlert.MsgType,
		"scope":         entry.CapAlert.Scope,
		"event":         info.Event,
		"urgency":       info.Urgency,
		"severity":      info.Severity,
		"certainty":     info.Certainty,
		"headline":      info.Headline,
		"source":        "NOAA CAP",
		"data_type":     "weather_alert",
		"update_frequency": "5 minutes",
		"coverage":      "United States",
	}

	// Add area information
	if len(info.Area) > 0 {
		metadata["area_description"] = info.Area[0].AreaDesc
	}

	// Add response types
	if len(info.ResponseType) > 0 {
		metadata["response_types"] = strings.Join(info.ResponseType, "; ")
	}

	// Add instructions if available
	if info.Instruction != "" {
		metadata["instructions"] = info.Instruction
	}

	// Add contact if available
	if info.Contact != "" {
		metadata["contact"] = info.Contact
	}

	// Add web link if available
	if info.Web != "" {
		metadata["web_link"] = info.Web
	}

	return metadata
}

// generateBadges generates event badges
func (p *NOAACAPProvider) generateBadges(info CAPInfo) []model.Badge {
	badges := []model.Badge{
		{
			Type:      model.BadgeTypeSource,
			Label:     "NOAA CAP",
			Timestamp: time.Now().UTC(),
		},
		{
			Type:      model.BadgeTypePrecision,
			Label:     "Text Inferred",
			Timestamp: time.Now().UTC(),
		},
		{
			Type:      model.BadgeTypeFreshness,
			Label:     "Real-time",
			Timestamp: time.Now().UTC(),
		},
	}

	// Add severity badge
	if info.Severity != "" {
		badges = append(badges, model.Badge{
			Type:      "severity",
			Label:     info.Severity,
			Timestamp: time.Now().UTC(),
		})
	}

	// Add urgency badge
	if info.Urgency != "" {
		badges = append(badges, model.Badge{
			Type:      "urgency",
			Label:     info.Urgency,
			Timestamp: time.Now().UTC(),
		})
	}

	// Add category badges
	for _, category := range info.Category {
		badges = append(badges, model.Badge{
			Type:      "category",
			Label:     category,
			Timestamp: time.Now().UTC(),
		})
	}

	return badges
}