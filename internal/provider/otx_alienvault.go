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

// OTXAlienVaultProvider fetches public threat intelligence pulses from AlienVault OTX
type OTXAlienVaultProvider struct {
	name     string
	apiURL   string
	interval time.Duration
}

// NewOTXAlienVaultProvider creates a new OTX AlienVault provider
func NewOTXAlienVaultProvider() *OTXAlienVaultProvider {
	return &OTXAlienVaultProvider{
		name:     "otx_alienvault",
		apiURL:   "https://otx.alienvault.com/api/v1/pulses/subscribed?limit=20&modified_since=",
		interval: 1800 * time.Second,
	}
}

// Name returns the provider identifier
func (p *OTXAlienVaultProvider) Name() string {
	return p.name
}

// Enabled returns whether the provider is enabled
func (p *OTXAlienVaultProvider) Enabled() bool {
	return true
}

// Interval returns the polling interval
func (p *OTXAlienVaultProvider) Interval() time.Duration {
	return p.interval
}

// otxPulsesResponse represents the OTX API response
type otxPulsesResponse struct {
	Results []otxPulse `json:"results"`
	Count   int        `json:"count"`
}

// otxPulse represents a single OTX pulse (threat report)
type otxPulse struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	AuthorName  string   `json:"author_name"`
	Created     string   `json:"created"`
	Modified    string   `json:"modified"`
	Tags        []string `json:"tags"`
	TLP         string   `json:"tlp"`
	Adversary   string   `json:"adversary"`
	References  []string `json:"references"`
	Indicators  []otxIndicator `json:"indicators"`
}

// otxIndicator represents an indicator of compromise
type otxIndicator struct {
	ID          int    `json:"id"`
	Indicator   string `json:"indicator"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

// Fetch retrieves recent public pulses from OTX
func (p *OTXAlienVaultProvider) Fetch(ctx context.Context) ([]*model.Event, error) {
	// Use the public activity feed (no API key needed)
	since := time.Now().UTC().Add(-24 * time.Hour).Format("2006-01-02T15:04:05")
	url := fmt.Sprintf("https://otx.alienvault.com/api/v1/pulses/activity?limit=20&modified_since=%s", since)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return []*model.Event{}, nil
	}

	req.Header.Set("User-Agent", "SENTINEL/3.0")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return []*model.Event{}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []*model.Event{}, nil
	}

	var data otxPulsesResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return []*model.Event{}, nil
	}

	var events []*model.Event
	for _, pulse := range data.Results {
		created, err := time.Parse("2006-01-02T15:04:05", pulse.Created)
		if err != nil {
			created, err = time.Parse("2006-01-02T15:04:05.000000", pulse.Created)
			if err != nil {
				created = time.Now().UTC()
			}
		}

		severity := p.determineSeverity(pulse)
		tags := strings.Join(pulse.Tags, ", ")
		iocCount := len(pulse.Indicators)

		desc := pulse.Description
		if len(desc) > 800 {
			desc = desc[:800] + "..."
		}

		event := &model.Event{
			Title:       fmt.Sprintf("OTX Pulse: %s", pulse.Name),
			Description: fmt.Sprintf("%s\n\nTags: %s\nIndicators: %d IOCs\nAuthor: %s\nTLP: %s", desc, tags, iocCount, pulse.AuthorName, pulse.TLP),
			Source:      p.name,
			SourceID:    fmt.Sprintf("otx_%s", pulse.ID),
			OccurredAt:  created,
			Location:    model.Point(0, 0), // Cyber threats are non-geographic
			Precision:   model.PrecisionUnknown,
			Category:    "cyber",
			Severity:    severity,
			Metadata: map[string]string{
				"pulse_id":  pulse.ID,
				"author":    pulse.AuthorName,
				"tags":      tags,
				"tlp":       pulse.TLP,
				"adversary": pulse.Adversary,
				"ioc_count": fmt.Sprintf("%d", iocCount),
				"source":    "AlienVault OTX",
			},
		}
		events = append(events, event)
	}

	return events, nil
}

func (p *OTXAlienVaultProvider) determineSeverity(pulse otxPulse) model.Severity {
	text := strings.ToLower(pulse.Name + " " + pulse.Description + " " + strings.Join(pulse.Tags, " "))
	switch {
	case strings.Contains(text, "apt") || strings.Contains(text, "ransomware") || strings.Contains(text, "zero-day"):
		return model.SeverityCritical
	case strings.Contains(text, "malware") || strings.Contains(text, "exploit") || strings.Contains(text, "botnet"):
		return model.SeverityHigh
	case strings.Contains(text, "phishing") || strings.Contains(text, "c2") || strings.Contains(text, "trojan"):
		return model.SeverityMedium
	default:
		return model.SeverityLow
	}
}
