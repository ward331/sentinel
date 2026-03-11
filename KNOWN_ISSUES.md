# SENTINEL V3 Known Issues

## Medium Severity

- **[MED] Financial overview may be partially wired.** The `GET /api/financial/overview` endpoint may still return incomplete data. The financial markets provider is registered but full integration is in progress (another agent is working on this).

- **[MED] `truth_score_min` and `country` query parameters on events are parsed but ignored.** They are accepted by the API but not passed to the storage query. Another agent is working on this.

- **[MED] `--export-config` and `--check-config` flags are documented but not yet implemented.** Use `GET /api/config` instead. Another agent is working on this.

- **[MED] Event log rotation endpoint has no automatic scheduling.** `POST /api/events/log/rotate` exists but must be called manually or via cron. Another agent is working on this.

## Low Severity

- **[LOW] OpenSky Enhanced provider not registered.** The enhanced OpenSky provider (with Bellingcat aircraft database) requires the aircraft database to be initialized separately. The basic OpenSky provider is registered instead.

- **[LOW] Some Tier 1 providers require manual key injection.** OpenSanctions and Global Fishing Watch take API keys via constructor, not from the config `keys` section.

- **[LOW] Setup wizard runs non-interactively in containers.** The terminal wizard requires stdin, which may not be available in Docker. Use `--config` with a pre-created config file instead.

## TODO Items

- Add Prometheus-compatible `/metrics` endpoint format
- Implement WebSocket support as an alternative to SSE
