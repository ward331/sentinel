package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/openclaw/sentinel-backend/internal/model"
)

// SWPCProvider fetches space weather data from NOAA Space Weather Prediction Center
type SWPCProvider struct {
	client *http.Client
	config *Config
}

// NewSWPCProvider creates a new SWPCProvider
func NewSWPCProvider(config *Config) *SWPCProvider {
	return &SWPCProvider{
		client: &http.Client{
			Timeout: 20 * time.Second,
		},
		config: config,
	}
}

// Fetch retrieves space weather data from SWPC
func (p *SWPCProvider) Fetch(ctx context.Context) ([]*model.Event, error) {
	var allEvents []*model.Event
	
	// Fetch from all SWPC sub-providers
	subProviders := []struct {
		name string
		url  string
		fetchFunc func(context.Context, string) ([]*model.Event, error)
	}{
		{"swpc_solar_wind", "https://services.swpc.noaa.gov/products/solar-wind/plasma-7-day.json", p.fetchSolarWind},
		{"swpc_kp_index", "https://services.swpc.noaa.gov/products/noaa-planetary-k-index.json", p.fetchKpIndex},
		{"swpc_alerts", "https://services.swpc.noaa.gov/products/alerts.json", p.fetchAlerts},
		{"swpc_goes_xray", "https://services.swpc.noaa.gov/json/goes/primary/xray-flares-latest.json", p.fetchGOESXray},
		{"swpc_forecast", "https://services.swpc.noaa.gov/text/3-day-forecast.txt", p.fetchForecast},
	}
	
	for _, sp := range subProviders {
		events, err := sp.fetchFunc(ctx, sp.url)
		if err != nil {
			fmt.Printf("Warning: Failed to fetch %s: %v\n", sp.name, err)
			continue
		}
		allEvents = append(allEvents, events...)
	}
	
	return allEvents, nil
}

// fetchSolarWind fetches solar wind plasma data
func (p *SWPCProvider) fetchSolarWind(ctx context.Context, url string) ([]*model.Event, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create solar wind request: %w", err)
	}
	
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch solar wind data: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("solar wind API returned status %d", resp.StatusCode)
	}
	
	var data [][]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode solar wind JSON: %w", err)
	}
	
	// Skip header row
	if len(data) < 2 {
		return nil, nil
	}
	
	// Get latest reading
	latest := data[len(data)-1]
	if len(latest) < 5 {
		return nil, fmt.Errorf("invalid solar wind data format")
	}
	
	// Parse values
	timestamp, _ := latest[0].(string)
	density, _ := strconv.ParseFloat(fmt.Sprint(latest[1]), 64)
	speed, _ := strconv.ParseFloat(fmt.Sprint(latest[2]), 64)
	temperature, _ := strconv.ParseFloat(fmt.Sprint(latest[4]), 64)
	
	// Create event
	event := &model.Event{
		Title:       "🌞 Solar Wind Activity",
		Description: p.generateSolarWindDescription(density, speed, temperature, timestamp),
		Source:      "swpc_solar_wind",
		SourceID:    fmt.Sprintf("solar_wind_%s", strings.ReplaceAll(timestamp, " ", "_")),
		OccurredAt:  p.parseTimestamp(timestamp),
		Location:    model.GeoJSON{Type: "Point", Coordinates: []float64{0.0, 0.0}}, // Space location
		Precision:   model.PrecisionExact,
		Magnitude:   p.calculateSolarWindMagnitude(density, speed),
		Category:    "space_weather",
		Severity:    p.determineSolarWindSeverity(speed, density),
		Metadata:    p.generateSolarWindMetadata(density, speed, temperature, timestamp),
		Badges:      p.generateSolarWindBadges(density, speed, timestamp),
	}
	
	return []*model.Event{event}, nil
}

// fetchKpIndex fetches planetary K-index data
func (p *SWPCProvider) fetchKpIndex(ctx context.Context, url string) ([]*model.Event, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kp index request: %w", err)
	}
	
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Kp index data: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Kp index API returned status %d", resp.StatusCode)
	}
	
	var data [][]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode Kp index JSON: %w", err)
	}
	
	// Skip header row
	if len(data) < 2 {
		return nil, nil
	}
	
	// Get latest reading
	latest := data[len(data)-1]
	if len(latest) < 3 {
		return nil, fmt.Errorf("invalid Kp index data format")
	}
	
	// Parse values
	timestamp, _ := latest[0].(string)
	kpStr, _ := latest[1].(string)
	kp, _ := strconv.ParseFloat(kpStr, 64)
	
	// Create event
	event := &model.Event{
		Title:       "🌌 Planetary K-Index: " + kpStr,
		Description: p.generateKpDescription(kp, timestamp),
		Source:      "swpc_kp_index",
		SourceID:    fmt.Sprintf("kp_%s", strings.ReplaceAll(timestamp, " ", "_")),
		OccurredAt:  p.parseTimestamp(timestamp),
		Location:    model.GeoJSON{Type: "Point", Coordinates: []float64{0.0, 90.0}}, // North pole for geomagnetic
		Precision:   model.PrecisionExact,
		Magnitude:   kp,
		Category:    "space_weather",
		Severity:    p.determineKpSeverity(kp),
		Metadata:    p.generateKpMetadata(kp, timestamp),
		Badges:      p.generateKpBadges(kp, timestamp),
	}
	
	return []*model.Event{event}, nil
}

// fetchAlerts fetches space weather alerts
func (p *SWPCProvider) fetchAlerts(ctx context.Context, url string) ([]*model.Event, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create alerts request: %w", err)
	}
	
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch alerts data: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("alerts API returned status %d", resp.StatusCode)
	}
	
	var alerts []SWPCAlert
	if err := json.NewDecoder(resp.Body).Decode(&alerts); err != nil {
		return nil, fmt.Errorf("failed to decode alerts JSON: %w", err)
	}
	
	var events []*model.Event
	for _, alert := range alerts {
		event := &model.Event{
			Title:       "🚨 " + alert.MessageType + ": " + alert.Category,
			Description: p.generateAlertDescription(alert),
			Source:      "swpc_alerts",
			SourceID:    alert.ID,
			OccurredAt:  p.parseTimestamp(alert.IssueTime),
			Location:    model.GeoJSON{Type: "Point", Coordinates: []float64{0.0, 0.0}},
			Precision:   model.PrecisionExact,
			Magnitude:   p.calculateAlertMagnitude(alert),
			Category:    "space_weather",
			Severity:    p.determineAlertSeverity(alert),
			Metadata:    p.generateAlertMetadata(alert),
			Badges:      p.generateAlertBadges(alert),
		}
		events = append(events, event)
	}
	
	return events, nil
}

// fetchGOESXray fetches GOES X-ray flare data
func (p *SWPCProvider) fetchGOESXray(ctx context.Context, url string) ([]*model.Event, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GOES X-ray request: %w", err)
	}
	
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch GOES X-ray data: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GOES X-ray API returned status %d", resp.StatusCode)
	}
	
	var flares []GOESXrayFlare
	if err := json.NewDecoder(resp.Body).Decode(&flares); err != nil {
		return nil, fmt.Errorf("failed to decode GOES X-ray JSON: %w", err)
	}
	
	var events []*model.Event
	for _, flare := range flares {
		// Only include significant flares
		if flare.MaxClass == "A" || flare.MaxClass == "B" {
			continue
		}
		
		event := &model.Event{
			Title:       "☀️ Solar Flare: " + flare.MaxClass + "-class",
			Description: p.generateFlareDescription(flare),
			Source:      "swpc_goes_xray",
			SourceID:    fmt.Sprintf("flare_%s_%s", flare.BeginTime, flare.MaxClass),
			OccurredAt:  p.parseTimestamp(flare.BeginTime),
			Location:    model.GeoJSON{Type: "Point", Coordinates: []float64{0.0, 0.0}},
			Precision:   model.PrecisionExact,
			Magnitude:   p.calculateFlareMagnitude(flare.MaxClass),
			Category:    "space_weather",
			Severity:    p.determineFlareSeverity(flare.MaxClass),
			Metadata:    p.generateFlareMetadata(flare),
			Badges:      p.generateFlareBadges(flare),
		}
		events = append(events, event)
	}
	
	return events, nil
}

// fetchForecast fetches 3-day space weather forecast
func (p *SWPCProvider) fetchForecast(ctx context.Context, url string) ([]*model.Event, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create forecast request: %w", err)
	}
	
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch forecast data: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("forecast API returned status %d", resp.StatusCode)
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read forecast text: %w", err)
	}
	
	forecastText := string(body)
	
	// Create event for forecast
	event := &model.Event{
		Title:       "📡 3-Day Space Weather Forecast",
		Description: p.generateForecastDescription(forecastText),
		Source:      "swpc_forecast",
		SourceID:    fmt.Sprintf("forecast_%d", time.Now().Unix()),
		OccurredAt:  time.Now().UTC(),
		Location:    model.GeoJSON{Type: "Point", Coordinates: []float64{0.0, 0.0}},
		Precision:   model.PrecisionApproximate,
		Magnitude:   5.0,
		Category:    "space_weather",
		Severity:    model.SeverityLow,
		Metadata:    p.generateForecastMetadata(forecastText),
		Badges:      p.generateForecastBadges(),
	}
	
	return []*model.Event{event}, nil
}

// Helper methods for solar wind
func (p *SWPCProvider) generateSolarWindDescription(density, speed, temperature float64, timestamp string) string {
	return fmt.Sprintf(`Solar Wind Conditions:
• Density: %.1f protons/cm³
• Speed: %.0f km/s
• Temperature: %.0f K
• Time: %s

Source: NOAA SWPC Solar Wind Plasma Monitor
Normal ranges: Density 1-10 protons/cm³, Speed 300-800 km/s`,
		density, speed, temperature, timestamp)
}

func (p *SWPCProvider) calculateSolarWindMagnitude(density, speed float64) float64 {
	magnitude := 3.0
	if speed > 600 {
		magnitude += (speed - 600) / 100
	}
	if density > 10 {
		magnitude += (density - 10) / 5
	}
	return magnitude
}

func (p *SWPCProvider) determineSolarWindSeverity(speed, density float64) string {
	if speed > 800 || density > 20 {
		return model.SeverityHigh
	}
	if speed > 600 || density > 10 {
		return model.SeverityMedium
	}
	return model.SeverityLow
}

// Helper methods for Kp index
func (p *SWPCProvider) generateKpDescription(kp float64, timestamp string) string {
	level := "Quiet"
	if kp >= 5 {
		level = "Minor Storm"
	}
	if kp >= 6 {
		level = "Moderate Storm"
	}
	if kp >= 7 {
		level = "Strong Storm"
	}
	if kp >= 8 {
		level = "Severe Storm"
	}
	if kp >= 9 {
		level = "Extreme Storm"
	}
	
	return fmt.Sprintf(`Planetary K-Index: %.1f (%s)
• Time: %s
• Scale: 0-9 (higher = more geomagnetic activity)
• Effects: Auroras, radio propagation, satellite operations

Source: NOAA SWPC Planetary K-Index`,
		kp, level, timestamp)
}

func (p *SWPCProvider) determineKpSeverity(kp float64) string {
	if kp >= 7 {
		return model.SeverityHigh
	}
	if kp >= 5 {
		return model.SeverityMedium
	}
	return model.SeverityLow
}

// Helper methods for alerts
type SWPCAlert struct {
	ID          string `json:"id"`
	IssueTime   string `json:"issueTime"`
	MessageType string `json:"messageType"`
	Category    string `json:"category"`
	Severity    string `json:"severity"`
	Source      string `json:"source"`
	Message     string `json:"message"`
}

func (p *SWPCProvider) generateAlertDescription(alert SWPCAlert) string {
	return fmt.Sprintf(`Space Weather Alert:
• Type: %s
• Category: %s
• Severity: %s
• Issue Time: %s
• Source: %s

Message: %s`,
		alert.MessageType, alert.Category, alert.Severity,
		alert.IssueTime, alert.Source, alert.Message)
}

func (p *SWPCProvider) calculateAlertMagnitude(alert SWPCAlert) float64 {
	magnitude := 5.0
	switch alert.Severity {
	case "Extreme":
		magnitude = 9.0
	case "Severe":
		magnitude = 8.0
	case "Strong":
		magnitude = 7.0
	case "Moderate":
		magnitude = 6.0
	case "Minor":
		magnitude = 5.0
	}
	return magnitude
}

func (p *SWPCProvider) determineAlertSeverity(alert SWPCAlert) string {
	switch alert.Severity {
	case "Extreme", "Severe":
		return model.SeverityCritical
	case "Strong":
		return model.SeverityHigh
	case "Moderate":
		return model.SeverityMedium
	default:
		return model.SeverityLow
	}
}

// Helper methods for GOES X-ray flares
type GOESXrayFlare struct {
	BeginTime string `json:"beginTime"`
	PeakTime  string `json:"peakTime"`
	EndTime   string `json:"endTime"`
	MaxClass  string `json:"maxClass"`
	Latitude  string `json:"latitude"`
	Longitude string `json:"longitude"`
}

func (p *SWPCProvider) generateFlareDescription(flare GOESXrayFlare) string {
	return fmt.Sprintf(`Solar X-ray Flare:
• Class: %s
• Begin: %s
• Peak: %s
• End: %s
• Location: %s° latitude, %s° longitude

Source: NOAA GOES X-ray Monitor
Flare classes: A, B, C (minor), M (moderate), X (major)`,
		flare.MaxClass, flare.BeginTime, flare.PeakTime, flare.EndTime,
		flare.Latitude, flare.Longitude)
}

func (p *SWPCProvider) calculateFlareMagnitude(class string) float64 {
	// Convert flare class to magnitude
	// A=1, B=2, C=3, M=4, X=5, with decimal for subclass
	if len(class) < 2 {
		return 3.0
	}
	
	baseClass := class[0:1]
	subclass := class[1:]
	
	magnitude := 3.0
	switch baseClass {
	case "A":
		magnitude = 1.0
	case "B":
		magnitude = 2.0
	case "C":
		magnitude = 3.0
	case "M":
		magnitude = 4.0
	case "X":
		magnitude = 5.0
	}
	
	// Add subclass (e.g., X1.5 = 5.0 + 1.5 = 6.5)
	if subclass != "" {
		if sub, err := strconv.ParseFloat(subclass, 64); err == nil {
			magnitude += sub
		}
	}
	
	return magnitude
}

func (p *SWPCProvider) determineFlareSeverity(class string) string {
	if len(class) < 1 {
		return model.SeverityLow
	}
	
	switch class[0:1] {
	case "X":
		return model.SeverityCritical
	case "M":
		return model.SeverityHigh
	case "C":
		return model.SeverityMedium
	default:
		return model.SeverityLow
	}
}

// Helper methods for forecast
func (p *SWPCProvider) generateForecastDescription(text string) string {
	// Extract first few lines for description
	lines := strings.Split(text, "\n")
	var summary strings.Builder
	
	summary.WriteString("NOAA SWPC 3-Day Space Weather Forecast\n\n")
	
	// Add first 10 lines or until empty line
	for i := 0; i < len(lines) && i < 10; i++ {
		if lines[i] == "" {
			break
		}
		summary.WriteString(lines[i] + "\n")
	}
	
	summary.WriteString("\nSource: NOAA Space Weather Prediction Center")
	
	return summary.String()
}

// Metadata generation methods
func (p *SWPCProvider) generateSolarWindMetadata(density, speed, temperature float64, timestamp string) map[string]string {
	return map[string]string{
		"source":      "NOAA SWPC Solar Wind",
		"timestamp":   timestamp,
		"density":     fmt.Sprintf("%.1f", density),
		"speed":       fmt.Sprintf("%.0f", speed),
		"temperature": fmt.Sprintf("%.0f", temperature),
		"units":       "density: protons/cm³, speed: km/s, temperature: K",
	}
}

func (p *SWPCProvider) generateKpMetadata(kp float64, timestamp string) map[string]string {
	return map[string]string{
		"source":    "NOAA SWPC Kp Index",
		"timestamp": timestamp,
		"kp_index":  fmt.Sprintf("%.1f", kp),
		"scale":     "0-9 (planetary geomagnetic activity)",
	}
}

func (p *SWPCProvider) generateAlertMetadata(alert SWPCAlert) map[string]string {
	return map[string]string{
		"id":           alert.ID,
		"source":       "NOAA SWPC Alerts",
		"issue_time":   alert.IssueTime,
		"message_type": alert.MessageType,
		"category":     alert.Category,
		"severity":     alert.Severity,
		"alert_source": alert.Source,
		"message":      alert.Message,
	}
}

func (p *SWPCProvider) generateFlareMetadata(flare GOESXrayFlare) map[string]string {
	return map[string]string{
		"source":     "NOAA GOES X-ray",
		"begin_time": flare.BeginTime,
		"peak_time":  flare.PeakTime,
		"end_time":   flare.EndTime,
		"class":      flare.MaxClass,
		"latitude":   flare.Latitude,
		"longitude":  flare.Longitude,
	}
}

func (p *SWPCProvider) generateForecastMetadata(text string) map[string]string {
	// Extract forecast period
	period := "3-day"
	re := regexp.MustCompile(`(?i)(\d+)-day`)
	if matches := re.FindStringSubmatch(text); len(matches) > 1 {
		period = matches[1] + "-day"
	}
	
	return map[string]string{
		"source":        "NOAA SWPC Forecast",
		"timestamp":     time.Now().UTC().Format(time.RFC3339),
		"forecast_type": "space_weather",
		"period":        period,
		"text_length":   fmt.Sprintf("%d", len(text)),
	}
}

// Badge generation methods
func (p *SWPCProvider) generateSolarWindBadges(density, speed float64, timestamp string) []model.Badge {
	t := p.parseTimestamp(timestamp)
	badges := []model.Badge{
		{
			Label:     "NOAA SWPC",
			Type:      "source",
			Timestamp: t,
		},
		{
			Label:     "Solar Wind",
			Type:      "space_weather",
			Timestamp: t,
		},
	}
	
	if speed > 600 {
		badges = append(badges, model.Badge{
			Label:     "High Speed",
			Type:      "condition",
			Timestamp: t,
		})
	}
	if density > 10 {
		badges = append(badges, model.Badge{
			Label:     "High Density",
			Type:      "condition",
			Timestamp: t,
		})
	}
	
	return badges
}

func (p *SWPCProvider) generateKpBadges(kp float64, timestamp string) []model.Badge {
	t := p.parseTimestamp(timestamp)
	badges := []model.Badge{
		{
			Label:     "NOAA SWPC",
			Type:      "source",
			Timestamp: t,
		},
		{
			Label:     "Kp Index",
			Type:      "geomagnetic",
			Timestamp: t,
		},
	}
	
	if kp >= 5 {
		badges = append(badges, model.Badge{
			Label:     "Geomagnetic Storm",
			Type:      "storm",
			Timestamp: t,
		})
	}
	if kp >= 7 {
		badges = append(badges, model.Badge{
			Label:     "Strong Storm",
			Type:      "intensity",
			Timestamp: t,
		})
	}
	
	return badges
}

func (p *SWPCProvider) generateAlertBadges(alert SWPCAlert) []model.Badge {
	t := p.parseTimestamp(alert.IssueTime)
	badges := []model.Badge{
		{
			Label:     "NOAA SWPC",
			Type:      "source",
			Timestamp: t,
		},
		{
			Label:     "Space Weather Alert",
			Type:      "alert",
			Timestamp: t,
		},
		{
			Label:     alert.Category,
			Type:      "category",
			Timestamp: t,
		},
	}
	
	// Add severity badge
	badges = append(badges, model.Badge{
		Label:     alert.Severity,
		Type:      "severity",
		Timestamp: t,
	})
	
	return badges
}

func (p *SWPCProvider) generateFlareBadges(flare GOESXrayFlare) []model.Badge {
	t := p.parseTimestamp(flare.BeginTime)
	badges := []model.Badge{
		{
			Label:     "NOAA GOES",
			Type:      "source",
			Timestamp: t,
		},
		{
			Label:     "Solar Flare",
			Type:      "solar",
			Timestamp: t,
		},
		{
			Label:     flare.MaxClass + "-class",
			Type:      "flare_class",
			Timestamp: t,
		},
	}
	
	// Add intensity badge
	if flare.MaxClass[0:1] == "X" {
		badges = append(badges, model.Badge{
			Label:     "Major Flare",
			Type:      "intensity",
			Timestamp: t,
		})
	} else if flare.MaxClass[0:1] == "M" {
		badges = append(badges, model.Badge{
			Label:     "Moderate Flare",
			Type:      "intensity",
			Timestamp: t,
		})
	}
	
	return badges
}

func (p *SWPCProvider) generateForecastBadges() []model.Badge {
	t := time.Now().UTC()
	return []model.Badge{
		{
			Label:     "NOAA SWPC",
			Type:      "source",
			Timestamp: t,
		},
		{
			Label:     "3-Day Forecast",
			Type:      "forecast",
			Timestamp: t,
		},
		{
			Label:     "Space Weather",
			Type:      "category",
			Timestamp: t,
		},
	}
}

// Utility methods
func (p *SWPCProvider) parseTimestamp(timestamp string) time.Time {
	if timestamp == "" {
		return time.Now().UTC()
	}
	
	// Try various timestamp formats
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		time.RFC3339,
		time.RFC1123,
	}
	
	for _, format := range formats {
		t, err := time.Parse(format, timestamp)
		if err == nil {
			return t.UTC()
		}
	}
	
	return time.Now().UTC()
}
