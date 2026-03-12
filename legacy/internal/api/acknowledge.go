package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// AcknowledgeEvent handles POST /api/events/{id}/acknowledge
func (h *Handler) AcknowledgeEvent(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// Extract event ID — the path is /api/events/{id}/acknowledge
	path := strings.TrimPrefix(r.URL.Path, "/api/events/")
	path = strings.TrimSuffix(path, "/acknowledge")
	if path == "" {
		http.Error(w, `{"error":"event ID required"}`, http.StatusBadRequest)
		if h.metrics != nil {
			h.metrics.RecordAPIError("/api/events/{id}/acknowledge")
		}
		return
	}

	// Verify event exists
	_, err := h.storage.GetEvent(r.Context(), path)
	if err != nil {
		http.Error(w, `{"error":"event not found"}`, http.StatusNotFound)
		if h.metrics != nil {
			h.metrics.RecordAPIError("/api/events/{id}/acknowledge")
		}
		return
	}

	// Update acknowledged flag in DB
	_, err = h.storage.DB().ExecContext(r.Context(),
		`UPDATE events SET acknowledged = 1 WHERE id = ?`, path)
	if err != nil {
		http.Error(w, `{"error":"failed to acknowledge event"}`, http.StatusInternalServerError)
		if h.metrics != nil {
			h.metrics.RecordAPIError("/api/events/{id}/acknowledge")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "acknowledged",
		"event_id": path,
	})

	if h.metrics != nil {
		h.metrics.RecordAPIRequest("/api/events/{id}/acknowledge", time.Since(startTime))
	}
}
