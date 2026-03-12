package notify

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"
	"time"
)

// AlertRule defines conditions and actions for auto-routing events to channels.
type AlertRule struct {
	ID            int64           `json:"id"`
	Name          string          `json:"name"`
	Conditions    RuleConditions  `json:"conditions"`
	Actions       RuleActions     `json:"actions"`
	Enabled       bool            `json:"enabled"`
	CreatedAt     time.Time       `json:"created_at"`
}

// RuleConditions defines when a rule fires.
type RuleConditions struct {
	Category     string   `json:"category,omitempty"`      // exact category match
	MinSeverity  string   `json:"min_severity,omitempty"`  // severity >= threshold
	Keywords     []string `json:"keywords,omitempty"`      // any keyword in title or body
	Sources      []string `json:"sources,omitempty"`       // event source matches
	MinMagnitude float64  `json:"min_magnitude,omitempty"` // e.g. earthquake magnitude
	LocationLat  float64  `json:"location_lat,omitempty"`  // center of radius check
	LocationLon  float64  `json:"location_lon,omitempty"`
	RadiusKm     float64  `json:"radius_km,omitempty"`     // radius in km (0 = no geo filter)
}

// RuleActions defines what to do when a rule fires.
type RuleActions struct {
	Channels []string `json:"channels"` // list of channel names, or ["*"] for all
}

// AlertRuleRaw is the database row with JSON blobs.
type AlertRuleRaw struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	ConditionsJSON string    `json:"conditions_json"`
	ActionsJSON    string    `json:"actions_json"`
	Enabled        int       `json:"enabled"`
	CreatedAt      time.Time `json:"created_at"`
}

// ToAlertRule converts a raw DB row into a typed AlertRule.
func (r *AlertRuleRaw) ToAlertRule() (*AlertRule, error) {
	rule := &AlertRule{
		ID:        r.ID,
		Name:      r.Name,
		Enabled:   r.Enabled == 1,
		CreatedAt: r.CreatedAt,
	}
	if err := json.Unmarshal([]byte(r.ConditionsJSON), &rule.Conditions); err != nil {
		return nil, fmt.Errorf("parse conditions for rule %d: %w", r.ID, err)
	}
	if err := json.Unmarshal([]byte(r.ActionsJSON), &rule.Actions); err != nil {
		return nil, fmt.Errorf("parse actions for rule %d: %w", r.ID, err)
	}
	return rule, nil
}

// EventData is a minimal event representation for rule evaluation.
// This avoids importing the model package directly.
type EventData struct {
	Title       string
	Description string
	Category    string
	Severity    string
	Source      string
	Magnitude   float64
	Lat         float64
	Lon         float64
}

// severityRank returns an integer rank for severity comparison.
func severityRank(s string) int {
	switch strings.ToLower(s) {
	case "info":
		return 1
	case "watch":
		return 2
	case "warning":
		return 3
	case "alert":
		return 4
	case "critical":
		return 5
	default:
		return 0
	}
}

// haversineKm computes the great-circle distance between two lat/lon points in km.
func haversineKm(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371.0 // Earth radius in km
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

// Matches checks whether an event satisfies all rule conditions.
// All non-empty conditions must match (AND logic).
func (r *AlertRule) Matches(evt EventData) bool {
	c := r.Conditions

	// Category check
	if c.Category != "" && !strings.EqualFold(c.Category, evt.Category) {
		return false
	}

	// Severity threshold
	if c.MinSeverity != "" && severityRank(evt.Severity) < severityRank(c.MinSeverity) {
		return false
	}

	// Source check
	if len(c.Sources) > 0 {
		found := false
		for _, s := range c.Sources {
			if strings.EqualFold(s, evt.Source) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Keyword check (OR within keywords)
	if len(c.Keywords) > 0 {
		found := false
		combined := strings.ToLower(evt.Title + " " + evt.Description)
		for _, kw := range c.Keywords {
			if strings.Contains(combined, strings.ToLower(kw)) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Magnitude threshold
	if c.MinMagnitude > 0 && evt.Magnitude < c.MinMagnitude {
		return false
	}

	// Geo-fence radius check
	if c.RadiusKm > 0 {
		dist := haversineKm(c.LocationLat, c.LocationLon, evt.Lat, evt.Lon)
		if dist > c.RadiusKm {
			return false
		}
	}

	return true
}

// RuleEngine evaluates alert rules against events and dispatches to matching channels.
type RuleEngine struct {
	rules      []*AlertRule
	dispatcher *Dispatcher
}

// NewRuleEngine creates a rule engine.
func NewRuleEngine(dispatcher *Dispatcher) *RuleEngine {
	return &RuleEngine{dispatcher: dispatcher}
}

// SetRules replaces the loaded rules.
func (re *RuleEngine) SetRules(rules []*AlertRule) {
	re.rules = rules
}

// AddRule adds a rule.
func (re *RuleEngine) AddRule(rule *AlertRule) {
	re.rules = append(re.rules, rule)
}

// Evaluate checks an event against all enabled rules and dispatches alerts
// for matching rules. Returns the combined log entries.
func (re *RuleEngine) Evaluate(ctx context.Context, evt EventData, eventID int64, eventURL string) []LogEntry {
	var allEntries []LogEntry

	for _, rule := range re.rules {
		if !rule.Enabled {
			continue
		}
		if !rule.Matches(evt) {
			continue
		}

		log.Printf("[rules] Rule %q matched event %d", rule.Name, eventID)

		alert := Alert{
			EventID:  eventID,
			Title:    evt.Title,
			Body:     evt.Description,
			Severity: evt.Severity,
			Category: evt.Category,
			URL:      eventURL,
		}

		// If actions specify "*" (all channels), dispatch to all.
		// Otherwise dispatch only to named channels.
		if len(rule.Actions.Channels) == 1 && rule.Actions.Channels[0] == "*" {
			entries := re.dispatcher.Dispatch(ctx, alert)
			allEntries = append(allEntries, entries...)
		} else {
			// Dispatch to specific channels only — create a filtered dispatcher
			for _, ch := range re.dispatcher.channels {
				if !ch.Enabled() {
					continue
				}
				for _, target := range rule.Actions.Channels {
					if strings.EqualFold(ch.Name(), target) {
						entry := LogEntry{
							Channel: ch.Name(),
							EventID: eventID,
							SentAt:  time.Now().UTC(),
						}
						if err := ch.Send(ctx, alert); err != nil {
							entry.Status = "error"
							entry.Error = err.Error()
						} else {
							entry.Status = "ok"
						}
						allEntries = append(allEntries, entry)
					}
				}
			}
		}
	}

	return allEntries
}
