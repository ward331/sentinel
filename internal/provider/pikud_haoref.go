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

// PikudHaOrefProvider fetches Israel Home Front Command (Pikud HaOref) Red Alert data
type PikudHaOrefProvider struct {
	name     string
	alertURL string
	interval time.Duration
}

// NewPikudHaOrefProvider creates a new Pikud HaOref provider
func NewPikudHaOrefProvider() *PikudHaOrefProvider {
	return &PikudHaOrefProvider{
		name:     "pikud_haoref",
		alertURL: "https://www.oref.org.il/WarningMessages/alert/alerts.json",
		interval: 5 * time.Second,
	}
}

// Name returns the provider identifier
func (p *PikudHaOrefProvider) Name() string {
	return p.name
}

// Enabled returns whether the provider is enabled
func (p *PikudHaOrefProvider) Enabled() bool {
	return true
}

// Interval returns the polling interval
func (p *PikudHaOrefProvider) Interval() time.Duration {
	return p.interval
}

// pikudAlert represents an alert from Pikud HaOref
type pikudAlert struct {
	ID        string `json:"id"`
	Cat       string `json:"cat"`
	Title     string `json:"title"`
	Data      []string `json:"data"`
	Desc      string `json:"desc"`
}

// Fetch retrieves active alerts from Pikud HaOref
func (p *PikudHaOrefProvider) Fetch(ctx context.Context) ([]*model.Event, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", p.alertURL, nil)
	if err != nil {
		return []*model.Event{}, nil
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; SENTINEL/3.0)")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Referer", "https://www.oref.org.il/")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return []*model.Event{}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []*model.Event{}, nil
	}

	var alerts []pikudAlert
	if err := json.NewDecoder(resp.Body).Decode(&alerts); err != nil {
		// Response may be empty or non-JSON when no alerts active
		return []*model.Event{}, nil
	}

	var events []*model.Event
	for _, alert := range alerts {
		severity := p.determineSeverity(alert)
		areas := strings.Join(alert.Data, ", ")

		event := &model.Event{
			Title:       fmt.Sprintf("Red Alert: %s — %s", alert.Title, areas),
			Description: fmt.Sprintf("Pikud HaOref alert: %s\nAreas: %s\n%s", alert.Title, areas, alert.Desc),
			Source:      p.name,
			SourceID:    fmt.Sprintf("pikud_%s", alert.ID),
			OccurredAt:  time.Now().UTC(),
			Location:    model.Point(34.78, 32.08), // Israel centroid
			Precision:   model.PrecisionApproximate,
			Category:    "conflict",
			Severity:    severity,
			Metadata: map[string]string{
				"alert_type": alert.Cat,
				"areas":      areas,
				"source":     "Pikud HaOref (Israel Home Front Command)",
			},
		}
		events = append(events, event)
	}

	return events, nil
}

func (p *PikudHaOrefProvider) determineSeverity(alert pikudAlert) model.Severity {
	cat := strings.ToLower(alert.Cat)
	switch {
	case strings.Contains(cat, "missile") || strings.Contains(cat, "rocket"):
		return model.SeverityCritical
	case strings.Contains(cat, "uav") || strings.Contains(cat, "drone"):
		return model.SeverityHigh
	case strings.Contains(cat, "infiltration"):
		return model.SeverityCritical
	default:
		return model.SeverityHigh
	}
}
