package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// SlackChannel sends alerts via a Slack incoming webhook.
type SlackChannel struct {
	webhookURL string
	enabled    bool
	client     *http.Client
}

// NewSlackChannel creates a Slack notification channel.
func NewSlackChannel(webhookURL string, enabled bool) *SlackChannel {
	return &SlackChannel{
		webhookURL: webhookURL,
		enabled:    enabled,
		client:     &http.Client{Timeout: 10 * time.Second},
	}
}

// Name returns "slack".
func (s *SlackChannel) Name() string { return "slack" }

// Enabled returns whether this channel is active.
func (s *SlackChannel) Enabled() bool { return s.enabled && s.webhookURL != "" }

// slackColorForSeverity returns a Slack attachment color hex string.
func slackColorForSeverity(severity string) string {
	switch severity {
	case "info":
		return "#00cc00"
	case "watch":
		return "#ffcc00"
	case "warning":
		return "#ff8800"
	case "alert":
		return "#ff0000"
	case "critical":
		return "#990000"
	default:
		return "#888888"
	}
}

// Send delivers an alert via Slack webhook.
func (s *SlackChannel) Send(ctx context.Context, alert Alert) error {
	if s.webhookURL == "" {
		return fmt.Errorf("slack not configured: missing webhook_url")
	}

	emoji := SeverityEmoji(alert.Severity)
	text := fmt.Sprintf("%s *SENTINEL %s*", emoji, strings.ToUpper(alert.Severity))

	fields := []map[string]interface{}{
		{"title": "Event", "value": alert.Title, "short": false},
	}
	if alert.Category != "" {
		fields = append(fields, map[string]interface{}{"title": "Category", "value": alert.Category, "short": true})
	}
	fields = append(fields, map[string]interface{}{"title": "Severity", "value": alert.Severity, "short": true})
	if alert.URL != "" {
		fields = append(fields, map[string]interface{}{"title": "Link", "value": fmt.Sprintf("<%s|View Event>", alert.URL), "short": true})
	}

	payload := map[string]interface{}{
		"text": text,
		"attachments": []map[string]interface{}{
			{
				"color":  slackColorForSeverity(alert.Severity),
				"text":   alert.Body,
				"fields": fields,
				"footer": "SENTINEL World Monitoring",
				"ts":     alert.OccurredAt.Unix(),
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("slack: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("slack: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("slack: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack: unexpected status %d", resp.StatusCode)
	}
	return nil
}

// Test sends a test message via Slack.
func (s *SlackChannel) Test(ctx context.Context) error {
	return s.Send(ctx, Alert{
		Title:    "SENTINEL notification test",
		Body:     "If you see this message, Slack notifications are working.",
		Severity: "info",
		Category: "test",
	})
}
