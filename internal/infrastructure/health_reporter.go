package infrastructure

import (
	"sync"
	"time"
)

// ProviderHealth tracks health metrics for a data provider
type ProviderHealth struct {
	Name string

	mu sync.RWMutex

	// Counters
	TotalRequests    int64
	SuccessfulRequests int64
	FailedRequests   int64

	// Timing
	TotalResponseTime time.Duration
	LastRequestTime   time.Time
	FirstRequestTime  time.Time

	// Status
	LastError     string
	LastErrorTime time.Time
	IsHealthy     bool
}

// NewProviderHealth creates a new health tracker for a provider
func NewProviderHealth(name string) *ProviderHealth {
	return &ProviderHealth{
		Name:             name,
		IsHealthy:        true,
		FirstRequestTime: time.Now(),
	}
}

// RecordSuccess records a successful request
func (h *ProviderHealth) RecordSuccess(responseTime time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.TotalRequests++
	h.SuccessfulRequests++
	h.TotalResponseTime += responseTime
	h.LastRequestTime = time.Now()
	h.IsHealthy = true
}

// RecordError records a failed request
func (h *ProviderHealth) RecordError(err error, responseTime time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.TotalRequests++
	h.FailedRequests++
	h.TotalResponseTime += responseTime
	h.LastRequestTime = time.Now()
	h.LastError = err.Error()
	h.LastErrorTime = time.Now()
	h.IsHealthy = false
}

// GetStats returns current health statistics
func (h *ProviderHealth) GetStats() ProviderStats {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var avgResponseTime time.Duration
	if h.TotalRequests > 0 {
		avgResponseTime = h.TotalResponseTime / time.Duration(h.TotalRequests)
	}

	var errorRate float64
	if h.TotalRequests > 0 {
		errorRate = float64(h.FailedRequests) / float64(h.TotalRequests) * 100
	}

	var uptime time.Duration
	if !h.FirstRequestTime.IsZero() {
		uptime = time.Since(h.FirstRequestTime)
	}

	return ProviderStats{
		Name:              h.Name,
		TotalRequests:     h.TotalRequests,
		SuccessfulRequests: h.SuccessfulRequests,
		FailedRequests:    h.FailedRequests,
		ErrorRate:         errorRate,
		AvgResponseTime:   avgResponseTime,
		LastRequestTime:   h.LastRequestTime,
		Uptime:            uptime,
		IsHealthy:         h.IsHealthy,
		LastError:         h.LastError,
		LastErrorTime:     h.LastErrorTime,
	}
}

// ProviderStats contains health statistics for a provider
type ProviderStats struct {
	Name              string        `json:"name"`
	TotalRequests     int64         `json:"total_requests"`
	SuccessfulRequests int64         `json:"successful_requests"`
	FailedRequests    int64         `json:"failed_requests"`
	ErrorRate         float64       `json:"error_rate"`
	AvgResponseTime   time.Duration `json:"avg_response_time"`
	LastRequestTime   time.Time     `json:"last_request_time"`
	Uptime            time.Duration `json:"uptime"`
	IsHealthy         bool          `json:"is_healthy"`
	LastError         string        `json:"last_error,omitempty"`
	LastErrorTime     time.Time     `json:"last_error_time,omitempty"`
}

// HealthReporter manages health tracking for multiple providers
type HealthReporter struct {
	mu        sync.RWMutex
	providers map[string]*ProviderHealth
}

// NewHealthReporter creates a new health reporter
func NewHealthReporter() *HealthReporter {
	return &HealthReporter{
		providers: make(map[string]*ProviderHealth),
	}
}

// GetOrCreateProvider gets or creates a health tracker for a provider
func (r *HealthReporter) GetOrCreateProvider(name string) *ProviderHealth {
	r.mu.Lock()
	defer r.mu.Unlock()

	if provider, exists := r.providers[name]; exists {
		return provider
	}

	provider := NewProviderHealth(name)
	r.providers[name] = provider
	return provider
}

// RecordSuccess records a successful request for a provider
func (r *HealthReporter) RecordSuccess(providerName string, responseTime time.Duration) {
	provider := r.GetOrCreateProvider(providerName)
	provider.RecordSuccess(responseTime)
}

// RecordError records a failed request for a provider
func (r *HealthReporter) RecordError(providerName string, err error, responseTime time.Duration) {
	provider := r.GetOrCreateProvider(providerName)
	provider.RecordError(err, responseTime)
}

// GetProviderStats returns statistics for a specific provider
func (r *HealthReporter) GetProviderStats(providerName string) (ProviderStats, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, exists := r.providers[providerName]
	if !exists {
		return ProviderStats{}, false
	}

	return provider.GetStats(), true
}

// GetAllStats returns statistics for all providers
func (r *HealthReporter) GetAllStats() map[string]ProviderStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := make(map[string]ProviderStats)
	for name, provider := range r.providers {
		stats[name] = provider.GetStats()
	}

	return stats
}

// GetHealthyProviders returns names of healthy providers
func (r *HealthReporter) GetHealthyProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var healthy []string
	for name, provider := range r.providers {
		if provider.IsHealthy {
			healthy = append(healthy, name)
		}
	}

	return healthy
}

// GetUnhealthyProviders returns names of unhealthy providers
func (r *HealthReporter) GetUnhealthyProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var unhealthy []string
	for name, provider := range r.providers {
		if !provider.IsHealthy {
			unhealthy = append(unhealthy, name)
		}
	}

	return unhealthy
}

// ResetProvider resets statistics for a provider
func (r *HealthReporter) ResetProvider(providerName string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.providers[providerName]; exists {
		// Create new provider with same name
		r.providers[providerName] = NewProviderHealth(providerName)
		return true
	}

	return false
}