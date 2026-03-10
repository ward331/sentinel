package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/openclaw/sentinel-backend/internal/model"
)

// IranConflictProvider fetches Iran conflict data from OSINT sources
type IranConflictProvider struct {
	name     string
	baseURL  string
	interval time.Duration
}




// NewIranConflictProvider creates a new Iran conflict data provider
func NewIranConflictProvider() *IranConflictProvider {
	return &IranConflictProvider{
		name:     "iranconflict",
		baseURL:  "https://raw.githubusercontent.com/danielrosehill/Iran-Israel-War-2026-OSINT-Data/main/data",
		interval: 15 * time.Minute,
	}
}


// Fetch retrieves Iran conflict data
func (p *IranConflictProvider) Fetch(ctx context.Context) ([]*model.Event, error) {
	// Fetch waves.json data
	waves, err := p.fetchWavesData(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch waves data: %w", err)
	}

	// Convert to events
	events := make([]*model.Event, 0, len(waves))
	for _, wave := range waves {
		event := p.waveToEvent(wave)
		if event != nil {
			events = append(events, event)
		}
	}

	return events, nil
}

// waveToEvent converts a wave to an event
func (p *IranConflictProvider) waveToEvent(wave *WaveData) *model.Event {
	if wave == nil {
		return nil
	}

	// Determine severity based on weapon type and target
	severity := p.determineSeverity(wave.WeaponType, wave.TargetType)

	// Create event
	event := &model.Event{
		ID:          fmt.Sprintf("iranconflict-%s-%d", wave.OperationName, wave.WaveNumber),
		Title:       fmt.Sprintf("Iran Conflict: %s", wave.OperationName),
		Description: p.generateDescription(wave),
		Source:      "iranconflict",
		SourceID:    fmt.Sprintf("wave-%d", wave.WaveNumber),
		OccurredAt:  wave.Date,
		Location: model.Location{
			Type:        "Point",
			Coordinates: []float64{wave.Longitude, wave.Latitude},
		},
		Precision: model.PrecisionExact,
		Magnitude: p.calculateMagnitude(wave),
		Category:  "conflict",
		Severity:  severity,
		Metadata: map[string]string{
			"wave_number":        fmt.Sprintf("%d", wave.WaveNumber),
			"operation_name":     wave.OperationName,
			"weapon_type":        wave.WeaponType,
			"target_type":        wave.TargetType,
			"interception_rate":  fmt.Sprintf("%.1f", wave.InterceptionRate),
			"impact_assessment":  wave.ImpactAssessment,
			"total_weapons":      fmt.Sprintf("%d", wave.TotalWeapons),
			"targets_destroyed":  fmt.Sprintf("%d", wave.TargetsDestroyed),
			"alert_tier":         "TIER 3",
			"data_source":        "Iran-Israel-War-2026-OSINT-Data",
			"conflict_region":    "Middle East",
			"primary_actors":     "Iran; Israel",
		},
		Badges: []model.Badge{
			{
				Type:      model.BadgeTypeSource,
				Label:     "OSINT Conflict Data",
				Timestamp: time.Now().UTC(),
			},
			{
				Type:      model.BadgeTypePrecision,
				Label:     "Exact",
				Timestamp: time.Now().UTC(),
			},
			{
				Type:      model.BadgeTypeFreshness,
				Label:     "Real-time",
				Timestamp: time.Now().UTC(),
			},
		},
	}

	return event
}

// determineSeverity determines event severity based on weapon and target
func (p *IranConflictProvider) determineSeverity(weaponType, targetType string) model.Severity {
	// High severity weapons
	highSeverityWeapons := map[string]bool{
		"ballistic_missile": true,
		"cruise_missile":    true,
		"drone_swarm":       true,
		"hypersonic":        true,
	}

	// High value targets
	highValueTargets := map[string]bool{
		"military_base":     true,
		"air_defense":       true,
		"nuclear_facility":  true,
		"government_building": true,
		"critical_infrastructure": true,
	}

	if highSeverityWeapons[weaponType] && highValueTargets[targetType] {
		return model.SeverityCritical
	} else if highSeverityWeapons[weaponType] || highValueTargets[targetType] {
		return model.SeverityHigh
	} else {
		return model.SeverityMedium
	}
}

// calculateMagnitude calculates event magnitude
func (p *IranConflictProvider) calculateMagnitude(wave *WaveData) float64 {
	// Base magnitude on weapon count and interception rate
	base := 3.0 // Base magnitude for conflict events
	
	// Adjust based on weapon count
	weaponFactor := float64(wave.TotalWeapons) / 10.0
	if weaponFactor > 5.0 {
		weaponFactor = 5.0
	}

	// Adjust based on interception rate (lower interception = higher impact)
	interceptionFactor := (100.0 - wave.InterceptionRate) / 20.0

	// Adjust based on targets destroyed
	destructionFactor := float64(wave.TargetsDestroyed) / 2.0

	return base + weaponFactor + interceptionFactor + destructionFactor
}

// generateDescription generates event description
func (p *IranConflictProvider) generateDescription(wave *WaveData) string {
	return fmt.Sprintf(
		"Operation: %s\nWave: %d\nWeapons: %s (%d total)\nTargets: %s\nInterception Rate: %.1f%%\nImpact: %s\nTargets Destroyed: %d",
		wave.OperationName,
		wave.WaveNumber,
		wave.WeaponType,
		wave.TotalWeapons,
		wave.TargetType,
		wave.InterceptionRate,
		wave.ImpactAssessment,
		wave.TargetsDestroyed,
	)
}

// fetchWavesData fetches waves.json from GitHub
func (p *IranConflictProvider) fetchWavesData(ctx context.Context) ([]*WaveData, error) {
	url := fmt.Sprintf("%s/waves.json", p.baseURL)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var waves []*WaveData
	if err := json.Unmarshal(body, &waves); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return waves, nil
}

// WaveData represents a strike wave from the OSINT dataset
type WaveData struct {
	WaveNumber       int       `json:"wave_number"`
	OperationName    string    `json:"operation_name"`
	Date             time.Time `json:"date"`
	WeaponType       string    `json:"weapon_type"`
	TargetType       string    `json:"target_type"`
	Latitude         float64   `json:"latitude"`
	Longitude        float64   `json:"longitude"`
	InterceptionRate float64   `json:"interception_rate"`
	ImpactAssessment string    `json:"impact_assessment"`
	TotalWeapons     int       `json:"total_weapons"`
	TargetsDestroyed int       `json:"targets_destroyed"`
}

// ISWProvider fetches Institute for the Study of War RSS feed
type ISWProvider struct {
	name     string
	feedURL  string
	interval time.Duration
}

// NewISWProvider creates a new ISW RSS feed provider
func NewISWProvider() *ISWProvider {
	return &ISWProvider{
		name:     "isw",
		feedURL:  "https://understandingwar.org/rss.xml",
		interval: 30 * time.Minute,
	}
}


// Fetch retrieves ISW RSS feed data
func (p *ISWProvider) Fetch(ctx context.Context) ([]*model.Event, error) {
	// This would parse RSS feed and filter for Iran/Israel/Middle East keywords
	// For now, return empty slice as placeholder
	return []*model.Event{}, nil
}

// IranStrikeMapPreset creates a preset for the Iran Strike Map
func IranStrikeMapPreset() map[string]interface{} {
	return map[string]interface{}{
		"name":        "Iran Conflict Tracking",
		"category":    "Conflict Tracking",
		"description": "Real-time tracking of Iran-Israel conflict events",
		"iframe": map[string]interface{}{
			"src":    "https://www.iranstrikemap.com",
			"width":  "100%",
			"height": "600px",
			"title":  "Iran Strike Map - Real-time conflict tracking",
		},
		"sources": []map[string]interface{}{
			{
				"name":        "OSINT Dataset",
				"url":         "https://github.com/danielrosehill/Iran-Israel-War-2026-OSINT-Data",
				"description": "Comprehensive OSINT data on Iran-Israel conflict",
				"update_frequency": "15 minutes",
			},
			{
				"name":        "ISW RSS Feed",
				"url":         "https://understandingwar.org/rss.xml",
				"description": "Institute for the Study of War analysis",
				"update_frequency": "30 minutes",
			},
			{
				"name":        "Iran Strike Map",
				"url":         "https://www.iranstrikemap.com",
				"description": "Interactive map of strike events",
				"type":        "iframe",
			},
		},
		"keywords": []string{
			"Iran", "Israel", "Middle East", "Conflict", "OSINT",
			"Missile", "Drone", "Strike", "Defense", "Interception",
		},
		"alert_tier": "TIER 3",
		"priority":   "high",
	}
}
