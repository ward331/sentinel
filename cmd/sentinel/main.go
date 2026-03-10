package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/openclaw/sentinel-backend/internal/api"
	"github.com/openclaw/sentinel-backend/internal/config"
	"github.com/openclaw/sentinel-backend/internal/health"
	"github.com/openclaw/sentinel-backend/internal/metrics"
	"github.com/openclaw/sentinel-backend/internal/poller"
	"github.com/openclaw/sentinel-backend/internal/provider"
	"github.com/openclaw/sentinel-backend/internal/storage"
)

var (
	configPath = flag.String("config", "", "Path to config file")
	dataDir    = flag.String("data-dir", "", "Override data directory")
	port       = flag.Int("port", 8080, "Server port (default 8080)")
	host       = flag.String("host", "", "Bind host (default 0.0.0.0)")
	version    = flag.Bool("version", false, "Print version and exit")
)

const Version = "2.0.0"

func main() {
	flag.Parse()
	
	if *version {
		fmt.Printf("SENTINEL v%s\n", Version)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Start server
	if err := startServer(cfg); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// loadConfig loads configuration with CLI overrides
func loadConfig() (*config.Config, error) {
	// Try to load V2 config first
	if *configPath != "" {
		cfg, err := config.LoadConfig(*configPath)
		if err == nil {
			return cfg, nil
		}
	}

	// Fall back to V1 config
	v1Config := config.LoadConfigV1()
	
	// Apply CLI overrides to V1 config
	// Port flag now has default 8080, so always use it
	v1Config.HTTPPort = fmt.Sprintf("%d", *port)
	if *host != "" {
		v1Config.HTTPHost = *host
	}
	if *dataDir != "" {
		// Update all paths in V1 config
		v1Config.DBPath = *dataDir + "/sentinel.db"
		v1Config.BackupDir = *dataDir + "/backups"
		v1Config.EventLogPath = *dataDir + "/events.ndjson"
	}
	
	// Convert V1 config to V2 config
	return config.MigrateFromV1(v1Config), nil
}

// startServer starts the HTTP server with poller integration
func startServer(cfg *config.Config) error {
	// Initialize storage
	store, err := storage.NewWithConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer store.Close()

	// Initialize OSINT storage
	osintStorage := storage.NewOSINTStorage(store.DB())
	ctx := context.Background()
	if err := osintStorage.CreateTable(ctx); err != nil {
		log.Printf("Warning: Failed to create OSINT table: %v", err)
	}
	if err := osintStorage.SeedBuiltinResources(ctx); err != nil {
		log.Printf("Warning: Failed to seed OSINT resources: %v", err)
	}

	// Create metrics and health registry
	metrics := metrics.NewMetrics()
	healthRegistry := health.NewHealthRegistry()

	// Create and start poller
	poller := initializePoller(store, cfg)
	if poller != nil {
		if err := poller.Start(); err != nil {
			log.Printf("Warning: Failed to start poller: %v", err)
		} else {
			defer poller.Stop()
			log.Printf("Poller started with %d providers", len(poller.GetProviderNames()))
		}
	}

	// Create API handler and router
	apiHandler := api.NewHandler(store, metrics, healthRegistry)
	router := apiHandler.Router()
	
	// Register OSINT resources routes
	osintHandler := api.NewOSINTResourcesHandler(osintStorage)
	osintHandler.RegisterRoutes(router.PathPrefix("/api/osint").Subrouter())

	// Create HTTP server
	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler: router,
	}

	log.Printf("Starting SENTINEL server on %s", srv.Addr)
	log.Printf("Data directory: %s", cfg.DataDir)
	log.Printf("Providers registered: %d", len(poller.GetProviderNames()))
	
	// Start server in background
	serverErr := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	select {
	case sig := <-sigChan:
		log.Printf("Received signal %v, shutting down...", sig)
	case err := <-serverErr:
		log.Printf("Server error: %v", err)
	}
	
	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}
	
	log.Println("Server shutdown complete")
	return nil
}

// initializePoller creates and registers all providers with the poller
func initializePoller(store storage.Storage, cfg *config.Config) *poller.Poller {
	p := poller.NewPoller(store)
	
	// Create provider config from main config
	providerConfig := &provider.Config{
		Enabled: true,
	}
	
	// Register all 25 providers
	registerProvider(p, "usgs", provider.NewUSGSProvider(providerConfig))
	registerProvider(p, "gdacs", provider.NewGDACSProvider(providerConfig))
	registerProvider(p, "noaa_cap", provider.NewNOAACAPProvider(providerConfig))
	registerProvider(p, "noaa_nws", provider.NewNOAANWSProvider(providerConfig))
	registerProvider(p, "tsunami", provider.NewTsunamiProvider(providerConfig))
	registerProvider(p, "volcano", provider.NewVolcanoProvider(providerConfig))
	registerProvider(p, "reliefweb", provider.NewReliefWebProvider(providerConfig))
	registerProvider(p, "opensky", provider.NewOpenSkyEnhancedProvider(providerConfig))
	registerProvider(p, "airplanes_live", provider.NewAirplanesLiveProvider(providerConfig))
	registerProvider(p, "adsb_one", provider.NewADSBOneProvider(providerConfig))
	registerProvider(p, "openmeteo", provider.NewOpenMeteoProvider(providerConfig))
	registerProvider(p, "iranconflict", provider.NewIranConflictProvider(providerConfig))
	registerProvider(p, "liveuamap", provider.NewLiveUAMapProvider(providerConfig))
	registerProvider(p, "gdelt", provider.NewGDELTProvider(providerConfig))
	registerProvider(p, "opensanctions", provider.NewOpenSanctionsProvider(providerConfig))
	registerProvider(p, "globalforestwatch", provider.NewGlobalForestWatchProvider(providerConfig))
	registerProvider(p, "globalfishingwatch", provider.NewGlobalFishingWatchProvider(providerConfig))
	registerProvider(p, "celestrak", provider.NewCelesTrakProvider(providerConfig))
	registerProvider(p, "swpc", provider.NewSWPCProvider(providerConfig))
	registerProvider(p, "who", provider.NewWHOProvider(providerConfig))
	registerProvider(p, "promed", provider.NewProMEDProvider(providerConfig))
	registerProvider(p, "nasa_firms", provider.NewNASAFIRMSProvider(providerConfig))
	registerProvider(p, "piracy_imb", provider.NewPiracyIMBProvider(providerConfig))
	registerProvider(p, "financial_markets", provider.NewFinancialMarketsProvider(providerConfig))
	
	return p
}

// registerProvider registers a provider with the poller
func registerProvider(p *poller.Poller, name string, prov provider.Provider) {
	p.RegisterProvider(name, prov)
	log.Printf("Registered provider: %s (interval: %v)", name, prov.Interval())
}