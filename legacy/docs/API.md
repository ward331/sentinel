# SENTINEL V3 API Reference

Base URL: `http://localhost:8080`

All responses are `application/json` unless otherwise noted. Timestamps use RFC 3339 format.

---

## Table of Contents

- [Health](#health)
- [Events](#events)
- [Providers](#providers)
- [Signal Board](#signal-board)
- [Configuration](#configuration)
- [News & Intelligence](#news--intelligence)
- [Financial](#financial)
- [Notifications](#notifications)
- [Alert Rules](#alert-rules)
- [Entity Search](#entity-search)
- [Correlations](#correlations)
- [Metrics](#metrics)

---

## Health

### `GET /api/health`

Returns server health status and uptime.

**Query Parameters:**

| Parameter  | Type   | Default | Description                          |
|------------|--------|---------|--------------------------------------|
| `detailed` | string | `""`    | Set to `"true"` for full health info |

**Response (simple):**

```json
{
  "status": "ok",
  "version": "v3.0.0",
  "timestamp": "2026-03-11T12:00:00Z",
  "uptime": 3600.5
}
```

**Response (detailed, `?detailed=true`):**

```json
{
  "status": "healthy",
  "uptime_seconds": 3600.5,
  "checks": {
    "database": { "status": "healthy", "latency_ms": 2 },
    "poller": { "status": "healthy", "active_providers": 21 }
  }
}
```

**Status Codes:**

| Code | Meaning       |
|------|---------------|
| 200  | Server is up  |

---

## Events

### `GET /api/events`

List events with filtering and pagination.

**Query Parameters:**

| Parameter          | Type   | Default | Description                                           |
|--------------------|--------|---------|-------------------------------------------------------|
| `limit`            | int    | 100     | Max results (1-1000)                                  |
| `offset`           | int    | 0       | Pagination offset                                     |
| `source`           | string | —       | Filter by source name (e.g. `usgs`, `gdacs`)          |
| `category`         | string | —       | Filter by category (e.g. `earthquake`, `aviation`)    |
| `severity`         | string | —       | Filter by severity: `low`, `medium`, `high`, `critical` |
| `min_magnitude`    | float  | —       | Minimum magnitude                                     |
| `max_magnitude`    | float  | —       | Maximum magnitude                                     |
| `q`                | string | —       | Full-text search query                                |
| `exclude_category` | string | —       | Exclude events of this category                       |
| `exclude_source`   | string | —       | Exclude events from this source                       |
| `start_time`       | string | —       | RFC 3339 start time                                   |
| `end_time`         | string | —       | RFC 3339 end time                                     |
| `bbox`             | string | —       | Bounding box: `minLon,minLat,maxLon,maxLat`           |
| `truth_score_min`  | int    | —       | Minimum truth score (1-5) (reserved, not yet wired)   |
| `country`          | string | —       | Country filter (reserved, not yet wired)              |

**Response:**

```json
{
  "events": [
    {
      "id": "a1b2c3d4-...",
      "title": "M 4.2 - 10 km NW of Ridgecrest, CA",
      "description": "Earthquake detected by USGS...",
      "source": "usgs",
      "source_id": "us7000abcd",
      "occurred_at": "2026-03-11T10:30:00Z",
      "ingested_at": "2026-03-11T10:30:05Z",
      "location": {
        "type": "Point",
        "coordinates": [-117.67, 35.63]
      },
      "precision": "exact",
      "magnitude": 4.2,
      "category": "earthquake",
      "severity": "medium",
      "metadata": {
        "depth_km": "10.5",
        "felt_reports": "42"
      },
      "badges": [
        { "label": "usgs", "type": "source", "timestamp": "2026-03-11T10:30:05Z" },
        { "label": "exact", "type": "precision", "timestamp": "2026-03-11T10:30:05Z" }
      ]
    }
  ],
  "total": 1,
  "limit": 100,
  "offset": 0
}
```

**Status Codes:**

| Code | Meaning              |
|------|----------------------|
| 200  | Success              |
| 500  | Database query error |

---

### `POST /api/events`

Create a new event manually.

**Request Body:**

```json
{
  "title": "Custom event title",
  "description": "Description of the event",
  "source": "manual",
  "source_id": "optional-dedup-key",
  "occurred_at": "2026-03-11T10:00:00Z",
  "location": {
    "type": "Point",
    "coordinates": [-73.935, 40.730]
  },
  "precision": "exact",
  "magnitude": 0,
  "category": "custom",
  "severity": "low",
  "metadata": {
    "key": "value"
  }
}
```

**Required Fields:** `title`, `description`, `source`, `occurred_at`, `location` (with `type` and `coordinates`), `precision`

**Precision Values:** `exact`, `polygon_area`, `approximate`, `text_inferred`, `unknown`

**Severity Values:** `low`, `medium`, `high`, `critical`

**Response:** `201 Created` with the full event object (same shape as GET).

**Status Codes:**

| Code | Meaning                 |
|------|-------------------------|
| 201  | Event created           |
| 400  | Validation error        |
| 500  | Storage error           |

---

### `GET /api/events/{id}`

Get a single event by ID.

**Response:** Full event object (same shape as list item).

**Status Codes:**

| Code | Meaning          |
|------|------------------|
| 200  | Success          |
| 400  | Missing ID       |
| 404  | Event not found  |
| 500  | Database error   |

---

### `POST /api/events/{id}/acknowledge`

Mark an event as acknowledged.

**Request Body:** None required.

**Response:**

```json
{
  "status": "acknowledged",
  "event_id": "a1b2c3d4-..."
}
```

**Status Codes:**

| Code | Meaning         |
|------|-----------------|
| 200  | Acknowledged    |
| 400  | Missing ID      |
| 404  | Event not found |
| 500  | Database error  |

---

### `GET /api/events/stream`

Server-Sent Events (SSE) stream of real-time events.

**Headers Sent:**

```
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive
```

**Event Types:**

| Type              | Description                           |
|-------------------|---------------------------------------|
| `new_event`       | A new event was ingested              |
| `event_update`    | An existing event was updated         |
| `correlation`     | A correlation flash was detected      |
| `signal_board`    | Signal board levels changed           |
| `provider_health` | Provider health status changed        |
| `anomaly`         | An anomaly was detected               |

**Example Stream:**

```
: connected

event: new_event
data: {"id":"a1b2...","title":"M 5.1 - Offshore Chile","source":"usgs",...}

event: new_event
data: {"id":"c3d4...","title":"Tropical Storm Warning","source":"noaa_nws",...}
```

**Connection:** Keep-alive. Reconnect on disconnect. The initial `: connected` comment confirms the SSE handshake.

---

## Providers

### `GET /api/providers`

List all registered data providers.

**Response:**

```json
{
  "providers": [
    {
      "name": "usgs",
      "interval_seconds": 60,
      "enabled": true
    },
    {
      "name": "gdacs",
      "interval_seconds": 60,
      "enabled": true
    }
  ],
  "total": 21
}
```

**Status Codes:**

| Code | Meaning                |
|------|------------------------|
| 200  | Success                |
| 503  | Poller not initialized |

---

### `GET /api/providers/health`

Get health statistics for all providers (success/failure counts, last run times).

**Response:**

```json
{
  "usgs": {
    "name": "usgs",
    "last_success": "2026-03-11T12:00:00Z",
    "last_error": null,
    "success_count": 120,
    "error_count": 2,
    "avg_latency_ms": 450
  }
}
```

**Status Codes:**

| Code | Meaning                      |
|------|------------------------------|
| 200  | Success                      |
| 503  | Health reporter not available |

---

### `GET /api/providers/healthy`

List providers currently in a healthy state.

**Response:**

```json
{
  "healthy_providers": ["usgs", "gdacs", "noaa_cap"],
  "count": 3
}
```

---

### `GET /api/providers/unhealthy`

List providers currently in an unhealthy state.

**Response:**

```json
{
  "unhealthy_providers": ["opensky"],
  "count": 1
}
```

---

### `GET /api/providers/stats`

Get statistics for a specific provider.

**Query Parameters:**

| Parameter | Type   | Description    |
|-----------|--------|----------------|
| (path)    | string | Provider name via URL path segment |

**Response:** Single provider stats object.

---

## Signal Board

### `GET /api/signal-board`

Returns DEFCON-style threat levels across five domains (0-5 scale).

**Response:**

```json
{
  "military": 1,
  "cyber": 2,
  "financial": 1,
  "natural": 1,
  "health": 0,
  "calculated_at": "2026-03-11T12:00:00Z"
}
```

**Domain Scale:**

| Level | Meaning             |
|-------|---------------------|
| 0     | Nominal / no threats |
| 1     | Low / routine        |
| 2     | Guarded / elevated   |
| 3     | Elevated / notable   |
| 4     | High / significant   |
| 5     | Critical / extreme   |

---

## Configuration

### `GET /api/config/ui`

Get UI feature flags and preferences.

**Response:**

```json
{
  "version": "3.0.0",
  "features": {
    "signal_board": true,
    "entity_tracking": true,
    "correlations": true,
    "news": true,
    "financial": true,
    "notifications": true,
    "alerts": true,
    "intel_briefing": true,
    "osint_resources": true
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
  }
}
```

---

### `GET /api/config`

Get the current configuration (secrets are redacted).

**Response:** Full config object with sensitive fields replaced by `"[REDACTED]"`.

---

### `POST /api/config`

Update configuration. Accepts a partial JSON object; only provided fields are updated.

**Request Body:** Partial config JSON.

**Response:**

```json
{
  "status": "success",
  "message": "Configuration updated"
}
```

**Status Codes:**

| Code | Meaning          |
|------|------------------|
| 200  | Updated          |
| 400  | Invalid JSON     |

---

## News & Intelligence

### `GET /api/news`

Get aggregated news items from RSS ingestion.

**Response:**

```json
{
  "items": [
    {
      "id": 1,
      "title": "Breaking: Major earthquake hits...",
      "url": "https://example.com/article",
      "description": "...",
      "source_name": "Reuters",
      "source_category": "news",
      "pub_date": "2026-03-11T10:00:00Z",
      "relevance_score": 8,
      "truth_score": 3
    }
  ],
  "total": 0
}
```

---

### `GET /api/intel/briefing`

Get AI-powered intelligence briefing summarizing current global situation.

**Response:**

```json
{
  "content": "SENTINEL Intelligence Briefing -- 2026-03-11\n\nNo significant events...",
  "generated_at": "2026-03-11T08:00:00Z",
  "type": "morning"
}
```

---

## Financial

### `GET /api/financial/overview`

Get financial market indicators snapshot.

**Response:**

```json
{
  "vix": 18.5,
  "btc_usd": 67500.00,
  "eth_usd": 3450.00,
  "oil_wti": 78.20,
  "gold": 2340.00,
  "yield_10y": 4.25,
  "yield_2y": 4.70,
  "curve_inverted": true,
  "fear_greed": 55,
  "timestamp": "2026-03-11T12:00:00Z"
}
```

---

## Notifications

### `GET /api/notifications/config`

Get notification channel status and configuration.

**Response:**

```json
{
  "telegram": { "enabled": false, "min_severity": "warning", "configured": false },
  "slack":    { "enabled": false, "min_severity": "warning", "configured": false },
  "discord":  { "enabled": false, "min_severity": "warning", "configured": false },
  "email":    { "enabled": false, "min_severity": "alert",   "configured": false },
  "ntfy":     { "enabled": false, "min_severity": "warning", "configured": false }
}
```

---

### `POST /api/notifications/config`

Update notification channel settings.

**Request Body:**

```json
{
  "telegram": {
    "enabled": true,
    "min_severity": "high"
  }
}
```

**Response:**

```json
{
  "status": "success",
  "message": "Notification config updated"
}
```

---

### `POST /api/notifications/test/{channel}`

Send a test notification through a specific channel.

**Path Parameters:**

| Parameter | Type   | Description                                                 |
|-----------|--------|-------------------------------------------------------------|
| `channel` | string | One of: `telegram`, `slack`, `discord`, `email`, `ntfy`, `pushover` |

**Response:**

```json
{
  "channel": "telegram",
  "status": "sent",
  "message": "Test notification dispatched to telegram"
}
```

**Status Codes:**

| Code | Meaning                    |
|------|----------------------------|
| 200  | Test sent                  |
| 400  | Unknown channel name       |

---

## Alert Rules

### `GET /api/alerts/rules`

List all alert rules.

**Response:**

```json
[
  {
    "id": "major-earthquake",
    "name": "Major Earthquake Alert",
    "description": "Alert for earthquakes magnitude 6.0 or higher",
    "enabled": true,
    "conditions": [
      { "field": "category", "operator": "equals", "value": "earthquake" },
      { "field": "magnitude", "operator": "gte", "value": 6.0 }
    ],
    "actions": [
      { "type": "log", "config": { "level": "warn" } }
    ],
    "created_at": "2026-03-11T00:00:00Z",
    "updated_at": "2026-03-11T00:00:00Z"
  }
]
```

---

### `POST /api/alerts/rules`

Create a new alert rule.

**Request Body:**

```json
{
  "id": "my-rule",
  "name": "High Severity Wildfire",
  "description": "Alert on high-severity wildfire events",
  "enabled": true,
  "conditions": [
    { "field": "category", "operator": "equals", "value": "wildfire" },
    { "field": "severity", "operator": "equals", "value": "high" }
  ],
  "actions": [
    { "type": "webhook", "config": { "url": "https://hooks.example.com/fire" } }
  ]
}
```

**Condition Fields:** `category`, `severity`, `source`, `magnitude`, `title`, or any metadata key.

**Condition Operators (string):** `equals`, `contains`, `starts_with`, `ends_with`

**Condition Operators (numeric):** `equals`, `gt`, `gte`, `lt`, `lte`

**Action Types:** `log`, `webhook`, `slack`, `discord`, `teams`, `email`

**Response:** `201 Created` with the rule object.

---

### `PUT /api/alerts/rules/{id}`

Update an existing alert rule.

**Request Body:** Full rule object (same shape as POST).

**Response:**

```json
{
  "status": "success",
  "message": "Rule updated",
  "id": "my-rule"
}
```

**Status Codes:**

| Code | Meaning        |
|------|----------------|
| 200  | Updated        |
| 400  | Invalid body   |
| 404  | Rule not found |
| 503  | Engine unavailable |

---

### `DELETE /api/alerts/rules/{id}`

Delete an alert rule.

**Response:** `204 No Content`

**Status Codes:**

| Code | Meaning        |
|------|----------------|
| 204  | Deleted        |
| 400  | Missing ID     |
| 404  | Rule not found |
| 503  | Engine unavailable |

---

## Entity Search

### `GET /api/entity/search`

Search for entities (aircraft, vessels, events) by keyword.

**Query Parameters:**

| Parameter | Type   | Required | Description          |
|-----------|--------|----------|----------------------|
| `q`       | string | Yes      | Search query string  |

**Response:**

```json
{
  "query": "Boeing 737",
  "results": [
    {
      "id": "a1b2c3d4-...",
      "type": "event",
      "name": "Boeing 737 squawk 7700 near LAX",
      "source": "airplanes_live",
      "lat": 33.94,
      "lon": -118.41,
      "last_seen": "2026-03-11T11:00:00Z"
    }
  ],
  "total": 1
}
```

**Entity Types:** `aircraft`, `vessel`, `satellite`, `event`

**Status Codes:**

| Code | Meaning                |
|------|------------------------|
| 200  | Success                |
| 400  | Missing `q` parameter  |
| 500  | Search failed          |

---

## Correlations

### `GET /api/correlations`

Get active correlation flashes (multi-source incident detections).

**Response:**

```json
{
  "correlations": [
    {
      "id": 1,
      "region_name": "Eastern Mediterranean",
      "lat": 34.5,
      "lon": 33.0,
      "radius_km": 50.0,
      "event_count": 7,
      "source_count": 3,
      "started_at": "2026-03-11T10:00:00Z",
      "last_event_at": "2026-03-11T10:45:00Z",
      "confirmed": false
    }
  ],
  "total": 1
}
```

A correlation flash fires when 3+ independent sources report events in the same geographic region within 60 minutes.

---

## Metrics

### `GET /api/metrics`

Get internal performance metrics.

**Response:**

```json
{
  "events_ingested": 4521,
  "events_broadcast": 4521,
  "api_requests": {
    "/api/events": { "count": 120, "avg_ms": 15 },
    "/api/health": { "count": 500, "avg_ms": 1 }
  },
  "alerts_triggered": 3,
  "alerts_processed": 3,
  "errors": {
    "/api/events": 0
  }
}
```

**Status Codes:**

| Code | Meaning              |
|------|----------------------|
| 200  | Success              |
| 503  | Metrics unavailable  |

---

## Common Error Format

All error responses return a plain-text or JSON error body:

```json
{"error": "description of the problem"}
```

Or plain text:

```
Failed to list events: database locked
```

---

## Authentication

Authentication is optional and disabled by default. When enabled via config:

- Send `Authorization: Bearer <api-key>` header
- The `/api/health` endpoint is always exempt from auth
- The `/api/events/stream` SSE endpoint is exempt from rate limiting

---

## Rate Limiting

When enabled (default: on), the rate limiter uses a token-bucket algorithm:

- **Default rate:** 100 requests/second
- **Burst:** 200 requests
- **Exempt paths:** `/api/health`, `/api/events/stream`
- Rate limit exceeded returns `429 Too Many Requests`

---

## CORS

All API endpoints return permissive CORS headers:

```
Access-Control-Allow-Origin: *
Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS
Access-Control-Allow-Headers: Content-Type, Authorization
```
