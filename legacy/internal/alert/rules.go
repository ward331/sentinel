package alert

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/openclaw/sentinel-backend/internal/model"
)

// Rule defines an alert rule
type Rule struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Enabled     bool        `json:"enabled"`
	Conditions  []Condition `json:"conditions"`
	Actions     []Action    `json:"actions"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// Condition defines a condition for triggering an alert
type Condition struct {
	Field     string      `json:"field"`
	Operator  string      `json:"operator"`
	Value     interface{} `json:"value"`
}

// Action defines an action to take when alert triggers
type Action struct {
	Type   string            `json:"type"`
	Config map[string]string `json:"config"`
}

// RuleStore is the interface the rule engine needs for persistence.
// Implemented by *storage.Storage.
type RuleStore interface {
	GetAlertRules() ([]model.AlertRule, error)
	GetAlertRule(id int64) (*model.AlertRule, error)
	CreateAlertRule(rule *model.AlertRule) error
	UpdateAlertRule(id int64, rule *model.AlertRule) error
	DeleteAlertRule(id int64) error
}

// RuleEngine evaluates events against rules
type RuleEngine struct {
	rules []Rule
	store RuleStore // nil = legacy in-memory only
}

// NewRuleEngine creates a new rule engine with hardcoded defaults (no persistence).
func NewRuleEngine() *RuleEngine {
	return &RuleEngine{
		rules: defaultRules(),
	}
}

// NewRuleEngineWithStore creates a rule engine backed by SQLite.
// It loads existing rules from the database. If the DB is empty it seeds
// the default rules and persists them.
func NewRuleEngineWithStore(store RuleStore) *RuleEngine {
	re := &RuleEngine{store: store}
	if err := re.loadFromDB(); err != nil {
		log.Printf("[alert] failed to load rules from DB, falling back to defaults: %v", err)
		re.rules = defaultRules()
		return re
	}
	// Seed defaults if DB was empty
	if len(re.rules) == 0 {
		log.Println("[alert] no rules in DB, seeding defaults")
		for _, r := range defaultRules() {
			re.rules = append(re.rules, r)
			if err := re.persistRule(&r); err != nil {
				log.Printf("[alert] failed to seed rule %q: %v", r.Name, err)
			}
		}
		// Reload so IDs are correct
		_ = re.loadFromDB()
	}
	log.Printf("[alert] loaded %d alert rules from DB", len(re.rules))
	return re
}

// loadFromDB reads all rules from storage and converts them to alert.Rule.
func (re *RuleEngine) loadFromDB() error {
	if re.store == nil {
		return nil
	}
	dbRules, err := re.store.GetAlertRules()
	if err != nil {
		return err
	}
	rules := make([]Rule, 0, len(dbRules))
	for _, dr := range dbRules {
		r, err := modelToRule(&dr)
		if err != nil {
			log.Printf("[alert] skipping rule %d (%s): %v", dr.ID, dr.Name, err)
			continue
		}
		rules = append(rules, *r)
	}
	re.rules = rules
	return nil
}

// persistRule converts an alert.Rule to model.AlertRule and inserts it.
// On success the rule's ID is updated to the DB-assigned value.
func (re *RuleEngine) persistRule(r *Rule) error {
	mr, err := ruleToModel(r)
	if err != nil {
		return err
	}
	if err := re.store.CreateAlertRule(mr); err != nil {
		return err
	}
	r.ID = strconv.FormatInt(mr.ID, 10)
	return nil
}

// defaultRules returns the hardcoded seed rules.
func defaultRules() []Rule {
	return []Rule{
		{
			ID:          "major-earthquake",
			Name:        "Major Earthquake Alert",
			Description: "Alert for earthquakes magnitude 6.0 or higher",
			Enabled:     true,
			Conditions: []Condition{
				{Field: "category", Operator: "equals", Value: "earthquake"},
				{Field: "magnitude", Operator: "gte", Value: 6.0},
			},
			Actions: []Action{
				{Type: "log", Config: map[string]string{"level": "warn"}},
			},
		},
		{
			ID:          "critical-severity",
			Name:        "Critical Severity Alert",
			Description: "Alert for events with critical severity",
			Enabled:     true,
			Conditions: []Condition{
				{Field: "severity", Operator: "equals", Value: "critical"},
			},
			Actions: []Action{
				{Type: "log", Config: map[string]string{"level": "error"}},
			},
		},
		{
			ID:          "usgs-major",
			Name:        "USGS Major Event",
			Description: "Alert for major USGS events (magnitude 5.0+)",
			Enabled:     true,
			Conditions: []Condition{
				{Field: "source", Operator: "equals", Value: "usgs"},
				{Field: "magnitude", Operator: "gte", Value: 5.0},
			},
			Actions: []Action{
				{Type: "log", Config: map[string]string{"level": "info"}},
			},
		},
	}
}

// Evaluate checks an event against all rules
func (e *RuleEngine) Evaluate(event *model.Event) []Rule {
	var triggered []Rule
	
	for _, rule := range e.rules {
		if !rule.Enabled {
			continue
		}
		
		if e.evaluateRule(event, rule) {
			triggered = append(triggered, rule)
			e.executeActions(event, rule)
		}
	}
	
	return triggered
}

// evaluateRule checks if an event matches a rule's conditions
func (e *RuleEngine) evaluateRule(event *model.Event, rule Rule) bool {
	for _, condition := range rule.Conditions {
		if !e.evaluateCondition(event, condition) {
			return false
		}
	}
	return true
}

// evaluateCondition checks a single condition against an event
func (e *RuleEngine) evaluateCondition(event *model.Event, condition Condition) bool {
	switch condition.Field {
	case "category":
		return e.compareString(event.Category, condition.Operator, condition.Value)
	case "severity":
		return e.compareString(string(event.Severity), condition.Operator, condition.Value)
	case "source":
		return e.compareString(event.Source, condition.Operator, condition.Value)
	case "magnitude":
		return e.compareFloat(event.Magnitude, condition.Operator, condition.Value)
	case "title":
		return e.compareString(event.Title, condition.Operator, condition.Value)
	default:
		// Check metadata
		if val, ok := event.Metadata[condition.Field]; ok {
			return e.compareString(val, condition.Operator, condition.Value)
		}
		return false
	}
}

// compareString compares string values
func (e *RuleEngine) compareString(value, operator string, expected interface{}) bool {
	expectedStr, ok := expected.(string)
	if !ok {
		return false
	}
	
	switch operator {
	case "equals":
		return value == expectedStr
	case "contains":
		// Simple contains check
		for i := 0; i <= len(value)-len(expectedStr); i++ {
			if value[i:i+len(expectedStr)] == expectedStr {
				return true
			}
		}
		return false
	case "starts_with":
		return len(value) >= len(expectedStr) && value[:len(expectedStr)] == expectedStr
	case "ends_with":
		return len(value) >= len(expectedStr) && value[len(value)-len(expectedStr):] == expectedStr
	default:
		return false
	}
}

// compareFloat compares float values
func (e *RuleEngine) compareFloat(value float64, operator string, expected interface{}) bool {
	expectedFloat, ok := expected.(float64)
	if !ok {
		// Try to convert from int
		if expectedInt, ok := expected.(int); ok {
			expectedFloat = float64(expectedInt)
		} else {
			return false
		}
	}
	
	switch operator {
	case "equals":
		return value == expectedFloat
	case "gte":
		return value >= expectedFloat
	case "gt":
		return value > expectedFloat
	case "lte":
		return value <= expectedFloat
	case "lt":
		return value < expectedFloat
	default:
		return false
	}
}

// executeActions executes all actions for a triggered rule
func (e *RuleEngine) executeActions(event *model.Event, rule Rule) {
	for _, action := range rule.Actions {
		switch action.Type {
		case "log":
			level := action.Config["level"]
			msg := fmt.Sprintf("[ALERT] Rule '%s' triggered for event: %s (ID: %s)", 
				rule.Name, event.Title, event.ID)
			
			switch level {
			case "error":
				fmt.Printf("ERROR: %s\n", msg)
			case "warn":
				fmt.Printf("WARN: %s\n", msg)
			case "info":
				fmt.Printf("INFO: %s\n", msg)
			default:
				fmt.Printf("ALERT: %s\n", msg)
			}
			
		case "webhook":
			e.executeWebhookAction(event, rule, action)
			
		case "slack":
			e.executeSlackAction(event, rule, action)
			
		case "discord":
			e.executeDiscordAction(event, rule, action)
			
		case "teams":
			e.executeTeamsAction(event, rule, action)
			
		case "email":
			e.executeEmailAction(event, rule, action)
		}
	}
}

// executeWebhookAction sends an HTTP POST to a webhook URL
func (e *RuleEngine) executeWebhookAction(event *model.Event, rule Rule, action Action) {
	url, ok := action.Config["url"]
	if !ok || url == "" {
		fmt.Printf("[ALERT] Webhook action missing URL for rule '%s'\n", rule.Name)
		return
	}
	
	// Prepare webhook payload
	payload := map[string]interface{}{
		"rule_id":      rule.ID,
		"rule_name":    rule.Name,
		"event_id":     event.ID,
		"event_title":  event.Title,
		"event_source": event.Source,
		"category":     event.Category,
		"severity":     event.Severity,
		"magnitude":    event.Magnitude,
		"occurred_at":  event.OccurredAt,
		"triggered_at": time.Now().UTC(),
		"full_event":   event,
	}
	
	jsonData, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("[ALERT] Failed to marshal webhook payload for rule '%s': %v\n", rule.Name, err)
		return
	}
	
	// Send HTTP POST request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("[ALERT] Failed to send webhook for rule '%s': %v\n", rule.Name, err)
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		fmt.Printf("[ALERT] Webhook sent successfully for rule '%s' (status: %d)\n", rule.Name, resp.StatusCode)
	} else {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("[ALERT] Webhook failed for rule '%s': %d - %s\n", rule.Name, resp.StatusCode, string(body))
	}
}

// executeEmailAction sends an email notification
func (e *RuleEngine) executeEmailAction(event *model.Event, rule Rule, action Action) {
	// For now, just log that email would be sent
	// In a full implementation, this would use SMTP configuration
	// from the global config and send actual emails
	
	to := action.Config["to"]
	if to == "" {
		to = "admin@example.com"
	}
	
	subject := fmt.Sprintf("[SENTINEL ALERT] %s: %s", rule.Name, event.Title)
	body := fmt.Sprintf(`
Alert Rule: %s
Event: %s
Source: %s
Category: %s
Severity: %s
Magnitude: %.2f
Time: %s
Location: %s
Description: %s

Full event details available in SENTINEL dashboard.
`,
		rule.Name,
		event.Title,
		event.Source,
		event.Category,
		event.Severity,
		event.Magnitude,
		event.OccurredAt.Format("2006-01-02 15:04:05 UTC"),
		e.formatLocation(event.Location),
		event.Description,
	)
	
	fmt.Printf("[ALERT] Email would be sent to %s for rule '%s'\n", to, rule.Name)
	fmt.Printf("Subject: %s\n", subject)
	fmt.Printf("Body: %s\n", body)
	
	// TODO: Integrate with actual SMTP configuration
	// This is a placeholder implementation
}

// executeSlackAction sends a notification to Slack
func (e *RuleEngine) executeSlackAction(event *model.Event, rule Rule, action Action) {
	webhookURL, ok := action.Config["webhook_url"]
	if !ok || webhookURL == "" {
		fmt.Printf("[ALERT] Slack action missing webhook_url for rule '%s'\n", rule.Name)
		return
	}
	
	channel := action.Config["channel"]
	if channel == "" {
		channel = "#alerts"
	}
	
	// Slack webhook payload
	payload := map[string]interface{}{
		"channel": channel,
		"text": fmt.Sprintf("🚨 *%s Alert*\n*Event:* %s\n*Source:* %s\n*Severity:* %s\n*Time:* %s\n*Rule:* %s",
			rule.Name,
			event.Title,
			event.Source,
			event.Severity,
			event.OccurredAt.Format("2006-01-02 15:04:05 UTC"),
			rule.Name,
		),
		"attachments": []map[string]interface{}{
			{
				"color": e.getSeverityColor(event.Severity),
				"fields": []map[string]string{
					{"title": "Category", "value": event.Category, "short": "true"},
					{"title": "Magnitude", "value": fmt.Sprintf("%.2f", event.Magnitude), "short": "true"},
					{"title": "Location", "value": e.formatLocation(event.Location), "short": "false"},
					{"title": "Description", "value": event.Description, "short": "false"},
				},
			},
		},
	}
	
	e.sendJSONWebhook(webhookURL, payload, "Slack")
}

// executeDiscordAction sends a notification to Discord
func (e *RuleEngine) executeDiscordAction(event *model.Event, rule Rule, action Action) {
	webhookURL, ok := action.Config["webhook_url"]
	if !ok || webhookURL == "" {
		fmt.Printf("[ALERT] Discord action missing webhook_url for rule '%s'\n", rule.Name)
		return
	}
	
	// Discord webhook payload
	payload := map[string]interface{}{
		"content": fmt.Sprintf("🚨 **%s Alert**", rule.Name),
		"embeds": []map[string]interface{}{
			{
				"title":       event.Title,
				"description": event.Description,
				"color":       e.getSeverityColorDecimal(event.Severity),
				"fields": []map[string]interface{}{
					{"name": "Source", "value": event.Source, "inline": "true"},
					{"name": "Category", "value": event.Category, "inline": "true"},
					{"name": "Severity", "value": string(event.Severity), "inline": "true"},
					{"name": "Magnitude", "value": fmt.Sprintf("%.2f", event.Magnitude), "inline": "true"},
					{"name": "Time", "value": event.OccurredAt.Format("2006-01-02 15:04:05 UTC"), "inline": "false"},
					{"name": "Location", "value": e.formatLocation(event.Location), "inline": "false"},
				},
				"footer": map[string]string{
					"text": fmt.Sprintf("Rule: %s | Event ID: %s", rule.Name, event.ID),
				},
				"timestamp": event.OccurredAt.Format(time.RFC3339),
			},
		},
	}
	
	e.sendJSONWebhook(webhookURL, payload, "Discord")
}

// executeTeamsAction sends a notification to Microsoft Teams
func (e *RuleEngine) executeTeamsAction(event *model.Event, rule Rule, action Action) {
	webhookURL, ok := action.Config["webhook_url"]
	if !ok || webhookURL == "" {
		fmt.Printf("[ALERT] Teams action missing webhook_url for rule '%s'\n", rule.Name)
		return
	}
	
	// Microsoft Teams webhook payload
	payload := map[string]interface{}{
		"@type": "MessageCard",
		"@context": "http://schema.org/extensions",
		"themeColor": e.getSeverityColor(event.Severity),
		"summary": fmt.Sprintf("%s Alert: %s", rule.Name, event.Title),
		"sections": []map[string]interface{}{
			{
				"activityTitle": fmt.Sprintf("🚨 %s Alert", rule.Name),
				"activitySubtitle": event.Title,
				"facts": []map[string]string{
					{"name": "Source", "value": event.Source},
					{"name": "Category", "value": event.Category},
					{"name": "Severity", "value": string(event.Severity)},
					{"name": "Magnitude", "value": fmt.Sprintf("%.2f", event.Magnitude)},
					{"name": "Time", "value": event.OccurredAt.Format("2006-01-02 15:04:05 UTC")},
					{"name": "Location", "value": e.formatLocation(event.Location)},
				},
				"text": event.Description,
			},
		},
		"potentialAction": []map[string]interface{}{
			{
				"@type": "OpenUri",
				"name": "View in Dashboard",
				"targets": []map[string]string{
					{"os": "default", "uri": fmt.Sprintf("http://localhost:8080/events/%s", event.ID)},
				},
			},
		},
	}
	
	e.sendJSONWebhook(webhookURL, payload, "Teams")
}

// sendJSONWebhook sends a JSON payload to a webhook URL
func (e *RuleEngine) sendJSONWebhook(url string, payload map[string]interface{}, service string) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("[ALERT] Failed to marshal %s webhook payload: %v\n", service, err)
		return
	}
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("[ALERT] Failed to send %s webhook: %v\n", service, err)
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		fmt.Printf("[ALERT] %s webhook sent successfully (status: %d)\n", service, resp.StatusCode)
	} else {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("[ALERT] %s webhook failed: %d - %s\n", service, resp.StatusCode, string(body))
	}
}

// getSeverityColor returns a color for Slack/Teams based on severity
func (e *RuleEngine) getSeverityColor(severity model.Severity) string {
	switch severity {
	case model.SeverityCritical:
		return "#ff0000" // Red
	case model.SeverityHigh:
		return "#ff9900" // Orange
	case model.SeverityMedium:
		return "#ffff00" // Yellow
	case model.SeverityLow:
		return "#00ff00" // Green
	default:
		return "#808080" // Gray
	}
}

// getSeverityColorDecimal returns a decimal color for Discord
func (e *RuleEngine) getSeverityColorDecimal(severity model.Severity) int {
	switch severity {
	case model.SeverityCritical:
		return 16711680 // Red
	case model.SeverityHigh:
		return 16753920 // Orange
	case model.SeverityMedium:
		return 16776960 // Yellow
	case model.SeverityLow:
		return 65280    // Green
	default:
		return 8421504  // Gray
	}
}

// GetRules returns all rules
func (e *RuleEngine) GetRules() []Rule {
	return e.rules
}

// AddRule adds a new rule, persisting to DB if a store is configured.
func (e *RuleEngine) AddRule(rule Rule) {
	if rule.CreatedAt.IsZero() {
		rule.CreatedAt = time.Now()
	}
	rule.UpdatedAt = time.Now()

	if e.store != nil {
		if err := e.persistRule(&rule); err != nil {
			log.Printf("[alert] failed to persist new rule %q: %v", rule.Name, err)
		}
	} else if rule.ID == "" {
		rule.ID = fmt.Sprintf("rule-%d", len(e.rules)+1)
	}

	e.rules = append(e.rules, rule)
}

// UpdateRule updates an existing rule, persisting to DB if a store is configured.
func (e *RuleEngine) UpdateRule(id string, updated Rule) bool {
	for i, rule := range e.rules {
		if rule.ID == id {
			updated.ID = id
			updated.CreatedAt = rule.CreatedAt
			updated.UpdatedAt = time.Now()

			if e.store != nil {
				dbID, err := strconv.ParseInt(id, 10, 64)
				if err == nil {
					mr, convErr := ruleToModel(&updated)
					if convErr == nil {
						if dbErr := e.store.UpdateAlertRule(dbID, mr); dbErr != nil {
							log.Printf("[alert] failed to persist update for rule %s: %v", id, dbErr)
						}
					}
				}
			}

			e.rules[i] = updated
			return true
		}
	}
	return false
}

// DeleteRule removes a rule, persisting to DB if a store is configured.
func (e *RuleEngine) DeleteRule(id string) bool {
	for i, rule := range e.rules {
		if rule.ID == id {
			if e.store != nil {
				dbID, err := strconv.ParseInt(id, 10, 64)
				if err == nil {
					if dbErr := e.store.DeleteAlertRule(dbID); dbErr != nil {
						log.Printf("[alert] failed to delete rule %s from DB: %v", id, dbErr)
					}
				}
			}
			e.rules = append(e.rules[:i], e.rules[i+1:]...)
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Conversion helpers between alert.Rule and model.AlertRule
// ---------------------------------------------------------------------------

// ruleToModel converts an alert.Rule to a model.AlertRule for DB storage.
func ruleToModel(r *Rule) (*model.AlertRule, error) {
	condJSON, err := json.Marshal(r.Conditions)
	if err != nil {
		return nil, fmt.Errorf("marshal conditions: %w", err)
	}
	actJSON, err := json.Marshal(r.Actions)
	if err != nil {
		return nil, fmt.Errorf("marshal actions: %w", err)
	}
	mr := &model.AlertRule{
		Name:       r.Name,
		Conditions: json.RawMessage(condJSON),
		Actions:    json.RawMessage(actJSON),
		Enabled:    r.Enabled,
		CreatedAt:  r.CreatedAt.UTC().Format(time.RFC3339),
	}
	if r.ID != "" {
		if dbID, err := strconv.ParseInt(r.ID, 10, 64); err == nil {
			mr.ID = dbID
		}
	}
	return mr, nil
}

// modelToRule converts a model.AlertRule (DB row) to an alert.Rule.
func modelToRule(mr *model.AlertRule) (*Rule, error) {
	r := &Rule{
		ID:      strconv.FormatInt(mr.ID, 10),
		Name:    mr.Name,
		Enabled: mr.Enabled,
	}
	if mr.CreatedAt != "" {
		if t, err := time.Parse(time.RFC3339, mr.CreatedAt); err == nil {
			r.CreatedAt = t
		}
	}
	r.UpdatedAt = r.CreatedAt

	if len(mr.Conditions) > 0 {
		if err := json.Unmarshal(mr.Conditions, &r.Conditions); err != nil {
			return nil, fmt.Errorf("unmarshal conditions: %w", err)
		}
	}
	if len(mr.Actions) > 0 {
		if err := json.Unmarshal(mr.Actions, &r.Actions); err != nil {
			return nil, fmt.Errorf("unmarshal actions: %w", err)
		}
	}
	return r, nil
}

// formatLocation formats a Location struct into a readable string
func (e *RuleEngine) formatLocation(loc model.Location) string {
	if loc.Type == "Point" {
		// Coordinates should be [lon, lat] for Point
		if coords, ok := loc.Coordinates.([]interface{}); ok && len(coords) >= 2 {
			lon, _ := coords[0].(float64)
			lat, _ := coords[1].(float64)
			return fmt.Sprintf("%.4f, %.4f", lat, lon)
		}
	}
	return "Unknown location"
}