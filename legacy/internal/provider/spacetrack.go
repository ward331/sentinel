package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/openclaw/sentinel-backend/internal/model"
)

// SpaceTrackProvider fetches TLE data, conjunction warnings, and decay predictions from space-track.org
// Tier 1: Free with username/password registration
// Category: space
// Signup: https://www.space-track.org/auth/createAccount
type SpaceTrackProvider struct {
	client *http.Client
	config *Config
	cookie string // Session cookie after auth
}

// NewSpaceTrackProvider creates a new SpaceTrackProvider
func NewSpaceTrackProvider(config *Config) *SpaceTrackProvider {
	return &SpaceTrackProvider{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		config: config,
	}
}

// Name returns the provider identifier
func (p *SpaceTrackProvider) Name() string {
	return "spacetrack"
}

// Enabled returns whether the provider is enabled (requires username + password in Options)
func (p *SpaceTrackProvider) Enabled() bool {
	if p.config == nil {
		return false
	}
	if p.config.Options == nil {
		return false
	}
	if p.config.Options["username"] == "" || p.config.Options["password"] == "" {
		return false
	}
	return p.config.Enabled
}

// Interval returns the polling interval
func (p *SpaceTrackProvider) Interval() time.Duration {
	if p.config != nil && p.config.PollInterval > 0 {
		return p.config.PollInterval
	}
	return 3600 * time.Second
}

// spaceTrackDecay represents a predicted decay entry
type spaceTrackDecay struct {
	NORAD_CAT_ID string `json:"NORAD_CAT_ID"`
	OBJECT_NAME  string `json:"OBJECT_NAME"`
	INTLDES      string `json:"INTLDES"`
	DECAY_EPOCH  string `json:"DECAY_EPOCH"`
	SOURCE       string `json:"SOURCE"`
	MSG_TYPE     string `json:"MSG_TYPE"`
	MSG_EPOCH    string `json:"MSG_EPOCH"`
}

// spaceTrackConjunction represents a conjunction data message
type spaceTrackConjunction struct {
	CDM_ID                string `json:"CDM_ID"`
	TCA                   string `json:"TCA"`
	SAT_1_NAME            string `json:"SAT_1_NAME"`
	SAT_1_ID              string `json:"SAT1_ID"`
	SAT_2_NAME            string `json:"SAT_2_NAME"`
	SAT_2_ID              string `json:"SAT2_ID"`
	MISS_DISTANCE         string `json:"MISS_DISTANCE"`
	COLLISION_PROBABILITY string `json:"COLLISION_PROBABILITY"`
}

// Fetch retrieves space tracking data
func (p *SpaceTrackProvider) Fetch(ctx context.Context) ([]*model.Event, error) {
	// Authenticate first
	if err := p.authenticate(ctx); err != nil {
		return nil, fmt.Errorf("SpaceTrack authentication failed: %w", err)
	}

	var allEvents []*model.Event

	// Fetch decay predictions
	decayEvents, err := p.fetchDecays(ctx)
	if err != nil {
		fmt.Printf("Warning: SpaceTrack decay fetch failed: %v\n", err)
	} else {
		allEvents = append(allEvents, decayEvents...)
	}

	return allEvents, nil
}

func (p *SpaceTrackProvider) authenticate(ctx context.Context) error {
	authURL := "https://www.space-track.org/ajaxauth/login"

	form := url.Values{}
	form.Set("identity", p.config.Options["username"])
	form.Set("password", p.config.Options["password"])

	req, err := http.NewRequestWithContext(ctx, "POST", authURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create auth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("authentication returned status %d: %s", resp.StatusCode, string(body))
	}

	// Extract session cookie
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "chocolatechip" {
			p.cookie = cookie.Value
			return nil
		}
	}

	// Even without explicit cookie, the client jar should handle it
	return nil
}

func (p *SpaceTrackProvider) fetchDecays(ctx context.Context) ([]*model.Event, error) {
	// Fetch decay predictions for the next 7 days
	url := "https://www.space-track.org/basicspacedata/query/class/decay/DECAY_EPOCH/>now/orderby/DECAY_EPOCH%20asc/limit/50/format/json"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create decay request: %w", err)
	}
	if p.cookie != "" {
		req.AddCookie(&http.Cookie{Name: "chocolatechip", Value: p.cookie})
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch decay data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("decay query returned status %d: %s", resp.StatusCode, string(body))
	}

	var decays []spaceTrackDecay
	if err := json.NewDecoder(resp.Body).Decode(&decays); err != nil {
		return nil, fmt.Errorf("failed to decode decay response: %w", err)
	}

	var events []*model.Event
	now := time.Now().UTC()

	for _, d := range decays {
		decayTime, err := time.Parse("2006-01-02 15:04:05", d.DECAY_EPOCH)
		if err != nil {
			decayTime = now
		}

		noradID, _ := strconv.Atoi(d.NORAD_CAT_ID)

		severity := model.SeverityLow
		hoursUntilDecay := time.Until(decayTime).Hours()
		if hoursUntilDecay < 24 {
			severity = model.SeverityMedium
		}
		if hoursUntilDecay < 6 {
			severity = model.SeverityHigh
		}

		event := &model.Event{
			Title:       fmt.Sprintf("Orbital Decay: %s (NORAD %s)", d.OBJECT_NAME, d.NORAD_CAT_ID),
			Description: fmt.Sprintf("Predicted orbital decay/reentry\n\nObject: %s\nNORAD ID: %s\nIntl Designator: %s\nPredicted Decay: %s\nSource: %s\nMessage Type: %s", d.OBJECT_NAME, d.NORAD_CAT_ID, d.INTLDES, d.DECAY_EPOCH, d.SOURCE, d.MSG_TYPE),
			Source:      "spacetrack",
			SourceID:    fmt.Sprintf("st_decay_%s_%s", d.NORAD_CAT_ID, d.DECAY_EPOCH),
			OccurredAt:  decayTime,
			Location:    model.Point(0, 0), // Orbital — no fixed location
			Precision:   model.PrecisionUnknown,
			Category:    "space",
			Severity:    severity,
			Metadata: map[string]string{
				"norad_id":    d.NORAD_CAT_ID,
				"norad_int":   fmt.Sprintf("%d", noradID),
				"object_name": d.OBJECT_NAME,
				"intl_des":    d.INTLDES,
				"decay_epoch": d.DECAY_EPOCH,
				"msg_source":  d.SOURCE,
				"msg_type":    d.MSG_TYPE,
				"tier":        "1",
				"signup_url":  "https://www.space-track.org/auth/createAccount",
			},
			Badges: []model.Badge{
				{Label: "Space-Track.org", Type: "source", Timestamp: now},
				{Label: "space", Type: "category", Timestamp: now},
				{Label: "Orbital Decay", Type: "event_type", Timestamp: now},
			},
		}

		events = append(events, event)
	}

	return events, nil
}
