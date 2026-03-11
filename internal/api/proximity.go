package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/openclaw/sentinel-backend/internal/engine"
)

// proximityEngine is the shared proximity alert instance.
// It is set via SetProximityEngine on the Handler.
var proximityEngine *engine.ProximityAlert

// SetProximityEngine attaches a proximity alert engine to the handler.
func (h *Handler) SetProximityEngine(pe *engine.ProximityAlert) {
	proximityEngine = pe
}

// GetProximityEvents handles GET /api/proximity/events
// Returns events within the configured proximity radius.
// Optional query params: minutes (default 60), severity.
func (h *Handler) GetProximityEvents(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	if proximityEngine == nil || !proximityEngine.Configured() {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"events":     []interface{}{},
			"total":      0,
			"configured": proximityEngine != nil && proximityEngine.Configured(),
			"message":    "Proximity alerts not configured. Set your home location first.",
		})
		if h.metrics != nil {
			h.metrics.RecordAPIRequest("/api/proximity/events", time.Since(startTime))
		}
		return
	}

	// Parse minutes lookback (default 60)
	minutes := 60
	if m := r.URL.Query().Get("minutes"); m != "" {
		if parsed, err := strconv.Atoi(m); err == nil && parsed > 0 && parsed <= 10080 {
			minutes = parsed
		}
	}

	// Fetch recent events from storage
	events, err := h.storage.GetRecentEvents(r.Context(), minutes)
	if err != nil {
		http.Error(w, `{"error":"failed to query events"}`, http.StatusInternalServerError)
		if h.metrics != nil {
			h.metrics.RecordAPIError("/api/proximity/events")
		}
		return
	}

	// Filter to nearby
	nearby := proximityEngine.FilterNearby(events)

	// Optional severity filter
	if sev := r.URL.Query().Get("severity"); sev != "" {
		filtered := make([]engine.ProximityEvent, 0, len(nearby))
		for _, pe := range nearby {
			if pe.Event.Severity == sev {
				filtered = append(filtered, pe)
			}
		}
		nearby = filtered
	}

	response := map[string]interface{}{
		"events":    nearby,
		"total":     len(nearby),
		"radius_km": proximityEngine.RadiusKm,
		"home_lat":  proximityEngine.HomeLat,
		"home_lon":  proximityEngine.HomeLon,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	if h.metrics != nil {
		h.metrics.RecordAPIRequest("/api/proximity/events", time.Since(startTime))
	}
}

// GetProximityConfig handles GET /api/proximity/config
// Returns the current proximity alert configuration.
func (h *Handler) GetProximityConfig(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	configured := proximityEngine != nil && proximityEngine.Configured()

	response := map[string]interface{}{
		"configured": configured,
	}

	if configured {
		response["lat"] = proximityEngine.HomeLat
		response["lon"] = proximityEngine.HomeLon
		response["radius_km"] = proximityEngine.RadiusKm
	} else {
		response["lat"] = 0.0
		response["lon"] = 0.0
		response["radius_km"] = engine.DefaultProximityRadiusKm
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	if h.metrics != nil {
		h.metrics.RecordAPIRequest("/api/proximity/config", time.Since(startTime))
	}
}

// UpdateProximityConfig handles POST /api/proximity/config
// Accepts JSON body with lat, lon, and optional radius_km.
func (h *Handler) UpdateProximityConfig(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	var body struct {
		Lat      float64 `json:"lat"`
		Lon      float64 `json:"lon"`
		RadiusKm float64 `json:"radius_km"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
		if h.metrics != nil {
			h.metrics.RecordAPIError("/api/proximity/config")
		}
		return
	}

	// Validate latitude and longitude ranges
	if body.Lat < -90 || body.Lat > 90 {
		http.Error(w, `{"error":"lat must be between -90 and 90"}`, http.StatusBadRequest)
		if h.metrics != nil {
			h.metrics.RecordAPIError("/api/proximity/config")
		}
		return
	}
	if body.Lon < -180 || body.Lon > 180 {
		http.Error(w, `{"error":"lon must be between -180 and 180"}`, http.StatusBadRequest)
		if h.metrics != nil {
			h.metrics.RecordAPIError("/api/proximity/config")
		}
		return
	}

	if body.RadiusKm <= 0 {
		body.RadiusKm = engine.DefaultProximityRadiusKm
	}

	// Update the proximity engine
	if proximityEngine == nil {
		proximityEngine = engine.NewProximityAlertRaw(body.Lat, body.Lon, body.RadiusKm, nil)
	} else {
		proximityEngine.UpdateLocation(body.Lat, body.Lon, body.RadiusKm)
	}

	// Also update the config if available
	if h.config != nil {
		h.config.Location.Lat = body.Lat
		h.config.Location.Lon = body.Lon
		h.config.Location.RadiusKm = body.RadiusKm
		h.config.Location.Set = true
	}

	response := map[string]interface{}{
		"status":    "success",
		"message":   "Proximity config updated",
		"lat":       body.Lat,
		"lon":       body.Lon,
		"radius_km": body.RadiusKm,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	if h.metrics != nil {
		h.metrics.RecordAPIRequest("/api/proximity/config", time.Since(startTime))
	}
}
