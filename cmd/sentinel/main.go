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

// startServer starts the HTTP server
func startServer(cfg *config.Config) error {
	// Initialize storage
	dbPath := filepath.Join(cfg.DataDir, "sentinel.db")
	store, err := storage.NewWithConfig(dbPath, true, 10)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer store.Close()

	// Initialize metrics and health
	metrics := metrics.NewMetrics()
	healthRegistry := health.NewHealthRegistry()

	// Create API handler and router
	apiHandler := api.NewHandler(store, metrics, healthRegistry)
	router := apiHandler.Router()

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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	log.Println("Server stopped gracefully")
	return nil
}