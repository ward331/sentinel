# SENTINEL V3 Technical Architecture

## System Overview

```
                         +-----------------------------------------+
                         |          SENTINEL V3 Binary              |
                         |                                         |
  Data Sources           |   +----------+    +------------------+  |       Clients
  (33+ providers)        |   | Poller   |--->| Storage (SQLite) |  |
       |                 |   | Engine   |    |   events         |  |     +----------+
       v                 |   +----------+    |   correlations   |  |     | Browser  |
  +---------+            |        |          |   anomalies      |  |     | (PWA)    |
  | USGS    |-----+      |        v          |   news_items     |  |---->| Leaflet  |
  | GDACS   |-----+----->|   +----------+   |   signal_board   |  |     | SSE      |
  | NOAA    |-----+      |   | Engine   |   +------------------+  |     +----------+
  | OpenSky |-----+      |   | - Correl.|        |               |
  | GDELT   |-----+      |   | - Truth  |        v               |     +----------+
  | SWPC    |-----+      |   | - Anomaly|   +----------+         |     | Telegram |
  | WHO     |-----+      |   | - Signal |   | HTTP API |         |---->| Slack    |
  | ...     |-----+      |   | - DeadR. |   | (mux)    |         |     | Discord  |
  +---------+            |   +----------+   +----------+         |     | ntfy     |
                         |                       |               |     | Email    |
                         |                       v               |     | Pushover |
                         |                  +----------+         |     +----------+
                         |                  | SSE      |         |
                         |                  | Broker   |---------|
                         |                  +----------+         |
                         +-----------------------------------------+
```

---

## Package Structure

```
sentinel-backend/
  cmd/
    sentinel/
      main.go              # Entry point, CLI flags, server startup
      web/                  # Embedded frontend files (go:embed)

  internal/
    api/                   # HTTP handlers and middleware
      handler.go           # Core event/provider/health handlers
      router.go            # gorilla/mux router factory
      stream.go            # SSE broker (new_event, correlation, etc.)
      signal_board.go      # GET /api/signal-board
      financial.go         # GET /api/financial/overview
      intel.go             # GET /api/intel/briefing, GET /api/news
      notifications.go     # Notification config and test endpoints
      alerts.go            # Alert rule CRUD (PUT, DELETE)
      entity.go            # GET /api/entity/search
      correlations.go      # GET /api/correlations
      acknowledge.go       # POST /api/events/{id}/acknowledge
      config_ui.go         # GET /api/config/ui
      settings.go          # GET/POST /api/config
      osint_resources.go   # OSINT resources sub-API
      filter_handler.go    # Filter API endpoints
      middleware.go         # Auth middleware
      ratelimit.go         # Rate limiting middleware
      cors.go              # CORS middleware
      auth.go              # Authentication config

    alert/
      rules.go             # Alert rule engine (evaluate, webhook, Slack, etc.)

    config/
      v2config.go          # Config struct, load/save, defaults
      config_v1.go         # V1 compatibility
      defaults.go          # Default values
      encrypt.go           # AES-256-GCM secret encryption
      migrate.go           # V1 -> V2 config migration

    engine/
      correlation.go       # Correlation Flash (multi-source incident detection)
      truth.go             # Truth Score calculator (1-5, cross-source)
      anomaly.go           # Anomaly Detector (rolling baseline spike)
      signal_board.go      # Signal Board (5-domain threat levels)
      dead_reckoning.go    # Dead Reckoning (projected positions)
      geo.go               # Haversine distance, geo utilities

    filter/
      engine.go            # Filter rule engine
      evaluator.go         # Rule evaluation logic
      interface.go         # Filter types and interfaces

    health/
      health.go            # Health check registry

    infrastructure/
      health_reporter.go   # Provider health stats
      ndjson_log.go        # NDJSON event log writer

    intel/
      briefing.go          # AI intelligence briefing generator
      news_agg.go          # News aggregation from RSS feeds

    logging/
      middleware.go        # Request logging middleware

    metrics/
      metrics.go           # Internal performance metrics

    model/
      event.go             # Event, Location, Badge, Severity types
      osint_resource.go    # OSINT resource model

    notify/
      dispatcher.go        # Notification dispatcher
      telegram.go          # Telegram channel
      slack.go             # Slack channel
      discord.go           # Discord channel
      email.go             # Email (SMTP) channel
      ntfy.go              # ntfy channel

    poller/
      poller.go            # Provider scheduling, dedup, stats

    provider/
      interface.go         # Provider interface and BaseProvider
      common.go            # Shared provider utilities
      usgs.go              # USGS earthquakes
      gdacs.go             # GDACS multi-hazard
      noaa_cap.go          # NOAA CAP alerts
      noaa_nws.go          # NOAA NWS weather
      tsunami.go           # Tsunami warnings
      volcano.go           # Volcanic activity
      opensky_enhanced.go  # OpenSky + aircraft ID
      airplanes_live.go    # Airplanes.live ADS-B
      adsb_one.go          # ADSB.one fallback
      openmeteo.go         # Open-Meteo weather
      gdelt.go             # GDELT news events
      liveuamap.go         # LiveUAMap conflicts
      iranconflict.go      # Iran conflict OSINT + ISW
      celestrak.go         # CelesTrak satellites
      swpc.go              # NOAA space weather
      who.go               # WHO disease outbreaks
      promed.go            # ProMED health
      nasa_firms.go        # NASA FIRMS fires
      piracy_imb.go        # Piracy reports
      financial_markets.go # Financial indicators
      reliefweb.go         # UN ReliefWeb
      ... (37 total)

    providers/
      aircraft/
        database.go        # Bellingcat ADS-B aircraft lookup DB

    server/
      shutdown.go          # Graceful shutdown handler

    service/
      installer.go         # Cross-platform service installer

    setup/
      wizard.go            # First-run interactive setup wizard

    storage/
      storage.go           # SQLite operations (CRUD, FTS, spatial)
      optimization.go      # Connection pooling, WAL mode, indexes
      engine_queries.go    # Engine-specific DB queries
      v3_migration.go      # V3 schema migration
      osint_storage.go     # OSINT resources DB layer

    tray/
      tray.go              # System tray icon (Linux/macOS/Windows)

  frontend/               # Frontend source (built into web/)
  web/                    # Compiled frontend (embedded via go:embed)
  manifests/              # OSINT resource manifests
  docs/                   # Documentation (this directory)
```

---

## Data Flow

### Event Ingestion Pipeline

```
External API  -->  Provider.Fetch()  -->  Poller  -->  Dedup Check
                                                          |
                                              (new event) |  (duplicate: skip)
                                                          v
                                                    Storage.StoreEvent()
                                                          |
                                                          v
                                              +----  Alert Engine  ----+
                                              |    (evaluate rules)    |
                                              |           |            |
                                              v           v            v
                                          Log Alert   Webhook     Notification
                                                                  Dispatcher
                                                          |
                                                          v
                                                   SSE Broker
                                                   .Broadcast()
                                                          |
                                              +-----------+-----------+
                                              |           |           |
                                              v           v           v
                                          Client 1   Client 2   Client N
                                          (browser)  (browser)  (mobile)
```

### Intelligence Pipeline

```
Events in DB
     |
     v
+----+----+    +----+----+    +----+----+    +----+----+
| Correlat.|    |  Truth   |    | Anomaly  |    | Signal  |
|  Engine  |    |  Score   |    | Detector |    |  Board  |
+----+----+    +----+----+    +----+----+    +----+----+
     |              |              |              |
     v              v              v              v
correlations   truth_score    anomalies     signal_board_log
   table       on events       table           table
     |              |              |              |
     +------+-------+------+------+              |
            |                                    |
            v                                    v
     /api/correlations                    /api/signal-board
     SSE: correlation                     SSE: signal_board
```

**Correlation Flash:** When 3+ independent sources report events within the same geographic radius (default 50 km) within 60 minutes, a correlation flash is created.

**Truth Score (1-5):** Each event gets a truth score based on cross-source confirmation:
- 1 = Single source only
- 2 = Two independent sources
- 3 = Three or more sources agree
- 4 = Confirmed by an authoritative source (USGS, NOAA, WHO)
- 5 = Multiple authoritative sources confirm

**Anomaly Detector:** Maintains a rolling 24-hour baseline of event rates per provider per region. Fires when the actual rate exceeds 3x the baseline (spike factor).

**Signal Board:** Aggregates threat levels across five domains (military, cyber, financial, natural, health), each rated 0-5.

**Dead Reckoning:** When an aircraft or vessel signal is lost, the engine projects the entity's position forward using its last known heading, speed, and elapsed time. Projections expire after a configurable number of minutes (default 30).

---

## Database Schema

### Core Tables

**events**
```sql
CREATE TABLE events (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT,
    source TEXT NOT NULL,
    source_id TEXT,
    occurred_at DATETIME NOT NULL,
    ingested_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    lat REAL, lon REAL,
    location_type TEXT,
    location_json TEXT,
    precision TEXT,
    magnitude REAL,
    category TEXT,
    severity TEXT,
    metadata_json TEXT,
    badges_json TEXT,
    truth_score INTEGER DEFAULT 1,
    acknowledged INTEGER DEFAULT 0
);
```

### V3 Intelligence Tables

**correlations** -- Multi-source incident groupings
```sql
CREATE TABLE correlations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    region_name TEXT,
    lat REAL, lon REAL,
    radius_km REAL,
    event_count INTEGER,
    source_count INTEGER,
    started_at DATETIME,
    last_event_at DATETIME,
    confirmed INTEGER DEFAULT 0,
    incident_name TEXT,
    events_json TEXT
);
```

**truth_confirmations** -- Cross-source confirmation records
```sql
CREATE TABLE truth_confirmations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    primary_event_id INTEGER,
    confirming_source TEXT,
    confirming_event_id INTEGER,
    confirmed_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

**anomalies** -- Spike detection log
```sql
CREATE TABLE anomalies (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    provider_name TEXT,
    region TEXT,
    expected_rate REAL,
    actual_rate REAL,
    spike_factor REAL,
    detected_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    resolved_at DATETIME
);
```

**signal_board_log** -- Historical threat level snapshots
```sql
CREATE TABLE signal_board_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    military INTEGER, cyber INTEGER, financial INTEGER,
    natural INTEGER, health INTEGER,
    calculated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

**notification_log** -- Audit trail for sent notifications
```sql
CREATE TABLE notification_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    channel TEXT,
    event_id INTEGER,
    sent_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    status TEXT, error TEXT
);
```

**alert_rules** -- Persisted alert rules
```sql
CREATE TABLE alert_rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT,
    conditions_json TEXT,
    actions_json TEXT,
    enabled INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

**briefing_log** -- AI briefing history
```sql
CREATE TABLE briefing_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    content TEXT,
    generated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    delivered_channels TEXT
);
```

**news_items** -- Aggregated news
```sql
CREATE TABLE news_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    url TEXT UNIQUE NOT NULL,
    description TEXT,
    source_name TEXT,
    source_category TEXT,
    pub_date DATETIME,
    ingested_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    relevance_score INTEGER DEFAULT 0,
    lat REAL, lon REAL,
    matched_event_id INTEGER,
    truth_score INTEGER DEFAULT 1
);
```

### Storage Features

- **WAL mode:** Write-Ahead Logging for concurrent read/write
- **FTS5:** Full-text search on event titles and descriptions
- **Connection pooling:** Optimized for Go's `database/sql` driver
- **Pure Go SQLite:** Uses `modernc.org/sqlite` (no CGO required)

---

## Performance Characteristics

| Metric | Target | Notes |
|--------|--------|-------|
| Binary size | ~20 MB | Stripped, statically linked |
| Startup time | < 2s | Database init + provider registration |
| Idle memory | ~50-80 MB | 21 providers active |
| Per-SSE client | ~2 MB | Buffered channel per client |
| Event ingestion | 1000+/min | Limited by provider fetch rates |
| API latency (P95) | < 50 ms | Simple queries on indexed columns |
| SQLite write throughput | ~500 events/sec | WAL mode, batched inserts |
| DB growth rate | ~10 MB/day | Default intervals, all providers |

### Tuning

- **Reduce memory:** Disable unused providers in config
- **Reduce disk I/O:** Increase provider poll intervals
- **Reduce DB size:** Lower `ui.data_retention_days` (default 30)
- **More SSE clients:** Increase the stream broker buffer size
- **Faster queries:** The storage layer auto-creates indexes on `source`, `category`, `severity`, `occurred_at`

---

## Concurrency Model

- **Main goroutine:** HTTP server (gorilla/mux)
- **Poller goroutine per provider:** Each provider runs in its own goroutine with its own timer
- **SSE broker:** Fan-out pattern with buffered channels (1000-event buffer)
- **Engine goroutines:** Correlation, truth, anomaly, and signal board each run periodic scans
- **Graceful shutdown:** Context cancellation propagates to all goroutines, with a 30-second timeout

---

## Technology Stack

| Component | Technology | Purpose |
|-----------|-----------|---------|
| Language | Go 1.24+ | Server, all business logic |
| HTTP Router | gorilla/mux | URL routing, path params |
| Database | modernc.org/sqlite | Pure Go SQLite driver |
| UUID | google/uuid | Event ID generation |
| Rate Limiting | golang.org/x/time | Token bucket rate limiter |
| System Tray | getlantern/systray | Desktop tray icon |
| Frontend | Embedded HTML/JS/CSS | Served via go:embed |
| Map | Leaflet.js | Default 2D map |
| Globe | CesiumJS (optional) | 3D globe (requires token) |
