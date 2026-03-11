package notify

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// NtfyChannel sends alerts via ntfy.sh push notifications.
type NtfyChannel struct {
	server  string
	topic   string
	enabled bool
	client  *http.Client
}

// NewNtfyChannel creates an ntfy notification channel.
func NewNtfyChannel(server, topic string, enabled bool) *NtfyChannel {
	if server == "" {
		server = "https://ntfy.sh"
	}
	return &NtfyChannel{
		server:  strings.TrimRight(server, "/"),
		topic:   topic,
		enabled: enabled,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

// Name returns "ntfy".
func (n *NtfyChannel) Name() string { return "ntfy" }

// Enabled returns whether this channel is active.
func (n *NtfyChannel) Enabled() bool { return n.enabled && n.topic != "" }

// ntfyPriority maps severity to ntfy priority (1-5).
func ntfyPriority(severity string) string {
	switch severity {
	case "info":
		return "2"
	case "watch":
		return "3"
	case "warning":
		return "4"
	case "alert":
		return "5"
	case "critical":
		return "5"
	default:
		return "3"
	}
}

// ntfyTags returns tags based on category for emoji display in ntfy clients.
func ntfyTags(category string) string {
	switch strings.ToLower(category) {
	case "earthquake":
		return "earth_americas,shake"
	case "fire", "wildfire":
		return "fire"
	case "weather", "storm", "hurricane", "tornado":
		return "cloud,wind_face"
	case "flood":
		return "ocean"
	case "volcano":
		return "volcano"
	case "tsunami":
		return "ocean,warning"
	case "conflict", "military":
		return "crossed_swords"
	case "cyber":
		return "shield"
	case "health", "disease":
		return "biohazard"
	case "space", "satellite":
		return "satellite"
	case "financial":
		return "chart_with_downwards_trend"
	case "test":
		return "test_tube"
	default:
		return "rotating_light"
	}
}

// Send delivers an alert via ntfy.
func (n *NtfyChannel) Send(ctx context.Context, alert Alert) error {
	if n.topic == "" {
		return fmt.Errorf("ntfy topic not configured")
	}

	url := fmt.Sprintf("%s/%s", n.server, n.topic)
	body := alert.Body
	if body == "" {
		body = alert.Title
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBufferString(body))
	if err != nil {
		return fmt.Errorf("ntfy: create request: %w", err)
	}

	req.Header.Set("Title", fmt.Sprintf("SENTINEL %s — %s", strings.ToUpper(alert.Severity), alert.Title))
	req.Header.Set("Priority", ntfyPriority(alert.Severity))
	req.Header.Set("Tags", ntfyTags(alert.Category))
	if alert.URL != "" {
		req.Header.Set("Click", alert.URL)
	}

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("ntfy: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ntfy: unexpected status %d", resp.StatusCode)
	}
	return nil
}

// Test sends a test notification via ntfy.
func (n *NtfyChannel) Test(ctx context.Context) error {
	return n.Send(ctx, Alert{
		Title:    "SENTINEL notification test",
		Body:     "If you see this notification, ntfy is working correctly.",
		Severity: "info",
		Category: "test",
	})
}
