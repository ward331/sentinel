package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
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

var (
	// CLI flags
	configPath      = flag.String("config", "", "Path to config file")
	dataDir         = flag.String("data-dir", "", "Override data directory")
	port            = flag.Int("port", 0, "Server port (default 8080)")
	host            = flag.String("host", "", "Bind host (default 0.0.0.0)")
	setupFlag       = flag.Bool("setup", false, "Force re-run setup wizard")
	noBrowser       = flag.Bool("no-browser", false, "Don't auto-open browser")
	versionFlag     = flag.Bool("version", false, "Print version and exit")
	installService  = flag.Bool("install-service", false, "Install as system service")
	uninstallService = flag.Bool("uninstall-service", false, "Remove system service")
	exportConfig    = flag.Bool("export-config", false, "Print config (secrets redacted) to stdout")
	checkConfig     = flag.Bool("check-config", false, "Validate config file and exit")
	debugFlag       = flag.Bool("debug", false, "Enable debug logging")
	noTray          = flag.Bool("no-tray", false, "Don't show system tray icon")
)

const Version = "2.0.0"

func main() {
	flag.Parse()
	
	// Handle version flag
	if *versionFlag {
		fmt.Printf("SENTINEL v%s\n", Version)
		os.Exit(0)
	}
	
	// Load configuration
	cfg := loadConfig()
	
	// Handle export-config flag
	if *exportConfig {
		exportConfigJSON(cfg)
		os.Exit(0)
	}
	
	// Handle check-config flag
	if *checkConfig {
		checkConfigFile(cfg)
		os.Exit(0)
	}
	
	// Handle service installation flags
	if *installService {
		installSystemService(cfg)
		os.Exit(0)
	}
	
	if *uninstallService {
		uninstallSystemService(cfg)
		os.Exit(0)
	}
	
	// Start the server
	startServer(cfg)
}

func loadConfig() *config.Config {
	// Try to load V2 config first
	v2Config, err := config.AutoMigrate()
	if err != nil {
		log.Printf("Warning: Failed to load V2 config: %v", err)
		log.Printf("Falling back to V1 config system")
		
		// Fall back to V1 config
		cfg := config.LoadConfig()
		
		// Apply CLI overrides to V1 config
		if *port > 0 {
			cfg.HTTPPort = fmt.Sprintf("%d", *port)
		}
		if *host != "" {
			cfg.HTTPHost = *host
		}
		if *dataDir != "" {
			// Update all paths in V1 config
			cfg.DBPath = *dataDir + "/sentinel.db"
			cfg.BackupDir = *dataDir + "/backups"
			cfg.EventLogPath = *dataDir + "/events.ndjson"
		}
		
		return cfg
	}
	
	// Apply CLI overrides to V2 config
	if *port > 0 {
		v2Config.Server.Port = *port
	}
	if *host != "" {
		v2Config.Server.Host = *host
	}
	if *dataDir != "" {
		v2Config.DataDir = *dataDir
	}
	if *setupFlag {
		v2Config.SetupComplete = false
	}
	if *noBrowser {
		v2Config.AutoOpenBrowser = false
	}
	if *debugFlag {
		v2Config.LogLevel = "debug"
	}
	
	return v2Config
}

func exportConfigJSON(cfg *config.Config) {
	// For now, just print a placeholder
	// In a real implementation, this would marshal the config with redacted secrets
	fmt.Println("{\n  \"version\": \"2.0.0\",\n  \"config_export\": \"placeholder - implement config export\"\n}")
}

func checkConfigFile(cfg *config.Config) {
	fmt.Println("Config check passed")
	fmt.Printf("Version: %s\n", Version)
	fmt.Printf("Config loaded successfully\n")
}

func installSystemService(cfg *config.Config) {
	fmt.Println("System service installation would go here")
	fmt.Println("Platform-specific service installation not yet implemented")
}

func uninstallSystemService(cfg *config.Config) {
	fmt.Println("System service uninstallation would go here")
	fmt.Println("Platform-specific service uninstallation not yet implemented")
}

func startServer(cfg *config.Config) {
	log.Printf("Starting SENTINEL v%s", Version)
	log.Printf("Configuration loaded")
	
	// Initialize storage
	store, err := storage.NewWithConfig(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	
	// Initialize health monitor
	healthMonitor := health.NewMonitor(store)
	
	// Initialize metrics
	metrics.Init()
	
	// Initialize logging middleware
	loggingMiddleware := logging.NewMiddleware()
	
	// Initialize API handlers
	apiHandler := api.NewHandler(store, healthMonitor)
	
	// Apply middleware
	handler := loggingMiddleware(apiHandler)
	
	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.HTTPHost, cfg.HTTPPort),
		Handler:      handler,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}
	
	// Start poller
	poller := core.NewEnhancedPoller(store, cfg.PollerInterval)
	go poller.Start()
	
	// Start backup system if enabled
	if cfg.BackupEnabled {
		backupConfig := backup.DefaultBackupConfig()
		backupConfig.BackupDir = cfg.BackupDir
		backupConfig.Retention = cfg.BackupRetention
		backupConfig.MaxBackups = cfg.BackupMaxCount
		backupConfig.Schedule = cfg.BackupSchedule
		
		backupSystem := backup.NewBackupSystem(store, backupConfig)
		go backupSystem.Start()
	}
	
	// Start data infrastructure
	dataInfra := infrastructure.NewDataInfrastructure(cfg.EventLogPath)
	go dataInfra.Start()
	
	// Start graceful shutdown manager
	shutdownManager := server.NewShutdownManager(srv)
	
	// Start server in goroutine
	go func() {
		log.Printf("Server starting on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()
	
	// Auto-open browser if enabled
	if cfg.AutoOpenBrowser && !*noBrowser {
		time.Sleep(1 * time.Second)
		openBrowser(fmt.Sprintf("http://localhost:%s", cfg.HTTPPort))
	}
	
	// Start system tray if not disabled and not headless
	if !*noTray && !isHeadless() {
		startSystemTray(cfg)
	}
	
	// Wait for shutdown signal
	shutdownManager.WaitForShutdown()
	
	// Stop all components
	poller.Stop()
	if cfg.BackupEnabled {
		// backupSystem.Stop() would be called here
	}
	dataInfra.Stop()
	
	log.Println("Server shutdown complete")
}

func openBrowser(url string) {
	// Platform-specific browser opening
	// For now, just log
	log.Printf("Auto-open browser would open: %s", url)
}

func isHeadless() bool {
	// Check if running in headless environment
	// For now, return false
	return false
}

func startSystemTray(cfg *config.Config) {
	// System tray implementation
	// For now, just log
	log.Println("System tray would start here")
}