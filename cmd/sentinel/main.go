package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/openclaw/sentinel-backend/internal/api"
	"github.com/openclaw/sentinel-backend/internal/config"
	"github.com/openclaw/sentinel-backend/internal/engine"
	"github.com/openclaw/sentinel-backend/internal/health"
	"github.com/openclaw/sentinel-backend/internal/metrics"
	"github.com/openclaw/sentinel-backend/internal/notify"
	"github.com/openclaw/sentinel-backend/internal/poller"
	"github.com/openclaw/sentinel-backend/internal/provider"
	"github.com/openclaw/sentinel-backend/internal/setup"
	"github.com/openclaw/sentinel-backend/internal/storage"
)

//go:embed web/*
var webFS embed.FS

var (
	configPath = flag.String("config", "", "Path to configuration file")
	dataDir    = flag.String("data-dir", "", "Data directory for database and files")
	port       = flag.Int("port", 8080, "Port to listen on")
	host       = flag.String("host", "localhost", "Host to bind to")
	version    = flag.Bool("version", false, "Show version information")
	wizard     = flag.Bool("wizard", false, "Run first-run setup wizard")
	noFrontend = flag.Bool("no-frontend", false, "API only — do not serve embedded frontend")
)

// Version is set during build via -ldflags.
var Version = "v3.0.0"

// tier1ProviderNames lists provider names that require API keys.
var tier1ProviderNames = map[string]bool{
	"adsbexchange":   true,
	"aisstream":      true,
	"acled":          true,
	"openweathermap": true,
	"nasa_firms_rt":  true,
	"spacetrack":     true,
	"alpha_vantage":  true,
	"finnhub":        true,
	"fred":           true,
	"shodan":         true,
	"abusech":        true,
}

func main() {
	flag.Parse()

	if *version {
		fmt.Printf("SENTINEL %s\n", Version)
		return
	}

	// ── 1. Load configuration ───────────────────────────────
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// ── 2. Wizard (first-run or --wizard) ───────────────────
	if *wizard || !cfg.SetupComplete {
		if *wizard {
			if err := setup.RunIfNeeded(cfg); err != nil {
				log.Fatalf("Setup wizard failed: %v", err)
			}
		}
	}

	// ── 3. Start the server ─────────────────────────────────
	if err := startServer(cfg); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// loadConfig loads configuration from file or CLI flags.
func loadConfig() (*config.Config, error) {
	cfg := config.DefaultConfig()

	if *configPath != "" {
		fileConfig, err := config.LoadConfig(*configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load config from %s: %w", *configPath, err)
		}
		cfg = fileConfig
	}

	if *dataDir != "" {
		cfg.DataDir = *dataDir
	}
	if *port != 8080 {
		cfg.Server.Port = *port
	}
	if *host != "localhost" {
		cfg.Server.Host = *host
	}

	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory %s: %w", cfg.DataDir, err)
	}

	return cfg, nil
}

// startServer initializes all V3 components and starts the HTTP server.
func startServer(cfg *config.Config) error {
	// ── 1. Storage ──────────────────────────────────────────
	dbPath := filepath.Join(cfg.DataDir, "sentinel.db")
	store, err := storage.NewWithConfig(dbPath, true, 10)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer store.Close()

	if err := storage.RunV3Migration(store.DB()); err != nil {
		return fmt.Errorf("V3 migration failed: %w", err)
	}

	// ── 2. Providers & Poller ───────────────────────────────
	pollerInstance := initializePoller(store, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pollerInstance.Start()
	defer pollerInstance.Stop()

	tier0Count, tier1Count := countProvidersByTier(pollerInstance)

	// ── 3. Intelligence Engines ─────────────────────────────
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

	// ── 4. Notification Dispatcher ──────────────────────────
	notifyDispatcher := initializeNotifications(cfg)

	// ── 4b. Proximity Alert Engine ──────────────────────────
	var proxAlert *engine.ProximityAlert
	if cfg.Location.Set {
		proxAlert = engine.NewProximityAlert(cfg.Location, func(title, body, severity string) {
			notifyDispatcher.Dispatch(context.Background(), notify.Alert{
				Title:      title,
				Body:       body,
				Severity:   severity,
				Category:   "proximity",
				OccurredAt: time.Now(),
			})
		})
		log.Printf("Proximity alerts enabled: %.4f,%.4f radius %.0fkm",
			cfg.Location.Lat, cfg.Location.Lon, proxAlert.RadiusKm)
	}

	// ── 5. API Handler ──────────────────────────────────────
	metricsInst := metrics.NewMetrics()
	healthRegistry := health.NewHealthRegistry()

	apiHandler := api.NewHandler(store, metricsInst, healthRegistry)
	apiHandler.SetPoller(pollerInstance)
	apiHandler.SetConfig(cfg)
	if proxAlert != nil {
		apiHandler.SetProximityEngine(proxAlert)
	}
	router := apiHandler.Router()

	// OSINT resources sub-router
	osintStorage := storage.NewOSINTStorage(store.DB())
	osintHandler := api.NewOSINTResourcesHandler(osintStorage)
	osintRouter := router.PathPrefix("/api/osint").Subrouter()
	osintHandler.RegisterRoutes(osintRouter)

	// ── 6. Embedded Frontend ────────────────────────────────
	frontendMode := "disabled"
	if !*noFrontend {
		webSub, fsErr := fs.Sub(webFS, "web")
		if fsErr != nil {
			log.Printf("WARNING: failed to mount embedded frontend: %v", fsErr)
		} else {
			router.PathPrefix("/").Handler(http.FileServer(http.FS(webSub)))
			frontendMode = "embedded"
		}
	}

	// ── 7. Middleware Stack ─────────────────────────────────
	var handler http.Handler = router

	handler = api.CORSMiddleware(handler)

	rlCfg := api.DefaultRateLimitConfig()
	rlCfg.Enabled = true
	rlCfg.RPS = 100
	rlCfg.Burst = 200
	handler = api.RateLimitMiddleware(rlCfg)(handler)

	authCfg := api.DefaultAuthConfig()
	authCfg.Enabled = cfg.Server.AuthEnabled
	if cfg.Server.AuthToken != "" {
		authCfg.APIKeys = []string{cfg.Server.AuthToken}
	}
	handler = api.AuthMiddleware(authCfg)(handler)

	// ── 8. Startup Summary ──────────────────────────────────
	enabledTotal := tier0Count + tier1Count
	engineSummary := strings.Join([]string{
		statusMark("correlation", true),
		statusMark("truth", true),
		statusMark("anomaly", true),
		statusMark("signal-board", cfg.SignalBoard.Enabled),
		statusMark("dead-reckoning", cfg.EntityTracking.Enabled),
		statusMark("proximity", proximityEnabled),
	}, " | ")

	enabledChannels := notifyDispatcher.EnabledChannelNames()
	notifySummary := "none"
	if len(enabledChannels) > 0 {
		notifySummary = strings.Join(enabledChannels, " | ")
	}

	log.Printf("\n"+
		"    SENTINEL %s starting...\n"+
		"    Port: %d\n"+
		"    Data: %s\n"+
		"    Providers: %d enabled (%d tier-0, %d tier-1)\n"+
		"    Engine: %s\n"+
		"    Notifications: %s\n"+
		"    Frontend: %s\n"+
		"    Ready.",
		Version, cfg.Server.Port, cfg.DataDir,
		enabledTotal, tier0Count, tier1Count,
		engineSummary, notifySummary, frontendMode,
	)

	// ── 9. HTTP Server ──────────────────────────────────────
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// ── 10. Wait for SIGTERM / SIGINT ───────────────────────
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		log.Printf("Received signal: %v", sig)
	case err := <-serverErr:
		log.Printf("Server error: %v", err)
	}

	// ── 11. Graceful Shutdown ───────────────────────────────
	log.Println("Shutting down...")
	cancel() // stops all engine goroutines via ctx

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	log.Println("Server stopped gracefully")
	return nil
}

// ─── Notification Dispatcher ────────────────────────────────────────────────

// initializeNotifications creates notification channels from config.
func initializeNotifications(cfg *config.Config) *notify.Dispatcher {
	var channels []notify.Channel

	if cfg.Telegram.Enabled {
		channels = append(channels,
			notify.NewTelegramChannel(cfg.Telegram.BotToken, cfg.Telegram.ChatID, true))
	}
	if cfg.Slack.Enabled {
		channels = append(channels,
			notify.NewSlackChannel(cfg.Slack.WebhookURL, true))
	}
	if cfg.Discord.Enabled {
		channels = append(channels,
			notify.NewDiscordChannel(cfg.Discord.WebhookURL, true))
	}
	if cfg.Ntfy.Enabled {
		channels = append(channels,
			notify.NewNtfyChannel(cfg.Ntfy.Server, cfg.Ntfy.Topic, true))
	}
	if cfg.Pushover.Enabled {
		channels = append(channels,
			notify.NewPushoverChannel(cfg.Pushover.UserKey, cfg.Pushover.AppToken, true))
	}
	if cfg.Email.Enabled {
		channels = append(channels,
			notify.NewEmailChannel(
				cfg.Email.SMTPHost, cfg.Email.SMTPPort,
				cfg.Email.FromAddress, cfg.Email.ToAddresses,
				cfg.Email.Username, "", // password decrypted at runtime if needed
				true,
			))
	}

	return notify.NewDispatcher(channels...)
}

// ─── Provider Registration ──────────────────────────────────────────────────

// initializePoller creates the poller and registers all providers.
func initializePoller(store *storage.Storage, cfg *config.Config) *poller.Poller {
	p := poller.NewPoller(store)

	tier0Cfg := &provider.Config{
		Enabled:      true,
		PollInterval: 5 * time.Minute,
	}

	// ── Tier 0: free, no API key required ──
	registerProvider(p, "usgs", provider.NewUSGSProvider(tier0Cfg))
	registerProvider(p, "gdacs", provider.NewGDACSProvider(tier0Cfg))
	registerProvider(p, "noaa_cap", provider.NewNOAACAPProvider(tier0Cfg))
	registerProvider(p, "noaa_nws", provider.NewNOAANWSProvider(tier0Cfg))
	registerProvider(p, "tsunami", provider.NewTsunamiProvider(tier0Cfg))
	registerProvider(p, "volcano", provider.NewVolcanoProvider(tier0Cfg))
	registerProvider(p, "reliefweb", provider.NewReliefWebProvider(tier0Cfg))
	registerProvider(p, "airplanes_live", provider.NewAirplanesLiveProvider(tier0Cfg))
	registerProvider(p, "adsb_one", provider.NewADSBOneProvider(tier0Cfg))
	registerProvider(p, "openmeteo", provider.NewOpenMeteoProvider())
	registerProvider(p, "iranconflict", provider.NewIranConflictProvider())
	registerProvider(p, "liveuamap", provider.NewLiveUAMapProvider(tier0Cfg))
	registerProvider(p, "gdelt", provider.NewGDELTProvider())
	registerProvider(p, "globalforestwatch", provider.NewGlobalForestWatchProvider())
	registerProvider(p, "celestrak", provider.NewCelesTrakProvider(tier0Cfg))
	registerProvider(p, "swpc", provider.NewSWPCProvider(tier0Cfg))
	registerProvider(p, "who", provider.NewWHOProvider(tier0Cfg))
	registerProvider(p, "promed", provider.NewProMEDProvider(tier0Cfg))
	registerProvider(p, "nasa_firms", provider.NewNASAFIRMSProvider(tier0Cfg))
	registerProvider(p, "piracy_imb", provider.NewPiracyIMBProvider(tier0Cfg))
	registerProvider(p, "financial_markets", provider.NewFinancialMarketsProvider(tier0Cfg))
	registerProvider(p, "opensanctions", provider.NewOpenSanctionsProvider(""))
	registerProvider(p, "pikud_haoref", provider.NewPikudHaOrefProvider())
	registerProvider(p, "ukraine_alerts", provider.NewUkraineAlertsProvider())
	registerProvider(p, "ukmto", provider.NewUKMTOProvider())
	registerProvider(p, "sec_edgar", provider.NewSECEdgarProvider())
	registerProvider(p, "cisa_kev", provider.NewCISAKEVProvider())
	registerProvider(p, "otx_alienvault", provider.NewOTXAlienVaultProvider())
	registerProvider(p, "bellingcat", provider.NewBellingcatProvider())
	registerProvider(p, "isw", provider.NewISWProvider())

	// ── Tier 1: free with API key — disabled if key missing ──
	tier1Cfg := &provider.Config{
		Enabled:      true,
		PollInterval: 5 * time.Minute,
		Options:      make(map[string]string),
	}
	registerProvider(p, "adsbexchange", provider.NewADSBExchangeProvider(tier1Cfg))
	registerProvider(p, "aisstream", provider.NewAISStreamProvider(tier1Cfg))
	registerProvider(p, "acled", provider.NewACLEDProvider(tier1Cfg))
	registerProvider(p, "openweathermap", provider.NewOpenWeatherMapProvider(tier1Cfg))
	registerProvider(p, "nasa_firms_rt", provider.NewNASAFIRMSRTProvider(tier1Cfg))
	registerProvider(p, "spacetrack", provider.NewSpaceTrackProvider(tier1Cfg))
	registerProvider(p, "alpha_vantage", provider.NewAlphaVantageProvider(tier1Cfg))
	registerProvider(p, "finnhub", provider.NewFinnhubProvider(tier1Cfg))
	registerProvider(p, "fred", provider.NewFREDProvider(tier1Cfg))
	registerProvider(p, "shodan", provider.NewShodanProvider(tier1Cfg))
	registerProvider(p, "abusech", provider.NewAbuseCHProvider(tier1Cfg))

	return p
}

// registerProvider registers a single provider with the poller.
func registerProvider(p *poller.Poller, name string, prov provider.Provider) {
	p.RegisterProvider(name, prov)
}

// countProvidersByTier counts enabled providers split by tier.
func countProvidersByTier(p *poller.Poller) (tier0, tier1 int) {
	for _, name := range p.GetProviderNames() {
		prov, ok := p.GetProvider(name)
		if !ok || !prov.Enabled() {
			continue
		}
		if tier1ProviderNames[name] {
			tier1++
		} else {
			tier0++
		}
	}
	return
}

// statusMark returns "name ok" or "name off" for the startup log.
func statusMark(name string, on bool) string {
	if on {
		return name + " ok"
	}
	return name + " off"
}
