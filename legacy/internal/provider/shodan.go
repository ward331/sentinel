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

// ShodanProvider fetches internet-facing device intelligence from Shodan
// Tier 1: Free with API key (limited queries)
// Category: cyber
// Signup: https://account.shodan.io/register
type ShodanProvider struct {
	client *http.Client
	config *Config
}

// NewShodanProvider creates a new ShodanProvider
func NewShodanProvider(config *Config) *ShodanProvider {
	return &ShodanProvider{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		config: config,
	}
}

// Name returns the provider identifier
func (p *ShodanProvider) Name() string {
	return "shodan"
}

// Enabled returns whether the provider is enabled (requires API key)
func (p *ShodanProvider) Enabled() bool {
	if p.config == nil || p.config.APIKey == "" {
		return false
	}
	return p.config.Enabled
}

// Interval returns the polling interval
func (p *ShodanProvider) Interval() time.Duration {
	if p.config != nil && p.config.PollInterval > 0 {
		return p.config.PollInterval
	}
	return 3600 * time.Second
}

// shodanExploitResponse represents Shodan exploit search results
type shodanExploitResponse struct {
	Matches []shodanExploit `json:"matches"`
	Total   int             `json:"total"`
}

type shodanExploit struct {
	ID          string   `json:"_id"`
	Description string   `json:"description"`
	Author      string   `json:"author"`
	Code        string   `json:"code"`
	Date        string   `json:"date"`
	Platform    string   `json:"platform"`
	Port        int      `json:"port"`
	Source      string   `json:"source"`
	Type        string   `json:"type"`
	CVE         []string `json:"cve"`
}

// shodanAlertResponse represents Shodan network alert info
type shodanAlertResponse struct {
	ID      string              `json:"id"`
	Name    string              `json:"name"`
	Filters shodanAlertFilters  `json:"filters"`
	Created string              `json:"created"`
	Expires int                 `json:"expires"`
}

type shodanAlertFilters struct {
	IP []string `json:"ip"`
}

// shodanHoneypotResponse represents Shodan honeyscore
type shodanHoneypotResult struct {
	IP        string  `json:"ip_str"`
	Port      int     `json:"port"`
	Org       string  `json:"org"`
	OS        string  `json:"os"`
	Product   string  `json:"product"`
	Version   string  `json:"version"`
	Transport string  `json:"transport"`
	Lat       float64 `json:"latitude"`
	Lon       float64 `json:"longitude"`
	Country   string  `json:"country_name"`
	City      string  `json:"city"`
	Vulns     map[string]struct {
		CVSS     float64  `json:"cvss"`
		Summary  string   `json:"summary"`
		Refs     []string `json:"references"`
		Verified bool     `json:"verified"`
	} `json:"vulns"`
}

type shodanSearchResponse struct {
	Matches []shodanHoneypotResult `json:"matches"`
	Total   int                    `json:"total"`
}

// Fetch retrieves cyber intelligence from Shodan
func (p *ShodanProvider) Fetch(ctx context.Context) ([]*model.Event, error) {
	var allEvents []*model.Event

	// Fetch recently exposed vulnerable services
	vulnEvents, err := p.fetchVulnerableServices(ctx)
	if err != nil {
		fmt.Printf("Warning: Shodan vulnerable services fetch failed: %v\n", err)
	} else {
		allEvents = append(allEvents, vulnEvents...)
	}

	return allEvents, nil
}

func (p *ShodanProvider) fetchVulnerableServices(ctx context.Context) ([]*model.Event, error) {
	// Query for recently found critical vulnerabilities
	queries := []struct {
		query string
		label string
	}{
		{"vuln:CVE-2024 country:US", "US Critical CVEs"},
		{"product:apache vuln:CVE-2024", "Apache Vulnerabilities"},
	}

	var events []*model.Event

	for _, q := range queries {
		url := fmt.Sprintf("https://api.shodan.io/shodan/host/search?key=%s&query=%s&minify=true",
			p.config.APIKey, q.query)

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			continue
		}

		resp, err := p.client.Do(req)
		if err != nil {
			continue
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			fmt.Printf("Warning: Shodan query '%s' returned status %d: %s\n", q.query, resp.StatusCode, string(body))
			continue
		}

		var data shodanSearchResponse
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		now := time.Now().UTC()
		for _, match := range data.Matches {
			if match.Lat == 0 && match.Lon == 0 {
				continue
			}

			vulnCount := len(match.Vulns)
			severity := model.SeverityLow
			if vulnCount > 5 {
				severity = model.SeverityCritical
			} else if vulnCount > 2 {
				severity = model.SeverityHigh
			} else if vulnCount > 0 {
				severity = model.SeverityMedium
			}

			title := fmt.Sprintf("Exposed %s (%s:%d) - %d vulns",
				match.Product, match.IP, match.Port, vulnCount)

			description := fmt.Sprintf("Shodan Device Intelligence\n\nIP: %s\nPort: %d\nOrg: %s\nOS: %s\nProduct: %s %s\nLocation: %s, %s\nVulnerabilities: %d",
				match.IP, match.Port, match.Org, match.OS, match.Product, match.Version,
				match.City, match.Country, vulnCount)

			event := &model.Event{
				Title:       title,
				Description: description,
				Source:      "shodan",
				SourceID:    fmt.Sprintf("shodan_%s_%d_%d", match.IP, match.Port, now.Unix()),
				OccurredAt:  now,
				Location:    model.Point(match.Lon, match.Lat),
				Precision:   model.PrecisionApproximate,
				Category:    "cyber",
				Severity:    severity,
				Metadata: map[string]string{
					"ip":         match.IP,
					"port":       fmt.Sprintf("%d", match.Port),
					"org":        match.Org,
					"os":         match.OS,
					"product":    match.Product,
					"version":    match.Version,
					"country":    match.Country,
					"city":       match.City,
					"vuln_count": fmt.Sprintf("%d", vulnCount),
					"query":      q.label,
					"tier":       "1",
					"signup_url": "https://account.shodan.io/register",
				},
				Badges: []model.Badge{
					{Label: "Shodan", Type: "source", Timestamp: now},
					{Label: "cyber", Type: "category", Timestamp: now},
					{Label: q.label, Type: "query", Timestamp: now},
				},
			}

			events = append(events, event)
			if len(events) >= 50 {
				return events, nil
			}
		}
	}

	return events, nil
}
