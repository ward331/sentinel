package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/openclaw/sentinel-backend/internal/model"
	"github.com/openclaw/sentinel-backend/internal/storage"
)

// OSINTResourcesHandler handles OSINT resource API endpoints
type OSINTResourcesHandler struct {
	storage *storage.OSINTStorage
}

// NewOSINTResourcesHandler creates a new OSINT resources handler
func NewOSINTResourcesHandler(osintStorage *storage.OSINTStorage) *OSINTResourcesHandler {
	return &OSINTResourcesHandler{
		storage: osintStorage,
	}
}

// RegisterRoutes registers OSINT resource routes
func (h *OSINTResourcesHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/resources", h.listResources).Methods("GET")
	r.HandleFunc("/resources/{id}", h.getResource).Methods("GET")
	r.HandleFunc("/resources", h.createResource).Methods("POST")
	r.HandleFunc("/resources/{id}", h.updateResource).Methods("PUT")
	r.HandleFunc("/resources/{id}", h.deleteResource).Methods("DELETE")
	r.HandleFunc("/resources/contextual/{eventId}", h.getContextualResources).Methods("GET")
	r.HandleFunc("/resources/categories", h.getCategories).Methods("GET")
	r.HandleFunc("/resources/platforms", h.getPlatforms).Methods("GET")
}

// listResources returns a list of OSINT resources with optional filtering
func (h *OSINTResourcesHandler) listResources(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Parse query parameters
	query := r.URL.Query()
	filters := make(map[string]interface{})
	
	if platform := query.Get("platform"); platform != "" {
		filters["platform"] = platform
	}
	if category := query.Get("category"); category != "" {
		filters["category"] = category
	}
	if credibility := query.Get("credibility"); credibility != "" {
		filters["credibility"] = credibility
	}
	if builtin := query.Get("builtin"); builtin != "" {
		if isBuiltin, err := strconv.ParseBool(builtin); err == nil {
			filters["is_builtin"] = isBuiltin
		}
	}
	if freeTier := query.Get("free_tier"); freeTier != "" {
		if isFree, err := strconv.ParseBool(freeTier); err == nil {
			filters["free_tier"] = isFree
		}
	}
	if tag := query.Get("tag"); tag != "" {
		filters["tag"] = tag
	}
	if search := query.Get("search"); search != "" {
		filters["search"] = search
	}
	
	// Parse pagination
	limit := 50
	offset := 0
	
	if limitStr := query.Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	
	if offsetStr := query.Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}
	
	// Get resources
	resources, err := h.storage.List(ctx, filters, limit, offset)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list resources: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Get total count
	total, err := h.storage.Count(ctx, filters)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to count resources: %v", err), http.StatusInternalServerError)
		return
	}
	
	response := map[string]interface{}{
		"resources": resources,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getResource returns a single OSINT resource by ID
func (h *OSINTResourcesHandler) getResource(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid resource ID", http.StatusBadRequest)
		return
	}
	
	resource, err := h.storage.GetByID(ctx, id)
	if err != nil {
		http.Error(w, "Resource not found", http.StatusNotFound)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resource)
}

// createResource creates a new OSINT resource
func (h *OSINTResourcesHandler) createResource(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	var input model.OSINTResourceInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Validate required fields
	if input.DisplayName == "" || input.ProfileURL == "" || input.Platform == "" || input.Category == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}
	
	// Convert input to resource
	resource := &model.OSINTResource{
		Platform:         input.Platform,
		Category:         input.Category,
		DisplayName:      input.DisplayName,
		ProfileURL:       input.ProfileURL,
		Description:      input.Description,
		Credibility:      input.Credibility,
		IsBuiltin:        input.IsBuiltin,
		Tags:             input.Tags,
		APIKeyRequired:   input.APIKeyRequired,
		FreeTier:         input.FreeTier,
		Notes:            input.Notes,
	}
	
	// Set default credibility if not provided
	if resource.Credibility == "" {
		resource.Credibility = model.CredibilityCommunity
	}
	
	// Insert resource
	if err := h.storage.Insert(ctx, resource); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create resource: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resource)
}

// updateResource updates an existing OSINT resource
func (h *OSINTResourcesHandler) updateResource(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid resource ID", http.StatusBadRequest)
		return
	}
	
	// Get existing resource
	existing, err := h.storage.GetByID(ctx, id)
	if err != nil {
		http.Error(w, "Resource not found", http.StatusNotFound)
		return
	}
	
	var input model.OSINTResourceInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Update resource fields
	existing.Platform = input.Platform
	existing.Category = input.Category
	existing.DisplayName = input.DisplayName
	existing.ProfileURL = input.ProfileURL
	existing.Description = input.Description
	existing.Credibility = input.Credibility
	existing.IsBuiltin = input.IsBuiltin
	existing.Tags = input.Tags
	existing.APIKeyRequired = input.APIKeyRequired
	existing.FreeTier = input.FreeTier
	existing.Notes = input.Notes
	
	// Update resource
	if err := h.storage.Update(ctx, existing); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update resource: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(existing)
}

// deleteResource deletes an OSINT resource
func (h *OSINTResourcesHandler) deleteResource(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid resource ID", http.StatusBadRequest)
		return
	}
	
	if err := h.storage.Delete(ctx, id); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete resource: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusNoContent)
}

// getContextualResources returns OSINT resources relevant to a specific event
func (h *OSINTResourcesHandler) getContextualResources(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	eventID := vars["eventId"]
	
	// In a real implementation, we would fetch the event from storage
	// and determine which OSINT resources are relevant based on:
	// - Event category (conflict, disaster, maritime, etc.)
	// - Event severity/tier
	// - Event location
	// - Event source
	
	// For now, return all built-in resources with contextual URLs
	filters := map[string]interface{}{
		"is_builtin": true,
	}
	
	resources, err := h.storage.List(ctx, filters, 100, 0)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get resources: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Enhance resources with contextual information
	enhancedResources := make([]map[string]interface{}, 0, len(resources))
	
	for _, resource := range resources {
		enhanced := map[string]interface{}{
			"id":           resource.ID,
			"platform":     resource.Platform,
			"category":     resource.Category,
			"display_name": resource.DisplayName,
			"profile_url":  resource.ProfileURL,
			"description":  resource.Description,
			"credibility":  resource.Credibility,
			"tags":         resource.Tags,
			"free_tier":    resource.FreeTier,
			"notes":        resource.Notes,
			"contextual_url": resource.ProfileURL,
			"icon":         h.getResourceIcon(resource),
			"label":        h.getResourceLabel(resource),
			"show_for_tier": h.getShowForTier(resource),
			"show_for_categories": h.getShowForCategories(resource),
		}
		
		// Add contextual parameters for Hunt Intelligence tools
		if strings.Contains(resource.ProfileURL, "birdhunt.huntintel.io") {
			enhanced["contextual_url"] = resource.ProfileURL + "?lat={lat}&lon={lon}"
			enhanced["label"] = "🐦 Find tweets from this location"
			enhanced["show_for_tier"] = []int{2, 3, 4, 5} // TIER 2+
			enhanced["show_for_categories"] = []string{"all"}
		}
		
		if strings.Contains(resource.ProfileURL, "instahunt.huntintel.io") {
			enhanced["contextual_url"] = resource.ProfileURL + "?lat={lat}&lon={lon}"
			enhanced["label"] = "📸 Find Instagram posts from this location"
			enhanced["show_for_tier"] = []int{1, 2, 3, 4, 5} // All tiers
			enhanced["show_for_categories"] = []string{"conflict", "disaster", "military"}
		}
		
		enhancedResources = append(enhancedResources, enhanced)
	}
	
	response := map[string]interface{}{
		"event_id":   eventID,
		"resources":  enhancedResources,
		"timestamp":  time.Now(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getCategories returns all unique categories
func (h *OSINTResourcesHandler) getCategories(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Get all resources
	resources, err := h.storage.List(ctx, nil, 1000, 0)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get resources: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Extract unique categories
	categories := make(map[string]bool)
	for _, resource := range resources {
		categories[resource.Category] = true
	}
	
	// Convert to slice
	categoryList := make([]string, 0, len(categories))
	for category := range categories {
		categoryList = append(categoryList, category)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(categoryList)
}

// getPlatforms returns all unique platforms
func (h *OSINTResourcesHandler) getPlatforms(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Get all resources
	resources, err := h.storage.List(ctx, nil, 1000, 0)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get resources: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Extract unique platforms
	platforms := make(map[string]bool)
	for _, resource := range resources {
		platforms[resource.Platform] = true
	}
	
	// Convert to slice
	platformList := make([]string, 0, len(platforms))
	for platform := range platforms {
		platformList = append(platformList, platform)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(platformList)
}

// getResourceIcon returns an icon for the resource based on platform/category
func (h *OSINTResourcesHandler) getResourceIcon(resource *model.OSINTResource) string {
	switch resource.Platform {
	case model.PlatformWeb:
		return "🌐"
	case model.PlatformAPI:
		return "🔌"
	case model.PlatformDataset:
		return "📊"
	case model.PlatformTool:
		return "🛠️"
	case model.PlatformRSS:
		return "📰"
	case model.PlatformMap:
		return "🗺️"
	default:
		return "📄"
	}
}

// getResourceLabel returns a human-readable label for the resource
func (h *OSINTResourcesHandler) getResourceLabel(resource *model.OSINTResource) string {
	// Use display name as default
	return resource.DisplayName
}

// getShowForTier returns which event tiers this resource should be shown for
func (h *OSINTResourcesHandler) getShowForTier(resource *model.OSINTResource) []int {
	// Default: show for all tiers
	tiers := []int{1, 2, 3, 4, 5}
	
	// Hunt Intelligence tools have specific tier requirements
	if strings.Contains(resource.ProfileURL, "birdhunt.huntintel.io") {
		// BirdHunt: TIER 2+
		tiers = []int{2, 3, 4, 5}
	}
	
	return tiers
}

// getShowForCategories returns which event categories this resource should be shown for
func (h *OSINTResourcesHandler) getShowForCategories(resource *model.OSINTResource) []string {
	// Default: show for all categories
	categories := []string{"all"}
	
	// Hunt Intelligence tools have specific category requirements
	if strings.Contains(resource.ProfileURL, "instahunt.huntintel.io") {
		// InstaHunt: conflict, disaster, military
		categories = []string{"conflict", "disaster", "military"}
	}
	
	return categories
}