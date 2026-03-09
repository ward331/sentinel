package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/openclaw/sentinel-backend/internal/api"
	"github.com/openclaw/sentinel-backend/internal/backup"
	"github.com/openclaw/sentinel-backend/internal/config"
	"github.com/openclaw/sentinel-backend/internal/core"
	"github.com/openclaw/sentinel-backend/internal/health"
	"github.com/openclaw/sentinel-backend/internal/infrastructure"
	"github.com/openclaw/sentinel-backend/internal/logging"
	"github.com/openclaw/sentinel-backend/internal/metrics"

	"github.com/openclaw/sentinel-backend/internal/storage"
	"github.com/openclaw/sentinel-backend/internal/server"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()
	
	log.Printf("Starting SENTINEL with configuration:")
	log.Printf("  Database: %s (pooling: %v, max connections: %d)", 
		cfg.DBPath, cfg.ConnectionPool, cfg.MaxConnections)
	log.Printf("  HTTP Server: %s:%s", cfg.HTTPHost, cfg.HTTPPort)
	log.Printf("  Poller Interval: %v", cfg.PollerInterval)
	log.Printf("  Rate Limiting: %v (%d RPS, %d burst)", 
		cfg.RateLimitEnabled, cfg.RateLimitRPS, cfg.RateLimitBurst)

	// Initialize storage with configuration
	store, err := storage.NewWithConfig(cfg.DBPath, cfg.ConnectionPool, cfg.MaxConnections)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()

	// Create metrics collector
	metrics := metrics.NewMetrics()

	// Create health registry
	healthRegistry := health.NewHealthRegistry()
	
	// Register health checkers
	healthRegistry.Register(health.NewDatabaseChecker(store.DB()))
	healthRegistry.Register(health.NewMemoryChecker(400)) // 400 MB threshold
	healthRegistry.Register(health.NewDiskChecker(cfg.DBPath))

	// Initialize data infrastructure components
	healthReporter := infrastructure.NewHealthReporter()
	
	// Create NDJSON event log
	eventLog, err := infrastructure.NewNDJSONLog(cfg.EventLogPath)
	if err != nil {
		log.Fatalf("Failed to create NDJSON event log: %v", err)
	}
	defer eventLog.Close()

	// Initialize API handler with infrastructure (creates stream broker)
	handler := api.NewHandlerWithInfrastructure(
		store, 
		metrics, 
		healthRegistry,
		healthReporter,
		eventLog,
	)

	// Initialize and start enhanced poller with data infrastructure
	poller := core.NewEnhancedPoller(
		store, 
		handler.Stream(), 
		metrics, 
		healthReporter,
		eventLog,
		cfg.PollerInterval,
	)
	
	// Create backup manager
	backupConfig := backup.BackupConfig{
		Enabled:    cfg.BackupEnabled,
		BackupDir:  cfg.BackupDir,
		Retention:  cfg.BackupRetention,
		MaxBackups: cfg.BackupMaxCount,
		Schedule:   cfg.BackupSchedule,
	}
	
	backupManager, err := backup.NewBackupManager(cfg.DBPath, backupConfig)
	if err != nil {
		log.Fatalf("Failed to create backup manager: %v", err)
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	go poller.Start(ctx)
	log.Printf("USGS live feed poller started (%v interval)", cfg.PollerInterval)
	
	// Start backup scheduler if enabled
	if cfg.BackupEnabled {
		go backupManager.StartScheduledBackups(ctx, cfg.BackupSchedule)
		log.Printf("Database backup scheduler started (%v interval)", cfg.BackupSchedule)
	}

	// Setup HTTP router with middleware
	var httpHandler http.Handler = http.NewServeMux()
	mux := httpHandler.(*http.ServeMux)
	
	// Apply CORS middleware (first in chain - handles preflight requests)
	httpHandler = api.CORSMiddleware(httpHandler)
	
	// Apply logging middleware (logs all requests)
	loggingConfig := logging.LoggingConfig{
		Enabled:     cfg.LoggingEnabled,
		Format:      cfg.LoggingFormat,
		LogLevel:    cfg.LoggingLevel,
		IncludeBody: false,
	}
	httpHandler = logging.LoggingMiddleware(loggingConfig)(httpHandler)
	
	// Apply rate limiting middleware if enabled
	if cfg.RateLimitEnabled {
		rateLimitConfig := api.RateLimitConfig{
			Enabled: cfg.RateLimitEnabled,
			RPS:     cfg.RateLimitRPS,
			Burst:   cfg.RateLimitBurst,
			ExemptPaths: []string{
				"/api/health",
				"/api/metrics",
			},
		}
		httpHandler = api.RateLimitMiddleware(rateLimitConfig)(httpHandler)
	}
	
	// API routes
	mux.HandleFunc("/api/events", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handler.ListEvents(w, r)
		case http.MethodPost:
			handler.CreateEvent(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/events/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handler.GetEvent(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/events/stream", handler.EventStream)
	mux.HandleFunc("/api/health", handler.HealthCheck)
	
	// Alert rule endpoints
	mux.HandleFunc("/api/alerts/rules", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handler.ListAlertRules(w, r)
		case http.MethodPost:
			handler.CreateAlertRule(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	
	// Metrics endpoint
	mux.HandleFunc("/api/metrics", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handler.GetMetrics(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Provider health endpoints
	mux.HandleFunc("/api/providers/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handler.GetProviderHealth(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/providers/healthy", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handler.GetHealthyProviders(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/providers/unhealthy", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handler.GetUnhealthyProviders(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/providers/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/stats") {
			handler.GetProviderStats(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Event log endpoints
	mux.HandleFunc("/api/event-log/info", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handler.GetEventLogInfo(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/event-log/rotate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handler.RotateEventLog(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Create HTTP server with configuration
	httpServer := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.HTTPHost, cfg.HTTPPort),
		Handler:      httpHandler, // Use handler with middleware
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Starting SENTINEL server on %s:%s", cfg.HTTPHost, cfg.HTTPPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Setup graceful shutdown manager
	shutdownManager := server.SetupGracefulShutdown(
		httpServer,
		cancel,
		store.Close,
		backupManager,
		server.GracefulShutdownConfig{
			Timeout: 30 * time.Second,
		},
	)
	
	// Wait for termination signal and shutdown gracefully
	shutdownManager.WaitForSignal()
	log.Println("Server stopped")
}

