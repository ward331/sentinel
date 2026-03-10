package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/openclaw/sentinel-backend/internal/alert"
	"github.com/openclaw/sentinel-backend/internal/config"
	"github.com/openclaw/sentinel-backend/internal/health"
	"github.com/openclaw/sentinel-backend/internal/infrastructure"
	"github.com/openclaw/sentinel-backend/internal/metrics"
	"github.com/openclaw/sentinel-backend/internal/model"
	"github.com/openclaw/sentinel-backend/internal/poller"
	"github.com/openclaw/sentinel-backend/internal/storage"
)

// Handler handles HTTP requests for the SENTINEL API
type Handler struct {
	storage        *storage.Storage
	stream         *StreamBroker
	alertEngine    *alert.RuleEngine
	metrics        *metrics.Metrics
	health         *health.HealthRegistry
	healthReporter *infrastructure.HealthReporter
	eventLog       *infrastructure.NDJSONLog
	poller         *poller.Poller
	config         *config.Config
	startTime      time.Time
}

// NewHandler creates a new API handler
func NewHandler(storage *storage.Storage, metrics *metrics.Metrics, healthRegistry *health.HealthRegistry) *Handler {
	return &Handler{
		storage:     storage,
		stream:      NewStreamBroker(),
		alertEngine: alert.NewRuleEngine(),
		metrics:     metrics,
		health:      healthRegistry,
		startTime:   time.Now(),
	}
}

// NewHandlerWithInfrastructure creates a new API handler with data infrastructure
func NewHandlerWithInfrastructure(
	storage *storage.Storage, 
	metrics *metrics.Metrics, 
	healthRegistry *health.HealthRegistry,
	healthReporter *infrastructure.HealthReporter,
	eventLog *infrastructure.NDJSONLog,
) *Handler {
	return &Handler{
		storage:        storage,
		stream:         NewStreamBroker(),
		alertEngine:    alert.NewRuleEngine(),
		metrics:        metrics,
		health:         healthRegistry,
		healthReporter: healthReporter,
		eventLog:       eventLog,
		startTime:      time.Now(),
	}
}

// Stream returns the stream broker for broadcasting events
func (h *Handler) Stream() *StreamBroker {
	return h.stream
}

// SetPoller sets the poller instance on the handler
func (h *Handler) SetPoller(p *poller.Poller) {
	h.poller = p
}

// SetConfig sets the config instance on the handler
func (h *Handler) SetConfig(cfg *config.Config) {
	h.config = cfg
}

// ListProviders handles GET /api/providers
func (h *Handler) ListProviders(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	if h.poller == nil {
		http.Error(w, `{"error": "Poller not available"}`, http.StatusServiceUnavailable)
		if h.metrics != nil {
			h.metrics.RecordAPIError("/api/providers")
		}
		return
	}

	names := h.poller.GetProviderNames()

	type providerInfo struct {
		Name            string `json:"name"`
		IntervalSeconds int    `json:"interval_seconds"`
		Enabled         bool   `json:"enabled"`
	}

	providers := make([]providerInfo, 0, len(names))
	for _, name := range names {
		prov, ok := h.poller.GetProvider(name)
		if !ok {
			continue
		}
		providers = append(providers, providerInfo{
			Name:            name,
			IntervalSeconds: int(prov.Interval().Seconds()),
			Enabled:         prov.Enabled(),
		})
	}

	response := map[string]interface{}{
		"providers": providers,
		"total":     len(providers),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	if h.metrics != nil {
		h.metrics.RecordAPIRequest("/api/providers", time.Since(startTime))
	}
}

// ListEvents handles GET /api/events
func (h *Handler) ListEvents(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	
	filter := parseListFilter(r)

	events, total, err := h.storage.ListEvents(r.Context(), filter)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list events: %v", err), http.StatusInternalServerError)
		if h.metrics != nil {
			h.metrics.RecordAPIError("/api/events")
		}
		return
	}

	response := map[string]interface{}{
		"events": events,
		"total":  total,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	
	// Record metrics
	if h.metrics != nil {
		h.metrics.RecordAPIRequest("/api/events", time.Since(startTime))
	}
}

// CreateEvent handles POST /api/events
func (h *Handler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	
	var input model.EventInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		if h.metrics != nil {
			h.metrics.RecordAPIError("/api/events")
		}
		return
	}

	// Validate required fields
	if err := validateEventInput(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		if h.metrics != nil {
			h.metrics.RecordAPIError("/api/events")
		}
		return
	}

	// Create event from input
	event := &model.Event{
		Title:       input.Title,
		Description: input.Description,
		Source:      input.Source,
		SourceID:    input.SourceID,
		OccurredAt:  input.OccurredAt,
		Location:    input.Location,
		Precision:   input.Precision,
		Magnitude:   input.Magnitude,
		Category:    input.Category,
		Severity:    input.Severity,
		Metadata:    input.Metadata,
	}

	// Add badges
	event.Badges = []model.Badge{
		{
			Label:     input.Source,
			Type:      model.BadgeTypeSource,
			Timestamp: time.Now().UTC(),
		},
		{
			Label:     string(input.Precision),
			Type:      model.BadgeTypePrecision,
			Timestamp: time.Now().UTC(),
		},
		{
			Label:     "just now",
			Type:      model.BadgeTypeFreshness,
			Timestamp: time.Now().UTC(),
		},
	}

	// Store event
	if err := h.storage.StoreEvent(r.Context(), event); err != nil {
		http.Error(w, fmt.Sprintf("Failed to store event: %v", err), http.StatusInternalServerError)
		if h.metrics != nil {
			h.metrics.RecordAPIError("/api/events")
		}
		return
	}

	// Record metrics for ingested event
	if h.metrics != nil {
		h.metrics.RecordEventIngested()
		h.metrics.RecordEventBroadcast()
	}

	// Evaluate alert rules
	if h.alertEngine != nil {
		triggered := h.alertEngine.Evaluate(event)
		if len(triggered) > 0 {
			log.Printf("Alert triggered for API-created event %s: %d rule(s) matched", event.ID, len(triggered))
			if h.metrics != nil {
				h.metrics.RecordAlertTriggered()
				h.metrics.RecordAlertProcessed()
			}
		}
	}

	// Broadcast to SSE stream
	h.stream.Broadcast(event)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(event)
	
	// Record API metrics
	if h.metrics != nil {
		h.metrics.RecordAPIRequest("/api/events", time.Since(startTime))
	}
}

// GetEvent handles GET /api/events/{id}
func (h *Handler) GetEvent(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	
	path := strings.TrimPrefix(r.URL.Path, "/api/events/")
	if path == "" {
		http.Error(w, "Event ID required", http.StatusBadRequest)
		if h.metrics != nil {
			h.metrics.RecordAPIError("/api/events/{id}")
		}
		return
	}

	event, err := h.storage.GetEvent(r.Context(), path)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			http.Error(w, "Event not found", http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf("Failed to get event: %v", err), http.StatusInternalServerError)
		}
		if h.metrics != nil {
			h.metrics.RecordAPIError("/api/events/{id}")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(event)
	
	// Record metrics
	if h.metrics != nil {
		h.metrics.RecordAPIRequest("/api/events/{id}", time.Since(startTime))
	}
}

// EventStream handles GET /api/events/stream
func (h *Handler) EventStream(w http.ResponseWriter, r *http.Request) {
	log.Printf("EventStream: New SSE connection from %s", r.RemoteAddr)
	
	// Set headers for Server-Sent Events
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Type")

	// Create a new client
	client := h.stream.NewClient()
	defer h.stream.RemoveClient(client)

	// Send initial comment to establish connection
	fmt.Fprintf(w, ": connected\n\n")
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	// Listen for events
	for {
		select {
		case event := <-client:
			data, err := json.Marshal(event)
			if err != nil {
				log.Printf("Failed to marshal event for SSE: %v", err)
				continue
			}
			fmt.Fprintf(w, "event: new\ndata: %s\n\n", data)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		case <-r.Context().Done():
			return
		}
	}
}

// HealthCheck handles GET /api/health
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	
	var response interface{}
	
	// Check if detailed health check is requested
	if r.URL.Query().Get("detailed") == "true" && h.health != nil {
		// Perform detailed health checks
		overallStatus, checks := h.health.OverallStatus(r.Context())
		healthResponse := health.NewHealthResponse(overallStatus, time.Since(h.startTime), checks)
		response = healthResponse
	} else {
		// Simple health check
		response = map[string]interface{}{
			"status":    "ok",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"uptime":    time.Since(h.startTime).Seconds(),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	
	// Record metrics
	if h.metrics != nil {
		h.metrics.RecordAPIRequest("/api/health", time.Since(startTime))
	}
}

var startTime = time.Now()

func parseListFilter(r *http.Request) storage.ListFilter {
	filter := storage.ListFilter{
		Limit:  100,
		Offset: 0,
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 1000 {
			filter.Limit = limit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}

	if source := r.URL.Query().Get("source"); source != "" {
		filter.Source = source
	}

	if category := r.URL.Query().Get("category"); category != "" {
		filter.Category = category
	}

	if severity := r.URL.Query().Get("severity"); severity != "" {
		filter.Severity = severity
	}

	if magStr := r.URL.Query().Get("min_magnitude"); magStr != "" {
		if mag, err := strconv.ParseFloat(magStr, 64); err == nil && mag >= 0 {
			filter.MinMagnitude = mag
		}
	}

	if magStr := r.URL.Query().Get("max_magnitude"); magStr != "" {
		if mag, err := strconv.ParseFloat(magStr, 64); err == nil && mag >= 0 {
			filter.MaxMagnitude = mag
		}
	}

	if query := r.URL.Query().Get("q"); query != "" {
		filter.Query = query
	}

	if startStr := r.URL.Query().Get("start_time"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			filter.StartTime = t
		}
	}

	if endStr := r.URL.Query().Get("end_time"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			filter.EndTime = t
		}
	}

	if bboxStr := r.URL.Query().Get("bbox"); bboxStr != "" {
		parts := strings.Split(bboxStr, ",")
		if len(parts) == 4 {
			var bbox []float64
			for _, part := range parts {
				if val, err := strconv.ParseFloat(strings.TrimSpace(part), 64); err == nil {
					bbox = append(bbox, val)
				}
			}
			if len(bbox) == 4 {
				filter.BBox = bbox
			}
		}
	}

	return filter
}

func validateEventInput(input *model.EventInput) error {
	if input.Title == "" {
		return fmt.Errorf("title is required")
	}
	if input.Description == "" {
		return fmt.Errorf("description is required")
	}
	if input.Source == "" {
		return fmt.Errorf("source is required")
	}
	if input.OccurredAt.IsZero() {
		return fmt.Errorf("occurred_at is required")
	}
	if input.Location.Type == "" {
		return fmt.Errorf("location.type is required")
	}
	if input.Location.Coordinates == nil {
		return fmt.Errorf("location.coordinates is required")
	}
	if input.Precision == "" {
		return fmt.Errorf("precision is required")
	}

	// Validate precision enum
	switch input.Precision {
	case model.PrecisionExact, model.PrecisionPolygonArea, model.PrecisionApproximate,
		model.PrecisionTextInferred, model.PrecisionUnknown:
		// valid
	default:
		return fmt.Errorf("invalid precision value: %s", input.Precision)
	}

	// Validate severity if provided
	if input.Severity != "" {
		switch input.Severity {
		case model.SeverityLow, model.SeverityMedium, model.SeverityHigh, model.SeverityCritical:
			// valid
		default:
			return fmt.Errorf("invalid severity value: %s", input.Severity)
		}
	}

	return nil
}

// ListAlertRules handles GET /api/alerts/rules
func (h *Handler) ListAlertRules(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	
	if h.alertEngine == nil {
		http.Error(w, "Alert engine not available", http.StatusServiceUnavailable)
		if h.metrics != nil {
			h.metrics.RecordAPIError("/api/alerts/rules")
		}
		return
	}

	rules := h.alertEngine.GetRules()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rules)
	
	// Record metrics
	if h.metrics != nil {
		h.metrics.RecordAPIRequest("/api/alerts/rules", time.Since(startTime))
	}
}

// CreateAlertRule handles POST /api/alerts/rules
func (h *Handler) CreateAlertRule(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	
	if h.alertEngine == nil {
		http.Error(w, "Alert engine not available", http.StatusServiceUnavailable)
		if h.metrics != nil {
			h.metrics.RecordAPIError("/api/alerts/rules")
		}
		return
	}

	var rule alert.Rule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		if h.metrics != nil {
			h.metrics.RecordAPIError("/api/alerts/rules")
		}
		return
	}

	h.alertEngine.AddRule(rule)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(rule)
	
	// Record metrics
	if h.metrics != nil {
		h.metrics.RecordAPIRequest("/api/alerts/rules", time.Since(startTime))
	}
}

// GetMetrics handles GET /api/metrics
func (h *Handler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	if h.metrics == nil {
		http.Error(w, "Metrics not available", http.StatusServiceUnavailable)
		return
	}

	metrics := h.metrics.Get()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// GetProviderHealth handles GET /api/providers/health
func (h *Handler) GetProviderHealth(w http.ResponseWriter, r *http.Request) {
	if h.healthReporter == nil {
		http.Error(w, `{"error": "Health reporter not available"}`, http.StatusServiceUnavailable)
		return
	}

	stats := h.healthReporter.GetAllStats()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// GetProviderStats handles GET /api/providers/{name}/stats
func (h *Handler) GetProviderStats(w http.ResponseWriter, r *http.Request) {
	if h.healthReporter == nil {
		http.Error(w, `{"error": "Health reporter not available"}`, http.StatusServiceUnavailable)
		return
	}

	// Extract provider name from path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 5 {
		http.Error(w, `{"error": "Invalid path"}`, http.StatusBadRequest)
		return
	}
	providerName := pathParts[4]

	stats, exists := h.healthReporter.GetProviderStats(providerName)
	if !exists {
		http.Error(w, `{"error": "Provider not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// GetEventLogInfo handles GET /api/event-log/info
func (h *Handler) GetEventLogInfo(w http.ResponseWriter, r *http.Request) {
	if h.eventLog == nil {
		http.Error(w, `{"error": "Event log not available"}`, http.StatusServiceUnavailable)
		return
	}

	stats, err := h.eventLog.Stats()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "Failed to get log stats: %v"}`, err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"path":         h.eventLog.GetLogPath(),
		"size_bytes":   stats.Size(),
		"modified":     stats.ModTime().Format(time.RFC3339),
		"is_directory": stats.IsDir(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// RotateEventLog handles POST /api/event-log/rotate
func (h *Handler) RotateEventLog(w http.ResponseWriter, r *http.Request) {
	if h.eventLog == nil {
		http.Error(w, `{"error": "Event log not available"}`, http.StatusServiceUnavailable)
		return
	}

	rotatedPath, err := h.eventLog.Rotate()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "Failed to rotate log: %v"}`, err), http.StatusInternalServerError)
		return
	}

	response := map[string]string{
		"message":      "Event log rotated successfully",
		"rotated_path": rotatedPath,
		"new_path":     h.eventLog.GetLogPath(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetHealthyProviders handles GET /api/providers/healthy
func (h *Handler) GetHealthyProviders(w http.ResponseWriter, r *http.Request) {
	if h.healthReporter == nil {
		http.Error(w, `{"error": "Health reporter not available"}`, http.StatusServiceUnavailable)
		return
	}

	healthy := h.healthReporter.GetHealthyProviders()
	response := map[string]interface{}{
		"healthy_providers": healthy,
		"count":             len(healthy),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetUnhealthyProviders handles GET /api/providers/unhealthy
func (h *Handler) GetUnhealthyProviders(w http.ResponseWriter, r *http.Request) {
	if h.healthReporter == nil {
		http.Error(w, `{"error": "Health reporter not available"}`, http.StatusServiceUnavailable)
		return
	}

	unhealthy := h.healthReporter.GetUnhealthyProviders()
	response := map[string]interface{}{
		"unhealthy_providers": unhealthy,
		"count":               len(unhealthy),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Router returns a configured gorilla/mux router with all API routes
func (h *Handler) Router() *mux.Router {
	r := mux.NewRouter()
	
	// Event log routes (if available) — MUST be before {id}
	if h.eventLog != nil {
		r.HandleFunc("/api/events/log/info", h.GetEventLogInfo).Methods("GET")
		r.HandleFunc("/api/events/log/rotate", h.RotateEventLog).Methods("POST")
	}

	// Event routes — stream MUST be before {id} to avoid matching "stream" as an id
	r.HandleFunc("/api/events/stream", h.EventStream).Methods("GET")
	r.HandleFunc("/api/events", h.ListEvents).Methods("GET")
	r.HandleFunc("/api/events", h.CreateEvent).Methods("POST")
	r.HandleFunc("/api/events/{id}", h.GetEvent).Methods("GET")
	
	// Health routes
	r.HandleFunc("/api/health", h.HealthCheck).Methods("GET")
	r.HandleFunc("/api/providers/healthy", h.GetHealthyProviders).Methods("GET")
	r.HandleFunc("/api/providers/unhealthy", h.GetUnhealthyProviders).Methods("GET")

	// Provider listing route
	r.HandleFunc("/api/providers", h.ListProviders).Methods("GET")

	// Config/settings routes
	if h.config != nil {
		settingsHandler := NewSettingsHandler(h.config)
		r.HandleFunc("/api/config", settingsHandler.ServeHTTP).Methods("GET", "POST")
	}
	
	// Alert routes
	r.HandleFunc("/api/alerts/rules", h.ListAlertRules).Methods("GET")
	r.HandleFunc("/api/alerts/rules", h.CreateAlertRule).Methods("POST")
	// Note: GetAlertRule, UpdateAlertRule, DeleteAlertRule methods not implemented yet
	
	// Metrics routes
	r.HandleFunc("/api/metrics", h.GetMetrics).Methods("GET")
	
	// Provider health routes
	r.HandleFunc("/api/providers/health", h.GetProviderHealth).Methods("GET")
	r.HandleFunc("/api/providers/stats", h.GetProviderStats).Methods("GET")
	
	return r
}