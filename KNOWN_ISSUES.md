# SENTINEL V3 Known Issues

## High Severity

- **[HIGH] Intel briefing is a placeholder.** The `GET /api/intel/briefing` endpoint returns static text. LLM integration for AI-generated briefings is not yet wired.

- **[HIGH] News endpoint returns empty.** The `GET /api/news` endpoint always returns `{"items":[], "total":0}`. RSS news ingestion pipeline is stubbed but not yet connected to the `news_items` table.

- **[HIGH] Financial overview returns static data.** The `GET /api/financial/overview` endpoint returns hardcoded placeholder values. The financial markets provider is registered but not yet wired to the API response.

## Medium Severity

- **[MED] Signal Board returns static threat levels.** The signal board engine is implemented but the calculation is not yet connected to real event data. Returns hardcoded defaults.

- **[MED] Notification dispatch is partially stubbed.** The `POST /api/notifications/test/{channel}` endpoint logs the intent but does not actually send through all channels. Telegram, Slack, Discord, and email dispatchers exist but are not fully integrated with the notification config API.

- **[MED] POST /api/notifications/config does not persist.** Updates are accepted but not saved to the config file. Restarting the server resets notification settings.

- **[MED] Alert rules are in-memory only.** Rules created via `POST /api/alerts/rules` are lost on server restart. The `alert_rules` table exists but the engine does not yet read/write from it.

- **[MED] Correlation engine not scheduled.** The correlation engine is initialized but not yet running on a periodic schedule. The `/api/correlations` endpoint queries the database table directly (which may be empty).

- **[MED] Truth score calculator not scheduled.** The truth score calculator is initialized but not running. All events default to `truth_score=1`.

- **[MED] Anomaly detector not scheduled.** The anomaly detector is initialized but not running. The `anomalies` table will remain empty.

- **[MED] `truth_score_min` and `country` query parameters on events are parsed but ignored.** They are accepted by the API but not passed to the storage query.

## Low Severity

- **[LOW] OpenSky Enhanced provider not registered.** The enhanced OpenSky provider (with Bellingcat aircraft database) requires the aircraft database to be initialized separately. The basic OpenSky provider is registered instead.

- **[LOW] Some Tier 1 providers require manual key injection.** OpenSanctions and Global Fishing Watch take API keys via constructor, not from the config `keys` section.

- **[LOW] Setup wizard runs non-interactively in containers.** The terminal wizard requires stdin, which may not be available in Docker. Use `--config` with a pre-created config file instead.

- **[LOW] System tray icon requires a desktop environment.** On headless servers, the systray package may log warnings. This is harmless.

- **[LOW] `--export-config` and `--check-config` flags are documented but not yet implemented.** Use `GET /api/config` instead.

- **[LOW] Event log rotation endpoint has no automatic scheduling.** `POST /api/events/log/rotate` exists but must be called manually or via cron.

## TODO Items

- Wire the intelligence engines (correlation, truth, anomaly, signal board) into the main scheduler loop
- Connect financial markets provider output to `/api/financial/overview`
- Implement news RSS ingestion into `news_items` table
- Persist alert rules to SQLite
- Persist notification config changes to disk
- Add `truth_score_min` and `country` to storage query filters
- Implement `--export-config` and `--check-config` CLI flags
- Add Prometheus-compatible `/metrics` endpoint format
- Implement WebSocket support as an alternative to SSE
- Add automatic event log rotation schedule
