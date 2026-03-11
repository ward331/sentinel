package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/openclaw/sentinel-backend/internal/alert"
)

// UpdateAlertRule handles PUT /api/alerts/rules/{id}
func (h *Handler) UpdateAlertRule(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	if h.alertEngine == nil {
		http.Error(w, `{"error":"alert engine not available"}`, http.StatusServiceUnavailable)
		if h.metrics != nil {
			h.metrics.RecordAPIError("/api/alerts/rules/{id}")
		}
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		http.Error(w, `{"error":"rule ID required"}`, http.StatusBadRequest)
		if h.metrics != nil {
			h.metrics.RecordAPIError("/api/alerts/rules/{id}")
		}
		return
	}

	var rule alert.Rule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"invalid request body: %v"}`, err), http.StatusBadRequest)
		if h.metrics != nil {
			h.metrics.RecordAPIError("/api/alerts/rules/{id}")
		}
		return
	}

	if !h.alertEngine.UpdateRule(id, rule) {
		http.Error(w, `{"error":"rule not found"}`, http.StatusNotFound)
		if h.metrics != nil {
			h.metrics.RecordAPIError("/api/alerts/rules/{id}")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Rule updated",
		"id":      id,
	})

	if h.metrics != nil {
		h.metrics.RecordAPIRequest("/api/alerts/rules/{id}", time.Since(startTime))
	}
}

// DeleteAlertRule handles DELETE /api/alerts/rules/{id}
func (h *Handler) DeleteAlertRule(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	if h.alertEngine == nil {
		http.Error(w, `{"error":"alert engine not available"}`, http.StatusServiceUnavailable)
		if h.metrics != nil {
			h.metrics.RecordAPIError("/api/alerts/rules/{id}")
		}
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		http.Error(w, `{"error":"rule ID required"}`, http.StatusBadRequest)
		if h.metrics != nil {
			h.metrics.RecordAPIError("/api/alerts/rules/{id}")
		}
		return
	}

	if !h.alertEngine.DeleteRule(id) {
		http.Error(w, `{"error":"rule not found"}`, http.StatusNotFound)
		if h.metrics != nil {
			h.metrics.RecordAPIError("/api/alerts/rules/{id}")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)

	if h.metrics != nil {
		h.metrics.RecordAPIRequest("/api/alerts/rules/{id}", time.Since(startTime))
	}
}
