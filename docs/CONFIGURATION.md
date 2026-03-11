# SENTINEL V3 Configuration Guide

SENTINEL uses a single JSON configuration file. On first launch, defaults are applied and the setup wizard can be run interactively.

---

## Config File Location

The config file path is platform-specific:

| Platform | Default Path |
|----------|-------------|
| Linux    | `~/.config/sentinel/config.json` |
| macOS    | `~/Library/Application Support/SENTINEL/config.json` |
| Windows  | `%APPDATA%\SENTINEL\config.json` |

Override with: `sentinel --config /path/to/config.json`

---

## Data Directory

The data directory holds the SQLite database, backups, event logs, and encryption keys:

| Platform | Default Path |
|----------|-------------|
| Linux    | `~/.local/share/sentinel/` |
| macOS    | `~/Library/Application Support/SENTINEL/data/` |
| Windows  | `%APPDATA%\SENTINEL\data\` |

Override with: `sentinel --data-dir /path/to/data`

Contents of the data directory:

```
sentinel/
  sentinel.db        # SQLite database
  events.ndjson      # NDJSON event log
  sentinel.log       # Application log
  backups/           # Automatic backups
  keys/              # Encryption keys
```

---

## CLI Flags

| Flag              | Default      | Description                          |
|-------------------|-------------|--------------------------------------|
| `--config`        | (platform)  | Path to config file                  |
| `--data-dir`      | (platform)  | Data directory override              |
| `--port`          | `8080`      | HTTP server port                     |
| `--host`          | `localhost` | Bind address                         |
| `--version`       | —           | Print version and exit               |
| `--wizard`        | —           | Run setup wizard                     |
| `--no-frontend`   | `false`     | API only, do not serve embedded web  |

CLI flags override config file values.

---

## Configuration Fields

### Top-Level

| Field             | Type   | Default    | Description                       |
|-------------------|--------|------------|-----------------------------------|
| `version`         | string | `"2.0.0"` | Config schema version             |
| `setup_complete`  | bool   | `false`    | Whether first-run wizard has run  |
| `data_dir`        | string | (platform) | Data directory path               |
| `log_level`       | string | `"info"`   | Log level: debug, info, warn, error |
| `auto_open_browser` | bool | `true`     | Open dashboard on launch          |
| `check_for_updates` | bool | `true`     | Check for new versions on startup |
| `cesium_token`    | string | `""`       | Cesium Ion access token (for 3D globe) |

### Server

| Field               | Type   | Default      | Description                |
|---------------------|--------|-------------|----------------------------|
| `server.port`       | int    | `8080`      | HTTP port                  |
| `server.host`       | string | `"0.0.0.0"` | Bind address               |
| `server.tls_enabled`| bool   | `false`      | Enable HTTPS               |
| `server.tls_cert`   | string | `""`         | TLS certificate path       |
| `server.tls_key`    | string | `""`         | TLS private key path       |
| `server.auth_enabled`| bool  | `false`      | Require API key auth       |
| `server.auth_token` | string | `""`         | Bearer token for API auth  |
| `server.dashboard_password` | string | `""` | Password for web dashboard |

### Providers

Each provider has:

| Field              | Type | Default | Description           |
|--------------------|------|---------|----------------------|
| `enabled`          | bool | varies  | Enable/disable provider |
| `interval_seconds` | int  | varies  | Polling interval      |
| `options`          | map  | `{}`    | Provider-specific options |

Example:

```json
{
  "providers": {
    "usgs": { "enabled": true, "interval_seconds": 60 },
    "gdacs": { "enabled": true, "interval_seconds": 60 },
    "opensky": { "enabled": true, "interval_seconds": 60 },
    "noaa_cap": { "enabled": true, "interval_seconds": 300 },
    "openmeteo": { "enabled": true, "interval_seconds": 600 },
    "gdelt": { "enabled": true, "interval_seconds": 900 },
    "celestrak": { "enabled": true, "interval_seconds": 21600 },
    "swpc": { "enabled": true, "interval_seconds": 60 },
    "who": { "enabled": true, "interval_seconds": 3600 },
    "promed": { "enabled": true, "interval_seconds": 1800 },
    "airplanes_live": { "enabled": true, "interval_seconds": 30 },
    "nasa_firms": { "enabled": true, "interval_seconds": 1800 },
    "piracy_imb": { "enabled": true, "interval_seconds": 3600 },
    "israel_alerts": { "enabled": true, "interval_seconds": 5 },
    "reliefweb": { "enabled": true, "interval_seconds": 600 },
    "iran_conflict": { "enabled": true, "interval_seconds": 900 },
    "isw": { "enabled": true, "interval_seconds": 1800 }
  }
}
```

See `docs/PROVIDERS.md` for the full list and what each provider does.

### API Keys

For Tier 1 providers that need an API key:

```json
{
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
  }
}
```

Keys stored in the config file can be encrypted. See [Secret Encryption](#secret-encryption) below.

---

## Notification Channels

### Telegram

```json
{
  "telegram": {
    "enabled": true,
    "bot_token": "123456:ABC-DEF",
    "chat_id": "-1001234567890",
    "min_severity": "warning",
    "digest_mode": false,
    "digest_interval_minutes": 60
  }
}
```

**Setup:**
1. Create a bot via [@BotFather](https://t.me/botfather)
2. Get the bot token
3. Add the bot to your group/channel
4. Get the chat ID (send a message, then check `https://api.telegram.org/bot<TOKEN>/getUpdates`)

### Slack

```json
{
  "slack": {
    "enabled": true,
    "webhook_url": "https://hooks.slack.com/services/T.../B.../...",
    "channel": "#sentinel-alerts",
    "min_severity": "warning"
  }
}
```

**Setup:**
1. Go to https://api.slack.com/apps and create an app
2. Enable Incoming Webhooks
3. Create a webhook for your channel
4. Copy the webhook URL

### Discord

```json
{
  "discord": {
    "enabled": true,
    "webhook_url": "https://discord.com/api/webhooks/...",
    "min_severity": "warning"
  }
}
```

**Setup:**
1. In your Discord server, go to channel Settings > Integrations > Webhooks
2. Create a webhook and copy the URL

### Email (SMTP)

```json
{
  "email": {
    "enabled": true,
    "method": "smtp",
    "smtp_host": "smtp.gmail.com",
    "smtp_port": 587,
    "smtp_tls": "starttls",
    "username": "you@gmail.com",
    "password_encrypted": "...",
    "from_address": "sentinel@yourdomain.com",
    "to_addresses": ["admin@yourdomain.com"],
    "min_severity": "alert"
  }
}
```

Supported methods: `smtp`, `gmail` (OAuth2), `sendgrid`, `mailgun`.

### ntfy

```json
{
  "ntfy": {
    "enabled": true,
    "server": "https://ntfy.sh",
    "topic": "my-sentinel-alerts",
    "min_severity": "warning"
  }
}
```

**Setup:**
1. Pick a unique topic name at https://ntfy.sh
2. Subscribe on your phone/desktop
3. No signup required for the public server

### Pushover

```json
{
  "pushover": {
    "enabled": true,
    "app_token": "your-app-token",
    "user_key": "your-user-key"
  }
}
```

**Setup:**
1. Register at https://pushover.net
2. Create an application
3. Copy the app token and your user key

---

## Secret Encryption

SENTINEL uses AES-256-GCM to encrypt sensitive fields (API keys, passwords, tokens) stored in `config.json`.

### How It Works

1. A 256-bit encryption key is stored in `<config-dir>/sentinel.key`
2. Sensitive values in config are stored as base64-encoded AES-256-GCM ciphertext
3. Fields ending in `_encrypted` (e.g. `password_encrypted`, `sendgrid_key_encrypted`) use this encryption
4. The key file has `0600` permissions (owner-only read/write)

### Generating the Key

The encryption key is automatically generated during the first-run wizard. To generate manually:

```bash
# The wizard generates the key automatically:
sentinel --wizard

# Or it is created when the application first needs to encrypt a value
```

The key file is stored at:
- Linux: `~/.config/sentinel/sentinel.key`
- macOS: `~/Library/Application Support/SENTINEL/sentinel.key`
- Windows: `%APPDATA%\SENTINEL\sentinel.key`

### Security Notes

- Never commit `sentinel.key` to version control
- Back up the key file separately from the config
- If the key is lost, encrypted values must be re-entered
- The key is base64-encoded on disk

---

## Signal Board Configuration

```json
{
  "signal_board": {
    "enabled": true
  }
}
```

The Signal Board provides a DEFCON-style threat posture display across five domains: military, cyber, financial, natural, and health. Each domain is rated 0-5.

---

## Entity Tracking Configuration

```json
{
  "entity_tracking": {
    "enabled": true,
    "dead_reckoning_mins": 30
  }
}
```

| Field                | Type | Default | Description                                        |
|----------------------|------|---------|----------------------------------------------------|
| `enabled`            | bool | `false` | Enable aircraft/vessel dead-reckoning projections  |
| `dead_reckoning_mins`| int  | `30`    | Minutes to project position after signal loss      |

---

## Location Configuration

Set your home location for proximity alerts:

```json
{
  "location": {
    "lat": 40.7128,
    "lon": -74.0060,
    "radius_km": 100,
    "timezone": "America/New_York",
    "set": true
  }
}
```

---

## UI Preferences

```json
{
  "ui": {
    "default_view": "globe",
    "default_preset": "Global Watch",
    "data_retention_days": 30,
    "sound_enabled": true,
    "sound_volume": 75,
    "ticker_enabled": true,
    "ticker_speed": "medium",
    "ticker_min_severity": "warning"
  }
}
```

| Field                | Type   | Default          | Description                         |
|----------------------|--------|------------------|-------------------------------------|
| `default_view`       | string | `"globe"`        | Initial map view (globe/map/list)   |
| `default_preset`     | string | `"Global Watch"` | Default filter preset               |
| `data_retention_days`| int    | `30`             | Days before old events are purged   |
| `sound_enabled`      | bool   | `true`           | Play alert sounds                   |
| `sound_volume`       | int    | `75`             | Alert volume (0-100)                |
| `ticker_enabled`     | bool   | `true`           | Show scrolling event ticker         |
| `ticker_speed`       | string | `"medium"`       | Ticker scroll speed                 |
| `ticker_min_severity`| string | `"warning"`      | Minimum severity for ticker display |

---

## Morning Briefing

```json
{
  "morning_briefing": {
    "enabled": true,
    "time_utc": "08:00",
    "delivery": ["telegram", "email"],
    "include_events": true,
    "include_conflicts": true,
    "include_space_weather": true,
    "include_financial": true,
    "include_iss_passes": true,
    "include_news": true
  }
}
```

---

## Weekly Digest

```json
{
  "weekly_digest": {
    "enabled": true,
    "day": "sunday",
    "time_utc": "08:00",
    "delivery": ["email"]
  }
}
```

---

## Notification Rules and Geofences

```json
{
  "notifications": {
    "rules": [
      {
        "id": "quake-alert",
        "name": "Large Earthquake Near Home",
        "enabled": true,
        "condition": {
          "category": "earthquake",
          "magnitude_gte": 5.0
        },
        "actions": ["telegram", "pushover"]
      }
    ],
    "geofences": [
      {
        "id": "home-area",
        "name": "Home 100km Radius",
        "enabled": true,
        "type": "circle",
        "center_lat": 40.7128,
        "center_lon": -74.0060,
        "radius_km": 100
      }
    ]
  }
}
```

---

## Environment Variable Overrides

While the JSON config file is the primary configuration method, some settings can be overridden with environment variables for container/CI deployments:

| Variable            | Maps To                     |
|--------------------|-----------------------------|
| `SENTINEL_PORT`    | `server.port`               |
| `SENTINEL_HOST`    | `server.host`               |
| `SENTINEL_DATA_DIR`| `data_dir`                  |
| `SENTINEL_DB_PATH` | Database file path          |
| `SENTINEL_LOG_LEVEL`| `log_level`                |

CLI flags take precedence over environment variables, which take precedence over config file values.

---

## Full Example Config

A complete config file with all defaults is generated on first run. You can export the current config (with secrets redacted) via:

```bash
sentinel --export-config
```

Or via the API:

```bash
curl http://localhost:8080/api/config
```
