# Changelog

All notable changes to SENTINEL are documented in this file.

## [3.0.0] -- 2026-03-11

### Added
- Single binary architecture with embedded frontend (go:embed)
- Mobile-first responsive design
- Signal Board -- DEFCON-style global threat posture across 5 domains (military, cyber, financial, natural, health)
- Correlation Flash -- multi-source incident detection (3+ sources, same region, 60-minute window)
- Truth Score -- source confirmation scoring (1-5 scale, cross-source validation)
- Anomaly Detector -- rolling 24-hour baseline with 3x spike threshold
- Dead Reckoning -- projected asset positions on signal loss (aircraft, vessels)
- Entity Tracker -- cross-source entity search
- Timeline Scrubber -- 24-hour event replay
- Intel Briefing -- AI-powered situational summary endpoint
- 33 Tier 0 (zero-key) providers:
  - Natural Disaster: USGS, GDACS, NOAA CAP, NOAA NWS, Tsunami, Volcano
  - Aviation: OpenSky Enhanced, Airplanes.live, ADSB.one
  - Weather: Open-Meteo
  - Conflict/OSINT: Iran Conflict, ISW, LiveUAMap, GDELT, Ukraine Alerts, Pikud HaOref
  - Space: CelesTrak, SWPC
  - Health: WHO, ProMED
  - Environmental: NASA FIRMS, Global Forest Watch
  - Financial: Financial Markets, SEC EDGAR
  - Maritime Security: Piracy IMB, UKMTO
  - Humanitarian: ReliefWeb
  - Cyber Security: CISA KEV, OTX AlienVault
  - Enrichment: Bellingcat ADS-B aircraft database (~500K registrations)
- 7 Tier 1 (free with key) providers:
  - ADS-B Exchange, AISStream, ACLED, OpenWeatherMap, OpenSanctions, Global Fishing Watch, NASA FIRMS RT
- 6 notification channels: Telegram, Email, Slack, Discord, ntfy, Pushover
- Alert rule engine with conditions (field/operator/value) and actions (log, webhook, Slack, Discord, Teams, email)
- Event acknowledgement system
- AES-256-GCM secret encryption for config values
- Cross-platform builds (Linux amd64/arm64, macOS amd64/arm64, Windows amd64)
- Cross-platform service installer (systemd, launchd, Windows sc.exe)
- System tray integration (Linux, macOS, Windows)
- First-run setup wizard (interactive terminal)
- Web-based settings page
- Docker support with multi-stage build
- Rate limiting middleware (token bucket, 100 req/s default)
- Auth middleware (optional API key)
- CORS middleware (permissive by default)
- NDJSON event log for external integration
- Provider health monitoring and statistics
- Performance metrics endpoint
- V3 schema migration (idempotent, safe to run multiple times)
- PWA support for mobile

### Changed
- Unified single repo (was split across 3 workspaces: backend, frontend, datainfra)
- Leaflet default map (was CesiumJS only; CesiumJS remains optional with token)
- Mobile-first design (was desktop-only)
- RAM target < 80 MB idle (was 400 MB+)
- Single JSON config file (was scattered env vars and config files)
- Pure Go SQLite driver (was CGO-dependent)
- gorilla/mux router (added path parameters, method routing)

### Removed
- Monolithic HTML dashboard files
- Hard CesiumJS dependency (now optional)
- Separate frontend/backend/datainfra repositories
- CGO build requirement
- Hardcoded paths and credentials

---

## [2.0.0] -- 2026-03-09

### Added
- V2 configuration system with platform-specific defaults
- CLI flags (--config, --data-dir, --port, --host, --version, --wizard, etc.)
- Service installer for systemd, launchd, Windows
- System tray icon with getlantern/systray
- Setup wizard (7-step interactive configuration)
- Web settings page with REST API
- Iran Conflict OSINT provider (GitHub dataset + ISW RSS)
- Bellingcat ADS-B aircraft identification database
- Enhanced OpenSky provider with military aircraft detection
- V1 to V2 config migration

### Changed
- Removed all hardcoded paths (/tmp/sentinel.db, etc.)
- Removed hardcoded Cesium tokens from HTML files
- Platform-specific default directories (Linux, macOS, Windows)

---

## [1.0.0] -- 2026-03-08

### Added
- Go backend with REST API and SSE streaming
- SQLite storage with FTS5 and R*Tree spatial indexing
- USGS, GDACS, OpenSky providers
- Alert system with rule evaluation
- Automatic backups
- CesiumJS 3D globe frontend
- Basic event filtering (source, category, severity, magnitude)
