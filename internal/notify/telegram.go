package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// TelegramChannel sends alerts via the Telegram Bot API.
type TelegramChannel struct {
	botToken string
	chatID   string
	enabled  bool
	client   *http.Client
}

// NewTelegramChannel creates a Telegram notification channel.
func NewTelegramChannel(botToken, chatID string, enabled bool) *TelegramChannel {
	return &TelegramChannel{
		botToken: botToken,
		chatID:   chatID,
		enabled:  enabled,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

// Name returns "telegram".
func (t *TelegramChannel) Name() string { return "telegram" }

// Enabled returns whether this channel is active.
func (t *TelegramChannel) Enabled() bool { return t.enabled && t.botToken != "" && t.chatID != "" }

// Send delivers an alert via Telegram using HTML formatting.
func (t *TelegramChannel) Send(ctx context.Context, alert Alert) error {
	if t.botToken == "" || t.chatID == "" {
		return fmt.Errorf("telegram not configured: missing bot_token or chat_id")
	}

	emoji := SeverityEmoji(alert.Severity)
	text := fmt.Sprintf(
		"%s <b>SENTINEL %s</b>\n\n<b>%s</b>\n%s",
		emoji, alert.Severity, alert.Title, alert.Body,
	)
	if alert.Category != "" {
		text += fmt.Sprintf("\n\nCategory: <code>%s</code>", alert.Category)
	}
	if alert.URL != "" {
		text += fmt.Sprintf("\n<a href=\"%s\">View Event</a>", alert.URL)
	}

	payload := map[string]interface{}{
		"chat_id":    t.chatID,
		"text":       text,
		"parse_mode": "HTML",
		"disable_web_page_preview": true,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("telegram: marshal payload: %w", err)
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.botToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("telegram: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("telegram: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram: unexpected status %d", resp.StatusCode)
	}
	return nil
}

// Test sends a test notification via Telegram.
func (t *TelegramChannel) Test(ctx context.Context) error {
	return t.Send(ctx, Alert{
		Title:    "SENTINEL notification test",
		Body:     "If you see this message, Telegram notifications are working.",
		Severity: "info",
		Category: "test",
	})
}
