# SENTINEL V3 Troubleshooting Guide

---

## Provider Errors

### Provider returns no events

**Symptoms:** A provider shows `success_count: 0` and events are never created.

**Possible causes:**
- The provider's upstream API is down or returning empty results
- Network firewall blocking outbound HTTPS
- The provider requires an API key that is not configured

**Fix:**
1. Check the provider's API directly:
   ```bash
   curl -s "https://earthquake.usgs.gov/earthquakes/feed/v1.0/summary/all_hour.geojson" | head -c 200
   ```
2. Check provider health: `GET /api/providers/health`
3. Check logs for HTTP error codes
4. For Tier 1 providers, verify the API key is set in config

### Provider returns errors repeatedly

**Symptoms:** `error_count` grows, `last_error` shows HTTP 429 or 403.

**Possible causes:**
- Rate limit exceeded on the upstream API
- API key expired or revoked
- IP blocked by upstream provider

**Fix:**
1. Increase the provider's `interval_seconds` in config
2. Rotate or refresh the API key
3. Check if your IP is on a block list (common with ADS-B Exchange)

### OpenSky returning 401 Unauthorized

OpenSky anonymous access is limited to 100 requests/day. Register for a free account at https://opensky-network.org/index.php/-/login to increase to 4000/day.

### GDELT returning large payloads

GDELT can return very large JSON responses. If memory spikes, increase the provider interval or add category filters in the provider options.

---

## Database Issues

### "database is locked"

**Cause:** Multiple writers attempting concurrent SQLite writes.

**Fix:**
- SENTINEL uses WAL mode by default, which should handle this. If the error persists:
  ```bash
  sqlite3 /path/to/sentinel.db "PRAGMA journal_mode=WAL;"
  ```
- Ensure only one SENTINEL instance is running against the same database file
- Check that no other process has the database open

### Database file growing too large

**Fix:**
1. Lower `ui.data_retention_days` in config (default: 30)
2. Run a manual vacuum:
   ```bash
   sqlite3 /path/to/sentinel.db "VACUUM;"
   ```
3. Disable high-frequency providers you do not need (aviation providers at 30s intervals produce the most data)

### Corrupted database

**Symptoms:** SQL errors, "malformed" messages, or segfaults.

**Fix:**
1. Check integrity:
   ```bash
   sqlite3 /path/to/sentinel.db "PRAGMA integrity_check;"
   ```
2. If corrupt, restore from backup:
   ```bash
   cp /path/to/sentinel/backups/latest.db /path/to/sentinel/sentinel.db
   ```
3. If no backup exists, delete the database and restart. SENTINEL will recreate the schema.

### Migration errors on startup

V3 migrations use `CREATE TABLE IF NOT EXISTS` and `ALTER TABLE ADD COLUMN`. If a migration fails:

1. Check logs for the specific SQL error
2. The most common cause is a schema mismatch from a partially applied migration
3. Back up the database, then try deleting and recreating:
   ```bash
   mv sentinel.db sentinel.db.broken
   # Restart SENTINEL -- it will create a fresh database
   ```

---

## Build Errors

### "modernc.org/sqlite: build constraints exclude all Go files"

**Cause:** CGO_ENABLED is set incorrectly.

**Fix:** SENTINEL uses the pure-Go SQLite driver. Build with:
```bash
CGO_ENABLED=0 go build ./cmd/sentinel/
```

### "go: module requires Go 1.24"

**Fix:** Update Go to 1.24 or later:
```bash
go install golang.org/dl/go1.24.0@latest
go1.24.0 download
```

### "cannot find package github.com/getlantern/systray"

**Fix:**
```bash
go mod download
# or
go mod tidy
```

### Cross-compilation fails for Windows

**Cause:** Some dependencies may have OS-specific build tags.

**Fix:** Use the Makefile targets which set the correct flags:
```bash
make build-windows
```

---

## Network / Firewall Issues

### Providers cannot reach upstream APIs

**Symptoms:** All providers show errors, "connection refused" or "timeout".

**Fix:**
1. Verify outbound HTTPS is allowed:
   ```bash
   curl -s https://earthquake.usgs.gov/earthquakes/feed/v1.0/summary/all_hour.geojson | head -c 50
   ```
2. If behind a corporate proxy, set `HTTP_PROXY` / `HTTPS_PROXY` environment variables
3. Ensure DNS resolution works: `nslookup earthquake.usgs.gov`

### SSE connections drop after 60 seconds

**Cause:** A reverse proxy or load balancer is timing out idle connections.

**Fix:**
- Nginx: Set `proxy_read_timeout 86400s;` and `proxy_buffering off;`
- Caddy: Set `flush_interval -1`
- HAProxy: Set `timeout tunnel 86400s`
- AWS ALB: Set idle timeout to maximum (4000 seconds)

### CORS errors in browser console

**Symptoms:** Browser blocks API calls with "Access-Control-Allow-Origin" errors.

**Fix:** SENTINEL includes permissive CORS headers by default (`Access-Control-Allow-Origin: *`). If you see CORS errors:
1. Verify you are hitting the SENTINEL server directly, not a proxy that strips headers
2. Check that your reverse proxy forwards CORS headers

---

## Performance Tuning

### High memory usage

**Expected:** 50-80 MB idle, up to 150 MB under load.

**If higher:**
1. Check the number of SSE clients: each adds ~2 MB
2. Disable high-frequency providers (aviation at 30s is the biggest contributor)
3. Reduce the event buffer size
4. Check for memory leaks with Go's pprof:
   ```bash
   curl http://localhost:8080/debug/pprof/heap > heap.prof
   go tool pprof heap.prof
   ```

### High CPU usage

**Expected:** < 5% idle on a 4-core system.

**If higher:**
1. Check which providers are consuming the most time via `/api/providers/health`
2. Increase poll intervals for expensive providers
3. Disable the anomaly detector or correlation engine if not needed

### Slow API responses

1. Check database size: `ls -lh /path/to/sentinel.db`
2. Rebuild indexes:
   ```bash
   sqlite3 /path/to/sentinel.db "REINDEX; ANALYZE;"
   ```
3. Enable WAL mode if not already:
   ```bash
   sqlite3 /path/to/sentinel.db "PRAGMA journal_mode=WAL;"
   ```
4. Check for long-running queries in logs

---

## Log Locations

| Deployment | Log Location |
|-----------|-------------|
| Foreground | stdout/stderr |
| Systemd | `journalctl -u sentinel` |
| Docker | `docker logs sentinel` |
| File logging | `<data-dir>/sentinel.log` |
| Event log | `<data-dir>/events.ndjson` |

### Reading Logs

SENTINEL logs use a structured format:

```
2026/03/11 12:00:00 [storage] V3 migration complete
2026/03/11 12:00:00 Registered provider: usgs (interval: 1m0s, enabled: true)
2026/03/11 12:00:00 Starting SENTINEL server on localhost:8080
2026/03/11 12:00:00 Poller started with 21 providers
```

Key log prefixes:
- `[storage]` -- Database operations
- `[ALERT]` -- Alert rule triggers
- `EventStream:` -- SSE client connections
- `Registered provider:` -- Provider startup

### Increasing Log Verbosity

Set `log_level` to `"debug"` in config or use the `--debug` CLI flag (if available).

---

## FAQ

### Can I run multiple SENTINEL instances?

Yes, but each must use a separate data directory and port:
```bash
./sentinel --port 8080 --data-dir /data/instance1 &
./sentinel --port 8081 --data-dir /data/instance2 &
```

They cannot share the same SQLite database file.

### How do I reset everything?

Delete the data directory and restart:
```bash
rm -rf ~/.local/share/sentinel/
./sentinel
```

This removes all events, configuration state, and database. The config file in `~/.config/sentinel/` is separate and preserved.

### How do I add a custom provider?

Implement the `Provider` interface in `internal/provider/`:
```go
type Provider interface {
    Fetch(ctx context.Context) ([]*model.Event, error)
    Name() string
    Interval() time.Duration
    Enabled() bool
}
```

Register it in `cmd/sentinel/main.go` inside `initializePoller()`. See `docs/PROVIDERS.md` for details.

### Why is my database empty after restart?

Check that `--data-dir` points to the same directory. If unset, SENTINEL uses platform defaults. The database path is logged on startup:
```
Database: /home/user/.local/share/sentinel/sentinel.db
```

### How do I export event data?

Use the API:
```bash
# Export as JSON
curl "http://localhost:8080/api/events?limit=1000" > events.json

# Export specific category
curl "http://localhost:8080/api/events?category=earthquake&limit=1000" > earthquakes.json
```

### Does SENTINEL work offline?

The server itself runs fine offline, but providers cannot fetch new data without network access. Existing events in the database remain accessible.

### How do I disable specific providers?

Edit `config.json`:
```json
{
  "providers": {
    "opensky": { "enabled": false, "interval_seconds": 60 }
  }
}
```

Or use the web settings page at http://localhost:8080/settings.
