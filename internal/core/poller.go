package core

import (
	"context"
	"log"
	"time"

	"github.com/openclaw/sentinel-backend/internal/alert"
	"github.com/openclaw/sentinel-backend/internal/api"
	"github.com/openclaw/sentinel-backend/internal/metrics"
	"github.com/openclaw/sentinel-backend/internal/model"
	"github.com/openclaw/sentinel-backend/internal/provider"
	"github.com/openclaw/sentinel-backend/internal/storage"
)

// Poller manages background polling of event providers
type Poller struct {
	storage     *storage.Storage
	stream      *api.StreamBroker
	alertEngine *alert.RuleEngine
	metrics     *metrics.Metrics
	providers   []model.Provider
	interval    time.Duration
}

// NewPoller creates a new poller
func NewPoller(storage *storage.Storage, stream *api.StreamBroker, metrics *metrics.Metrics, interval time.Duration) *Poller {
	return &Poller{
		storage:     storage,
		stream:      stream,
		alertEngine: alert.NewRuleEngine(),
		metrics:     metrics,
		interval:    interval,
		providers: []model.Provider{
			provider.NewUSGSProvider(),
			provider.NewGDACSProvider(),
			// Add more providers here as needed
		},
	}
}

// Start begins polling all providers in the background
func (p *Poller) Start(ctx context.Context) {
	log.Printf("Starting poller with %d provider(s), interval: %v", len(p.Providers()), p.interval)
	
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

// pollAll polls all registered providers
func (p *Poller) pollAll() {
	for _, provider := range p.providers {
		go p.pollProvider(provider)
	}
}

// pollProvider polls a single provider
func (p *Poller) pollProvider(provider model.Provider) {
	providerName := provider.Name()
	
	log.Printf("Polling %s provider...", providerName)
	
	// Record provider poll
	if p.metrics != nil {
		p.metrics.RecordProviderPoll(providerName)
	}
	
	events, err := provider.Fetch(context.Background())
	if err != nil {
		log.Printf("Failed to fetch from %s: %v", providerName, err)
		if p.metrics != nil {
			p.metrics.RecordProviderError(providerName)
		}
		return
	}
	
	log.Printf("%s returned %d event(s)", providerName, len(events))
	
	// Record provider events
	if p.metrics != nil {
		p.metrics.RecordProviderEvents(providerName, len(events))
	}
	
	newCount := 0
	filteredCount := 0
	for _, event := range events {
		// Check if event already exists by source_id
		if event.SourceID != "" {
			existing, err := p.storage.GetEventBySourceID(context.Background(), event.Source, event.SourceID)
			if err == nil && existing != nil {
				// Event already exists, skip
				filteredCount++
				if p.metrics != nil {
					p.metrics.RecordEventFiltered()
				}
				continue
			}
		}
		
		// Store new event
		if err := p.storage.StoreEvent(context.Background(), event); err != nil {
			log.Printf("Failed to store event from %s: %v", providerName, err)
			continue
		}
		
		// Record ingested event
		if p.metrics != nil {
			p.metrics.RecordEventIngested()
		}
		
		// Evaluate alert rules
		if p.alertEngine != nil {
			triggered := p.alertEngine.Evaluate(event)
			if len(triggered) > 0 {
				log.Printf("Alert triggered for event %s: %d rule(s) matched", event.ID, len(triggered))
				if p.metrics != nil {
					p.metrics.RecordAlertTriggered()
					p.metrics.RecordAlertProcessed()
				}
			}
		}
		
		// Broadcast to SSE clients
		if p.stream != nil {
			p.stream.Broadcast(event)
			log.Printf("Broadcasted new event from %s to SSE clients: %s", providerName, event.ID)
			if p.metrics != nil {
				p.metrics.RecordEventBroadcast()
			}
		}
		
		newCount++
	}
	
	if newCount > 0 {
		log.Printf("Ingested %d new event(s) from %s (filtered: %d)", newCount, providerName, filteredCount)
	} else {
		log.Printf("No new events from %s (filtered: %d)", providerName, filteredCount)
	}
}

// Providers returns the list of registered providers
func (p *Poller) Providers() []model.Provider {
	return p.providers
}