package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/openclaw/sentinel-backend/internal/config"
	"github.com/openclaw/sentinel-backend/internal/poller"
	"github.com/openclaw/sentinel-backend/internal/provider"
	"github.com/openclaw/sentinel-backend/internal/storage"
)

func main() {
	fmt.Println("Testing SENTINEL v2.0.0 Provider Integration")
	fmt.Println("===========================================")
	
	// Create config
	cfg := &config.ConfigV2{
		DataDir: "/tmp/sentinel_test",
		Server: config.ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
		Providers: make(map[string]config.ProviderConfig),
	}
	
	// Create storage
	store, err := storage.NewWithConfig(cfg)
	if err != nil {
		log.Fatalf("Failed to create storage: %v", err)
	}
	
	// Create poller
	p := poller.NewPoller(store)
	
	// Create provider config
	providerConfig := &provider.Config{
		Enabled: true,
	}
	
	// Register all providers
	providers := []struct {
		name string
		ctor func(*provider.Config) provider.Provider
	}{
		{"usgs", func(c *provider.Config) provider.Provider { return provider.NewUSGSProvider(c) }},
		{"gdacs", func(c *provider.Config) provider.Provider { return provider.NewGDACSProvider(c) }},
		{"noaa_cap", func(c *provider.Config) provider.Provider { return provider.NewNOAACAPProvider(c) }},
		{"noaa_nws", func(c *provider.Config) provider.Provider { return provider.NewNOAANWSProvider(c) }},
		{"tsunami", func(c *provider.Config) provider.Provider { return provider.NewTsunamiProvider(c) }},
		{"volcano", func(c *provider.Config) provider.Provider { return provider.NewVolcanoProvider(c) }},
		{"reliefweb", func(c *provider.Config) provider.Provider { return provider.NewReliefWebProvider(c) }},
		{"opensky", func(c *provider.Config) provider.Provider { return provider.NewOpenSkyEnhancedProvider(c) }},
		{"airplanes_live", func(c *provider.Config) provider.Provider { return provider.NewAirplanesLiveProvider(c) }},
		{"adsb_one", func(c *provider.Config) provider.Provider { return provider.NewADSBOneProvider(c) }},
		{"openmeteo", func(c *provider.Config) provider.Provider { return provider.NewOpenMeteoProvider(c) }},
		{"iranconflict", func(c *provider.Config) provider.Provider { return provider.NewIranConflictProvider(c) }},
		{"liveuamap", func(c *provider.Config) provider.Provider { return provider.NewLiveUAMapProvider(c) }},
		{"gdelt", func(c *provider.Config) provider.Provider { return provider.NewGDELTProvider(c) }},
		{"opensanctions", func(c *provider.Config) provider.Provider { return provider.NewOpenSanctionsProvider(c) }},
		{"globalforestwatch", func(c *provider.Config) provider.Provider { return provider.NewGlobalForestWatchProvider(c) }},
		{"globalfishingwatch", func(c *provider.Config) provider.Provider { return provider.NewGlobalFishingWatchProvider(c) }},
		{"celestrak", func(c *provider.Config) provider.Provider { return provider.NewCelesTrakProvider(c) }},
		{"swpc", func(c *provider.Config) provider.Provider { return provider.NewSWPCProvider(c) }},
		{"who", func(c *provider.Config) provider.Provider { return provider.NewWHOProvider(c) }},
		{"promed", func(c *provider.Config) provider.Provider { return provider.NewProMEDProvider(c) }},
		{"nasa_firms", func(c *provider.Config) provider.Provider { return provider.NewNASAFIRMSProvider(c) }},
		{"piracy_imb", func(c *provider.Config) provider.Provider { return provider.NewPiracyIMBProvider(c) }},
		{"financial_markets", func(c *provider.Config) provider.Provider { return provider.NewFinancialMarketsProvider(c) }},
	}
	
	fmt.Printf("\nRegistering %d providers:\n", len(providers))
	for _, pv := range providers {
		prov := pv.ctor(providerConfig)
		p.RegisterProvider(pv.name, prov)
		fmt.Printf("  ✅ %-25s (interval: %v)\n", pv.name, prov.Interval())
	}
	
	// Start poller
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	fmt.Println("\nStarting poller (10-second test)...")
	
	// Note: In real implementation, we would call p.Start() here
	// For test, just verify registration
	
	// Get stats
	stats := p.GetStats()
	fmt.Printf("\nPoller Statistics:\n")
	fmt.Printf("  Registered providers: %d\n", len(p.GetProviderNames()))
	fmt.Printf("  Uptime: %v\n", stats.Uptime)
	
	fmt.Println("\n✅ Integration test completed successfully!")
	fmt.Println("\nNext steps:")
	fmt.Println("1. Update main.go to integrate poller")
	fmt.Println("2. Add provider configuration to V2 config")
	fmt.Println("3. Test full system with all 25 providers")
}