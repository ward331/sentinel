package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/openclaw/sentinel-backend/internal/filter"
	"github.com/openclaw/sentinel-backend/internal/model"
)

// FilterHandler handles filter management API endpoints
type FilterHandler struct {
	filterEngine filter.FilterEngine
}

// NewFilterHandler creates a new filter handler
func NewFilterHandler(filterEngine filter.FilterEngine) *FilterHandler {
	return &FilterHandler{
		filterEngine: filterEngine,
	}
}

// RegisterRoutes registers filter routes
func (h *FilterHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/filters", h.listFilters)
	mux.HandleFunc("POST /api/filters", h.createFilter)
	mux.HandleFunc("GET /api/filters/{id}", h.getFilter)
	mux.HandleFunc("PUT /api/filters/{id}", h.updateFilter)
	mux.HandleFunc("DELETE /api/filters/{id}", h.deleteFilter)
	mux.HandleFunc("POST /api/filters/{id}/enable", h.enableFilter)
	mux.HandleFunc("POST /api/filters/{id}/disable", h.disableFilter)
	mux.HandleFunc("POST /api/filters/test", h.testFilter)
	mux.HandleFunc("GET /api/filters/{id}/stats", h.getFilterStats)
}

// listFilters returns all filters
func (h *FilterHandler) listFilters(w http.ResponseWriter, r *http.Request) {
	filters, err := h.filterEngine.ListFilters(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"filters": filters,
		"count":   len(filters),
	})
}

// getFilter returns a specific filter
func (h *FilterHandler) getFilter(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Filter ID is required", http.StatusBadRequest)
		return
	}

	filter, err := h.filterEngine.GetFilter(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, filter)
}

// createFilter creates a new filter
func (h *FilterHandler) createFilter(w http.ResponseWriter, r *http.Request) {
	var newFilter filter.Filter
	if err := json.NewDecoder(r.Body).Decode(&newFilter); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Set timestamps
	now := time.Now()
	newFilter.CreatedAt = now
	newFilter.UpdatedAt = now

	// Validate filter
	if newFilter.Name == "" {
		http.Error(w, "Filter name is required", http.StatusBadRequest)
		return
	}

	if len(newFilter.Conditions) == 0 {
		http.Error(w, "At least one condition is required", http.StatusBadRequest)
		return
	}

	// Add filter
	if err := h.filterEngine.AddFilter(r.Context(), &newFilter); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusCreated, newFilter)
}

// updateFilter updates an existing filter
func (h *FilterHandler) updateFilter(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Filter ID is required", http.StatusBadRequest)
		return
	}

	var updatedFilter filter.Filter
	if err := json.NewDecoder(r.Body).Decode(&updatedFilter); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Preserve ID and timestamps
	updatedFilter.ID = id
	updatedFilter.UpdatedAt = time.Now()

	if err := h.filterEngine.UpdateFilter(r.Context(), id, &updatedFilter); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, updatedFilter)
}

// deleteFilter deletes a filter
func (h *FilterHandler) deleteFilter(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Filter ID is required", http.StatusBadRequest)
		return
	}

	if err := h.filterEngine.RemoveFilter(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// enableFilter enables a filter
func (h *FilterHandler) enableFilter(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Filter ID is required", http.StatusBadRequest)
		return
	}

	if err := h.filterEngine.EnableFilter(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "enabled"})
}

// disableFilter disables a filter
func (h *FilterHandler) disableFilter(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Filter ID is required", http.StatusBadRequest)
		return
	}

	if err := h.filterEngine.DisableFilter(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "disabled"})
}

// testFilter tests a filter against sample events
func (h *FilterHandler) testFilter(w http.ResponseWriter, r *http.Request) {
	var testRequest struct {
		Filter    filter.Filter `json:"filter"`
		TestEvent model.Event   `json:"test_event"`
	}

	if err := json.NewDecoder(r.Body).Decode(&testRequest); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Create a temporary filter engine for testing
	// In a real implementation, we'd use the actual engine
	// For now, we'll simulate the test
	matches := []string{}
	if testRequest.Filter.Name != "" {
		matches = append(matches, "Filter would match this event")
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"matches":      matches,
		"match_count":  len(matches),
		"filter_name":  testRequest.Filter.Name,
		"event_title":  testRequest.TestEvent.Title,
	})
}

// getFilterStats returns statistics for a filter
func (h *FilterHandler) getFilterStats(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Filter ID is required", http.StatusBadRequest)
		return
	}

	matchCount, err := h.filterEngine.MatchCount(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"filter_id":   id,
		"match_count": matchCount,
	})
}

// respondJSON sends a JSON response
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}