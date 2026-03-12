package metrics

import (
	"sync"
	"time"
)

// Metrics tracks system metrics
type Metrics struct {
	mu sync.RWMutex

	// Event metrics
	eventsProcessedTotal int64
	eventsByProvider     map[string]int64

	// Timing metrics
	startTime time.Time
}

// NewMetrics creates a new metrics tracker
func NewMetrics() *Metrics {
	return &Metrics{
		eventsByProvider: make(map[string]int64),
		startTime:        time.Now(),
	}
}

// IncrementEventsProcessed increments the count of processed events for a provider
func (m *Metrics) IncrementEventsProcessed(provider string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.eventsProcessedTotal++
	m.eventsByProvider[provider]++
}

// Get returns current metrics
func (m *Metrics) Get() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	uptime := time.Since(m.startTime)

	return map[string]interface{}{
		"events_processed_total": m.eventsProcessedTotal,
		"events_by_provider":     m.eventsByProvider,
		"uptime_seconds":         uptime.Seconds(),
		"start_time":             m.startTime.Format(time.RFC3339),
	}
}

// Reset resets all metrics
func (m *Metrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.eventsProcessedTotal = 0
	m.eventsByProvider = make(map[string]int64)
	m.startTime = time.Now()
}

// RecordAPIError records an API error
func (m *Metrics) RecordAPIError(endpoint string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Stub method - just increment a counter
	m.eventsProcessedTotal++ // Using existing counter for simplicity
}

// RecordAPIRequest records an API request
func (m *Metrics) RecordAPIRequest(endpoint string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Stub method - just increment a counter
	m.eventsProcessedTotal++ // Using existing counter for simplicity
}

// RecordEventIngested records an event ingestion
func (m *Metrics) RecordEventIngested() {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Stub method - just increment a counter
	m.eventsProcessedTotal++
}

// RecordEventBroadcast records an event broadcast
func (m *Metrics) RecordEventBroadcast() {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Stub method - just increment a counter
	m.eventsProcessedTotal++
}

// RecordAlertTriggered records an alert trigger
func (m *Metrics) RecordAlertTriggered() {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Stub method - just increment a counter
	m.eventsProcessedTotal++
}

// RecordAlertProcessed records an alert processing
func (m *Metrics) RecordAlertProcessed() {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Stub method - just increment a counter
	m.eventsProcessedTotal++
}

// RecordProviderPoll records a provider poll
func (m *Metrics) RecordProviderPoll(providerName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Stub method - just increment a counter
	m.eventsProcessedTotal++
}

// RecordProviderError records a provider error
func (m *Metrics) RecordProviderError(providerName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Stub method - just increment a counter
	m.eventsProcessedTotal++
}

// RecordProviderEvents records provider events
func (m *Metrics) RecordProviderEvents(providerName string, count int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Stub method - just increment a counter
	m.eventsProcessedTotal += int64(count)
}

// RecordEventFiltered records filtered events
func (m *Metrics) RecordEventFiltered() {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Stub method - just increment a counter
	m.eventsProcessedTotal++
}