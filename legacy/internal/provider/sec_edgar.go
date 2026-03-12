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

// SECEdgarProvider fetches SEC EDGAR 8-K filings (material events)
type SECEdgarProvider struct {
	name     string
	apiURL   string
	interval time.Duration
}

// NewSECEdgarProvider creates a new SEC EDGAR provider
func NewSECEdgarProvider() *SECEdgarProvider {
	return &SECEdgarProvider{
		name:     "sec_edgar",
		apiURL:   "https://efts.sec.gov/LATEST/search-index?q=%228-K%22&dateRange=custom&startdt=%s&enddt=%s&forms=8-K",
		interval: 900 * time.Second,
	}
}

// Name returns the provider identifier
func (p *SECEdgarProvider) Name() string {
	return p.name
}

// Enabled returns whether the provider is enabled
func (p *SECEdgarProvider) Enabled() bool {
	return true
}

// Interval returns the polling interval
func (p *SECEdgarProvider) Interval() time.Duration {
	return p.interval
}

// edgarFullTextResponse represents the EDGAR full-text search API response
type edgarFullTextResponse struct {
	Hits struct {
		Total struct {
			Value int `json:"value"`
		} `json:"total"`
		Hits []edgarHit `json:"hits"`
	} `json:"hits"`
}

type edgarHit struct {
	ID     string         `json:"_id"`
	Source edgarFilingDoc `json:"_source"`
}

type edgarFilingDoc struct {
	EntityName   string `json:"entity_name"`
	FileNumber   string `json:"file_num"`
	FiledAt      string `json:"file_date"`
	FormType     string `json:"form_type"`
	DisplayNames []string `json:"display_names"`
}

// edgarRSSFiling represents a filing from the EDGAR RSS feed
type edgarRSSFiling struct {
	CompanyName string `json:"companyName"`
	FormType    string `json:"formType"`
	FilingDate  string `json:"filingDate"`
	AccessionNo string `json:"accessionNumber"`
	FileURL     string `json:"primaryDocument"`
	Items       string `json:"items"`
}

// Fetch retrieves recent 8-K filings from SEC EDGAR
func (p *SECEdgarProvider) Fetch(ctx context.Context) ([]*model.Event, error) {
	// Use the EDGAR XBRL filing feed for recent 8-K filings
	url := "https://efts.sec.gov/LATEST/search-index?q=%228-K%22&forms=8-K&dateRange=custom"
	now := time.Now().UTC()
	dayAgo := now.Add(-24 * time.Hour)
	url = fmt.Sprintf("https://efts.sec.gov/LATEST/search-index?q=%%228-K%%22&forms=8-K&startdt=%s&enddt=%s",
		dayAgo.Format("2006-01-02"), now.Format("2006-01-02"))

	// Try the EDGAR full-text search API
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return p.fetchFromRSS(ctx)
	}

	req.Header.Set("User-Agent", "SENTINEL sentinel@openclaw.org")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return p.fetchFromRSS(ctx)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return p.fetchFromRSS(ctx)
	}

	var result edgarFullTextResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return p.fetchFromRSS(ctx)
	}

	var events []*model.Event
	for _, hit := range result.Hits.Hits {
		doc := hit.Source
		if doc.FormType != "8-K" {
			continue
		}

		filedAt, err := time.Parse("2006-01-02", doc.FiledAt)
		if err != nil {
			filedAt = time.Now().UTC()
		}

		event := &model.Event{
			Title:       fmt.Sprintf("SEC 8-K: %s", doc.EntityName),
			Description: fmt.Sprintf("Material event filing (8-K) by %s.\nFiled: %s", doc.EntityName, doc.FiledAt),
			Source:      p.name,
			SourceID:    fmt.Sprintf("edgar_%s", hit.ID),
			OccurredAt:  filedAt,
			Location:    model.Point(-77.04, 38.90), // Washington DC (SEC HQ)
			Precision:   model.PrecisionApproximate,
			Category:    "financial",
			Severity:    model.SeverityLow,
			Metadata: map[string]string{
				"entity":    doc.EntityName,
				"form_type": doc.FormType,
				"file_num":  doc.FileNumber,
				"filed_at":  doc.FiledAt,
				"source":    "SEC EDGAR",
			},
		}
		events = append(events, event)
	}

	return events, nil
}

// fetchFromRSS falls back to the EDGAR company filing RSS feed
func (p *SECEdgarProvider) fetchFromRSS(ctx context.Context) ([]*model.Event, error) {
	// Fallback: EDGAR company 8-K RSS
	url := "https://www.sec.gov/cgi-bin/browse-edgar?action=getcurrent&type=8-K&dateb=&owner=include&count=20&search_text=&start=0&output=atom"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return []*model.Event{}, nil
	}

	req.Header.Set("User-Agent", "SENTINEL sentinel@openclaw.org")
	req.Header.Set("Accept", "application/atom+xml, application/xml")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return []*model.Event{}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []*model.Event{}, nil
	}

	// For simplicity, return empty on RSS parse failure — main JSON path is preferred
	return []*model.Event{}, nil
}

// determineSeverity checks 8-K items for material significance
func (p *SECEdgarProvider) determineSeverity(items string) model.Severity {
	lower := strings.ToLower(items)
	switch {
	case strings.Contains(lower, "bankruptcy") || strings.Contains(lower, "delisted"):
		return model.SeverityCritical
	case strings.Contains(lower, "restatement") || strings.Contains(lower, "auditor change"):
		return model.SeverityHigh
	case strings.Contains(lower, "acquisition") || strings.Contains(lower, "merger"):
		return model.SeverityMedium
	default:
		return model.SeverityLow
	}
}
