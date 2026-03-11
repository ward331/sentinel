package api

import (
	"encoding/json"
	"net/http"
	"time"
)

// GetIntelBriefing handles GET /api/intel/briefing
func (h *Handler) GetIntelBriefing(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// Placeholder briefing — LLM integration in Stage G5.
	response := map[string]interface{}{
		"content": "SENTINEL Intelligence Briefing — " + time.Now().UTC().Format("2006-01-02") +
			"\n\nNo significant events to report at this time. " +
			"All monitored domains are operating within normal parameters. " +
			"Signal board levels remain nominal across military, cyber, financial, natural, and health domains.\n\n" +
			"This is a placeholder briefing. AI-generated briefings will be available once the LLM integration is complete.",
		"generated_at": time.Now().UTC(),
		"type":         "morning",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	if h.metrics != nil {
		h.metrics.RecordAPIRequest("/api/intel/briefing", time.Since(startTime))
	}
}

// GetNews handles GET /api/news
func (h *Handler) GetNews(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// Placeholder — news items from news_items table or RSS ingestion (Stage G4).
	response := map[string]interface{}{
		"items": []interface{}{},
		"total": 0,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	if h.metrics != nil {
		h.metrics.RecordAPIRequest("/api/news", time.Since(startTime))
	}
}
