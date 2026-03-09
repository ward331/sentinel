package core

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/openclaw/sentinel-backend/internal/alert"
	"github.com/openclaw/sentinel-backend/internal/api"
	"github.com/openclaw/sentinel-backend/internal/infrastructure"
	"github.com/openclaw/sentinel-backend/internal/metrics"
	"github.com/openclaw/sentinel-backend/internal/model"
	"github.com/openclaw/sentinel-backend/internal/provider"
	"github.com/openclaw/sentinel-backend/internal/storage"
)

// EnhancedPoller manages background polling with data infrastructure features
type EnhancedPoller struct {
	storage       *storage.Storage
	stream        *api.StreamBroker
	alertEngine   *alert.RuleEngine
	metrics       *metrics.Metrics
	healthReporter *infrastructure.HealthReporter
	eventLog      *infrastructure.NDJSONLog
	providers     []model.Provider
	interval      time.Duration
}

// NewEnhancedPoller creates a new enhanced poller with data infrastructure
func NewEnhancedPoller(
	storage *storage.Storage,
	stream *api.StreamBroker,
	metrics *metrics.Metrics,
	healthReporter *infrastructure.HealthReporter,
	eventLog *infrastructure.NDJSONLog,
	interval time.Duration,
) *EnhancedPoller {
	// Create OpenSky provider (credentials can be empty for public API)
	openskyProvider := provider.NewOpenSkyProvider("", "") // Public API doesn't require auth
	
	return &EnhancedPoller{
		storage:       storage,
		stream:        stream,
		alertEngine:   alert.NewRuleEngine(),
		metrics:       metrics,
		healthReporter: healthReporter,
		eventLog:      eventLog,
		interval:      interval,
		providers: []model.Provider{
			provider.NewUSGSProvider(),
			provider.NewGDACSProvider(),
			openskyProvider,
		},
	}
}

// Start begins polling all providers in the background
func (p *EnhancedPoller) Start(ctx context.Context) {
	log.Printf("Starting enhanced poller with %d provider(s), interval: %v", len(p.Providers()), p.interval)
	log.Printf("Data infrastructure: NDJSON log at %s", p.eventLog.GetLogPath())
	
	// Log initial provider health
	p.logProviderHealth()
	
	// Run initial poll immediately
	p.pollAll()
	
	// Start periodic polling
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Poller stopping...")
			return
		case <-ticker.C:
			p.pollAll()
		}
	}
}

// pollAll polls all providers
func (p *EnhancedPoller) pollAll() {
	for _, provider := range p.providers {
		go p.pollProvider(provider)
	}
}

// pollProvider polls a single provider
func (p *EnhancedPoller) pollProvider(provider model.Provider) {
	startTime := time.Now()
	
	// Use longer timeout for OpenSky which fetches many flights
	timeout := 30 * time.Second
	if provider.Name() == "opensky" {
		timeout = 60 * time.Second
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	log.Printf("Polling %s provider (timeout: %v)...", provider.Name(), timeout)

	var events []*model.Event
	var err error

	// Record success or error with timing
	defer func() {
		responseTime := time.Since(startTime)
		if err != nil {
			p.healthReporter.RecordError(provider.Name(), err, responseTime)
			log.Printf("❌ Failed to poll %s: %v (took %v)", provider.Name(), err, responseTime)
		} else {
			p.healthReporter.RecordSuccess(provider.Name(), responseTime)
			log.Printf("✅ Successfully polled %s: %d events (took %v)", provider.Name(), len(events), responseTime)
		}
	}()

	// Fetch events from provider
	events, err = provider.Fetch(ctx)
	if err != nil {
		return
	}

	// Process each event
	newEvents := 0
	for _, event := range events {
		// Check if event already exists
		existing, err := p.storage.GetEventBySourceID(ctx, event.Source, event.SourceID)
		if err == nil && existing != nil {
			continue // Skip duplicate
		}

		// Store event in database
		if err := p.storage.StoreEvent(ctx, event); err != nil {
			log.Printf("Failed to store event from %s: %v", provider.Name(), err)
			continue
		}

		// Append to NDJSON log
		if p.eventLog != nil {
			if err := p.eventLog.AppendEvent(event); err != nil {
				log.Printf("Failed to append event to NDJSON log: %v", err)
			}
		}

		// Broadcast to SSE clients
		if p.stream != nil {
			p.stream.Broadcast(event)
		}

		// Evaluate alert rules
		if p.alertEngine != nil {
			p.alertEngine.Evaluate(event)
		}

		// Update metrics
		if p.metrics != nil {
			p.metrics.IncrementEventsProcessed(provider.Name())
		}

		newEvents++
	}

	if newEvents > 0 {
		log.Printf("Ingested %d new events from %s", newEvents, provider.Name())
	}
}

// Providers returns the list of providers
func (p *EnhancedPoller) Providers() []model.Provider {
	return p.providers
}

// GetHealthStats returns health statistics for all providers
func (p *EnhancedPoller) GetHealthStats() map[string]infrastructure.ProviderStats {
	return p.healthReporter.GetAllStats()
}

// GetProviderNames returns names of all providers
func (p *EnhancedPoller) GetProviderNames() []string {
	names := make([]string, len(p.providers))
	for i, provider := range p.providers {
		names[i] = provider.Name()
	}
	return names
}

// logProviderHealth logs current health status of all providers
func (p *EnhancedPoller) logProviderHealth() {
	stats := p.healthReporter.GetAllStats()
	
	log.Println("=== Provider Health Status ===")
	for name, stat := range stats {
		status := "✅ HEALTHY"
		if !stat.IsHealthy {
			status = "❌ UNHEALTHY"
		}
		log.Printf("%s: %s (Requests: %d, Error Rate: %.1f%%, Avg Response: %v)",
			name, status, stat.TotalRequests, stat.ErrorRate, stat.AvgResponseTime)
	}
	log.Println("=============================")
}

// RotateEventLog rotates the NDJSON event log
func (p *EnhancedPoller) RotateEventLog() (string, error) {
	if p.eventLog == nil {
		return "", fmt.Errorf("event log not initialized")
	}
	
	return p.eventLog.Rotate()
}

// GetEventLogPath returns the path to the event log
func (p *EnhancedPoller) GetEventLogPath() string {
	if p.eventLog == nil {
		return ""
	}
	return p.eventLog.GetLogPath()
}

// Close closes the event log
func (p *EnhancedPoller) Close() error {
	if p.eventLog != nil {
		return p.eventLog.Close()
	}
	return nil
}