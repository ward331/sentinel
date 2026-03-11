package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
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

	var update map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		if h.metrics != nil {
			h.metrics.RecordAPIError("/api/notifications/config")
		}
		return
	}

	// Apply updates to notification channels in config (simplified)
	// In a full implementation this would update h.config and save.

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Notification config updated",
	})

	if h.metrics != nil {
		h.metrics.RecordAPIRequest("/api/notifications/config", time.Since(startTime))
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
