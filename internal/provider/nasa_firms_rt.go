package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/openclaw/sentinel-backend/internal/model"
)

// NASAFIRMSRTProvider fetches real-time fire detection data using a NASA FIRMS MAP key
// Tier 1: Free with MAP key (requires registration)
// Category: wildfire
// Signup: https://firms.modaps.eosdis.nasa.gov/map/#d:24hrs
type NASAFIRMSRTProvider struct {
	client *http.Client
	config *Config
}

// NewNASAFIRMSRTProvider creates a new NASAFIRMSRTProvider
func NewNASAFIRMSRTProvider(config *Config) *NASAFIRMSRTProvider {
	return &NASAFIRMSRTProvider{
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		config: config,
	}
}

// Name returns the provider identifier
func (p *NASAFIRMSRTProvider) Name() string {
	return "nasa_firms_rt"
}

// Enabled returns whether the provider is enabled (requires MAP key)
func (p *NASAFIRMSRTProvider) Enabled() bool {
	if p.config == nil || p.config.APIKey == "" {
		return false
	}
	return p.config.Enabled
}

// Interval returns the polling interval
func (p *NASAFIRMSRTProvider) Interval() time.Duration {
	if p.config != nil && p.config.PollInterval > 0 {
		return p.config.PollInterval
	}
	return 600 * time.Second
}

// Fetch retrieves real-time fire data from NASA FIRMS using VIIRS and MODIS
func (p *NASAFIRMSRTProvider) Fetch(ctx context.Context) ([]*model.Event, error) {
	var allEvents []*model.Event

	// Fetch both VIIRS and MODIS data
	sources := []struct {
		name    string
		dataset string
	}{
		{"VIIRS_SNPP_NRT", "VIIRS_SNPP_NRT"},
		{"MODIS_NRT", "MODIS_NRT"},
	}

	for _, src := range sources {
		url := fmt.Sprintf("https://firms.modaps.eosdis.nasa.gov/api/area/csv/%s/%s/world/1",
			p.config.APIKey, src.dataset)

		events, err := p.fetchCSV(ctx, url, src.name)
		if err != nil {
			fmt.Printf("Warning: NASA FIRMS RT %s fetch failed: %v\n", src.name, err)
			continue
		}
		allEvents = append(allEvents, events...)
	}

	return allEvents, nil
}

func (p *NASAFIRMSRTProvider) fetchCSV(ctx context.Context, url, source string) ([]*model.Event, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create NASA FIRMS RT request: %w", err)
	}
	req.Header.Set("User-Agent", "SENTINEL/3.0")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch NASA FIRMS RT data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("NASA FIRMS RT returned status %d: %s", resp.StatusCode, string(body))
	}

	csvData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read NASA FIRMS RT response: %w", err)
	}

	return p.parseCSV(string(csvData), source)
}

func (p *NASAFIRMSRTProvider) parseCSV(csvData, source string) ([]*model.Event, error) {
	lines := strings.Split(csvData, "\n")
	if len(lines) <= 1 {
		return nil, nil
	}

	headers := strings.Split(lines[0], ",")
	headerMap := make(map[string]int)
	for i, h := range headers {
		headerMap[strings.TrimSpace(h)] = i
	}

	var events []*model.Event
	maxEvents := 200

	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		fields := strings.Split(line, ",")
		if len(fields) < len(headers) {
			continue
		}

		lat, err := strconv.ParseFloat(fields[headerMap["latitude"]], 64)
		if err != nil {
			continue
		}
		lon, err := strconv.ParseFloat(fields[headerMap["longitude"]], 64)
		if err != nil {
			continue
		}

		brightness := 0.0
		if idx, ok := headerMap["bright_ti4"]; ok && idx < len(fields) {
			brightness, _ = strconv.ParseFloat(fields[idx], 64)
		}
		if brightness == 0 {
			if idx, ok := headerMap["brightness"]; ok && idx < len(fields) {
				brightness, _ = strconv.ParseFloat(fields[idx], 64)
			}
		}

		frp := 0.0
		if idx, ok := headerMap["frp"]; ok && idx < len(fields) {
			frp, _ = strconv.ParseFloat(fields[idx], 64)
		}

		confidence := ""
		if idx, ok := headerMap["confidence"]; ok && idx < len(fields) {
			confidence = strings.ToLower(fields[idx])
		}

		acqDate := ""
		if idx, ok := headerMap["acq_date"]; ok && idx < len(fields) {
			acqDate = fields[idx]
		}
		acqTime := ""
		if idx, ok := headerMap["acq_time"]; ok && idx < len(fields) {
			acqTime = fields[idx]
		}

		occurredAt := time.Now().UTC()
		if acqDate != "" && acqTime != "" {
			if len(acqTime) == 3 {
				acqTime = "0" + acqTime
			}
			if t, err := time.Parse("2006-01-02 1504", acqDate+" "+acqTime); err == nil {
				occurredAt = t.UTC()
			}
		}

		severity := p.determineSeverity(brightness, frp)

		title := fmt.Sprintf("Fire Detection (%s) - %.0f K, %.1f MW", source, brightness, frp)
		if strings.Contains(confidence, "high") {
			title = "High-confidence " + title
		}

		event := &model.Event{
			Title:       title,
			Description: fmt.Sprintf("NASA FIRMS Real-time fire detection via %s\n\nBrightness: %.0f K\nFRP: %.1f MW\nConfidence: %s\nLocation: %.4f, %.4f\nTime: %s %s UTC", source, brightness, frp, confidence, lat, lon, acqDate, acqTime),
			Source:      "nasa_firms_rt",
			SourceID:    fmt.Sprintf("firms_rt_%s_%.4f_%.4f_%s", source, lat, lon, acqDate),
			OccurredAt:  occurredAt,
			Location:    model.Point(lon, lat),
			Precision:   model.PrecisionExact,
			Category:    "wildfire",
			Severity:    severity,
			Metadata: map[string]string{
				"sensor":     source,
				"brightness": fmt.Sprintf("%.0f", brightness),
				"frp":        fmt.Sprintf("%.1f", frp),
				"confidence": confidence,
				"tier":       "1",
				"signup_url": "https://firms.modaps.eosdis.nasa.gov/map/",
			},
			Badges: []model.Badge{
				{Label: "NASA FIRMS RT", Type: "source", Timestamp: occurredAt},
				{Label: "wildfire", Type: "category", Timestamp: occurredAt},
				{Label: source, Type: "sensor", Timestamp: occurredAt},
			},
		}

		events = append(events, event)
		if len(events) >= maxEvents {
			break
		}
	}

	return events, nil
}

func (p *NASAFIRMSRTProvider) determineSeverity(brightness, frp float64) model.Severity {
	if brightness > 400 || frp > 1000 {
		return model.SeverityCritical
	}
	if brightness > 330 || frp > 100 {
		return model.SeverityHigh
	}
	if brightness > 300 || frp > 10 {
		return model.SeverityMedium
	}
	return model.SeverityLow
}
