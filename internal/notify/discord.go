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

// DiscordChannel sends alerts via a Discord webhook.
type DiscordChannel struct {
	webhookURL string
	enabled    bool
	client     *http.Client
}

// NewDiscordChannel creates a Discord notification channel.
func NewDiscordChannel(webhookURL string, enabled bool) *DiscordChannel {
	return &DiscordChannel{
		webhookURL: webhookURL,
		enabled:    enabled,
		client:     &http.Client{Timeout: 10 * time.Second},
	}
}

// Name returns "discord".
func (d *DiscordChannel) Name() string { return "discord" }

// Enabled returns whether this channel is active.
func (d *DiscordChannel) Enabled() bool { return d.enabled && d.webhookURL != "" }

// discordEmbed represents a Discord embed object.
type discordEmbed struct {
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Color       int            `json:"color"`
	Fields      []discordField `json:"fields,omitempty"`
	Footer      *discordFooter `json:"footer,omitempty"`
	Timestamp   string         `json:"timestamp,omitempty"`
}

type discordField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

type discordFooter struct {
	Text string `json:"text"`
}

// Send delivers an alert via Discord webhook with an embed.
func (d *DiscordChannel) Send(ctx context.Context, alert Alert) error {
	if d.webhookURL == "" {
		return fmt.Errorf("discord not configured: missing webhook_url")
	}

	emoji := SeverityEmoji(alert.Severity)
	title := fmt.Sprintf("%s SENTINEL %s", emoji, strings.ToUpper(alert.Severity))

	fields := []discordField{
		{Name: "Event", Value: alert.Title, Inline: false},
	}
	if alert.Category != "" {
		fields = append(fields, discordField{Name: "Category", Value: alert.Category, Inline: true})
	}
	fields = append(fields, discordField{Name: "Severity", Value: alert.Severity, Inline: true})
	if alert.URL != "" {
		fields = append(fields, discordField{Name: "Link", Value: fmt.Sprintf("[View Event](%s)", alert.URL), Inline: true})
	}

	ts := alert.OccurredAt.Format(time.RFC3339)
	if alert.OccurredAt.IsZero() {
		ts = time.Now().UTC().Format(time.RFC3339)
	}

	embed := discordEmbed{
		Title:       title,
		Description: alert.Body,
		Color:       SeverityColor(alert.Severity),
		Fields:      fields,
		Footer:      &discordFooter{Text: "SENTINEL World Monitoring"},
		Timestamp:   ts,
	}

	payload := map[string]interface{}{
		"embeds": []discordEmbed{embed},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("discord: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("discord: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("discord: send: %w", err)
	}
	defer resp.Body.Close()

	// Discord returns 204 No Content on success
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("discord: unexpected status %d", resp.StatusCode)
	}
	return nil
}

// Test sends a test message via Discord.
func (d *DiscordChannel) Test(ctx context.Context) error {
	return d.Send(ctx, Alert{
		Title:    "SENTINEL notification test",
		Body:     "If you see this message, Discord notifications are working.",
		Severity: "info",
		Category: "test",
	})
}
