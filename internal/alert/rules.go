package alert

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

// RuleEngine evaluates events against rules
type RuleEngine struct {
	rules []Rule
}

// NewRuleEngine creates a new rule engine
func NewRuleEngine() *RuleEngine {
	return &RuleEngine{
		rules: []Rule{
			// Default rules
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
			
		case "email":
			// TODO: Implement email action
			fmt.Printf("[ALERT] Email would be sent for rule '%s' (event: %s)\n", 
				rule.Name, event.ID)
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

// GetRules returns all rules
func (e *RuleEngine) GetRules() []Rule {
	return e.rules
}

// AddRule adds a new rule
func (e *RuleEngine) AddRule(rule Rule) {
	if rule.ID == "" {
		rule.ID = fmt.Sprintf("rule-%d", len(e.rules)+1)
	}
	if rule.CreatedAt.IsZero() {
		rule.CreatedAt = time.Now()
	}
	rule.UpdatedAt = time.Now()
	e.rules = append(e.rules, rule)
}

// UpdateRule updates an existing rule
func (e *RuleEngine) UpdateRule(id string, updated Rule) bool {
	for i, rule := range e.rules {
		if rule.ID == id {
			updated.ID = id
			updated.CreatedAt = rule.CreatedAt
			updated.UpdatedAt = time.Now()
			e.rules[i] = updated
			return true
		}
	}
	return false
}

// DeleteRule removes a rule
func (e *RuleEngine) DeleteRule(id string) bool {
	for i, rule := range e.rules {
		if rule.ID == id {
			e.rules = append(e.rules[:i], e.rules[i+1:]...)
			return true
		}
	}
	return false
}