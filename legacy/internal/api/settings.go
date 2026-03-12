package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

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
		"status":  "success",
		"message": "Settings updated",
	})
}

// providerFieldMap maps JSON key names to the struct field names in ProvidersConfig.
func providerFieldMap() map[string]string {
	return map[string]string{
		"usgs":            "USGS",
		"gdacs":           "GDACS",
		"opensky":         "OpenSky",
		"noaa_cap":        "NOAACAP",
		"openmeteo":       "OpenMeteo",
		"gdelt":           "GDELT",
		"celestrak":       "Celestrak",
		"swpc":            "SWPC",
		"who":             "WHO",
		"promed":          "ProMED",
		"airplanes_live":  "AirplanesLive",
		"nasa_firms":      "NASAFIRMS",
		"piracy_imb":      "PiracyIMB",
		"israel_alerts":   "IsraelAlerts",
		"reliefweb":       "ReliefWeb",
		"vix":             "VIX",
		"oil_price":       "OilPrice",
		"crypto":          "Crypto",
		"sec_edgar":       "SECEdgar",
		"ofac_sdn":        "OFACSDN",
		"treasury_yields": "TreasuryYields",
		"news_rss":        "NewsRSS",
		"iran_conflict":   "IranConflict",
		"isw":             "ISW",
	}
}

// keysFieldMap maps JSON key names to the struct field names in KeysConfig.
func keysFieldMap() map[string]string {
	return map[string]string{
		"adsbexchange":  "Adsbexchange",
		"aisstream":     "Aisstream",
		"acled":         "Acled",
		"openweather":   "Openweather",
		"nasa":          "Nasa",
		"spacetrack":    "Spacetrack",
		"marinetraffic": "Marinetraffic",
		"vesselfinder":  "Vesselfinder",
		"n2yo":          "N2yo",
		"shodan":        "Shodan",
		"cloudflare":    "Cloudflare",
		"ukrainealerts": "Ukrainealerts",
		"alpha_vantage": "AlphaVantage",
		"finnhub":       "Finnhub",
		"fred":          "Fred",
		"polygon":       "Polygon",
	}
}

// getProviderConfig reads a ProviderConfig from ProvidersConfig by JSON key name using reflection.
func getProviderConfig(providers *config.ProvidersConfig, jsonKey string) (config.ProviderConfig, bool) {
	fm := providerFieldMap()
	fieldName, ok := fm[jsonKey]
	if !ok {
		return config.ProviderConfig{}, false
	}
	v := reflect.ValueOf(providers).Elem()
	f := v.FieldByName(fieldName)
	if !f.IsValid() {
		return config.ProviderConfig{}, false
	}
	return f.Interface().(config.ProviderConfig), true
}

// setProviderConfig writes a ProviderConfig into ProvidersConfig by JSON key name using reflection.
func setProviderConfig(providers *config.ProvidersConfig, jsonKey string, pc config.ProviderConfig) bool {
	fm := providerFieldMap()
	fieldName, ok := fm[jsonKey]
	if !ok {
		return false
	}
	v := reflect.ValueOf(providers).Elem()
	f := v.FieldByName(fieldName)
	if !f.IsValid() || !f.CanSet() {
		return false
	}
	f.Set(reflect.ValueOf(pc))
	return true
}

// getSafeConfig returns config without sensitive data
func (h *SettingsHandler) getSafeConfig() map[string]interface{} {
	// Build providers map with all providers
	providersMap := make(map[string]interface{})
	for jsonKey := range providerFieldMap() {
		pc, ok := getProviderConfig(&h.config.Providers, jsonKey)
		if !ok {
			continue
		}
		opts := pc.Options
		if opts == nil {
			opts = map[string]string{}
		}
		providersMap[jsonKey] = map[string]interface{}{
			"enabled":          pc.Enabled,
			"interval_seconds": pc.IntervalSeconds,
			"options":          opts,
		}
	}

	// Build keys_configured map — shows which keys are set without exposing values
	keysConfigured := make(map[string]bool)
	keysVal := reflect.ValueOf(&h.config.Keys).Elem()
	for jsonKey, fieldName := range keysFieldMap() {
		f := keysVal.FieldByName(fieldName)
		if f.IsValid() {
			keysConfigured[jsonKey] = f.String() != ""
		}
	}

	serverURL := fmt.Sprintf("http://%s:%d", h.config.Server.Host, h.config.Server.Port)
	if h.config.Server.TLSEnabled {
		serverURL = fmt.Sprintf("https://%s:%d", h.config.Server.Host, h.config.Server.Port)
	}

	data := map[string]interface{}{
		"server_url":      serverURL,
		"version":         h.config.Version,
		"setup_complete":  h.config.SetupComplete,
		"data_dir":        h.config.DataDir,
		"log_level":       h.config.LogLevel,
		"auto_open_browser": h.config.AutoOpenBrowser,
		"check_for_updates": h.config.CheckForUpdates,
		"server": map[string]interface{}{
			"port":         h.config.Server.Port,
			"host":         h.config.Server.Host,
			"tls_enabled":  h.config.Server.TLSEnabled,
			"auth_enabled": h.config.Server.AuthEnabled,
		},
		"ui": map[string]interface{}{
			"default_view":        h.config.UI.DefaultView,
			"default_preset":      h.config.UI.DefaultPreset,
			"data_retention_days": h.config.UI.DataRetentionDays,
			"sound_enabled":       h.config.UI.SoundEnabled,
			"sound_volume":        h.config.UI.SoundVolume,
			"ticker_enabled":      h.config.UI.TickerEnabled,
			"ticker_speed":        h.config.UI.TickerSpeed,
			"ticker_min_severity": h.config.UI.TickerMinSeverity,
		},
		"location": map[string]interface{}{
			"lat":      h.config.Location.Lat,
			"lon":      h.config.Location.Lon,
			"timezone": h.config.Location.Timezone,
			"set":      h.config.Location.Set,
		},
		"providers":      providersMap,
		"keys_configured": keysConfigured,
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
		if preset, ok := ui["default_preset"].(string); ok {
			h.config.UI.DefaultPreset = preset
		}
		if days, ok := ui["data_retention_days"].(float64); ok {
			h.config.UI.DataRetentionDays = int(days)
		}
		if soundEnabled, ok := ui["sound_enabled"].(bool); ok {
			h.config.UI.SoundEnabled = soundEnabled
		}
		if volume, ok := ui["sound_volume"].(float64); ok {
			h.config.UI.SoundVolume = int(volume)
		}
		if tickerEnabled, ok := ui["ticker_enabled"].(bool); ok {
			h.config.UI.TickerEnabled = tickerEnabled
		}
		if speed, ok := ui["ticker_speed"].(string); ok {
			h.config.UI.TickerSpeed = speed
		}
		if minSev, ok := ui["ticker_min_severity"].(string); ok {
			h.config.UI.TickerMinSeverity = minSev
		}
	}

	// Apply location settings
	if loc, ok := update["location"].(map[string]interface{}); ok {
		if lat, ok := loc["lat"].(float64); ok {
			h.config.Location.Lat = lat
			h.config.Location.Set = true
		}
		if lon, ok := loc["lon"].(float64); ok {
			h.config.Location.Lon = lon
			h.config.Location.Set = true
		}
		if tz, ok := loc["timezone"].(string); ok {
			h.config.Location.Timezone = tz
		}
	}

	// Apply provider settings — handle ALL providers dynamically
	if providers, ok := update["providers"].(map[string]interface{}); ok {
		for jsonKey, val := range providers {
			provUpdate, ok := val.(map[string]interface{})
			if !ok {
				continue
			}
			// Get current config for this provider
			pc, found := getProviderConfig(&h.config.Providers, jsonKey)
			if !found {
				continue
			}
			if enabled, ok := provUpdate["enabled"].(bool); ok {
				pc.Enabled = enabled
			}
			if interval, ok := provUpdate["interval_seconds"].(float64); ok {
				pc.IntervalSeconds = int(interval)
			}
			if opts, ok := provUpdate["options"].(map[string]interface{}); ok {
				if pc.Options == nil {
					pc.Options = make(map[string]string)
				}
				for k, v := range opts {
					if sv, ok := v.(string); ok {
						pc.Options[k] = sv
					}
				}
			}
			setProviderConfig(&h.config.Providers, jsonKey, pc)
		}
	}

	// Apply keys updates — accept key name → key value, save to KeysConfig
	if keys, ok := update["keys"].(map[string]interface{}); ok {
		keysVal := reflect.ValueOf(&h.config.Keys).Elem()
		for jsonKey, val := range keys {
			keyValue, ok := val.(string)
			if !ok {
				continue
			}
			fieldName, ok := keysFieldMap()[jsonKey]
			if !ok {
				continue
			}
			f := keysVal.FieldByName(fieldName)
			if f.IsValid() && f.CanSet() {
				f.SetString(keyValue)
			}
		}
	}
}
