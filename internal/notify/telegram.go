package notify

import (
	"fmt"
)

// TelegramChannel sends alerts via the Telegram Bot API.
type TelegramChannel struct {
	botToken string
	chatID   string
}

// NewTelegramChannel creates a Telegram notification channel.
func NewTelegramChannel(botToken, chatID string) *TelegramChannel {
	return &TelegramChannel{botToken: botToken, chatID: chatID}
}

// Name returns "telegram".
func (t *TelegramChannel) Name() string { return "telegram" }

// Send delivers an alert via Telegram.
// Stub — actual HTTP call to api.telegram.org in Stage G5.
func (t *TelegramChannel) Send(alert Alert) error {
	if t.botToken == "" || t.chatID == "" {
		return fmt.Errorf("telegram not configured")
	}
	// TODO: POST to https://api.telegram.org/bot<token>/sendMessage
	return nil
}
