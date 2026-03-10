# SENTINEL Backend - Deployment Guide

## 🚀 Production Deployment

This guide covers deploying SENTINEL Backend in production environments.

## 📋 Prerequisites

### System Requirements
- **OS**: Linux (Ubuntu 20.04+, CentOS 7+, or similar)
- **CPU**: 2+ cores
- **RAM**: 1 GB minimum (400 MB for Go, 600 MB for SQLite)
- **Disk**: 5 GB minimum (2 GB database + 3 GB backups/logs)
- **Network**: Outbound HTTPS access to provider APIs

### Software Dependencies
- **Go**: 1.21 or later (for building from source)
- **SQLite**: 3.35+ (modernc.org/sqlite driver included)
- **Systemd**: For service management (Linux)
- **Logrotate**: For log management

## 📦 Installation Methods

### Method 1: Binary Deployment (Recommended)

1. **Download the binary:**
```bash
wget https://github.com/openclaw/sentinel-backend/releases/latest/download/sentinel-linux-amd64
chmod +x sentinel-linux-amd64
sudo mv sentinel-linux-amd64 /usr/local/bin/sentinel
```

2. **Create configuration directory:**
```bash
sudo mkdir -p /etc/sentinel
sudo mkdir -p /var/lib/sentinel
sudo mkdir -p /var/log/sentinel
```

3. **Create configuration file:**
```bash
sudo tee /etc/sentinel/sentinel.env << EOF
# Core Configuration
SENTINEL_DB_PATH=/var/lib/sentinel/events.db
SENTINEL_HTTP_PORT=8080
SENTINEL_HTTP_HOST=0.0.0.0

# Performance
SENTINEL_CONNECTION_POOL=true
SENTINEL_MAX_CONNECTIONS=10
SENTINEL_RATE_LIMIT_ENABLED=true
SENTINEL_RATE_LIMIT_RPS=100
SENTINEL_RATE_LIMIT_BURST=200

# Operations
SENTINEL_LOGGING_ENABLED=true
SENTINEL_LOG_LEVEL=info
SENTINEL_LOG_FILE=/var/log/sentinel/sentinel.log
SENTINEL_BACKUP_ENABLED=true
SENTINEL_BACKUP_DIR=/var/lib/sentinel/backups
SENTINEL_BACKUP_SCHEDULE=6h
SENTINEL_BACKUP_RETENTION_DAYS=7

# Providers
SENTINEL_POLLER_INTERVAL=60s

# Alert Rules (optional)
SENTINEL_ALERT_WEBHOOK_URL=https://hooks.slack.com/services/...
SENTINEL_ALERT_EMAIL_FROM=alerts@example.com
SENTINEL_ALERT_EMAIL_TO=team@example.com
EOF
```

### Method 2: Build from Source

1. **Clone and build:**
```bash
git clone https://github.com/openclaw/sentinel-backend.git
cd sentinel-backend
make build
sudo cp sentinel /usr/local/bin/
```

2. **Install dependencies:**
```bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install -y ca-certificates

# CentOS/RHEL
sudo yum install -y ca-certificates
```

## 🛠️ Service Configuration

### Systemd Service (Linux)

Create systemd service file:

```bash
sudo tee /etc/systemd/system/sentinel.service << EOF
[Unit]
Description=SENTINEL Backend Service
After=network.target
StartLimitIntervalSec=0

[Service]
Type=simple
Restart=always
RestartSec=1
User=sentinel
Group=sentinel
EnvironmentFile=/etc/sentinel/sentinel.env
WorkingDirectory=/var/lib/sentinel
ExecStart=/usr/local/bin/sentinel
StandardOutput=append:/var/log/sentinel/sentinel.log
StandardError=append:/var/log/sentinel/sentinel-error.log

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ReadWritePaths=/var/lib/sentinel /var/log/sentinel

[Install]
WantedBy=multi-user.target
EOF
```

Create dedicated user:

```bash
sudo useradd -r -s /bin/false -m -d /var/lib/sentinel sentinel
sudo chown -R sentinel:sentinel /var/lib/sentinel /var/log/sentinel
```

### Docker Deployment

Create Dockerfile:

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o sentinel ./cmd/sentinel

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/sentinel .
COPY --from=builder /app/api/openapi.yaml ./api/
EXPOSE 8080
CMD ["./sentinel"]
```

Docker Compose example:

```yaml
version: '3.8'
services:
  sentinel:
    build: .
    ports:
      - "8080:8080"
    environment:
      - SENTINEL_DB_PATH=/data/events.db
      - SENTINEL_HTTP_PORT=8080
      - SENTINEL_HTTP_HOST=0.0.0.0
    volumes:
      - sentinel-data:/data
      - sentinel-backups:/backups
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/api/health"]
      interval: 30s
      timeout: 10s
      retries: 3

volumes:
  sentinel-data:
  sentinel-backups:
```

## 🔧 Configuration Reference

### Environment Variables

#### Core Configuration
| Variable | Default | Description |
|----------|---------|-------------|
| `SENTINEL_DB_PATH` | `/tmp/sentinel.db` | SQLite database file path |
| `SENTINEL_HTTP_PORT` | `8080` | HTTP server port |
| `SENTINEL_HTTP_HOST` | `0.0.0.0` | HTTP server bind address |
| `SENTINEL_HTTP_READ_TIMEOUT` | `30s` | HTTP read timeout |
| `SENTINEL_HTTP_WRITE_TIMEOUT` | `30s` | HTTP write timeout |
| `SENTINEL_HTTP_IDLE_TIMEOUT` | `60s` | HTTP idle timeout |

#### Performance
| Variable | Default | Description |
|----------|---------|-------------|
| `SENTINEL_CONNECTION_POOL` | `true` | Enable SQLite connection pooling |
| `SENTINEL_MAX_CONNECTIONS` | `5` | Maximum database connections |
| `SENTINEL_RATE_LIMIT_ENABLED` | `false` | Enable rate limiting |
| `SENTINEL_RATE_LIMIT_RPS` | `100` | Requests per second limit |
| `SENTINEL_RATE_LIMIT_BURST` | `200` | Burst limit |
| `SENTINEL_RATE_LIMIT_EXEMPT_PATHS` | `/api/health,/api/events/stream` | Paths exempt from rate limiting |

#### Operations
| Variable | Default | Description |
|----------|---------|-------------|
| `SENTINEL_LOGGING_ENABLED` | `true` | Enable structured logging |
| `SENTINEL_LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |
| `SENTINEL_LOG_FORMAT` | `text` | Log format (text, json) |
| `SENTINEL_LOG_FILE` | `stdout` | Log file path (stdout for console) |
| `SENTINEL_BACKUP_ENABLED` | `true` | Enable automatic backups |
| `SENTINEL_BACKUP_DIR` | `./backups` | Backup directory |
| `SENTINEL_BACKUP_SCHEDULE` | `24h` | Backup interval (e.g., 6h, 24h) |
| `SENTINEL_BACKUP_RETENTION_DAYS` | `7` | Days to keep backups |

#### Providers
| Variable | Default | Description |
|----------|---------|-------------|
| `SENTINEL_POLLER_INTERVAL` | `60s` | Poller interval |
| `SENTINEL_USGS_ENABLED` | `true` | Enable USGS provider |
| `SENTINEL_GDACS_ENABLED` | `true` | Enable GDACS provider |
| `SENTINEL_USGS_FEED_URL` | USGS URL | Custom USGS feed URL |
| `SENTINEL_GDACS_FEED_URL` | GDACS URL | Custom GDACS feed URL |

#### Alert System
| Variable | Default | Description |
|----------|---------|-------------|
| `SENTINEL_ALERT_WEBHOOK_URL` | `` | Webhook URL for alerts |
| `SENTINEL_ALERT_EMAIL_FROM` | `` | Email sender for alerts |
| `SENTINEL_ALERT_EMAIL_TO` | `` | Email recipient for alerts |
| `SENTINEL_ALERT_SMTP_HOST` | `` | SMTP host for email alerts |
| `SENTINEL_ALERT_SMTP_PORT` | `587` | SMTP port |
| `SENTINEL_ALERT_SMTP_USER` | `` | SMTP username |
| `SENTINEL_ALERT_SMTP_PASS` | `` | SMTP password |

## 🔒 Security Configuration

### Firewall Rules
```bash
# Allow HTTP port
sudo ufw allow 8080/tcp

# Or if behind reverse proxy
sudo ufw allow from 192.168.1.0/24 to any port 8080
```

### Reverse Proxy (Nginx)

```nginx
upstream sentinel_backend {
    server 127.0.0.1:8080;
    keepalive 32;
}

server {
    listen 80;
    server_name sentinel.example.com;
    
    # Redirect to HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name sentinel.example.com;
    
    ssl_certificate /etc/letsencrypt/live/sentinel.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/sentinel.example.com/privkey.pem;
    
    # SSL configuration
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES256-GCM-SHA512:DHE-RSA-AES256-GCM-SHA512;
    ssl_prefer_server_ciphers off;
    
    # Security headers
    add_header X-Frame-Options DENY;
    add_header X-Content-Type-Options nosniff;
    add_header X-XSS-Protection "1; mode=block";
    
    # Proxy configuration
    location / {
        proxy_pass http://sentinel_backend;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # SSE support
        proxy_buffering off;
        proxy_cache off;
        
        # Timeouts
        proxy_connect_timeout 7d;
        proxy_send_timeout 7d;
        proxy_read_timeout 7d;
    }
    
    # Health check endpoint
    location /api/health {
        proxy_pass http://sentinel_backend/api/health;
        access_log off;
    }
}
```

### SSL/TLS Certificates

```bash
# Using Let's Encrypt with Certbot
sudo apt-get install certbot python3-certbot-nginx
sudo certbot --nginx -d sentinel.example.com
```

## 📊 Monitoring Setup

### Health Checks
```bash
# Manual health check
curl -f http://localhost:8080/api/health

# Cron job for monitoring
*/5 * * * * curl -f http://localhost:8080/api/health || systemctl restart sentinel
```

### Log Management

Logrotate configuration:

```bash
sudo tee /etc/logrotate.d/sentinel << EOF
/var/log/sentinel/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    create 640 sentinel sentinel
    sharedscripts
    postrotate
        systemctl reload sentinel > /dev/null 2>&1 || true
    endscript
}
EOF
```

### Metrics Collection

Prometheus configuration:

```yaml
scrape_configs:
  - job_name: 'sentinel'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/api/metrics'
    scrape_interval: 15s
```

Grafana dashboard example metrics:
- API request rate
- Event ingestion rate
- Memory usage
- Database size
- SSE client connections
- Error rates

## 🔄 Backup and Recovery

### Manual Backup
```bash
# Create backup
sentinel-backup --output /backups/sentinel-$(date +%Y%m%d-%H%M%S).db

# List backups
sentinel-backup --list

# Restore from backup
sentinel-backup --restore /backups/sentinel-20240308-120000.db
```

### Automated Backup Script
```bash
#!/bin/bash
BACKUP_DIR="/var/lib/sentinel/backups"
RETENTION_DAYS=7
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# Create backup
cp /var/lib/sentinel/events.db "$BACKUP_DIR/sentinel_$TIMESTAMP.db"

# Compress backup
gzip "$BACKUP_DIR/sentinel_$TIMESTAMP.db"

# Clean old backups
find "$BACKUP_DIR" -name "sentinel_*.db.gz" -mtime +$RETENTION_DAYS -delete

# Log backup
echo "$(date): Backup created: sentinel_$TIMESTAMP.db.gz" >> /var/log/sentinel/backup.log
```

### Disaster Recovery

1. **Stop the service:**
```bash
sudo systemctl stop sentinel
```

2. **Restore database:**
```bash
# Copy backup to database location
sudo cp /backups/sentinel-latest.db /var/lib/sentinel/events.db
sudo chown sentinel:sentinel /var/lib/sentinel/events.db
```

3. **Start the service:**
```bash
sudo systemctl start sentinel
```

4. **Verify recovery:**
```bash
curl http://localhost:8080/api/health
```

## 📈 Scaling

### Vertical Scaling
- Increase RAM to 2-4 GB for larger datasets
- Use SSD storage for better I/O performance
- Increase CPU cores for higher concurrency

### Horizontal Scaling Considerations
1. **Load Balancer**: Distribute traffic across multiple instances
2. **Shared Storage**: Use network-attached storage for database
3. **Session Affinity**: Required for SSE connections
4. **Database**: Consider migrating to PostgreSQL for multi-instance deployment

### High Availability Setup
```nginx
upstream sentinel_backend {
    server 192.168.1.101:8080;
    server 192.168.1.102:8080;
    server 192.168.1.103:8080;
    
    # Session affinity for SSE
    hash $remote_addr consistent;
    
    # Health checks
    check interval=3000 rise=2 fall=3 timeout=1000;
}
```

## 🚨 Troubleshooting Production Issues

### Service Won't Start
```bash
# Check logs
sudo journalctl -u sentinel -f

# Check permissions
sudo ls -la /var/lib/sentinel/events.db

# Check port availability
sudo netstat -tulpn | grep :8080
```

### High Memory Usage
```bash
# Monitor memory
top -p $(pgrep sentinel)

# Check for memory leaks
curl http://localhost:8080/debug/pprof/heap > heap.pprof
go tool pprof heap.pprof
```

### Database Issues
```bash
# Check database integrity
sqlite3 /var/lib/sentinel/events.db "PRAGMA integrity_check;"

# Check database size
du -h /var/lib/sentinel/events.db

# Rebuild indexes if slow
sqlite3 /var/lib/sentinel/events.db "REINDEX;"
```

### No Events from Providers
```bash
# Check network connectivity
curl -v https://earthquake.usgs.gov/earthquakes/feed/v1.0/summary/all_hour.geojson

# Check poller logs
grep poller /var/log/sentinel/sentinel.log

# Test provider manually
sentinel-test-provider --provider usgs
```

## 🔄 Updates and Maintenance

### Update Procedure
1. **Backup current installation:**
```bash
sudo systemctl stop sentinel
cp /var/lib/sentinel/events.db /backups/events.db.bak
```

2. **Update binary:**
```bash
wget https://github.com/openclaw/sentinel-backend/releases/latest/download/sentinel-linux-amd64
sudo cp sentinel-linux-amd64 /usr/local/bin/sentinel
sudo chmod +x /usr/local/bin/sentinel
```

3. **Restart service:**
```bash
sudo systemctl start sentinel
sudo systemctl status sentinel
```

4. **Verify update:**
```bash
curl http://localhost:8080/api/health
```

### Database Maintenance
```bash
# Weekly maintenance script
#!/bin/bash
DB_PATH="/var/lib/sentinel/events.db"

# Vacuum to reclaim space
sqlite3 "$DB_PATH" "VACUUM;"

# Analyze for query optimization
sqlite3 "$DB_PATH" "ANALYZE;"

# Update statistics
sqlite3 "$DB_PATH" "UPDATE sqlite_stat1;"
```

## 📞 Support

### Getting Help
- **Documentation**: Check README.md and this guide
- **Issues**: GitHub issue tracker
- **Community**: OpenClaw Discord server
- **Email**: support@example.com

### Common Support Scenarios
1. **Installation issues**: Check prerequisites and permissions
2. **Performance problems**: Review configuration and system resources
3. **Data not updating**: Verify provider API connectivity
4. **Alert not working**: Check webhook/email configuration

## 📝 Changelog

Keep track of updates and changes in `CHANGELOG.md`.

---

**Next Steps After Deployment:**
1. Configure monitoring and alerts
2. Set up regular backup verification
3. Establish update schedule
4. Document incident response procedures
5. Train operations team

For additional help, refer to the [README.md](./README.md) or contact support.
