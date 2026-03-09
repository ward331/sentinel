package api

import (
	"encoding/json"
	"net/http"

	"github.com/openclaw/sentinel-backend/internal/config"
)

// SettingsHandler handles settings API endpoints
type SettingsHandler struct {
	config *config.Config
}

// NewSettingsHandler creates a new settings handler
func NewSettingsHandler(cfg *config.Config) *SettingsHandler {
	return &SettingsHandler{
		config: cfg,
	}
}

// ServeHTTP handles HTTP requests
func (h *SettingsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getSettings(w, r)
	case http.MethodPost:
		h.updateSettings(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getSettings returns current settings
func (h *SettingsHandler) getSettings(w http.ResponseWriter, r *http.Request) {
	// Return settings (without sensitive data)
	safeConfig := h.getSafeConfig()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(safeConfig)
}

// updateSettings updates settings
func (h *SettingsHandler) updateSettings(w http.ResponseWriter, r *http.Request) {
	var update map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Apply updates to config
	h.applyUpdates(update)

	// Save config
	if err := config.SaveConfig(h.config, ""); err != nil {
		http.Error(w, "Failed to save settings", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
		"message": "Settings updated",
	})
}

// getSafeConfig returns config without sensitive data
func (h *SettingsHandler) getSafeConfig() map[string]interface{} {
	// Convert config to map for safe serialization
	// In a real implementation, this would redact sensitive fields
	data := map[string]interface{}{
		"version":         h.config.Version,
		"setup_complete":  h.config.SetupComplete,
		"data_dir":        h.config.DataDir,
		"log_level":       h.config.LogLevel,
		"auto_open_browser": h.config.AutoOpenBrowser,
		"check_for_updates": h.config.CheckForUpdates,
		"server": map[string]interface{}{
			"port":        h.config.Server.Port,
			"host":        h.config.Server.Host,
			"tls_enabled": h.config.Server.TLSEnabled,
			"auth_enabled": h.config.Server.AuthEnabled,
		},
		"ui": map[string]interface{}{
			"default_view":          h.config.UI.DefaultView,
			"default_preset":        h.config.UI.DefaultPreset,
			"data_retention_days":   h.config.UI.DataRetentionDays,
			"sound_enabled":         h.config.UI.SoundEnabled,
			"sound_volume":          h.config.UI.SoundVolume,
			"ticker_enabled":        h.config.UI.TickerEnabled,
			"ticker_speed":          h.config.UI.TickerSpeed,
			"ticker_min_severity":   h.config.UI.TickerMinSeverity,
		},
		"location": map[string]interface{}{
			"lat":       h.config.Location.Lat,
			"lon":       h.config.Location.Lon,
			"timezone":  h.config.Location.Timezone,
			"set":       h.config.Location.Set,
		},
		"providers": map[string]interface{}{
			"usgs": map[string]interface{}{
				"enabled":          h.config.Providers.USGS.Enabled,
				"interval_seconds": h.config.Providers.USGS.IntervalSeconds,
			},
			"gdacs": map[string]interface{}{
				"enabled":          h.config.Providers.GDACS.Enabled,
				"interval_seconds": h.config.Providers.GDACS.IntervalSeconds,
			},
			"opensky": map[string]interface{}{
				"enabled":          h.config.Providers.OpenSky.Enabled,
				"interval_seconds": h.config.Providers.OpenSky.IntervalSeconds,
			},
			"iran_conflict": map[string]interface{}{
				"enabled":          h.config.Providers.IranConflict.Enabled,
				"interval_seconds": h.config.Providers.IranConflict.IntervalSeconds,
			},
			"isw": map[string]interface{}{
				"enabled":          h.config.Providers.ISW.Enabled,
				"interval_seconds": h.config.Providers.ISW.IntervalSeconds,
			},
		},
	}

	return data
}

// applyUpdates applies updates to config
func (h *SettingsHandler) applyUpdates(update map[string]interface{}) {
	// Apply server settings
	if server, ok := update["server"].(map[string]interface{}); ok {
		if port, ok := server["port"].(float64); ok {
			h.config.Server.Port = int(port)
		}
		if host, ok := server["host"].(string); ok {
			h.config.Server.Host = host
		}
	}

	// Apply UI settings
	if ui, ok := update["ui"].(map[string]interface{}); ok {
		if view, ok := ui["default_view"].(string); ok {
			h.config.UI.DefaultView = view
		}
		if soundEnabled, ok := ui["sound_enabled"].(bool); ok {
			h.config.UI.SoundEnabled = soundEnabled
		}
		if volume, ok := ui["sound_volume"].(float64); ok {
			h.config.UI.SoundVolume = int(volume)
		}
	}

	// Apply provider settings
	if providers, ok := update["providers"].(map[string]interface{}); ok {
		// USGS
		if usgs, ok := providers["usgs"].(map[string]interface{}); ok {
			if enabled, ok := usgs["enabled"].(bool); ok {
				h.config.Providers.USGS.Enabled = enabled
			}
		}
		// GDACS
		if gdacs, ok := providers["gdacs"].(map[string]interface{}); ok {
			if enabled, ok := gdacs["enabled"].(bool); ok {
				h.config.Providers.GDACS.Enabled = enabled
			}
		}
		// OpenSky
		if opensky, ok := providers["opensky"].(map[string]interface{}); ok {
			if enabled, ok := opensky["enabled"].(bool); ok {
				h.config.Providers.OpenSky.Enabled = enabled
			}
		}
		// Iran Conflict
		if iranConflict, ok := providers["iran_conflict"].(map[string]interface{}); ok {
			if enabled, ok := iranConflict["enabled"].(bool); ok {
				h.config.Providers.IranConflict.Enabled = enabled
			}
		}
		// ISW
		if isw, ok := providers["isw"].(map[string]interface{}); ok {
			if enabled, ok := isw["enabled"].(bool); ok {
				h.config.Providers.ISW.Enabled = enabled
			}
		}
	}
}