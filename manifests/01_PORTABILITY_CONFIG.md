# MANIFEST 01 — PORTABILITY, CONFIG & SINGLE BINARY
# ===================================================
# Covers: Stage 1 (portability scrub + config) and Stage 2 (CLI + tray + wizard)
# Read by agent during Stage 1 AND Stage 2.

════════════════════════════════════════════════════════════════
STAGE 1-A — PORTABILITY SCRUB
════════════════════════════════════════════════════════════════

Audit EVERY .go, .html, .js, .json, .sh, .yaml, .toml file.
Replace ALL hardcoded machine-specific values.

PATHS TO REPLACE:
  /home/ed/            → use runtime config system (cfg.DataDir / cfg.ConfigDir)
  /tmp/sentinel.db     → filepath.Join(cfg.DataDir, "sentinel.db")
  /tmp/sentinel*.ndjson→ filepath.Join(cfg.DataDir, "events.ndjson")
  /tmp/sentinel-backups→ filepath.Join(cfg.DataDir, "backups")
  /tmp/sentinel.log    → filepath.Join(cfg.DataDir, "sentinel.log")
  /etc/*.key           → filepath.Join(cfg.ConfigDir, "keys", keyname+".key")
  /etc/cesium.key      → cfg.CesiumToken (from config, not filesystem)
  /etc/sentinel-smtp.json → embedded in main config struct

NETWORK TO REPLACE:
  172.31.5.10          → remove from all source, docs only note as example
  172.31.5.58          → remove from all source
  All frontend JS with localhost:8080 or 127.0.0.1:8080:
    Replace with: window.location.protocol + '//' + window.location.hostname + ':' + SENTINEL_API_PORT
    Where SENTINEL_API_PORT is injected by backend as Go template variable in HTML

CREDENTIALS TO REMOVE:
  Telegram bot token   → cfg.Telegram.BotToken
  Telegram user ID     → cfg.Telegram.ChatID
  CesiumJS Ion token   → cfg.CesiumToken
  Any SMTP credentials → cfg.Email struct
  Any API keys in source → cfg.Keys map

VERIFICATION (run after scrub):
  grep -r "/home/" --include="*.go" --include="*.html" --include="*.js" .
  grep -r "172\.31\." --include="*.go" --include="*.html" .
  grep -r "eyJhbGci" . (Cesium token)
  grep -r "localhost:8080" --include="*.js" --include="*.html" .
  All must return 0 results (docs/ exempt)

════════════════════════════════════════════════════════════════
STAGE 1-B — UNIFIED CONFIG SYSTEM
════════════════════════════════════════════════════════════════

Create: internal/config/config.go

Config file location priority:
  1. --config flag
  2. $SENTINEL_CONFIG env var
  3. ./sentinel.config.json (next to binary)
  4. Platform default:
       Linux:   ~/.config/sentinel/config.json
       macOS:   ~/Library/Application Support/SENTINEL/config.json
       Windows: %APPDATA%\SENTINEL\config.json

Data directory priority:
  1. --data-dir flag
  2. cfg.DataDir from config
  3. Platform default:
       Linux:   ~/.local/share/sentinel/
       macOS:   ~/Library/Application Support/SENTINEL/data/
       Windows: %APPDATA%\SENTINEL\data\

Full config struct (sentinel.config.json):
{
  "version": "2.0.0",
  "setup_complete": false,
  "data_dir": "",
  "log_level": "info",
  "auto_open_browser": true,
  "check_for_updates": true,
  "cesium_token": "",

  "server": {
    "port": 8080,
    "host": "0.0.0.0",
    "tls_enabled": false,
    "tls_cert": "",
    "tls_key": "",
    "auth_enabled": false,
    "auth_token": "",
    "dashboard_password": ""
  },

  "telegram": {
    "enabled": false,
    "bot_token": "",
    "chat_id": "",
    "min_severity": "warning",
    "digest_mode": false,
    "digest_interval_minutes": 60
  },

  "slack": {
    "enabled": false,
    "webhook_url": "",
    "channel": "#sentinel-alerts",
    "min_severity": "warning"
  },

  "discord": {
    "enabled": false,
    "webhook_url": "",
    "min_severity": "warning"
  },

  "ntfy": {
    "enabled": false,
    "server": "https://ntfy.sh",
    "topic": "",
    "min_severity": "warning"
  },

  "pushover": {
    "enabled": false,
    "app_token": "",
    "user_key": ""
  },

  "email": {
    "enabled": false,
    "method": "",
    "smtp_host": "",
    "smtp_port": 587,
    "smtp_tls": "starttls",
    "username": "",
    "password_encrypted": "",
    "from_address": "",
    "to_addresses": [],
    "gmail_client_id": "",
    "gmail_client_secret": "",
    "gmail_refresh_token": "",
    "sendgrid_key_encrypted": "",
    "mailgun_key_encrypted": "",
    "mailgun_domain": "",
    "min_severity": "alert"
  },

  "keys": {
    "adsbexchange": "",
    "aisstream": "",
    "acled": "",
    "openweather": "",
    "nasa": "",
    "spacetrack": "",
    "marinetraffic": "",
    "vesselfinder": "",
    "n2yo": "",
    "shodan": "",
    "cloudflare": "",
    "ukrainealerts": "",
    "alpha_vantage": "",
    "finnhub": "",
    "fred": "",
    "polygon": ""
  },

  "providers": {
    "usgs":          {"enabled": true,  "interval_seconds": 60},
    "gdacs":         {"enabled": true,  "interval_seconds": 60},
    "opensky":       {"enabled": true,  "interval_seconds": 60},
    "noaa_cap":      {"enabled": true,  "interval_seconds": 300},
    "openmeteo":     {"enabled": true,  "interval_seconds": 600},
    "gdelt":         {"enabled": true,  "interval_seconds": 900},
    "celestrak":     {"enabled": true,  "interval_seconds": 21600},
    "swpc":          {"enabled": true,  "interval_seconds": 60},
    "who":           {"enabled": true,  "interval_seconds": 3600},
    "promed":        {"enabled": true,  "interval_seconds": 1800},
    "airplanes_live":{"enabled": true,  "interval_seconds": 30},
    "nasa_firms":    {"enabled": true,  "interval_seconds": 1800},
    "piracy_imb":    {"enabled": true,  "interval_seconds": 3600},
    "israel_alerts": {"enabled": true,  "interval_seconds": 5},
    "reliefweb":     {"enabled": true,  "interval_seconds": 600},
    "vix":           {"enabled": true,  "interval_seconds": 60},
    "oil_price":     {"enabled": true,  "interval_seconds": 300},
    "crypto":        {"enabled": true,  "interval_seconds": 60},
    "sec_edgar":     {"enabled": true,  "interval_seconds": 900},
    "ofac_sdn":      {"enabled": true,  "interval_seconds": 3600},
    "treasury_yields":{"enabled": true, "interval_seconds": 3600},
    "news_rss":      {"enabled": true,  "interval_seconds": 1800}
  },

  "notifications": {
    "rules": [],
    "geofences": []
  },

  "morning_briefing": {
    "enabled": false,
    "time_utc": "08:00",
    "delivery": ["telegram"],
    "include_events": true,
    "include_conflicts": true,
    "include_space_weather": true,
    "include_financial": true,
    "include_iss_passes": true,
    "include_news": true
  },

  "weekly_digest": {
    "enabled": false,
    "day": "sunday",
    "time_utc": "08:00",
    "delivery": ["email"]
  },

  "ui": {
    "default_view": "globe",
    "default_preset": "Global Watch",
    "data_retention_days": 30,
    "sound_enabled": true,
    "sound_volume": 75,
    "ticker_enabled": true,
    "ticker_speed": "medium",
    "ticker_min_severity": "warning"
  },

  "location": {
    "lat": 0.0,
    "lon": 0.0,
    "timezone": "UTC",
    "set": false
  }
}

Config encryption (internal/crypto/crypto.go):
- AES-256-GCM for passwords and API keys stored in config
- Encryption key = HKDF(machine-id + username, "sentinel-v2")
- Machine ID: /etc/machine-id (Linux), IOPlatformUUID (macOS),
              MachineGuid from registry (Windows)
- Fallback if machine-id unavailable: SHA256(hostname + username)
- Encrypted fields stored as base64 with "enc:" prefix
- enc:BASE64DATA — detected on read, decrypted transparently
- Plain text accepted on write, always stored encrypted
- GET /api/config returns keys masked: "sk-••••1234" (last 4 chars)

Config auto-migrate:
- On every startup: read config, add any missing keys with defaults
- Never remove unknown keys (forward-compat)
- Log "config migrated: added N new fields" if migration occurred

════════════════════════════════════════════════════════════════
STAGE 1-C — SINGLE BINARY ARCHITECTURE
════════════════════════════════════════════════════════════════

Merge serve_frontend.go into main sentinel binary.

Directory structure:
  web/
    index.html          (renamed from sentinel_dashboard.html)
    setup.html          (new — setup wizard)
    settings.html       (new — settings page)
    feed.html           (new — standalone feed view)
    media.html          (new — media wall)
    stats.html          (new — statistics dashboard)
    manifest.json       (PWA)
    service-worker.js   (PWA)
    icons/              (app icons: 16,32,64,128,192,512px PNG)
    static/             (CSS, JS bundles if any)

In main.go:
  //go:embed web
  var webFS embed.FS

HTTP routing (single port, default 8080):
  /           → web/index.html
  /feed       → web/feed.html
  /media      → web/media.html
  /settings   → web/settings.html
  /stats      → web/stats.html
  /setup      → web/setup.html
  /static/*   → web/static/
  /api/*      → existing API handlers
  /api/events/stream → SSE handler

All HTML files receive template injection:
  {{.APIPort}}     — port number
  {{.Version}}     — build version
  {{.Platform}}    — linux/darwin/windows

Delete serve_frontend.go after merge.

════════════════════════════════════════════════════════════════
STAGE 2-A — CLI FLAGS
════════════════════════════════════════════════════════════════

All flags via standard library flag package:

  --config string           Path to config file
  --data-dir string         Override data directory
  --port int                Server port (default 8080)
  --host string             Bind host (default 0.0.0.0)
  --setup                   Force re-run setup wizard
  --no-browser              Don't auto-open browser
  --version                 Print version and exit
  --install-service         Install as system service
  --uninstall-service       Remove system service
  --export-config           Print config (secrets redacted) to stdout
  --check-config            Validate config file and exit
  --debug                   Enable debug logging

════════════════════════════════════════════════════════════════
STAGE 2-B — SYSTEM SERVICE INSTALLER
════════════════════════════════════════════════════════════════

internal/service/ package with platform-specific files:

service_linux.go:
  Writes ~/.config/systemd/user/sentinel.service
  Runs: systemctl --user daemon-reload
        systemctl --user enable sentinel
        systemctl --user start sentinel
  Reports status after start.

service_darwin.go:
  Writes ~/Library/LaunchAgents/io.sentinel.monitor.plist
  Runs: launchctl load ~/Library/LaunchAgents/io.sentinel.monitor.plist
        launchctl start io.sentinel.monitor

service_windows.go:
  Uses golang.org/x/sys/windows/svc
  Service name: "SentinelMonitor"
  Display name: "SENTINEL Global Monitor"
  Description: "SENTINEL real-time global situational awareness"
  Start type: Automatic delayed
  --install-service: installs + starts
  --uninstall-service: stops + removes

════════════════════════════════════════════════════════════════
STAGE 2-C — SYSTEM TRAY (WINDOWS + MACOS)
════════════════════════════════════════════════════════════════

internal/tray/tray.go using github.com/getlantern/systray

Icon embedded from web/icons/tray-16.png (create placeholder)
Icon colors by alert tier:
  Normal:   blue
  Watch:    amber
  Alert:    red
  Critical: alternating red/dark every 500ms (goroutine)

Menu:
  "SENTINEL v{VERSION}"  (disabled, title)
  "● {N} active events"  (disabled, live counter — update every 30s)
  ─────────────────────
  "Open Dashboard"        → open http://localhost:{port}
  "Mute Alerts" toggle
  ─────────────────────
  "Settings"              → open http://localhost:{port}/settings
  "Restart"               → graceful restart
  "Quit"                  → graceful shutdown

Tray only runs if not --no-tray flag and not running as service.
Gracefully skip if display not available (headless Linux servers).

════════════════════════════════════════════════════════════════
STAGE 2-D — FIRST-RUN SETUP WIZARD
════════════════════════════════════════════════════════════════

Trigger: cfg.SetupComplete == false OR --setup flag
Action:  Start server, open http://localhost:{port}/setup in browser
         All /api/setup/* endpoints bypass auth during setup

web/setup.html — multi-step wizard, dark theme, mobile-friendly

STEP 1 — WELCOME
  SENTINEL v2 logo
  Platform detected (shown)
  "Setup takes about 3 minutes"
  [Begin Setup] button

STEP 2 — CESIUM TOKEN (required)
  Explanation + free signup link: https://ion.cesium.com/signup
  Platform-specific instructions:
    "1. Create free account at the link above
     2. Access Tokens → copy your Default Token
     3. Paste below"
  Token input + [Test Token] button (validates via Cesium API)
  Cannot proceed without valid token.

STEP 3 — YOUR LOCATION (optional, improves ISS passes + local alerts)
  Lat/lon inputs OR [Use My Location] (browser geolocation API)
  Timezone selector (auto-detected from browser)
  "Used for: ISS pass predictions, local weather, ham radio stations nearby"
  [Skip] button

STEP 4 — NOTIFICATION METHOD
  Multi-select cards: [Telegram] [Slack] [Email] [Discord] [ntfy.sh] [Skip]
  Shows substep for each selected method:

  TELEGRAM:
    Step-by-step BotFather instructions with exact commands
    Bot token input + [Fetch My Chat ID] auto-button + [Test Alert]

  SLACK:
    Webhook URL instructions + input + [Test]

  EMAIL:
    Method radio: Gmail OAuth2 / Gmail App Password / Generic SMTP /
                  SendGrid / Mailgun
    Appropriate fields per method + [Test]

  DISCORD:
    Webhook URL instructions + input + [Test]

  NTFY.SH:
    Topic input + ntfy app links + [Send Test]

STEP 5 — FINANCIAL ALERTS (new in V2)
  "Would you like financial market alerts?"
  ○ Yes — add market alerts to my feed
  ○ No — skip financial data
  If Yes:
    "Which markets? (select all that apply)"
    ☑ US Equities (VIX, S&P circuit breakers)
    ☑ Commodities (Oil, Gold, Wheat — geopolitical correlation)
    ☑ Cryptocurrency (Bitcoin, Ethereum flash crashes)
    ☑ Currencies (USD/major forex flash crashes)
    ☑ Bonds (Treasury yield movements)
    ☑ Sanctions (OFAC SDN list updates)
    ☑ SEC Filings (major enforcement actions)
    Alert threshold: "Alert me when price moves more than [5]% in [1 hour ▾]"
    Optional keys section (free tiers):
      Alpha Vantage (25 req/day free): [signup link] [key input]
      Finnhub (60 req/min free): [signup link] [key input]
      FRED (Federal Reserve data, free): [signup link] [key input]
    Note: "SENTINEL works without these keys using free public endpoints.
           Keys unlock higher update frequency and more data."

STEP 6 — OPTIONAL DATA KEYS
  Table: Source | Adds | Free Signup | Key | Test
  (see manifests/02_DATA_PROVIDERS.md for full table)

STEP 7 — SATELLITE TRACKING
  "How many satellites to track?"
  ○ Essential (~50: ISS, GPS, weather)   — Recommended for most
  ○ Standard (~8,000: all active)
  ○ Everything (~35,000: includes debris) — High memory usage

STEP 8 — REVIEW & LAUNCH
  Summary of configured options with ✅/○ per item
  [Launch SENTINEL] → save config, start providers, redirect to /
  [Go Back] to revise

POST-SETUP:
  cfg.SetupComplete = true
  cfg.save()
  Redirect to http://localhost:{port}/
  Show "Welcome to SENTINEL" banner for first 60 seconds

Setup wizard re-accessible: Settings → Advanced → [Re-run Setup Wizard]

════════════════════════════════════════════════════════════════
STAGE 2-E — SETTINGS PAGE
════════════════════════════════════════════════════════════════

web/settings.html — full settings management
Accessible via ⚙️ icon in header of all views.

Tabs:
  [API Keys] [Notifications] [Financial] [Geofences] [Display]
  [Thresholds] [Providers] [Security] [Data] [OSINT Resources] [Advanced]

Each tab loads/saves via GET/PATCH /api/config.
Implement all tabs. Full detail in respective manifests.
Key principle: every config option must be reachable from settings.

NEW IN V2 — [FINANCIAL] tab:
  Market subscriptions toggles (equities/commodities/crypto/forex/bonds)
  Price alert thresholds per asset class
  API key management for financial sources
  Geopolitical correlation: enable/disable overlay
  Watchlist: add specific tickers (e.g. "OIL", "XAUUSD", "BTC")

NEW IN V2 — [OSINT RESOURCES] tab:
  Platform profile suggestions (manage list, mark as followed)
  Ham radio presets (save frequency/region combos)
  WebSDR favorites (saved receiver URLs)
  Custom media sources for media wall
