package notify

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// PushoverChannel sends alerts via the Pushover API.
type PushoverChannel struct {
	userKey  string
	apiToken string
	enabled  bool
	client   *http.Client
}

// NewPushoverChannel creates a Pushover notification channel.
func NewPushoverChannel(userKey, apiToken string, enabled bool) *PushoverChannel {
	return &PushoverChannel{
		userKey:  userKey,
		apiToken: apiToken,
		enabled:  enabled,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

// Name returns "pushover".
func (p *PushoverChannel) Name() string { return "pushover" }

// Enabled returns whether this channel is active.
func (p *PushoverChannel) Enabled() bool {
	return p.enabled && p.userKey != "" && p.apiToken != ""
}

// pushoverPriority maps severity to Pushover priorities (-2 to 2).
// -2 = no notification, -1 = quiet, 0 = normal, 1 = high, 2 = emergency
func pushoverPriority(severity string) string {
	switch severity {
	case "info":
		return "-1"
	case "watch":
		return "0"
	case "warning":
		return "0"
	case "alert":
		return "1"
	case "critical":
		return "2"
	default:
		return "0"
	}
}

// Send delivers an alert via Pushover.
func (p *PushoverChannel) Send(ctx context.Context, alert Alert) error {
	if p.userKey == "" || p.apiToken == "" {
		return fmt.Errorf("pushover not configured: missing user_key or api_token")
	}

	emoji := SeverityEmoji(alert.Severity)
	title := fmt.Sprintf("%s SENTINEL %s", emoji, strings.ToUpper(alert.Severity))

	form := url.Values{}
	form.Set("token", p.apiToken)
	form.Set("user", p.userKey)
	form.Set("title", title)
	form.Set("message", fmt.Sprintf("%s\n\n%s", alert.Title, alert.Body))
	form.Set("priority", pushoverPriority(alert.Severity))
	form.Set("html", "1")

	// Emergency priority requires retry and expire parameters
	if pushoverPriority(alert.Severity) == "2" {
		form.Set("retry", "300")  // retry every 5 minutes
		form.Set("expire", "3600") // expire after 1 hour
	}

	if alert.URL != "" {
		form.Set("url", alert.URL)
		form.Set("url_title", "View Event")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.pushover.net/1/messages.json",
		strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("pushover: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("pushover: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("pushover: unexpected status %d", resp.StatusCode)
	}
	return nil
}

// Test sends a test notification via Pushover.
func (p *PushoverChannel) Test(ctx context.Context) error {
	return p.Send(ctx, Alert{
		Title:    "SENTINEL notification test",
		Body:     "If you see this notification, Pushover is working correctly.",
		Severity: "info",
		Category: "test",
	})
}
