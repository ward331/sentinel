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

// UkraineAlertsProvider fetches Ukraine air raid alerts from the public alerts API
type UkraineAlertsProvider struct {
	name     string
	apiURL   string
	interval time.Duration
}

// NewUkraineAlertsProvider creates a new Ukraine alerts provider
func NewUkraineAlertsProvider() *UkraineAlertsProvider {
	return &UkraineAlertsProvider{
		name:     "ukraine_alerts",
		apiURL:   "https://alerts.com.ua/api/states",
		interval: 300 * time.Second,
	}
}

// Name returns the provider identifier
func (p *UkraineAlertsProvider) Name() string {
	return p.name
}

// Enabled returns whether the provider is enabled
func (p *UkraineAlertsProvider) Enabled() bool {
	return true
}

// Interval returns the polling interval
func (p *UkraineAlertsProvider) Interval() time.Duration {
	return p.interval
}

// ukraineAlertState represents an oblast alert state
type ukraineAlertState struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	NameEN    string `json:"name_en"`
	Alert     bool   `json:"alert"`
	ChangedAt string `json:"changed"`
}

// ukraineAlertsResponse is the API response
type ukraineAlertsResponse struct {
	States    []ukraineAlertState `json:"states"`
	LastUpdate string             `json:"last_update"`
}

// oblastCoords maps Ukrainian oblasts to approximate centroids
var oblastCoords = map[string][2]float64{
	"Kyiv City":           {30.52, 50.45},
	"Kyiv Oblast":         {30.50, 50.35},
	"Kharkiv Oblast":      {36.25, 49.99},
	"Dnipropetrovsk Oblast": {35.04, 48.46},
	"Odesa Oblast":        {30.73, 46.48},
	"Donetsk Oblast":      {37.80, 48.01},
	"Zaporizhzhia Oblast": {35.14, 47.84},
	"Lviv Oblast":         {24.03, 49.84},
	"Mykolaiv Oblast":     {31.99, 46.97},
	"Kherson Oblast":      {33.50, 46.63},
	"Poltava Oblast":      {34.55, 49.59},
	"Cherkasy Oblast":     {32.06, 49.44},
	"Sumy Oblast":         {34.80, 51.03},
	"Chernihiv Oblast":    {31.29, 51.49},
	"Zhytomyr Oblast":     {28.66, 50.25},
	"Vinnytsia Oblast":    {28.47, 49.23},
	"Rivne Oblast":        {26.25, 50.62},
	"Ivano-Frankivsk Oblast": {24.71, 48.92},
	"Ternopil Oblast":     {25.59, 49.55},
	"Volyn Oblast":        {24.32, 50.75},
	"Khmelnytskyi Oblast": {27.00, 49.42},
	"Chernivtsi Oblast":   {25.94, 48.29},
	"Zakarpattia Oblast":  {23.00, 48.62},
	"Kirovohrad Oblast":   {32.26, 48.51},
	"Luhansk Oblast":      {39.30, 48.57},
	"Crimea":              {34.10, 44.95},
}

// Fetch retrieves active air raid alerts across Ukraine
func (p *UkraineAlertsProvider) Fetch(ctx context.Context) ([]*model.Event, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", p.apiURL, nil)
	if err != nil {
		return []*model.Event{}, nil
	}

	req.Header.Set("User-Agent", "SENTINEL/3.0")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return []*model.Event{}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []*model.Event{}, nil
	}

	var data ukraineAlertsResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return []*model.Event{}, nil
	}

	var events []*model.Event
	for _, state := range data.States {
		if !state.Alert {
			continue
		}

		name := state.NameEN
		if name == "" {
			name = state.Name
		}

		coords, ok := oblastCoords[name]
		if !ok {
			coords = [2]float64{31.16, 48.38} // Ukraine centroid
		}

		changedAt, err := time.Parse(time.RFC3339, state.ChangedAt)
		if err != nil {
			changedAt = time.Now().UTC()
		}

		event := &model.Event{
			Title:       fmt.Sprintf("Air Raid Alert: %s", name),
			Description: fmt.Sprintf("Active air raid alert in %s, Ukraine.\nAlert activated: %s", name, changedAt.Format(time.RFC3339)),
			Source:      p.name,
			SourceID:    fmt.Sprintf("ua_alert_%d_%s", state.ID, changedAt.Format("20060102T150405")),
			OccurredAt:  changedAt,
			Location:    model.Point(coords[0], coords[1]),
			Precision:   model.PrecisionApproximate,
			Category:    "conflict",
			Severity:    p.determineSeverity(name),
			Metadata: map[string]string{
				"oblast":  name,
				"source":  "alerts.com.ua",
				"country": "Ukraine",
			},
		}
		events = append(events, event)
	}

	return events, nil
}

func (p *UkraineAlertsProvider) determineSeverity(oblast string) model.Severity {
	lower := strings.ToLower(oblast)
	// Frontline oblasts get higher severity
	frontline := []string{"donetsk", "luhansk", "zaporizhzhia", "kherson", "kharkiv"}
	for _, f := range frontline {
		if strings.Contains(lower, f) {
			return model.SeverityCritical
		}
	}
	if strings.Contains(lower, "kyiv") {
		return model.SeverityHigh
	}
	return model.SeverityMedium
}
