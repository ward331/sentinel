package model

// NotificationConfig holds all notification channel configurations.
type NotificationConfig struct {
	Telegram TelegramConfig `json:"telegram"`
	Email    EmailConfig    `json:"email"`
	Slack    SlackConfig    `json:"slack"`
	Discord  DiscordConfig  `json:"discord"`
	Ntfy     NtfyConfig     `json:"ntfy"`
	Pushover PushoverConfig `json:"pushover"`
}

// TelegramConfig holds Telegram bot notification settings.
type TelegramConfig struct {
	Enabled  bool   `json:"enabled"`
	BotToken string `json:"bot_token"`
	ChatID   string `json:"chat_id"`
}

// EmailConfig holds email notification settings.
type EmailConfig struct {
	Enabled    bool   `json:"enabled"`
	SMTPHost   string `json:"smtp_host"`
	SMTPPort   int    `json:"smtp_port"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	FromAddr   string `json:"from_addr"`
	ToAddr     string `json:"to_addr"`
}

// SlackConfig holds Slack webhook notification settings.
type SlackConfig struct {
	Enabled    bool   `json:"enabled"`
	WebhookURL string `json:"webhook_url"`
	Channel    string `json:"channel"`
}

// DiscordConfig holds Discord webhook notification settings.
type DiscordConfig struct {
	Enabled    bool   `json:"enabled"`
	WebhookURL string `json:"webhook_url"`
}

// NtfyConfig holds ntfy notification settings.
type NtfyConfig struct {
	Enabled  bool   `json:"enabled"`
	ServerURL string `json:"server_url"`
	Topic    string `json:"topic"`
	Token    string `json:"token,omitempty"`
}

// PushoverConfig holds Pushover notification settings.
type PushoverConfig struct {
	Enabled bool   `json:"enabled"`
	AppKey  string `json:"app_key"`
	UserKey string `json:"user_key"`
}

// NotificationLog records a sent notification for audit purposes.
type NotificationLog struct {
	ID      int64  `json:"id"`
	Channel string `json:"channel"`
	EventID int64  `json:"event_id"`
	SentAt  string `json:"sent_at"`
	Status  string `json:"status"`
	Error   string `json:"error,omitempty"`
}
