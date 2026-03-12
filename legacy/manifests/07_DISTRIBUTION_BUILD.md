# MANIFEST 07 — DISTRIBUTION, BUILD SYSTEM & DOCUMENTATION
# ===========================================================
# Covers: Stage 9 (cross-platform Makefile, installers, Docker)
#         Stage 10 (all documentation)

════════════════════════════════════════════════════════════════
MAKEFILE — CROSS-PLATFORM BUILD SYSTEM
════════════════════════════════════════════════════════════════

Create Makefile in repo root:

Variables:
  VERSION    := $(shell git describe --tags --always --dirty 2>/dev/null || echo "2.0.0")
  BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
  LDFLAGS    := -X main.Version=$(VERSION) \
                -X main.BuildTime=$(BUILD_TIME) \
                -s -w
  CGO        := CGO_ENABLED=0

Targets:
  make build           — build for current platform
  make build-linux     — linux/amd64
  make build-linux-arm — linux/arm64
  make build-mac       — darwin/amd64
  make build-mac-arm   — darwin/arm64 (Apple Silicon)
  make build-windows   — windows/amd64 → sentinel.exe
  make build-all       — all 5 targets
  make test            — go test ./...
  make smoke           — existing smoke tests
  make clean           — rm -rf dist/
  make release         — build-all + package + docs
  make installer-windows — Inno Setup compile (see below)
  make installer-mac     — create-dmg (see below)
  make package-linux     — tar.gz with install scripts
  make docker            — docker build
  make docs              — validate all docs exist and non-empty

All builds:
  CGO_ENABLED=0 -trimpath -s -w
  Output: dist/sentinel-{os}-{arch}[.exe]

Output structure:
  dist/
    sentinel-linux-amd64
    sentinel-linux-arm64
    sentinel-darwin-amd64
    sentinel-darwin-arm64
    sentinel-windows-amd64.exe
    SENTINEL-Setup-v{VERSION}-windows-x64.exe
    SENTINEL-v{VERSION}-macos-x64.dmg
    SENTINEL-v{VERSION}-macos-arm64.dmg
    sentinel-v{VERSION}-linux-amd64.tar.gz
    sentinel-v{VERSION}-linux-arm64.tar.gz

════════════════════════════════════════════════════════════════
WINDOWS INSTALLER — Inno Setup
════════════════════════════════════════════════════════════════

File: installers/windows/sentinel.iss

[Setup]
  AppName=SENTINEL
  AppVersion={VERSION}
  AppPublisher=SENTINEL Project
  DefaultDirName={autopf}\SENTINEL
  DefaultGroupName=SENTINEL
  UninstallDisplayIcon={app}\sentinel.exe
  Compression=lzma2
  SolidCompression=yes
  OutputDir=..\..\dist
  OutputBaseFilename=SENTINEL-Setup-v{VERSION}-windows-x64
  WizardStyle=modern
  LicenseFile=..\..\LICENSE

[Files]
  sentinel.exe, LICENSE, README.md, INSTALL-WINDOWS.md

[Icons]
  Start Menu: SENTINEL, Open Dashboard, Uninstall
  Desktop shortcut: SENTINEL

[Run]
  sentinel.exe --install-service (runhidden)
  sentinel.exe --setup (postinstall, opens setup wizard)

[UninstallRun]
  sentinel.exe --uninstall-service (runhidden)

Windows Firewall rule (add/remove via installer):
  netsh advfirewall firewall add rule name="SENTINEL" dir=in action=allow
        program="{app}\sentinel.exe" protocol=TCP localport=8080

Windows Service (internal/service/service_windows.go):
  Service name: "SentinelMonitor"
  Display name: "SENTINEL Global Monitor"
  Description: "SENTINEL real-time global situational awareness"
  Start type: Automatic delayed
  Recovery: restart on failure

System Tray:
  Runs when binary starts interactively (not as service)
  Auto-open browser on first start unless --no-browser
  Tray icon changes color by current alert tier

════════════════════════════════════════════════════════════════
MACOS INSTALLER
════════════════════════════════════════════════════════════════

Files in installers/macos/:

LaunchAgent: io.sentinel.monitor.plist
  Label: io.sentinel.monitor
  ProgramArguments: [/Applications/SENTINEL/sentinel, --no-browser]
  RunAtLoad: true
  KeepAlive: true
  Logs: ~/Library/Logs/SENTINEL/sentinel.log

install.sh:
  mkdir -p ~/Applications/SENTINEL
  cp sentinel ~/Applications/SENTINEL/
  chmod +x ~/Applications/SENTINEL/sentinel
  cp io.sentinel.monitor.plist ~/Library/LaunchAgents/
  launchctl load ~/Library/LaunchAgents/io.sentinel.monitor.plist
  launchctl start io.sentinel.monitor
  sleep 2
  open http://localhost:8080/setup

uninstall.sh:
  launchctl stop io.sentinel.monitor
  launchctl unload ~/Library/LaunchAgents/io.sentinel.monitor.plist
  rm -f ~/Library/LaunchAgents/io.sentinel.monitor.plist
  rm -rf ~/Applications/SENTINEL
  echo "Data preserved at ~/Library/Application Support/SENTINEL/"

create-dmg.sh:
  Uses: npm install -g create-dmg OR brew install create-dmg
  Background: dark branded background image
  Window: 600x400, icon positions SENTINEL app + Applications symlink
  Output: dist/SENTINEL-v{VERSION}-macos-{arch}.dmg

--install-service on macOS: copies plist + runs launchctl
--uninstall-service: runs uninstall steps

Gatekeeper workaround (shown in installer + docs):
  Option 1: Right-click → Open → Open Anyway
  Option 2: System Settings → Privacy & Security → Open Anyway
  Option 3 (terminal): xattr -d com.apple.quarantine /Applications/SENTINEL/sentinel

Self-signing:
  codesign --force --sign - dist/sentinel-darwin-{arch}
  Ad-hoc signature — trusted on current machine, prevents most Gatekeeper dialogs

════════════════════════════════════════════════════════════════
LINUX PACKAGING
════════════════════════════════════════════════════════════════

--install-service writes:
  ~/.config/systemd/user/sentinel.service
  systemctl --user daemon-reload && enable && start

installers/linux/install.sh:
  Copies binary to ~/.local/bin/
  Runs --install-service
  Creates desktop entry: ~/.local/share/applications/sentinel.desktop
  Opens http://localhost:8080/setup

installers/linux/sentinel.desktop:
  Name=SENTINEL
  Comment=Global Situational Awareness Platform
  Exec=sentinel
  Icon=sentinel
  Terminal=false
  Type=Application
  Categories=Network;Security;

make package-linux:
  Creates dist/sentinel-v{VERSION}-linux-{arch}.tar.gz containing:
    sentinel (binary)
    install.sh
    uninstall.sh
    LICENSE
    README.md
    INSTALL-LINUX.md

════════════════════════════════════════════════════════════════
DOCKER
════════════════════════════════════════════════════════════════

Dockerfile:
  FROM scratch
  COPY dist/sentinel-linux-amd64 /sentinel
  VOLUME ["/data"]
  EXPOSE 8080
  ENV SENTINEL_DATA_DIR=/data
  ENTRYPOINT ["/sentinel", "--no-browser"]

docker-compose.yml:
  version: '3.8'
  services:
    sentinel:
      image: sentinel:latest
      build: .
      ports: ["8080:8080"]
      volumes:
        - ./data:/data
        - ./sentinel.config.json:/sentinel.config.json:ro
      environment:
        - SENTINEL_CONFIG=/sentinel.config.json
      restart: unless-stopped

.dockerignore: dist/ docs/ installers/ *.md (keep only source + web/)

════════════════════════════════════════════════════════════════
AUTO-UPDATE CHECKER
════════════════════════════════════════════════════════════════

On startup (unless --skip-update-check or cfg.CheckForUpdates == false):
  GET https://api.github.com/repos/sentinel-project/sentinel/releases/latest
  Compare tag_name vs main.Version
  If newer: log "Update available: {version}"
  Show banner in dashboard: "Update available v{new} → Download"
  Banner links to GitHub releases page
  Never auto-update. User clicks to download manually.
  Check at most once per startup.

════════════════════════════════════════════════════════════════
DOCUMENTATION — ALL FILES
════════════════════════════════════════════════════════════════

Generate ALL docs as agent completes each section.
Final check: all files exist and are non-empty.
All docs use clear markdown, no jargon, suitable for non-technical users.

docs/README.md (also copy to root README.md):
  Project description
  Feature list (all V2 features)
  Screenshots section (placeholder with [screenshot] markers)
  Quick start (3 commands)
  View modes overview
  Platform download links
  License badge

docs/INSTALL-WINDOWS.md:
  Prerequisites: Windows 10/11 x64
  Download and run installer
  Installer steps walkthrough
  First-run wizard walkthrough
  Accessing dashboard
  Start/stop service (services.msc + CLI)
  Firewall notes
  Uninstall
  Troubleshooting:
    Service won't start → check Event Viewer
    Port 8080 in use → netstat -ano | findstr 8080
    Windows Defender flags binary → add exclusion steps
    Setup wizard doesn't open → manual http://localhost:8080/setup

docs/INSTALL-MACOS.md:
  Prerequisites: macOS 11+ (Intel or Apple Silicon)
  Download DMG, drag to Applications
  Gatekeeper workaround (step by step with screenshots)
  First-run wizard
  LaunchAgent: start/stop/restart commands
  Uninstall
  Troubleshooting:
    Gatekeeper blocks app → xattr command
    Port in use → lsof -i :8080
    M1/M2 vs Intel note (separate binaries)
    Permission denied → chmod +x steps

docs/INSTALL-LINUX.md:
  Binary install (recommended — 3 steps)
  Docker install (2 commands)
  Build from source
  Systemd service setup
  Reverse proxy with nginx (example config)
  Uninstall

docs/INSTALL-DOCKER.md:
  Prerequisites: Docker + Docker Compose
  Quick start: docker-compose up
  Environment variables reference
  Volume mounts
  Port configuration
  Updating to new version

docs/CONFIGURATION.md:
  Every config option: name | type | default | description | example
  Environment variables that override config
  CLI flags reference table
  API key configuration per provider
  Config file locations per platform
  Config encryption notes

docs/PROVIDERS.md:
  Table: Provider | Category | Key Required | Free Tier | Signup URL | Update Rate | What it shows
  Per provider section: description, what it adds, how to get key (step by step), rate limits

docs/FINANCIAL-ALERTS.md:
  What financial alerts SENTINEL provides
  Free data sources used
  Alert types and thresholds
  Setting up optional API keys (Alpha Vantage, Finnhub, FRED)
  Geopolitical correlation feature
  Watchlist management
  Disclaimer: SENTINEL is a situational awareness tool, not financial advice

docs/NOTIFICATIONS.md:
  Telegram setup (step by step with exact BotFather commands)
  Slack setup
  Discord setup
  Email setup (per method with screenshots)
  ntfy.sh setup
  Pushover setup
  Alert rules configuration
  Geofence alerts
  Morning briefing setup
  Weekly digest setup

docs/FEED-DASHBOARD.md:
  Five view modes explained
  Text alert feed usage
  Subscription setup walkthrough
  Keyword watchlist tips
  Asset tracking guide
  Incident threading explanation
  Financial alerts in feed

docs/OSINT-RESOURCES.md:
  Overview of OSINT Resources section
  Platform profile suggestions (how to use, how to add your own)
  State media warning policy
  Ham radio monitoring guide:
    What WebSDR is and how to use it (with steps)
    What KiwiSDR is and how to use it
    What Broadcastify is
    How SENTINEL suggests frequencies based on events
    Software defined radio basics (no hardware required with WebSDR)
    RTL-SDR hardware recommendation (optional, $25)
  Signal types guide (HFDL, ACARS, AIS, ADS-B)
  Online decoder tools
  Frequency reference table
  Curated OSINT platform guides:
    X/Twitter list management for OSINT
    Telegram OSINT channel safety notes
    Reddit multireddit setup for OSINT
  Disclaimer: SENTINEL suggests resources, all content must be independently verified

docs/MEDIA-WALL.md:
  View mode setup
  Supported stream types (YouTube live, HLS)
  Grid layout options
  Media presets guide
  OSINT YouTube channel directory
  Scanner feeds guide (Broadcastify)

docs/API.md:
  Base URL, authentication
  Every endpoint: method | path | description | request | response | curl example
  SSE stream format
  Error codes

docs/TROUBLESHOOTING.md:
  Globe doesn't load → Cesium token
  No events showing → provider health check steps
  Telegram not working → bot/token debugging
  Email not sending → SMTP testing steps
  High memory → satellite layer performance
  Port in use → platform-specific steps
  Database locked → WAL mode note
  Financial data missing → API key check
  WebSDR links not working → public receiver availability notes

docs/CHANGELOG.md:
  v2.0.0 — complete feature list
  v1.0.0 — previous features (from earlier build)

docs/CONTRIBUTING.md:
  How to add a data provider
  How to add a notification channel
  How to add OSINT profile suggestions
  How to add radio frequencies
  Build instructions
  Testing

docs/SECURITY.md:
  How to report issues
  What data SENTINEL collects (none sent externally)
  API key storage (AES-256-GCM encrypted at rest)
  Dashboard auth options
  Network exposure notes (bind to localhost for local-only)

docs/OSINT-PROFILES-DIRECTORY.md:
  Full directory of all pre-loaded OSINT profile suggestions
  Organized by: Military | Naval | Aviation | Space | Cyber | Financial | Disaster | Geopolitics
  Per platform section
  How to add your own
  Note on state media accounts (verify independently)

docs/RADIO-FREQUENCY-GUIDE.md:
  Complete frequency reference table (all bands from radio_frequencies table)
  Aviation frequencies guide
  Maritime frequencies guide
  Military HF networks guide
  Data signals guide (HFDL, ACARS, AIS)
  How to use WebSDR (step by step with screenshots)
  How to use KiwiSDR
  How to use Broadcastify
  RTL-SDR hardware guide (optional)
  Software guide (SDR++, GQRX, SDR#)

LICENSE:
  MIT License — full text

SECURITY.md (root):
  Copy of docs/SECURITY.md

DEPENDENCIES.md:
  All Go dependencies with version + license
  All JS CDN dependencies with version
  All external APIs used (links to their terms)
