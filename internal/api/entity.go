package api

import (
	"encoding/json"
	"net/http"
	"time"
)

// EntitySearchResult represents a single entity search hit.
type EntitySearchResult struct {
	ID       string  `json:"id"`
	Type     string  `json:"type"` // "aircraft", "vessel", "satellite", "event"
	Name     string  `json:"name"`
	Source   string  `json:"source"`
	Lat      float64 `json:"lat,omitempty"`
	Lon      float64 `json:"lon,omitempty"`
	LastSeen string  `json:"last_seen,omitempty"`
}

// SearchEntities handles GET /api/entity/search?q=
func (h *Handler) SearchEntities(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, `{"error":"query parameter 'q' is required"}`, http.StatusBadRequest)
		if h.metrics != nil {
			h.metrics.RecordAPIError("/api/entity/search")
		}
		return
	}

	// Search events in storage that match the query.
	// A full implementation would also search aircraft/vessel tracking tables.
	filter := parseListFilter(r)
	filter.Query = query
	filter.Limit = 20

	events, _, err := h.storage.ListEvents(r.Context(), filter)
	if err != nil {
		http.Error(w, `{"error":"search failed"}`, http.StatusInternalServerError)
		if h.metrics != nil {
			h.metrics.RecordAPIError("/api/entity/search")
		}
		return
	}

	results := make([]EntitySearchResult, 0, len(events))
	for _, ev := range events {
		result := EntitySearchResult{
			ID:     ev.ID,
			Type:   "event",
			Name:   ev.Title,
			Source: ev.Source,
		}
		// Extract lat/lon from Point coordinates
		if coords, ok := ev.Location.Coordinates.([]interface{}); ok && len(coords) >= 2 {
			if lon, ok := coords[0].(float64); ok {
				result.Lon = lon
			}
			if lat, ok := coords[1].(float64); ok {
				result.Lat = lat
			}
		}
		result.LastSeen = ev.OccurredAt.Format(time.RFC3339)
		results = append(results, result)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"query":   query,
		"results": results,
		"total":   len(results),
	})

	if h.metrics != nil {
		h.metrics.RecordAPIRequest("/api/entity/search", time.Since(startTime))
	}
}
