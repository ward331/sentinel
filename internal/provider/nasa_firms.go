package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/openclaw/sentinel-backend/internal/model"
)

// NASAFIRMSProvider fetches fire detection data from NASA FIRMS
type NASAFIRMSProvider struct {
	client *http.Client
	config *Config
}

// NewNASAFIRMSProvider creates a new NASAFIRMSProvider
func NewNASAFIRMSProvider(config *Config) *NASAFIRMSProvider {
	return &NASAFIRMSProvider{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		config: config,
	}
}

// Fetch retrieves fire detection data from NASA FIRMS API
func (p *NASAFIRMSProvider) Fetch(ctx context.Context) ([]*model.Event, error) {
	// NASA FIRMS API for near real-time fire/hotspot data
	// Using VIIRS 375m data (most sensitive for fire detection)
	url := "https://firms.modaps.eosdis.nasa.gov/api/area/csv/eea9d7e5d9f4b36c8b8c7a1d3e2f4a5/VIIRS_SNPP_NRT/world/1"
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create NASA FIRMS request: %w", err)
	}
	
	req.Header.Set("User-Agent", "SENTINEL/2.0 (https://github.com/ward331/sentinel)")
	req.Header.Set("Accept", "text/csv, application/json")
	
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch NASA FIRMS data: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("NASA FIRMS returned status %d: %s", resp.StatusCode, string(body))
	}
	
	// Read CSV data
	csvData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read NASA FIRMS response: %w", err)
	}
	
	return p.parseCSVData(string(csvData))
}

// parseCSVData parses NASA FIRMS CSV data into events
func (p *NASAFIRMSProvider) parseCSVData(csvData string) ([]*model.Event, error) {
	lines := strings.Split(csvData, "\n")
	if len(lines) <= 1 {
		return []*model.Event{}, nil // No data
	}
	
	var events []*model.Event
	
	// Parse header
	headers := strings.Split(lines[0], ",")
	headerMap := make(map[string]int)
	for i, header := range headers {
		headerMap[strings.TrimSpace(header)] = i
	}
	
	// Parse data rows
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		
		fields := strings.Split(line, ",")
		if len(fields) < len(headers) {
			continue // Skip malformed lines
		}
		
		event, err := p.parseFireEvent(fields, headerMap)
		if err != nil {
			continue // Skip individual parsing errors
		}
		
		events = append(events, event)
	}
	
	return events, nil
}

// parseFireEvent parses a single fire detection event from CSV fields
func (p *NASAFIRMSProvider) parseFireEvent(fields []string, headerMap map[string]int) (*model.Event, error) {
	// Extract required fields
	latitude, err := strconv.ParseFloat(fields[headerMap["latitude"]], 64)
	if err != nil {
		return nil, fmt.Errorf("invalid latitude: %w", err)
	}
	
	longitude, err := strconv.ParseFloat(fields[headerMap["longitude"]], 64)
	if err != nil {
		return nil, fmt.Errorf("invalid longitude: %w", err)
	}
	
	// Parse date/time
	acqDate := fields[headerMap["acq_date"]]
	acqTime := fields[headerMap["acq_time"]]
	occurredAt, err := p.parseDateTime(acqDate, acqTime)
	if err != nil {
		occurredAt = time.Now().UTC()
	}
	
	// Extract fire properties
	brightness := 0.0
	if idx, ok := headerMap["bright_ti4"]; ok {
		if val, err := strconv.ParseFloat(fields[idx], 64); err == nil {
			brightness = val
		}
	}
	
	frp := 0.0 // Fire Radiative Power (MW)
	if idx, ok := headerMap["frp"]; ok {
		if val, err := strconv.ParseFloat(fields[idx], 64); err == nil {
			frp = val
		}
	}
	
	confidence := "low"
	if idx, ok := headerMap["confidence"]; ok {
		confVal := strings.ToLower(fields[idx])
		if strings.Contains(confVal, "high") {
			confidence = "high"
		} else if strings.Contains(confVal, "nominal") {
			confidence = "medium"
		}
	}
	
	// Generate event
	event := &model.Event{
		Title:       p.generateTitle(brightness, frp, confidence),
		Description: p.generateDescription(fields, headerMap),
		Source:      "nasa_firms",
		SourceID:    p.generateSourceID(fields, headerMap),
		OccurredAt:  occurredAt,
		Location: model.GeoJSON{
			Type:        "Point",
			Coordinates: []float64{longitude, latitude},
		},
		Precision: model.PrecisionExact,
		Magnitude: p.calculateMagnitude(brightness, frp, confidence),
		Category:  "environmental",
		Severity:  p.determineSeverity(brightness, frp),
		Metadata:  p.generateMetadata(fields, headerMap),
		Badges:    p.generateBadges(confidence, occurredAt),
	}
	
	return event, nil
}

// generateTitle creates a title for the fire detection event
func (p *NASAFIRMSProvider) generateTitle(brightness, frp float64, confidence string) string {
	var title strings.Builder
	
	// Add fire emoji
	title.WriteString("🔥 ")
	
	// Add confidence level
	switch confidence {
	case "high":
		title.WriteString("High-confidence ")
	case "medium":
		title.WriteString("Medium-confidence ")
	default:
		title.WriteString("Low-confidence ")
	}
	
	// Add fire type based on brightness
	if brightness > 400 {
		title.WriteString("Intense Wildfire")
	} else if brightness > 330 {
		title.WriteString("Wildfire")
	} else if brightness > 300 {
		title.WriteString("Fire")
	} else {
		title.WriteString("Thermal Anomaly")
	}
	
	// Add FRP if significant
	if frp > 100 {
		title.WriteString(fmt.Sprintf(" (%.0f MW)", frp))
	} else if frp > 10 {
		title.WriteString(fmt.Sprintf(" (%.1f MW)", frp))
	}
	
	return title.String()
}

// generateDescription creates a description for the fire detection event
func (p *NASAFIRMSProvider) generateDescription(fields []string, headerMap map[string]int) string {
	var builder strings.Builder
	
	// Extract key information
	latitude := fields[headerMap["latitude"]]
	longitude := fields[headerMap["longitude"]]
	acqDate := fields[headerMap["acq_date"]]
	acqTime := fields[headerMap["acq_time"]]
	
	brightness := "N/A"
	if idx, ok := headerMap["bright_ti4"]; ok {
		brightness = fields[idx]
	}
	
	frp := "N/A"
	if idx, ok := headerMap["frp"]; ok {
		frp = fields[idx]
	}
	
	confidence := "N/A"
	if idx, ok := headerMap["confidence"]; ok {
		confidence = fields[idx]
	}
	
	satellite := "VIIRS SNPP"
	if idx, ok := headerMap["satellite"]; ok {
		satellite = fields[idx]
	}
	
	// Build description
	builder.WriteString(fmt.Sprintf("NASA FIRMS Fire Detection\n\n"))
	builder.WriteString(fmt.Sprintf("Location: %s°N, %s°E\n", latitude, longitude))
	builder.WriteString(fmt.Sprintf("Detection time: %s %s UTC\n", acqDate, acqTime))
	builder.WriteString(fmt.Sprintf("Brightness temperature: %s K\n", brightness))
	builder.WriteString(fmt.Sprintf("Fire radiative power: %s MW\n", frp))
	builder.WriteString(fmt.Sprintf("Confidence: %s\n", confidence))
	builder.WriteString(fmt.Sprintf("Satellite: %s\n", satellite))
	builder.WriteString(fmt.Sprintf("Instrument: VIIRS (375m resolution)\n\n"))
	
	// Add interpretation
	brightnessVal, _ := strconv.ParseFloat(brightness, 64)
	if brightnessVal > 400 {
		builder.WriteString("⚠️ High brightness suggests intense wildfire activity.\n")
	} else if brightnessVal > 330 {
		builder.WriteString("⚠️ Moderate brightness indicates wildfire.\n")
	} else if brightnessVal > 300 {
		builder.WriteString("Thermal anomaly detected - possible fire.\n")
	} else {
		builder.WriteString("Low-temperature thermal anomaly.\n")
	}
	
	// Add source information
	builder.WriteString(fmt.Sprintf("\nSource: NASA FIRMS (Fire Information for Resource Management System)"))
	builder.WriteString(fmt.Sprintf("\nData: Near real-time VIIRS 375m active fire/hotspot data"))
	
	return builder.String()
}

// generateSourceID creates a unique source ID for the fire detection
func (p *NASAFIRMSProvider) generateSourceID(fields []string, headerMap map[string]int) string {
	// Use latitude, longitude, and acquisition time as unique identifier
	latitude := fields[headerMap["latitude"]]
	longitude := fields[headerMap["longitude"]]
	acqDate := fields[headerMap["acq_date"]]
	acqTime := fields[headerMap["acq_time"]]
	
	// Clean up values for ID
	latClean := strings.ReplaceAll(latitude, ".", "_")
	lonClean := strings.ReplaceAll(longitude, ".", "_")
	dateClean := strings.ReplaceAll(acqDate, "-", "")
	timeClean := strings.ReplaceAll(acqTime, ":", "")
	
	return fmt.Sprintf("nasa_firms_%s_%s_%s_%s", latClean, lonClean, dateClean, timeClean)
}

// parseDateTime parses NASA FIRMS date and time strings
func (p *NASAFIRMSProvider) parseDateTime(dateStr, timeStr string) (time.Time, error) {
	// Date format: YYYY-MM-DD
	// Time format: HHMM (24-hour, no colon)
	if len(timeStr) == 3 {
		timeStr = "0" + timeStr // Pad 3-digit times
	}
	
	timeLayout := "2006-01-02 1504"
	datetimeStr := fmt.Sprintf("%s %s", dateStr, timeStr)
	
	t, err := time.Parse(timeLayout, datetimeStr)
	if err != nil {
		return time.Now().UTC(), err
	}
	
	return t.UTC(), nil
}

// calculateMagnitude calculates event magnitude based on fire properties
func (p *NASAFIRMSProvider) calculateMagnitude(brightness, frp float64, confidence string) float64 {
	magnitude := 3.0 // Base for thermal anomalies
	
	// Adjust based on brightness
	if brightness > 400 {
		magnitude += 3.0
	} else if brightness > 330 {
		magnitude += 2.0
	} else if brightness > 300 {
		magnitude += 1.0
	}
	
	// Adjust based on FRP (Fire Radiative Power)
	if frp > 1000 {
		magnitude += 3.0
	} else if frp > 100 {
		magnitude += 2.0
	} else if frp > 10 {
		magnitude += 1.0
	} else if frp > 1 {
		magnitude += 0.5
	}
	
	// Adjust based on confidence
	switch confidence {
	case "high":
		magnitude += 1.0
	case "medium":
		magnitude += 0.5
	}
	
	return magnitude
}

// determineSeverity determines event severity based on fire intensity
func (p *NASAFIRMSProvider) determineSeverity(brightness, frp float64) string {
	if brightness > 400 || frp > 1000 {
		return model.SeverityCritical
	} else if brightness > 330 || frp > 100 {
		return model.SeverityHigh
	} else if brightness > 300 || frp > 10 {
		return model.SeverityMedium
	}
	return model.SeverityLow
}

// generateMetadata creates metadata for the fire detection event
func (p *NASAFIRMSProvider) generateMetadata(fields []string, headerMap map[string]int) map[string]string {
	metadata := make(map[string]string)
	
	// Copy all CSV fields to metadata
	for header, idx := range headerMap {
		if idx < len(fields) {
			metadata[header] = fields[idx]
		}
	}
	
	// Add derived metadata
	metadata["source"] = "NASA FIRMS"
	metadata["instrument"] = "VIIRS"
	metadata["resolution"] = "375m"
	metadata["data_type"] = "active_fire"
	metadata["timestamp"] = time.Now().UTC().Format(time.RFC3339)
	
	// Calculate additional derived values
	if brightness, err := strconv.ParseFloat(metadata["bright_ti4"], 64); err == nil {
		if brightness > 400 {
			metadata["intensity"] = "intense"
		} else if brightness > 330 {
			metadata["intensity"] = "high"
		} else if brightness > 300 {
			metadata["intensity"] = "medium"
		} else {
			metadata["intensity"] = "low"
		}
	}
	
	if frp, err := strconv.ParseFloat(metadata["frp"], 64); err == nil {
		if frp > 1000 {
			metadata["frp_category"] = "megafire"
		} else if frp > 100 {
			metadata["frp_category"] = "large_fire"
		} else if frp > 10 {
			metadata["frp_category"] = "medium_fire"
		} else if frp > 1 {
			metadata["frp_category"] = "small_fire"
		} else {
			metadata["frp_category"] = "thermal_anomaly"
		}
	}
	
	return metadata
}

// generateBadges creates badges for the fire detection event
func (p *NASAFIRMSProvider) generateBadges(confidence string, timestamp time.Time) []model.Badge {
	badges := []model.Badge{
		{
			Label:     "NASA FIRMS",
			Type:      "source",
			Timestamp: timestamp,
		},
		{
			Label:     "Satellite Fire Detection",
			Type:      "detection_method",
			Timestamp: timestamp,
		},
		{
			Label:     "VIIRS 375m",
			Type:      "instrument",
			Timestamp: timestamp,
		},
	}
	
	// Add confidence badge
	switch confidence {
	case "high":
		badges = append(badges, model.Badge{
			Label:     "High Confidence",
			Type:      "confidence",
			Timestamp: timestamp,
		})
	case "medium":
		badges = append(badges, model.Badge{
			Label:     "Medium Confidence",
			Type:      "confidence",
			Timestamp: timestamp,
		})
	default:
		badges = append(badges, model.Badge{
			Label:     "Low Confidence",
			Type:      "confidence",
			Timestamp: timestamp,
		})
	}
	
	return badges
}