package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/openclaw/sentinel-backend/internal/model"
)

// AbuseCHProvider fetches malware samples and botnet C2 tracking data from abuse.ch
// Tier 1: Free with API key
// Category: cyber
// Signup: https://bazaar.abuse.ch/api/ and https://feodotracker.abuse.ch/
type AbuseCHProvider struct {
	client *http.Client
	config *Config
}

// NewAbuseCHProvider creates a new AbuseCHProvider
func NewAbuseCHProvider(config *Config) *AbuseCHProvider {
	return &AbuseCHProvider{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		config: config,
	}
}

// Name returns the provider identifier
func (p *AbuseCHProvider) Name() string {
	return "abusech"
}

// Enabled returns whether the provider is enabled (requires API key)
func (p *AbuseCHProvider) Enabled() bool {
	if p.config == nil || p.config.APIKey == "" {
		return false
	}
	return p.config.Enabled
}

// Interval returns the polling interval
func (p *AbuseCHProvider) Interval() time.Duration {
	if p.config != nil && p.config.PollInterval > 0 {
		return p.config.PollInterval
	}
	return 1800 * time.Second
}

// bazaarResponse represents the MalwareBazaar API response
type bazaarResponse struct {
	QueryStatus string         `json:"query_status"`
	Data        []bazaarSample `json:"data"`
}

type bazaarSample struct {
	SHA256Hash     string `json:"sha256_hash"`
	MD5Hash        string `json:"md5_hash"`
	SHA1Hash       string `json:"sha1_hash"`
	FirstSeen      string `json:"first_seen"`
	LastSeen       string `json:"last_seen"`
	FileName       string `json:"file_name"`
	FileType       string `json:"file_type"`
	FileSize       int    `json:"file_size"`
	Signature      string `json:"signature"`
	Reporter       string `json:"reporter"`
	Tags           []string `json:"tags"`
	OriginCountry  string `json:"origin_country"`
	DeliveryMethod string `json:"delivery_method"`
}

// feodoEntry represents a Feodo Tracker botnet C2 entry
type feodoEntry struct {
	ID            int    `json:"id"`
	IPAddress     string `json:"ip_address"`
	Port          int    `json:"port"`
	Status        string `json:"status"`
	Hostname      string `json:"hostname"`
	ASNumber      int    `json:"as_number"`
	ASName        string `json:"as_name"`
	Country       string `json:"country"`
	FirstSeen     string `json:"first_seen"`
	LastOnline    string `json:"last_online"`
	Malware       string `json:"malware"`
}

type feodoResponse struct {
	QueryStatus string       `json:"query_status"`
	Data        []feodoEntry `json:"data"`
}

// Fetch retrieves malware and botnet data from abuse.ch
func (p *AbuseCHProvider) Fetch(ctx context.Context) ([]*model.Event, error) {
	var allEvents []*model.Event

	// Fetch recent malware samples from MalwareBazaar
	bazaarEvents, err := p.fetchBazaar(ctx)
	if err != nil {
		fmt.Printf("Warning: MalwareBazaar fetch failed: %v\n", err)
	} else {
		allEvents = append(allEvents, bazaarEvents...)
	}

	// Fetch Feodo Tracker botnet C2 data
	feodoEvents, err := p.fetchFeodo(ctx)
	if err != nil {
		fmt.Printf("Warning: FeodoTracker fetch failed: %v\n", err)
	} else {
		allEvents = append(allEvents, feodoEvents...)
	}

	return allEvents, nil
}

func (p *AbuseCHProvider) fetchBazaar(ctx context.Context) ([]*model.Event, error) {
	url := "https://mb-api.abuse.ch/api/v1/"

	body := "query=get_recent&selector=100"

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create MalwareBazaar request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("API-KEY", p.config.APIKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch MalwareBazaar data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("MalwareBazaar returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var data bazaarResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode MalwareBazaar response: %w", err)
	}

	var events []*model.Event
	now := time.Now().UTC()

	for _, sample := range data.Data {
		firstSeen, err := time.Parse("2006-01-02 15:04:05", sample.FirstSeen)
		if err != nil {
			firstSeen = now
		}

		severity := model.SeverityMedium
		if sample.Signature != "" {
			severity = model.SeverityHigh
		}

		tags := strings.Join(sample.Tags, ", ")
		title := fmt.Sprintf("Malware: %s (%s)", sample.Signature, sample.FileType)
		if sample.Signature == "" {
			title = fmt.Sprintf("Malware Sample: %s (%s)", sample.FileName, sample.FileType)
		}

		event := &model.Event{
			Title:       title,
			Description: fmt.Sprintf("MalwareBazaar Sample\n\nSignature: %s\nFile: %s (%s, %d bytes)\nSHA256: %s\nFirst Seen: %s\nDelivery: %s\nTags: %s\nReporter: %s\nOrigin: %s", sample.Signature, sample.FileName, sample.FileType, sample.FileSize, sample.SHA256Hash, sample.FirstSeen, sample.DeliveryMethod, tags, sample.Reporter, sample.OriginCountry),
			Source:      "abusech_bazaar",
			SourceID:    fmt.Sprintf("bazaar_%s", sample.SHA256Hash[:16]),
			OccurredAt:  firstSeen,
			Location:    model.Point(0, 0), // No specific location for malware samples
			Precision:   model.PrecisionUnknown,
			Category:    "cyber",
			Severity:    severity,
			Metadata: map[string]string{
				"sha256":          sample.SHA256Hash,
				"md5":             sample.MD5Hash,
				"file_name":       sample.FileName,
				"file_type":       sample.FileType,
				"signature":       sample.Signature,
				"reporter":        sample.Reporter,
				"origin_country":  sample.OriginCountry,
				"delivery_method": sample.DeliveryMethod,
				"tier":            "1",
				"signup_url":      "https://bazaar.abuse.ch/api/",
			},
			Badges: []model.Badge{
				{Label: "MalwareBazaar", Type: "source", Timestamp: firstSeen},
				{Label: "cyber", Type: "category", Timestamp: firstSeen},
				{Label: sample.FileType, Type: "file_type", Timestamp: firstSeen},
			},
		}

		events = append(events, event)
	}

	return events, nil
}

func (p *AbuseCHProvider) fetchFeodo(ctx context.Context) ([]*model.Event, error) {
	url := "https://feodotracker.abuse.ch/downloads/ipblocklist_recommended.json"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create FeodoTracker request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch FeodoTracker data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("FeodoTracker returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var entries []feodoEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("failed to decode FeodoTracker response: %w", err)
	}

	var events []*model.Event
	now := time.Now().UTC()
	maxEvents := 50

	for _, entry := range entries {
		firstSeen, err := time.Parse("2006-01-02 15:04:05", entry.FirstSeen)
		if err != nil {
			firstSeen = now
		}

		severity := model.SeverityHigh
		if entry.Status == "online" {
			severity = model.SeverityCritical
		}

		event := &model.Event{
			Title:       fmt.Sprintf("Botnet C2: %s (%s:%d) - %s", entry.Malware, entry.IPAddress, entry.Port, entry.Status),
			Description: fmt.Sprintf("Feodo Tracker Botnet C2 Server\n\nMalware: %s\nIP: %s:%d\nStatus: %s\nHostname: %s\nASN: AS%d (%s)\nCountry: %s\nFirst Seen: %s\nLast Online: %s", entry.Malware, entry.IPAddress, entry.Port, entry.Status, entry.Hostname, entry.ASNumber, entry.ASName, entry.Country, entry.FirstSeen, entry.LastOnline),
			Source:      "abusech_feodo",
			SourceID:    fmt.Sprintf("feodo_%d", entry.ID),
			OccurredAt:  firstSeen,
			Location:    model.Point(0, 0), // GeoIP would be needed for accurate placement
			Precision:   model.PrecisionUnknown,
			Category:    "cyber",
			Severity:    severity,
			Metadata: map[string]string{
				"ip":          entry.IPAddress,
				"port":        fmt.Sprintf("%d", entry.Port),
				"status":      entry.Status,
				"malware":     entry.Malware,
				"hostname":    entry.Hostname,
				"asn":         fmt.Sprintf("AS%d", entry.ASNumber),
				"as_name":     entry.ASName,
				"country":     entry.Country,
				"first_seen":  entry.FirstSeen,
				"last_online": entry.LastOnline,
				"tier":        "1",
				"signup_url":  "https://feodotracker.abuse.ch/",
			},
			Badges: []model.Badge{
				{Label: "FeodoTracker", Type: "source", Timestamp: firstSeen},
				{Label: "cyber", Type: "category", Timestamp: firstSeen},
				{Label: entry.Malware, Type: "malware_family", Timestamp: firstSeen},
			},
		}

		events = append(events, event)
		if len(events) >= maxEvents {
			break
		}
	}

	return events, nil
}
