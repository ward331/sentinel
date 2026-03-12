# SENTINEL V3 Deployment Guide

SENTINEL compiles to a single binary with an embedded web frontend. It uses SQLite for storage, so there is no external database to manage.

---

## Quick Start

```bash
# Download the binary for your platform
curl -LO https://github.com/openclaw/sentinel-backend/releases/latest/download/sentinel-linux-amd64
chmod +x sentinel-linux-amd64

# Run it
./sentinel-linux-amd64
```

That is it. SENTINEL will:
1. Create a data directory at `~/.local/share/sentinel/`
2. Initialize a SQLite database
3. Start polling 21+ free data providers
4. Serve the dashboard at http://localhost:8080

---

## Build from Source

Requirements: Go 1.24+

```bash
git clone https://github.com/openclaw/sentinel-backend.git
cd sentinel-backend
make build
./bin/sentinel-linux-amd64
```

### Cross-Platform Builds

```bash
# Build all platforms at once
make build-all

# Individual targets
make build-linux         # linux/amd64
make build-linux-arm     # linux/arm64 (Raspberry Pi, etc.)
make build-mac           # darwin/amd64
make build-mac-arm       # darwin/arm64 (Apple Silicon)
make build-windows       # windows/amd64
```

All binaries are statically linked (`CGO_ENABLED=0`) and have zero runtime dependencies.

### Release Packaging

```bash
make dist       # Create tar.gz/zip archives
make checksum   # Generate SHA256SUMS.txt
make release    # Full release pipeline
```

---

## Docker

### Docker Run

```bash
docker build -t sentinel:latest .
docker run -d \
  --name sentinel \
  -p 8080:8080 \
  -v sentinel-data:/app/data \
  sentinel:latest
```

### Docker Compose

```yaml
version: "3.8"

services:
  sentinel:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - sentinel-data:/app/data
      - ./config.json:/app/config.json:ro
    command: ["--host", "0.0.0.0", "--config", "/app/config.json", "--data-dir", "/app/data"]
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:8080/api/health"]
      interval: 30s
      timeout: 5s
      retries: 3
    deploy:
      resources:
        limits:
          memory: 128M

volumes:
  sentinel-data:
```

### Dockerfile

The included `Dockerfile` uses a multi-stage build:

```dockerfile
# Build stage
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags "-X main.Version=3.0.0 -s -w" \
    -o /sentinel ./cmd/sentinel/

# Runtime stage
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
RUN adduser -D -h /app sentinel
USER sentinel
WORKDIR /app
COPY --from=builder /sentinel /usr/local/bin/sentinel
EXPOSE 8080
VOLUME ["/app/data"]
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD wget -qO- http://localhost:8080/api/health || exit 1
ENTRYPOINT ["sentinel"]
CMD ["--host", "0.0.0.0", "--data-dir", "/app/data"]
```

The runtime image is ~15 MB. The sentinel binary is ~20 MB.

---

## Systemd Service (Linux)

Create `/etc/systemd/system/sentinel.service`:

```ini
[Unit]
Description=SENTINEL Global Event Monitor
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=sentinel
Group=sentinel
ExecStart=/usr/local/bin/sentinel --host 0.0.0.0 --port 8080 --data-dir /var/lib/sentinel
Restart=always
RestartSec=5
WorkingDirectory=/var/lib/sentinel

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/sentinel

# Resource limits
MemoryMax=256M
CPUQuota=50%

[Install]
WantedBy=multi-user.target
```

Install:

```bash
# Create user and directories
sudo useradd -r -s /bin/false -d /var/lib/sentinel sentinel
sudo mkdir -p /var/lib/sentinel
sudo chown sentinel:sentinel /var/lib/sentinel

# Install binary
sudo cp bin/sentinel-linux-amd64 /usr/local/bin/sentinel

# Enable and start
sudo systemctl daemon-reload
sudo systemctl enable --now sentinel

# Check status
sudo systemctl status sentinel
sudo journalctl -u sentinel -f
```

### User-Level Systemd (no root)

Create `~/.config/systemd/user/sentinel.service`:

```ini
[Unit]
Description=SENTINEL Global Event Monitor
After=network-online.target

[Service]
Type=simple
ExecStart=%h/.local/bin/sentinel --port 8080
Restart=always
RestartSec=5

[Install]
WantedBy=default.target
```

```bash
make install-linux   # Installs to ~/.local/bin/sentinel
systemctl --user daemon-reload
systemctl --user enable --now sentinel
```

---

## Reverse Proxy

### Nginx

```nginx
server {
    listen 443 ssl http2;
    server_name sentinel.example.com;

    ssl_certificate     /etc/letsencrypt/live/sentinel.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/sentinel.example.com/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # SSE support (critical)
        proxy_buffering off;
        proxy_cache off;
        proxy_read_timeout 86400s;
        proxy_set_header Connection "";
    }
}
```

### Caddy

```
sentinel.example.com {
    reverse_proxy localhost:8080 {
        flush_interval -1
    }
}
```

Caddy automatically handles TLS via Let's Encrypt. The `flush_interval -1` disables response buffering, which is required for SSE.

---

## Raspberry Pi / ARM Deployment

SENTINEL runs well on Raspberry Pi 3B+ and newer:

```bash
# Build for ARM64
make build-linux-arm

# Copy to Pi
scp bin/sentinel-linux-arm64 pi@raspberrypi:~/sentinel

# Run on Pi
ssh pi@raspberrypi
chmod +x ~/sentinel
./sentinel --port 8080
```

**Expected Performance (Pi 4, 4GB RAM):**
- Startup: ~2 seconds
- Idle memory: ~40-60 MB
- CPU: < 5% with all providers active
- SQLite DB: grows ~10 MB/day at default intervals

For headless Pi deployment, use the systemd service file above.

---

## Windows

### Manual

1. Download `sentinel-windows-amd64.exe`
2. Run: `sentinel-windows-amd64.exe --port 8080`
3. Open http://localhost:8080

### Windows Service

Use the built-in service installer:

```cmd
sentinel.exe --install-service
```

This creates a Windows service via `sc.exe`. To remove:

```cmd
sentinel.exe --uninstall-service
```

Config and data are stored in `%APPDATA%\SENTINEL\`.

---

## macOS

### Manual

```bash
./sentinel-darwin-arm64 --port 8080
```

### launchd Service

The built-in installer creates a launchd plist:

```bash
./sentinel --install-service
```

This creates `/Library/LaunchDaemons/com.sentinel.monitor.plist`. To remove:

```bash
./sentinel --uninstall-service
```

---

## Upgrading from V2

SENTINEL V3 is backward-compatible with V2 databases and config files.

### Migration Steps

1. Stop the V2 service
2. Back up your database and config:
   ```bash
   cp ~/.local/share/sentinel/sentinel.db ~/.local/share/sentinel/sentinel.db.v2backup
   cp ~/.config/sentinel/config.json ~/.config/sentinel/config.json.v2backup
   ```
3. Replace the binary with V3
4. Start the service

V3 automatically runs schema migrations on startup (adding `correlations`, `truth_confirmations`, `anomalies`, `signal_board_log`, `notification_log`, `alert_rules`, `briefing_log`, and `news_items` tables, plus `truth_score` and `acknowledged` columns on `events`). All migrations use `CREATE TABLE IF NOT EXISTS` and `ALTER TABLE ADD COLUMN` (idempotent).

### What Changed from V2

- New V3 intelligence engine tables (auto-created)
- Signal Board, Correlation Flash, Truth Score features
- Entity tracking and dead reckoning
- New providers added
- Mobile-first frontend redesign
- Config additions: `signal_board`, `entity_tracking` sections

### What Did Not Change

- Config file format (JSON, same location)
- Data directory structure
- Existing event data (preserved)
- API backward compatibility (all V2 endpoints still work)
- CLI flags

---

## Health Monitoring

### Health Check Endpoint

```bash
curl http://localhost:8080/api/health
# {"status":"ok","version":"v3.0.0","timestamp":"...","uptime":3600}

curl "http://localhost:8080/api/health?detailed=true"
# Returns component-level health information
```

### Monitoring with cron

```bash
# Add to crontab
*/5 * * * * curl -sf http://localhost:8080/api/health > /dev/null || systemctl --user restart sentinel
```

### Docker Health Check

Already included in the Dockerfile:

```dockerfile
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD wget -qO- http://localhost:8080/api/health || exit 1
```

---

## Resource Requirements

| Deployment  | CPU     | RAM    | Disk   |
|-------------|---------|--------|--------|
| Minimum     | 1 core  | 64 MB  | 100 MB |
| Recommended | 2 cores | 128 MB | 1 GB   |
| Heavy use   | 4 cores | 256 MB | 5 GB   |

RAM usage scales primarily with the number of active SSE clients and event volume. The SQLite database grows at approximately 10 MB/day with default provider intervals.

---

## TLS/HTTPS

### Built-in TLS

```json
{
  "server": {
    "tls_enabled": true,
    "tls_cert": "/etc/letsencrypt/live/sentinel.example.com/fullchain.pem",
    "tls_key": "/etc/letsencrypt/live/sentinel.example.com/privkey.pem"
  }
}
```

### Let's Encrypt with Certbot

```bash
sudo certbot certonly --standalone -d sentinel.example.com
```

For most deployments, using a reverse proxy (Caddy or Nginx) for TLS termination is simpler and recommended.
