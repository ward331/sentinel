package provider

import (
	"context"
	"time"

	"github.com/openclaw/sentinel-backend/internal/model"
)

// Provider defines the interface that all data providers must implement
type Provider interface {
	// Fetch retrieves events from the provider's data source
	Fetch(ctx context.Context) ([]*model.Event, error)
	
	// Name returns the provider's unique identifier
	Name() string
	
	// Interval returns the recommended polling interval
	Interval() time.Duration
	
	// Enabled returns whether the provider is enabled
	Enabled() bool
}

// Config holds provider configuration
type Config struct {
	// Provider-specific configuration
	APIKey     string
	Endpoint   string
	Location   string
	BoundingBox []float64 // [minLon, minLat, maxLon, maxLat]
	
	// Polling configuration
	PollInterval time.Duration
	Enabled      bool
	
	// Provider-specific options
	Options map[string]string
}

// BaseProvider provides common functionality for all providers
type BaseProvider struct {
	config *Config
	name   string
}

// NewBaseProvider creates a new BaseProvider
func NewBaseProvider(name string, config *Config) *BaseProvider {
	return &BaseProvider{
		name:   name,
		config: config,
	}
}

// Name returns the provider's name
func (p *BaseProvider) Name() string {
	return p.name
}

// Interval returns the polling interval
func (p *BaseProvider) Interval() time.Duration {
	if p.config != nil && p.config.PollInterval > 0 {
		return p.config.PollInterval
	}
	return 5 * time.Minute // Default interval
}

// Enabled returns whether the provider is enabled
func (p *BaseProvider) Enabled() bool {
	if p.config != nil {
		return p.config.Enabled
	}
	return true // Default to enabled
}

// GetConfig returns the provider's configuration
func (p *BaseProvider) GetConfig() *Config {
	return p.config
}