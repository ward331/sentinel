package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/openclaw/sentinel-backend/internal/api"
	"github.com/openclaw/sentinel-backend/internal/config"
	"github.com/openclaw/sentinel-backend/internal/health"
	"github.com/openclaw/sentinel-backend/internal/metrics"
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

// startServer starts the HTTP server
func startServer(cfg *config.Config) error {
	// Initialize storage
	store, err := storage.NewWithConfig(cfg.DataDir+"/sentinel.db", true, 10)
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
	
	// Start server in background
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-context.Background().Done()
	
	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	return srv.Shutdown(ctx)
}