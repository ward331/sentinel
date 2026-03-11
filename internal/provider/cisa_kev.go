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

// CISAKEVProvider fetches CISA Known Exploited Vulnerabilities catalog
type CISAKEVProvider struct {
	name     string
	apiURL   string
	interval time.Duration
}

// NewCISAKEVProvider creates a new CISA KEV provider
func NewCISAKEVProvider() *CISAKEVProvider {
	return &CISAKEVProvider{
		name:     "cisa_kev",
		apiURL:   "https://www.cisa.gov/sites/default/files/feeds/known_exploited_vulnerabilities.json",
		interval: 3600 * time.Second,
	}
}

// Name returns the provider identifier
func (p *CISAKEVProvider) Name() string {
	return p.name
}

// Enabled returns whether the provider is enabled
func (p *CISAKEVProvider) Enabled() bool {
	return true
}

// Interval returns the polling interval
func (p *CISAKEVProvider) Interval() time.Duration {
	return p.interval
}

// cisaKEVCatalog represents the CISA KEV JSON catalog
type cisaKEVCatalog struct {
	Title           string          `json:"title"`
	CatalogVersion  string          `json:"catalogVersion"`
	DateReleased    string          `json:"dateReleased"`
	Count           int             `json:"count"`
	Vulnerabilities []cisaKEVEntry  `json:"vulnerabilities"`
}

// cisaKEVEntry represents a single KEV entry
type cisaKEVEntry struct {
	CVEID              string `json:"cveID"`
	VendorProject      string `json:"vendorProject"`
	Product            string `json:"product"`
	VulnerabilityName  string `json:"vulnerabilityName"`
	DateAdded          string `json:"dateAdded"`
	ShortDescription   string `json:"shortDescription"`
	RequiredAction     string `json:"requiredAction"`
	DueDate            string `json:"dueDate"`
	KnownRansomware    string `json:"knownRansomwareCampaignUse"`
	Notes              string `json:"notes"`
}

// Fetch retrieves recent KEV entries from CISA
func (p *CISAKEVProvider) Fetch(ctx context.Context) ([]*model.Event, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", p.apiURL, nil)
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

	var catalog cisaKEVCatalog
	if err := json.NewDecoder(resp.Body).Decode(&catalog); err != nil {
		return []*model.Event{}, nil
	}

	// Only return entries added in the last 7 days
	cutoff := time.Now().UTC().AddDate(0, 0, -7)
	var events []*model.Event

	for _, vuln := range catalog.Vulnerabilities {
		dateAdded, err := time.Parse("2006-01-02", vuln.DateAdded)
		if err != nil {
			continue
		}
		if dateAdded.Before(cutoff) {
			continue
		}

		severity := p.determineSeverity(vuln)

		event := &model.Event{
			Title:       fmt.Sprintf("CISA KEV: %s — %s %s", vuln.CVEID, vuln.VendorProject, vuln.Product),
			Description: fmt.Sprintf("%s\n\nVulnerability: %s\nVendor: %s | Product: %s\nRequired Action: %s\nDue Date: %s\nRansomware Use: %s", vuln.ShortDescription, vuln.VulnerabilityName, vuln.VendorProject, vuln.Product, vuln.RequiredAction, vuln.DueDate, vuln.KnownRansomware),
			Source:      p.name,
			SourceID:    fmt.Sprintf("cisa_kev_%s", vuln.CVEID),
			OccurredAt:  dateAdded,
			Location:    model.Point(-77.04, 38.90), // Washington DC
			Precision:   model.PrecisionApproximate,
			Category:    "cyber",
			Severity:    severity,
			Metadata: map[string]string{
				"cve_id":            vuln.CVEID,
				"vendor":            vuln.VendorProject,
				"product":           vuln.Product,
				"date_added":        vuln.DateAdded,
				"due_date":          vuln.DueDate,
				"known_ransomware":  vuln.KnownRansomware,
				"source":            "CISA Known Exploited Vulnerabilities",
			},
		}
		events = append(events, event)
	}

	return events, nil
}

func (p *CISAKEVProvider) determineSeverity(vuln cisaKEVEntry) model.Severity {
	if strings.ToLower(vuln.KnownRansomware) == "known" {
		return model.SeverityCritical
	}
	desc := strings.ToLower(vuln.ShortDescription + " " + vuln.VulnerabilityName)
	switch {
	case strings.Contains(desc, "remote code execution") || strings.Contains(desc, "rce"):
		return model.SeverityCritical
	case strings.Contains(desc, "privilege escalation") || strings.Contains(desc, "authentication bypass"):
		return model.SeverityHigh
	case strings.Contains(desc, "information disclosure") || strings.Contains(desc, "xss"):
		return model.SeverityMedium
	default:
		return model.SeverityHigh // KEVs are actively exploited, default high
	}
}
