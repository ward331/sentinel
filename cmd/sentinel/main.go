package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
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
	configPath = flag.String("config", "", "Path to configuration file")
	dataDir    = flag.String("data-dir", "", "Data directory for database and files")
	port       = flag.Int("port", 8080, "Port to listen on")
	host       = flag.String("host", "localhost", "Host to bind to")
	version    = flag.Bool("version", false, "Show version information")
)

// Version is set during build
var Version = "v2.0.0"

func main() {
	flag.Parse()

	if *version {
		fmt.Printf("SENTINEL %s\n", Version)
		return
	}

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Start server
	if err := startServer(cfg); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// loadConfig loads configuration from file or CLI flags
func loadConfig() (*config.Config, error) {
	// Start with default config
	cfg := config.DefaultConfig()

	// Load from file if specified
	if *configPath != "" {
		fileConfig, err := config.LoadConfig(*configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load config from %s: %w", *configPath, err)
		}
		cfg = fileConfig
	}

	// Override with CLI flags
	if *dataDir != "" {
		cfg.DataDir = *dataDir
	}
	if *port != 8080 { // Only override if not default
		cfg.Server.Port = *port
	}
	if *host != "localhost" { // Only override if not default
		cfg.Server.Host = *host
	}

	// Ensure data directory exists
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory %s: %w", cfg.DataDir, err)
	}

	return cfg, nil
}

// startServer starts the HTTP server with poller integration
func startServer(cfg *config.Config) error {
	// Initialize storage
	dbPath := filepath.Join(cfg.DataDir, "sentinel.db")
	store, err := storage.NewWithConfig(dbPath, true, 10)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer store.Close()

	// Initialize poller
	pollerInstance := initializePoller(store, cfg)
	
	// Start poller
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	pollerInstance.Start()
	defer pollerInstance.Stop()
	log.Printf("Poller started with %d providers", len(pollerInstance.GetProviderNames()))

	// Initialize metrics and health
	metrics := metrics.NewMetrics()
	healthRegistry := health.NewHealthRegistry()

	// Create API handler and router
	apiHandler := api.NewHandler(store, metrics, healthRegistry)
	router := apiHandler.Router()

	// Initialize OSINT storage and add routes
	osintStorage := storage.NewOSINTStorage(store.DB())
	osintHandler := api.NewOSINTResourcesHandler(osintStorage)
	osintRouter := router.PathPrefix("/api/osint").Subrouter()
	osintHandler.RegisterRoutes(osintRouter)
	log.Printf("OSINT resources API initialized")

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		log.Printf("Starting SENTINEL server on %s", server.Addr)
		log.Printf("Data directory: %s", cfg.DataDir)
		log.Printf("Database: %s", dbPath)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		log.Printf("Received signal: %v", sig)
	case err := <-serverErr:
		log.Printf("Server error: %v", err)
	}

	// Graceful shutdown
	log.Println("Shutting down server...")
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	log.Println("Server stopped gracefully")
	return nil
}

// initializePoller creates and configures the poller with all providers
func initializePoller(store *storage.Storage, cfg *config.Config) *poller.Poller {
	p := poller.NewPoller(store)
	
	// Create provider config from main config
	providerConfig := &provider.Config{
		Enabled:      true,
		PollInterval: 5 * time.Minute, // Default interval
	}
	
	// Register all providers
	registerProvider(p, "usgs", provider.NewUSGSProvider(providerConfig))
	registerProvider(p, "gdacs", provider.NewGDACSProvider(providerConfig))
	registerProvider(p, "noaa_cap", provider.NewNOAACAPProvider(providerConfig))
	registerProvider(p, "noaa_nws", provider.NewNOAANWSProvider(providerConfig))
	registerProvider(p, "tsunami", provider.NewTsunamiProvider(providerConfig))
	registerProvider(p, "volcano", provider.NewVolcanoProvider(providerConfig))
	registerProvider(p, "reliefweb", provider.NewReliefWebProvider(providerConfig))
	// registerProvider(p, "opensky", provider.NewOpenSkyEnhancedProvider(nil)) // Requires aircraft database
	registerProvider(p, "airplanes_live", provider.NewAirplanesLiveProvider(providerConfig))
	registerProvider(p, "adsb_one", provider.NewADSBOneProvider(providerConfig))
	registerProvider(p, "openmeteo", provider.NewOpenMeteoProvider())
	registerProvider(p, "iranconflict", provider.NewIranConflictProvider())
	registerProvider(p, "liveuamap", provider.NewLiveUAMapProvider(providerConfig))
	registerProvider(p, "gdelt", provider.NewGDELTProvider())
	// registerProvider(p, "opensanctions", provider.NewOpenSanctionsProvider("")) // Requires API key
	registerProvider(p, "globalforestwatch", provider.NewGlobalForestWatchProvider())
	// registerProvider(p, "globalfishingwatch", provider.NewGlobalFishingWatchProvider("")) // Requires API key
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
	log.Printf("Registered provider: %s (interval: %v, enabled: %v)", name, prov.Interval(), prov.Enabled())
}