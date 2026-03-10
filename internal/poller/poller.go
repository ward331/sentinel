package poller

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/openclaw/sentinel-backend/internal/model"
	"github.com/openclaw/sentinel-backend/internal/provider"
	"github.com/openclaw/sentinel-backend/internal/storage"
)

// Poller manages the scheduling and execution of provider polling
type Poller struct {
	storage    storage.Storage
	providers  map[string]provider.Provider
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	eventChan  chan *model.Event
	stats      *Stats
}

// Stats holds polling statistics
type Stats struct {
	mu               sync.RWMutex
	TotalPolls       int64
	SuccessfulPolls  int64
	FailedPolls      int64
	TotalEvents      int64
	ProviderStats    map[string]*ProviderStats
	LastPollTime     time.Time
	Uptime           time.Duration
}

// ProviderStats holds statistics for a specific provider
type ProviderStats struct {
	Polls       int64
	Successes   int64
	Failures    int64
	Events      int64
	LastPoll    time.Time
	LastSuccess time.Time
	LastError   string
}

// NewPoller creates a new Poller instance
func NewPoller(storage storage.Storage) *Poller {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &Poller{
		storage:   storage,
		providers: make(map[string]provider.Provider),
		ctx:       ctx,
		cancel:    cancel,
		eventChan: make(chan *model.Event, 1000),
		stats: &Stats{
			ProviderStats: make(map[string]*ProviderStats),
			LastPollTime:  time.Now(),
		},
	}
}

// RegisterProvider registers a provider with the poller
func (p *Poller) RegisterProvider(name string, provider provider.Provider) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	p.providers[name] = provider
	
	// Initialize stats for this provider
	p.stats.mu.Lock()
	p.stats.ProviderStats[name] = &ProviderStats{}
	p.stats.mu.Unlock()
	
	log.Printf("Registered provider: %s (interval: %v)", name, provider.Interval())
}

// Start begins polling all registered providers
func (p *Poller) Start() error {
	log.Println("Starting poller...")
	
	// Start event processor
	p.wg.Add(1)
	go p.processEvents()
	
	// Start polling for each provider
	for name, prov := range p.providers {
		if !prov.Enabled() {
			log.Printf("Provider %s is disabled, skipping", name)
			continue
		}
		
		p.wg.Add(1)
		go p.pollProvider(name, prov)
	}
	
	// Start stats updater
	p.wg.Add(1)
	go p.updateStats()
	
	log.Printf("Poller started with %d providers", len(p.providers))
	return nil
}

// Stop gracefully stops the poller
func (p *Poller) Stop() {
	log.Println("Stopping poller...")
	
	// Cancel context to stop all goroutines
	p.cancel()
	
	// Wait for all goroutines to finish
	p.wg.Wait()
	
	// Close event channel
	close(p.eventChan)
	
	log.Println("Poller stopped")
}

// pollProvider continuously polls a single provider
func (p *Poller) pollProvider(name string, prov provider.Provider) {
	defer p.wg.Done()
	
	interval := prov.Interval()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	log.Printf("Starting polling for %s (interval: %v)", name, interval)
	
	// Initial poll
	p.executePoll(name, prov)
	
	for {
		select {
		case <-p.ctx.Done():
			log.Printf("Stopping polling for %s", name)
			return
		case <-ticker.C:
			p.executePoll(name, prov)
		}
	}
}

// executePoll performs a single poll for a provider
func (p *Poller) executePoll(name string, prov provider.Provider) {
	p.stats.mu.Lock()
	p.stats.TotalPolls++
	p.stats.ProviderStats[name].Polls++
	p.stats.ProviderStats[name].LastPoll = time.Now()
	p.stats.LastPollTime = time.Now()
	p.stats.mu.Unlock()
	
	ctx, cancel := context.WithTimeout(p.ctx, 30*time.Second)
	defer cancel()
	
	log.Printf("Polling provider: %s", name)
	
	startTime := time.Now()
	events, err := prov.Fetch(ctx)
	duration := time.Since(startTime)
	
	p.stats.mu.Lock()
	defer p.stats.mu.Unlock()
	
	if err != nil {
		p.stats.FailedPolls++
		p.stats.ProviderStats[name].Failures++
		p.stats.ProviderStats[name].LastError = err.Error()
		
		log.Printf("Poll failed for %s: %v (duration: %v)", name, err, duration)
		return
	}
	
	p.stats.SuccessfulPolls++
	p.stats.ProviderStats[name].Successes++
	p.stats.ProviderStats[name].LastSuccess = time.Now()
	p.stats.ProviderStats[name].LastError = ""
	
	log.Printf("Poll successful for %s: %d events (duration: %v)", name, len(events), duration)
	
	// Send events to processor
	for _, event := range events {
		p.eventChan <- event
		p.stats.TotalEvents++
		p.stats.ProviderStats[name].Events++
	}
}

// processEvents processes events from the event channel
func (p *Poller) processEvents() {
	defer p.wg.Done()
	
	log.Println("Starting event processor")
	
	for {
		select {
		case <-p.ctx.Done():
			log.Println("Stopping event processor")
			return
		case event := <-p.eventChan:
			p.storeEvent(event)
		}
	}
}

// storeEvent stores an event in the database
func (p *Poller) storeEvent(event *model.Event) {
	ctx, cancel := context.WithTimeout(p.ctx, 5*time.Second)
	defer cancel()
	
	// Check for duplicate events
	existing, err := p.storage.GetEventBySourceID(ctx, event.Source, event.SourceID)
	if err == nil && existing != nil {
		// Event already exists, skip
		return
	}
	
	// Store the event
	if err := p.storage.StoreEvent(ctx, event); err != nil {
		log.Printf("Failed to store event %s/%s: %v", event.Source, event.SourceID, err)
		return
	}
	
	log.Printf("Stored event: %s/%s", event.Source, event.SourceID)
}

// updateStats periodically updates poller statistics
func (p *Poller) updateStats() {
	defer p.wg.Done()
	
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	startTime := time.Now()
	
	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.stats.mu.Lock()
			p.stats.Uptime = time.Since(startTime)
			p.stats.mu.Unlock()
		}
	}
}

// GetStats returns current polling statistics
func (p *Poller) GetStats() *Stats {
	p.stats.mu.RLock()
	defer p.stats.mu.RUnlock()
	
	// Create a copy of stats
	statsCopy := &Stats{
		TotalPolls:      p.stats.TotalPolls,
		SuccessfulPolls: p.stats.SuccessfulPolls,
		FailedPolls:     p.stats.FailedPolls,
		TotalEvents:     p.stats.TotalEvents,
		LastPollTime:    p.stats.LastPollTime,
		Uptime:          p.stats.Uptime,
		ProviderStats:   make(map[string]*ProviderStats),
	}
	
	// Copy provider stats
	for name, stat := range p.stats.ProviderStats {
		statsCopy.ProviderStats[name] = &ProviderStats{
			Polls:       stat.Polls,
			Successes:   stat.Successes,
			Failures:    stat.Failures,
			Events:      stat.Events,
			LastPoll:    stat.LastPoll,
			LastSuccess: stat.LastSuccess,
			LastError:   stat.LastError,
		}
	}
	
	return statsCopy
}

// GetProviderNames returns a list of registered provider names
func (p *Poller) GetProviderNames() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	names := make([]string, 0, len(p.providers))
	for name := range p.providers {
		names = append(names, name)
	}
	return names
}

// GetProvider returns a provider by name
func (p *Poller) GetProvider(name string) (provider.Provider, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	prov, exists := p.providers[name]
	return prov, exists
}

// SetProviderEnabled enables or disables a provider
func (p *Poller) SetProviderEnabled(name string, enabled bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	prov, exists := p.providers[name]
	if !exists {
		return fmt.Errorf("provider %s not found", name)
	}
	
	// This would need to update the provider's config
	// For now, we'll just log the change
	log.Printf("Provider %s enabled: %v", name, enabled)
	
	_ = prov // Keep compiler happy
	
	return nil
}