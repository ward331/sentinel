# SENTINEL

Real-time global event monitoring in a single binary. Earthquakes, conflicts, flights, weather, cyber threats, financial markets -- all on one dashboard.

<!-- ![SENTINEL Dashboard](docs/screenshot.png) -->

---

## Key Features

- **Single binary, zero config** -- download and run, no database server or dependencies
- **33+ data providers** -- USGS, GDACS, NOAA, OpenSky, GDELT, WHO, NASA FIRMS, and more
- **Zero API keys required** -- all Tier 0 providers work out of the box
- **Signal Board** -- DEFCON-style threat posture across 5 domains
- **Correlation Flash** -- automatic multi-source incident detection
- **Truth Score** -- cross-source confirmation scoring (1-5)
- **Anomaly Detection** -- rolling baseline spike detection
- **Dead Reckoning** -- projected asset positions when signal is lost
- **6 notification channels** -- Telegram, Slack, Discord, Email, ntfy, Pushover
- **Alert rules** -- configurable conditions and actions per event
- **Real-time SSE streaming** -- live events pushed to all connected clients
- **Mobile-first design** -- responsive dashboard with PWA support
- **Cross-platform** -- Linux, macOS, Windows, ARM (Raspberry Pi)
- **< 80 MB RAM** -- lightweight enough for a Raspberry Pi

---

## Quick Start

```bash
# 1. Download
curl -LO https://github.com/openclaw/sentinel-backend/releases/latest/download/sentinel-linux-amd64
chmod +x sentinel-linux-amd64

# 2. Run
./sentinel-linux-amd64

# 3. Open
open http://localhost:8080
```

Or build from source:

```bash
git clone https://github.com/openclaw/sentinel-backend.git
cd sentinel-backend
make build
./bin/sentinel-linux-amd64
```

---

## Architecture

```
Data Sources (33+)  -->  Poller  -->  SQLite  -->  REST API / SSE  -->  Dashboard
                                        |
                                  Intelligence
                                  - Correlation
                                  - Truth Score
                                  - Anomaly Detection
                                  - Signal Board
```

SENTINEL is a single Go binary with an embedded web frontend. It uses SQLite (pure Go, no CGO) for storage and polls data sources on configurable intervals. Events are streamed to clients via Server-Sent Events (SSE).

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for the full technical breakdown.

---

## Documentation

| Document | Description |
|----------|-------------|
| [API Reference](docs/API.md) | Complete REST API documentation |
| [Provider Catalog](docs/PROVIDERS.md) | All 33+ data providers with endpoints and formats |
| [Configuration Guide](docs/CONFIGURATION.md) | Config file, CLI flags, notification setup |
| [Deployment Guide](docs/DEPLOYMENT.md) | Docker, systemd, reverse proxy, Raspberry Pi |
| [Architecture](docs/ARCHITECTURE.md) | System design, data flow, database schema |
| [Troubleshooting](docs/TROUBLESHOOTING.md) | Common issues and fixes |
| [Dependencies](docs/DEPENDENCIES.md) | Go module inventory with licenses |
| [Changelog](docs/CHANGELOG.md) | Version history |
| [Known Issues](KNOWN_ISSUES.md) | Current limitations and TODOs |

---

## Configuration

SENTINEL works with zero configuration. To customize, create a `config.json`:

```json
{
  "server": { "port": 8080, "host": "0.0.0.0" },
  "providers": {
    "usgs": { "enabled": true, "interval_seconds": 60 },
    "opensky": { "enabled": false }
  },
  "telegram": {
    "enabled": true,
    "bot_token": "your-bot-token",
    "chat_id": "-1001234567890"
  }
}
```

```bash
./sentinel --config config.json
```

See [docs/CONFIGURATION.md](docs/CONFIGURATION.md) for all options.

---

## Docker

```bash
docker build -t sentinel .
docker run -d -p 8080:8080 -v sentinel-data:/app/data sentinel
```

---

## API

```bash
# Health check
curl http://localhost:8080/api/health

# List events
curl http://localhost:8080/api/events?category=earthquake&limit=10

# SSE stream
curl -N http://localhost:8080/api/events/stream

# Signal board
curl http://localhost:8080/api/signal-board

# Search entities
curl http://localhost:8080/api/entity/search?q=Boeing
```

See [docs/API.md](docs/API.md) for the complete reference.

---

## Building

```bash
make build            # Linux amd64
make build-all        # All platforms
make test             # Run tests
make smoke            # Build + smoke test
make docker           # Build Docker image
make release          # Full release pipeline with checksums
```

---

## License

MIT License. See [LICENSE](LICENSE) for details.

---

## Contributing

Contributions are welcome.

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Make your changes
4. Run tests: `make test`
5. Run the smoke test: `make smoke`
6. Commit with a clear message
7. Open a pull request

Please follow Go conventions (`gofmt`, `go vet`) and add tests for new features.

### Adding a New Provider

1. Create `internal/provider/yourprovider.go` implementing the `Provider` interface
2. Register it in `cmd/sentinel/main.go` inside `initializePoller()`
3. Add its config to `internal/config/v2config.go` in `ProvidersConfig`
4. Document it in `docs/PROVIDERS.md`
5. Add a smoke test case
