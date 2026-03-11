package main

import (
	"context"
	"embed"
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
	"github.com/openclaw/sentinel-backend/internal/engine"
	"github.com/openclaw/sentinel-backend/internal/health"
	"github.com/openclaw/sentinel-backend/internal/metrics"
	"github.com/openclaw/sentinel-backend/internal/poller"
	"github.com/openclaw/sentinel-backend/internal/provider"
	"github.com/openclaw/sentinel-backend/internal/setup"
	"github.com/openclaw/sentinel-backend/internal/storage"
)

//go:embed web/*
var webFS embed.FS

var (
	configPath  = flag.String("config", "", "Path to configuration file")
	dataDir     = flag.String("data-dir", "", "Data directory for database and files")
	port        = flag.Int("port", 8080, "Port to listen on")
	host        = flag.String("host", "localhost", "Host to bind to")
	version     = flag.Bool("version", false, "Show version information")
	wizard      = flag.Bool("wizard", false, "Run first-run setup wizard")
	noFrontend  = flag.Bool("no-frontend", false, "API only — do not serve embedded frontend")
)

// Version is set during build
var Version = "v3.0.0"

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

	// Run wizard if requested or first run
	if *wizard || !cfg.SetupComplete {
		if *wizard {
			if err := setup.RunIfNeeded(cfg); err != nil {
				log.Fatalf("Setup wizard failed: %v", err)
			}
		}
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
	log.Printf("SENTINEL %s starting...", Version)

	// ── 1. Storage ──────────────────────────────────────────
	dbPath := filepath.Join(cfg.DataDir, "sentinel.db")
	store, err := storage.NewWithConfig(dbPath, true, 10)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer store.Close()

	// Run V3 schema migration (safe to run multiple times)
	if err := storage.RunV3Migration(store.DB()); err != nil {
		return fmt.Errorf("V3 migration failed: %w", err)
	}

	// ── 2. Providers & Poller ───────────────────────────────
	pollerInstance := initializePoller(store, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pollerInstance.Start()
	defer pollerInstance.Stop()
	log.Printf("Poller started with %d providers", len(pollerInstance.GetProviderNames()))

	// ── 3. Engine (V3) ──────────────────────────────────────
	correlationEngine := engine.NewCorrelationEngine()
	correlationEngine.SetStorage(store)
	correlationEngine.Start(ctx)
	defer correlationEngine.Stop()

	truthCalc := engine.NewTruthScoreCalculator()
	truthCalc.SetStorage(store)
	truthCalc.Start(ctx)
	defer truthCalc.Stop()

	anomalyDetector := engine.NewAnomalyDetector()
	anomalyDetector.SetStorage(store)
	anomalyDetector.Start(ctx)
	defer anomalyDetector.Stop()

	signalBoard := engine.NewSignalBoard(cfg.SignalBoard.Enabled)
	signalBoard.SetStorage(store)
	signalBoard.Start(ctx)
	defer signalBoard.Stop()

	drMins := 30
	if cfg.EntityTracking.DeadReckoningMins > 0 {
		drMins = cfg.EntityTracking.DeadReckoningMins
	}
	deadReckoning := engine.NewDeadReckoningEngine(drMins)
	deadReckoning.SetStorage(store)
	deadReckoning.Start(ctx)
	defer deadReckoning.Stop()

	log.Printf("Intelligence engines started: correlation, truth, anomaly, signal_board, dead_reckoning")

	// ── 4. API ──────────────────────────────────────────────
	metricsInst := metrics.NewMetrics()
	healthRegistry := health.NewHealthRegistry()

	apiHandler := api.NewHandler(store, metricsInst, healthRegistry)
	apiHandler.SetPoller(pollerInstance)
	apiHandler.SetConfig(cfg)
	router := apiHandler.Router()

	// OSINT resources
	osintStorage := storage.NewOSINTStorage(store.DB())
	osintHandler := api.NewOSINTResourcesHandler(osintStorage)
	osintRouter := router.PathPrefix("/api/osint").Subrouter()
	osintHandler.RegisterRoutes(osintRouter)
	log.Printf("OSINT resources API initialized")
	log.Printf("Filter API: Basic filtering available via query parameters")

	// ── 5. Middleware ───────────────────────────────────────
	var handler http.Handler = router

	handler = api.CORSMiddleware(handler)

	rateLimitConfig := api.DefaultRateLimitConfig()
	rateLimitConfig.Enabled = true
	rateLimitConfig.RPS = 100
	rateLimitConfig.Burst = 200
	handler = api.RateLimitMiddleware(rateLimitConfig)(handler)

	authConfig := api.DefaultAuthConfig()
	authConfig.Enabled = false
	authConfig.APIKeys = []string{"test-api-key-123"}
	handler = api.AuthMiddleware(authConfig)(handler)

	log.Printf("Security middleware applied: CORS=%v, RateLimit=%v, Auth=%v",
		true, rateLimitConfig.Enabled, authConfig.Enabled)

	// ── 6. Serve ────────────────────────────────────────────
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("Starting SENTINEL server on %s", server.Addr)
		log.Printf("Data directory: %s", cfg.DataDir)
		log.Printf("Database: %s", dbPath)
		if !*noFrontend {
			log.Printf("Frontend: embedded (web/)")
		} else {
			log.Printf("Frontend: disabled (--no-frontend)")
		}
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Wait for interrupt
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
		PollInterval: 5 * time.Minute,
	}

	// Register all providers
	registerProvider(p, "usgs", provider.NewUSGSProvider(providerConfig))
	registerProvider(p, "gdacs", provider.NewGDACSProvider(providerConfig))
	registerProvider(p, "noaa_cap", provider.NewNOAACAPProvider(providerConfig))
	registerProvider(p, "noaa_nws", provider.NewNOAANWSProvider(providerConfig))
	registerProvider(p, "tsunami", provider.NewTsunamiProvider(providerConfig))
	registerProvider(p, "volcano", provider.NewVolcanoProvider(providerConfig))
	registerProvider(p, "reliefweb", provider.NewReliefWebProvider(providerConfig))
	registerProvider(p, "airplanes_live", provider.NewAirplanesLiveProvider(providerConfig))
	registerProvider(p, "adsb_one", provider.NewADSBOneProvider(providerConfig))
	registerProvider(p, "openmeteo", provider.NewOpenMeteoProvider())
	registerProvider(p, "iranconflict", provider.NewIranConflictProvider())
	registerProvider(p, "liveuamap", provider.NewLiveUAMapProvider(providerConfig))
	registerProvider(p, "gdelt", provider.NewGDELTProvider())
	registerProvider(p, "globalforestwatch", provider.NewGlobalForestWatchProvider())
	registerProvider(p, "celestrak", provider.NewCelesTrakProvider(providerConfig))
	registerProvider(p, "swpc", provider.NewSWPCProvider(providerConfig))
	registerProvider(p, "who", provider.NewWHOProvider(providerConfig))
	registerProvider(p, "promed", provider.NewProMEDProvider(providerConfig))
	registerProvider(p, "nasa_firms", provider.NewNASAFIRMSProvider(providerConfig))
	registerProvider(p, "piracy_imb", provider.NewPiracyIMBProvider(providerConfig))
	registerProvider(p, "financial_markets", provider.NewFinancialMarketsProvider(providerConfig))
	registerProvider(p, "opensanctions", provider.NewOpenSanctionsProvider(""))
	registerProvider(p, "pikud_haoref", provider.NewPikudHaOrefProvider())
	registerProvider(p, "ukraine_alerts", provider.NewUkraineAlertsProvider())
	registerProvider(p, "ukmto", provider.NewUKMTOProvider())
	registerProvider(p, "sec_edgar", provider.NewSECEdgarProvider())
	registerProvider(p, "cisa_kev", provider.NewCISAKEVProvider())
	registerProvider(p, "otx_alienvault", provider.NewOTXAlienVaultProvider())
	registerProvider(p, "bellingcat", provider.NewBellingcatProvider())
	registerProvider(p, "isw", provider.NewISWProvider())

	return p
}

// registerProvider registers a provider with the poller
func registerProvider(p *poller.Poller, name string, prov provider.Provider) {
	p.RegisterProvider(name, prov)
	log.Printf("Registered provider: %s (interval: %v, enabled: %v)", name, prov.Interval(), prov.Enabled())
}
