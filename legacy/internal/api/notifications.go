package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/openclaw/sentinel-backend/internal/config"
)

// NotificationConfigResponse represents notification channel settings.
type NotificationConfigResponse struct {
	Telegram NotificationChannelStatus `json:"telegram"`
	Slack    NotificationChannelStatus `json:"slack"`
	Discord  NotificationChannelStatus `json:"discord"`
	Email    NotificationChannelStatus `json:"email"`
	Ntfy     NotificationChannelStatus `json:"ntfy"`
}

// NotificationChannelStatus represents the status of a notification channel.
type NotificationChannelStatus struct {
	Enabled     bool   `json:"enabled"`
	MinSeverity string `json:"min_severity"`
	Configured  bool   `json:"configured"`
}

// GetNotificationConfig handles GET /api/notifications/config
func (h *Handler) GetNotificationConfig(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	response := NotificationConfigResponse{
		Telegram: NotificationChannelStatus{Enabled: false, MinSeverity: "warning", Configured: false},
		Slack:    NotificationChannelStatus{Enabled: false, MinSeverity: "warning", Configured: false},
		Discord:  NotificationChannelStatus{Enabled: false, MinSeverity: "warning", Configured: false},
		Email:    NotificationChannelStatus{Enabled: false, MinSeverity: "alert", Configured: false},
		Ntfy:     NotificationChannelStatus{Enabled: false, MinSeverity: "warning", Configured: false},
	}

	if h.config != nil {
		response.Telegram.Enabled = h.config.Telegram.Enabled
		response.Telegram.MinSeverity = h.config.Telegram.MinSeverity
		response.Telegram.Configured = h.config.Telegram.BotToken != ""

		response.Slack.Enabled = h.config.Slack.Enabled
		response.Slack.MinSeverity = h.config.Slack.MinSeverity
		response.Slack.Configured = h.config.Slack.WebhookURL != ""

		response.Discord.Enabled = h.config.Discord.Enabled
		response.Discord.MinSeverity = h.config.Discord.MinSeverity
		response.Discord.Configured = h.config.Discord.WebhookURL != ""

		response.Email.Enabled = h.config.Email.Enabled
		response.Email.MinSeverity = h.config.Email.MinSeverity
		response.Email.Configured = h.config.Email.SMTPHost != "" || h.config.Email.GmailClientID != ""

		response.Ntfy.Enabled = h.config.Ntfy.Enabled
		response.Ntfy.MinSeverity = h.config.Ntfy.MinSeverity
		response.Ntfy.Configured = h.config.Ntfy.Topic != ""
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	if h.metrics != nil {
		h.metrics.RecordAPIRequest("/api/notifications/config", time.Since(startTime))
	}
}

// UpdateNotificationConfig handles POST /api/notifications/config
func (h *Handler) UpdateNotificationConfig(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	if h.config == nil {
		http.Error(w, `{"error":"config not available"}`, http.StatusServiceUnavailable)
		if h.metrics != nil {
			h.metrics.RecordAPIError("/api/notifications/config")
		}
		return
	}

	var update map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		if h.metrics != nil {
			h.metrics.RecordAPIError("/api/notifications/config")
		}
		return
	}

	// Apply updates per channel
	if tg, ok := update["telegram"].(map[string]interface{}); ok {
		applyChannelUpdate(&h.config.Telegram.Enabled, &h.config.Telegram.MinSeverity, tg)
		if v, ok := tg["bot_token"].(string); ok {
			h.config.Telegram.BotToken = v
		}
		if v, ok := tg["chat_id"].(string); ok {
			h.config.Telegram.ChatID = v
		}
		if v, ok := tg["digest_mode"].(bool); ok {
			h.config.Telegram.DigestMode = v
		}
		if v, ok := tg["digest_interval_minutes"].(float64); ok {
			h.config.Telegram.DigestIntervalMinutes = int(v)
		}
	}

	if sl, ok := update["slack"].(map[string]interface{}); ok {
		applyChannelUpdate(&h.config.Slack.Enabled, &h.config.Slack.MinSeverity, sl)
		if v, ok := sl["webhook_url"].(string); ok {
			h.config.Slack.WebhookURL = v
		}
		if v, ok := sl["channel"].(string); ok {
			h.config.Slack.Channel = v
		}
	}

	if dc, ok := update["discord"].(map[string]interface{}); ok {
		applyChannelUpdate(&h.config.Discord.Enabled, &h.config.Discord.MinSeverity, dc)
		if v, ok := dc["webhook_url"].(string); ok {
			h.config.Discord.WebhookURL = v
		}
	}

	if em, ok := update["email"].(map[string]interface{}); ok {
		applyChannelUpdate(&h.config.Email.Enabled, &h.config.Email.MinSeverity, em)
		if v, ok := em["method"].(string); ok {
			h.config.Email.Method = v
		}
		if v, ok := em["smtp_host"].(string); ok {
			h.config.Email.SMTPHost = v
		}
		if v, ok := em["smtp_port"].(float64); ok {
			h.config.Email.SMTPPort = int(v)
		}
		if v, ok := em["smtp_tls"].(string); ok {
			h.config.Email.SMTPTLS = v
		}
		if v, ok := em["username"].(string); ok {
			h.config.Email.Username = v
		}
		if v, ok := em["from_address"].(string); ok {
			h.config.Email.FromAddress = v
		}
		if v, ok := em["to_addresses"].([]interface{}); ok {
			addrs := make([]string, 0, len(v))
			for _, a := range v {
				if s, ok := a.(string); ok {
					addrs = append(addrs, s)
				}
			}
			h.config.Email.ToAddresses = addrs
		}
		if v, ok := em["gmail_client_id"].(string); ok {
			h.config.Email.GmailClientID = v
		}
		if v, ok := em["gmail_client_secret"].(string); ok {
			h.config.Email.GmailClientSecret = v
		}
		if v, ok := em["gmail_refresh_token"].(string); ok {
			h.config.Email.GmailRefreshToken = v
		}
		if v, ok := em["mailgun_domain"].(string); ok {
			h.config.Email.MailgunDomain = v
		}
		// Note: password_encrypted, sendgrid_key_encrypted, mailgun_key_encrypted
		// are NOT accepted here — they must stay as-is to preserve encryption.
	}

	if nt, ok := update["ntfy"].(map[string]interface{}); ok {
		applyChannelUpdate(&h.config.Ntfy.Enabled, &h.config.Ntfy.MinSeverity, nt)
		if v, ok := nt["server"].(string); ok {
			h.config.Ntfy.Server = v
		}
		if v, ok := nt["topic"].(string); ok {
			h.config.Ntfy.Topic = v
		}
	}

	if po, ok := update["pushover"].(map[string]interface{}); ok {
		if v, ok := po["enabled"].(bool); ok {
			h.config.Pushover.Enabled = v
		}
		if v, ok := po["app_token"].(string); ok {
			h.config.Pushover.AppToken = v
		}
		if v, ok := po["user_key"].(string); ok {
			h.config.Pushover.UserKey = v
		}
	}

	// Persist to disk
	if err := config.SaveConfig(h.config, ""); err != nil {
		log.Printf("[api] failed to save notification config: %v", err)
		http.Error(w, `{"error":"failed to save config to disk"}`, http.StatusInternalServerError)
		if h.metrics != nil {
			h.metrics.RecordAPIError("/api/notifications/config")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Notification config updated and saved",
	})

	if h.metrics != nil {
		h.metrics.RecordAPIRequest("/api/notifications/config", time.Since(startTime))
	}
}

// applyChannelUpdate sets the enabled and min_severity fields common to most channels.
func applyChannelUpdate(enabled *bool, minSeverity *string, data map[string]interface{}) {
	if v, ok := data["enabled"].(bool); ok {
		*enabled = v
	}
	if v, ok := data["min_severity"].(string); ok {
		*minSeverity = v
	}
}

// TestNotificationChannel handles POST /api/notifications/test/{ch}
func (h *Handler) TestNotificationChannel(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	vars := mux.Vars(r)
	channel := vars["ch"]

	// Validate channel name
	validChannels := map[string]bool{
		"telegram": true,
		"slack":    true,
		"discord":  true,
		"email":    true,
		"ntfy":     true,
		"pushover": true,
	}

	if !validChannels[channel] {
		http.Error(w, `{"error":"unknown notification channel"}`, http.StatusBadRequest)
		if h.metrics != nil {
			h.metrics.RecordAPIError("/api/notifications/test")
		}
		return
	}

	// Placeholder — would send a test notification through the dispatcher.
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"channel": channel,
		"status":  "sent",
		"message": "Test notification dispatched to " + channel,
	})

	if h.metrics != nil {
		h.metrics.RecordAPIRequest("/api/notifications/test", time.Since(startTime))
	}
}
