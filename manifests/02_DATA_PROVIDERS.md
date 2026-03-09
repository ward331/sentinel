# MANIFEST 02 — DATA PROVIDERS (ALL SOURCES)
# ============================================
# Covers: Stage 3 — All zero-key and optional key data providers
# Existing providers are documented here for reference.
# New providers are clearly marked NEW.

════════════════════════════════════════════════════════════════
ZERO-KEY PROVIDERS (enable on first run, no config needed)
════════════════════════════════════════════════════════════════

NATURAL DISASTERS:
  USGS Earthquakes
    URL: https://earthquake.usgs.gov/earthquakes/feed/v1.0/summary/all_hour.geojson
    Interval: 60s | Status: ALREADY IMPLEMENTED
    Events: magnitude, depth, felt reports, USGS event page link

  GDACS Multi-Hazard
    URL: https://www.gdacs.org/xml/rss.xml
    Interval: 60s | Status: ALREADY IMPLEMENTED
    Events: storms, floods, earthquakes, volcanic, droughts

  NOAA CAP Alerts
    URL: https://alerts.weather.gov/cap/us.php?x=1
    Interval: 300s | Status: NEW — implement
    Events: US weather alerts, tornado warnings, flash floods

  Pacific Tsunami Warning Center
    URL: https://ptwc.weather.gov/ptwc/ptwc.php?type=rss
    Interval: 120s | Status: NEW
    Events: tsunami watches, warnings, bulletins

  Volcano Discovery
    URL: https://www.volcanodiscovery.com/earthquakes/rss/large_quakes_worldwide.rss
    Interval: 300s | Status: NEW
    Events: volcanic activity reports

WEATHER:
  Open-Meteo (completely free, no key ever)
    URL: https://api.open-meteo.com/v1/forecast?latitude={lat}&longitude={lon}&hourly=temperature_2m,precipitation,windspeed_10m&current_weather=true
    Interval: 600s | Status: NEW
    Events: current weather at configured location
    Note: Use cfg.Location.Lat/Lon; if not set, skip silently

  NOAA NWS API
    URL: https://api.weather.gov/alerts/active
    Interval: 300s | Status: NEW
    Events: active US weather alerts with polygon areas
    Headers: User-Agent required — use "SENTINEL/2.0 contact@example.com"

MILITARY AVIATION (zero key):
  Airplanes.live (primary — unfiltered MLAT)
    URL: https://api.airplanes.live/v2/mil
    Interval: 30s | Status: NEW
    Events: military aircraft positions, type, callsign, squawk
    Better coverage than OpenSky for military

  ADSB.one (fallback)
    URL: https://api.adsb.one/v2/mil
    Interval: 30s | Status: NEW
    Use if Airplanes.live returns error

  OpenSky Network
    URL: https://opensky-network.org/api/states/all
    Interval: 60s | Status: ALREADY IMPLEMENTED
    Keep as tertiary fallback

  Alert logic:
    Squawk 7700 (emergency): TIER 3 ALERT
    Squawk 7600 (radio failure): TIER 2 WARNING
    Squawk 7500 (hijack): TIER 4 CRITICAL
    Unusual pattern detection: racetrack near conflict zone → TIER 2

CONFLICT/OSINT:
  GDELT GKG
    URL: http://data.gdeltproject.org/gdeltv2/lastupdate.txt → parse CSV link
    Interval: 900s | Status: NEW
    Filter CAMEO codes: 18x (assault), 19x (fight), 20x (mass violence)
    Parse: actor, location, date, source URL, goldstein scale

  ReliefWeb Disasters
    URL: https://api.reliefweb.int/v1/disasters?appname=sentinel&limit=50
    Interval: 600s | Status: NEW
    Events: humanitarian disasters, crisis declarations

MISSILE / AIR RAID ALERTS:
  Israel Pikud HaOref — Active alerts
    URL: https://www.oref.org.il/WarningMessages/alert/alerts.json
    Interval: 5s | Status: NEW — HIGH PRIORITY
    Events: current active air raid alerts by city
    Alert on any response: TIER 4 CRITICAL

  Israel Pikud HaOref — Alert history
    URL: https://www.oref.org.il/Shared/Ajax/GetAlarmsHistory.aspx
    Interval: 30s | Status: NEW
    Events: recent alert history for context

  Ukraine Alerts (zero-key baseline)
    URL: https://api.alerts.in.ua/v1/alerts/active.json
    Interval: 10s | Status: NEW
    Headers: X-API-Key header required — but if /etc/ukrainealerts.key absent,
             use public RSS: https://alerts.in.ua/feed.xml (300s)
    Events: oblast-level air raid alerts, all-clear
    Alert on active: TIER 4 CRITICAL

SATELLITES:
  CelesTrak Active Satellites
    URL: https://celestrak.org/NORAD/elements/gp.php?GROUP=active&FORMAT=json
    Interval: 21600s (6 hours) | Status: NEW
    ~8000 objects, TLE elements for SGP4 propagation

  CelesTrak Stations (ISS etc)
    URL: https://celestrak.org/NORAD/elements/gp.php?GROUP=stations&FORMAT=json
    Interval: 21600s | Status: NEW

  CelesTrak Starlink
    URL: https://celestrak.org/NORAD/elements/gp.php?GROUP=starlink&FORMAT=json
    Interval: 21600s | Status: NEW

  CelesTrak GPS Ops
    URL: https://celestrak.org/NORAD/elements/gp.php?GROUP=gps-ops&FORMAT=json
    Interval: 21600s | Status: NEW

  Satellite position calculation:
    SGP4 propagation in Go — recalculate positions every 10s in memory
    Do NOT store positions in DB (too many rows)
    Only store conjunction alerts and reentry predictions

SPACE WEATHER (all NOAA SWPC, zero key):
  Solar Wind: https://services.swpc.noaa.gov/products/solar-wind/mag-1-day.json
  Kp Index:   https://services.swpc.noaa.gov/json/planetary_k_index_1m.json
  Alerts:     https://services.swpc.noaa.gov/products/alerts.json
  GOES X-ray: https://services.swpc.noaa.gov/json/goes/primary/xrays-1-day.json
  Forecast:   https://services.swpc.noaa.gov/products/3-day-forecast.json
  All: Interval 60s | Status: NEW

  Alert thresholds:
    Kp >= 5: TIER 1 WATCH
    Kp >= 7: TIER 2 WARNING
    Kp >= 9: TIER 3 ALERT (rare — major geomagnetic storm)
    X-class flare: TIER 2 WARNING
    M5+ flare: TIER 1 WATCH

DISEASE / OUTBREAK:
  WHO Disease Outbreak News
    URL: https://www.who.int/api/news/diseaseoutbreaknews
    Interval: 3600s | Status: NEW

  ProMED RSS
    URL: https://www.promedmail.org/promed/rss
    Interval: 1800s | Status: NEW

WILDFIRE:
  NASA FIRMS (no key for basic, 24hr CSV)
    URL: https://firms.modaps.eosdis.nasa.gov/active_fire/noaa-20-viirs-c2/csv/J1_VIIRS_C2_Global_24h.csv
    Interval: 1800s | Status: NEW
    Parse: lat, lon, brightness, confidence, satellite, acq_date, acq_time
    Alert threshold: confidence >= 75%

MARITIME PIRACY:
  ICC IMB Piracy Reporting Centre
    URL: https://www.icc-ccs.org/piracy-reporting-centre/live-piracy-report/rss
    Interval: 3600s | Status: NEW

  UKMTO Maritime Security
    URL: https://www.maritimeglobalsecurity.org/feed/
    Interval: 1800s | Status: NEW

FINANCIAL (zero-key baseline — see MANIFEST 03 for full detail):
  CBOE VIX History
    URL: https://cdn.cboe.com/api/global/us_indices/daily_prices/VIX_History.csv
    Interval: 60s | Status: NEW (handled in manifest 03)

  CoinGecko Crypto (no key ever)
    URL: https://api.coingecko.com/api/v3/simple/price?ids=bitcoin,ethereum&vs_currencies=usd&include_24hr_change=true
    Interval: 60s | Status: NEW (handled in manifest 03)

  US Treasury Yields (free)
    URL: https://home.treasury.gov/resource-center/data-chart-center/interest-rates/pages/xml
    Interval: 3600s | Status: NEW (handled in manifest 03)

  OFAC SDN List (sanctions)
    URL: https://www.treasury.gov/ofac/downloads/sdnlist.txt
    Interval: 3600s | Status: NEW (handled in manifest 03)

════════════════════════════════════════════════════════════════
OPTIONAL KEY-BASED PROVIDERS
════════════════════════════════════════════════════════════════

Check cfg.Keys for each. If key empty: skip provider, log "Provider X disabled (no API key)".
In settings, each provider shows: ● Active / ○ No Key / ✕ Error

  ADS-B Exchange
    Key: cfg.Keys["adsbexchange"]
    URL: https://adsbexchange.com/api/aircraft/v2/mil/
    Interval: 30s
    Adds: unfiltered military MLAT + ADS-B (better than airplanes.live)
    Free signup: rapidapi.com/adsbexchange

  AISStream.io (ship tracking WebSocket)
    Key: cfg.Keys["aisstream"]
    URL: wss://stream.aisstream.io/v0/stream
    Adds: real-time AIS vessel positions globally
    Free signup: aisstream.io

  ACLED (conflict data)
    Key: cfg.Keys["acled"]
    URL: https://api.acleddata.com/acled/read?key={KEY}&limit=500&event_date={DATE}
    Interval: 1800s
    Adds: conflict events with actor names, fatalities, event type
    Free signup: acleddata.com

  OpenWeatherMap
    Key: cfg.Keys["openweather"]
    URL: https://api.openweathermap.org/data/2.5/weather + tile layers
    Interval: 300s
    Adds: weather map tile layers on globe, current conditions worldwide
    Free tier: 60 req/min, 1M req/month
    Free signup: openweathermap.org/api

  NASA (FIRMS real-time)
    Key: cfg.Keys["nasa"]
    URL: https://firms.modaps.eosdis.nasa.gov/mapserver/mapkey_status/?MAP_KEY={KEY}
    Adds: upgrades FIRMS from 24h CSV to real-time streaming (1h latency)
    Free signup: firms.modaps.eosdis.nasa.gov/api

  Space-Track.org (military TLEs)
    Key: cfg.Keys["spacetrack"] (format: "username:password")
    URL: https://www.space-track.org/basicspacedata/
    Adds: classified/military satellite TLEs not on CelesTrak
    Free signup: space-track.org/documentation#register

  Ukraine Alerts Full
    Key: cfg.Keys["ukrainealerts"]
    URL: https://api.alerts.in.ua/v1/alerts/active.json + SSE stream
    Adds: real-time SSE stream (vs polling), oblast-level granularity
    Free signup: devs.alerts.in.ua

  Alpha Vantage (financial)
    Key: cfg.Keys["alpha_vantage"]
    Free tier: 25 req/day
    Free signup: alphavantage.co
    See MANIFEST 03 for usage

  Finnhub (financial)
    Key: cfg.Keys["finnhub"]
    Free tier: 60 req/min
    Free signup: finnhub.io
    See MANIFEST 03 for usage

  FRED (Federal Reserve Economic Data)
    Key: cfg.Keys["fred"]
    Free: unlimited
    Free signup: fred.stlouisfed.org/docs/api/api_key.html
    See MANIFEST 03 for usage

  Polygon.io (financial)
    Key: cfg.Keys["polygon"]
    Free tier: 5 req/min, EOD data only
    Free signup: polygon.io
    See MANIFEST 03 for usage

════════════════════════════════════════════════════════════════
PROVIDER HEALTH SYSTEM
════════════════════════════════════════════════════════════════

internal/providers/health.go tracks per-provider:
  LastFetch      time.Time
  LastSuccess    time.Time
  ErrorStreak    int
  EventsPerHour  float64
  Status         string  // "active", "degraded", "failed", "disabled", "no_key"
  RateLimit      RateLimitInfo

GET /api/providers/health returns all provider statuses.

Alert if ErrorStreak >= 3: log warning "Provider X failing: {last_error}"
Alert if ErrorStreak >= 10: SENTINEL system alert (TIER 1) shown in feed:
  "⚠️ Data provider {name} has been unavailable for {duration}"

Display in Settings → Providers tab:
  Each provider: status dot + name + last seen + events/hr + [Force Refresh] [Disable]
  Rate limit bar if applicable
  Error message if ErrorStreak > 0

════════════════════════════════════════════════════════════════
PROVIDER SETUP IN settings.html — [API KEYS] TAB
════════════════════════════════════════════════════════════════

Table columns: Source | Category | What it adds | Free tier | Key (masked) | Status | Actions

Each row:
  [Get Free Key →]  link to signup page (opens new tab)
  Key input field   (reveals on [Edit])
  [Test]            calls actual API with provided key, shows ✅/❌ + quota info
  [Save]            encrypts + stores
  [Delete]          removes key

Pre-populate signup URLs in code — these are stable URLs:
  adsbexchange:   https://rapidapi.com/adsbexchange
  aisstream:      https://aisstream.io
  acled:          https://developer.acleddata.com
  openweather:    https://home.openweathermap.org/users/sign_up
  nasa:           https://firms.modaps.eosdis.nasa.gov/api/area/
  spacetrack:     https://www.space-track.org/documentation#register
  ukrainealerts:  https://devs.alerts.in.ua
  alpha_vantage:  https://www.alphavantage.co/support/#api-key
  finnhub:        https://finnhub.io/register
  fred:           https://fred.stlouisfed.org/docs/api/api_key.html
  polygon:        https://polygon.io/signup
