package api

import (
	"encoding/json"
	"net/http"
	"time"
)

// GetUIConfig handles GET /api/config/ui
func (h *Handler) GetUIConfig(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	response := map[string]interface{}{
		"version": "3.0.0",
		"features": map[string]bool{
			"signal_board":    true,
			"entity_tracking": true,
			"correlations":    true,
			"news":            true,
			"financial":       true,
			"notifications":   true,
			"alerts":          true,
			"intel_briefing":  true,
			"osint_resources": true,
		},
	}

	if h.config != nil {
		response["version"] = h.config.Version
		response["features"] = map[string]bool{
			"signal_board":    h.config.SignalBoard.Enabled,
			"entity_tracking": h.config.EntityTracking.Enabled,
			"correlations":    true,
			"news":            h.config.Providers.NewsRSS.Enabled,
			"financial":       h.config.Providers.VIX.Enabled || h.config.Providers.Crypto.Enabled,
			"notifications":   h.config.Telegram.Enabled || h.config.Slack.Enabled || h.config.Discord.Enabled || h.config.Email.Enabled,
			"alerts":          true,
			"intel_briefing":  h.config.MorningBriefing.Enabled,
			"osint_resources": true,
		}
		response["ui"] = map[string]interface{}{
			"default_view":        h.config.UI.DefaultView,
			"default_preset":      h.config.UI.DefaultPreset,
			"data_retention_days": h.config.UI.DataRetentionDays,
			"sound_enabled":       h.config.UI.SoundEnabled,
			"sound_volume":        h.config.UI.SoundVolume,
			"ticker_enabled":      h.config.UI.TickerEnabled,
			"ticker_speed":        h.config.UI.TickerSpeed,
			"ticker_min_severity": h.config.UI.TickerMinSeverity,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	if h.metrics != nil {
		h.metrics.RecordAPIRequest("/api/config/ui", time.Since(startTime))
	}
}
