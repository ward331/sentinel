# SENTINEL V3 Provider Catalog

SENTINEL ships with 33+ data providers across two tiers. Tier 0 providers require zero API keys and work out of the box. Tier 1 providers require a free API key signup.

---

## Provider Summary

| # | Provider | Category | Tier | Default Interval | Data Format |
|---|----------|----------|------|------------------|-------------|
| 1 | USGS | Natural Disaster | 0 | 60s | GeoJSON |
| 2 | GDACS | Natural Disaster | 0 | 60s | JSON |
| 3 | NOAA CAP | Natural Disaster | 0 | 300s (5m) | XML/CAP |
| 4 | NOAA NWS | Weather | 0 | 300s (5m) | JSON |
| 5 | Tsunami (NOAA NTWC) | Natural Disaster | 0 | 300s (5m) | JSON/RSS |
| 6 | Volcano (Smithsonian GVP) | Natural Disaster | 0 | 300s (5m) | JSON |
| 7 | ReliefWeb (UN OCHA) | Humanitarian | 0 | 600s (10m) | JSON |
| 8 | OpenSky Network | Aviation | 0 | 60s | JSON |
| 9 | Airplanes.live | Aviation | 0 | 30s | JSON |
| 10 | ADSB.one | Aviation | 0 | 30s | JSON |
| 11 | Open-Meteo | Weather | 0 | 600s (10m) | JSON |
| 12 | GDELT | Conflict/OSINT | 0 | 900s (15m) | JSON |
| 13 | LiveUAMap | Conflict/OSINT | 0 | 300s (5m) | JSON |
| 14 | Iran Conflict OSINT | Conflict/OSINT | 0 | 900s (15m) | JSON |
| 15 | ISW (Institute for the Study of War) | Conflict/OSINT | 0 | 1800s (30m) | RSS/XML |
| 16 | CelesTrak | Space/Satellite | 0 | 21600s (6h) | TLE/JSON |
| 17 | SWPC (NOAA Space Weather) | Space/Weather | 0 | 60s | JSON |
| 18 | WHO Disease Outbreak News | Health | 0 | 3600s (1h) | RSS/XML |
| 19 | ProMED | Health | 0 | 1800s (30m) | RSS/XML |
| 20 | NASA FIRMS | Environmental | 0 | 1800s (30m) | CSV/JSON |
| 21 | Piracy IMB | Maritime Security | 0 | 3600s (1h) | RSS/XML |
| 22 | Financial Markets | Financial | 0 | 60s | JSON |
| 23 | Global Forest Watch | Environmental | 0 | 3600s (1h) | JSON |
| 24 | SEC EDGAR | Financial | 0 | 900s (15m) | JSON/RSS |
| 25 | Bellingcat ADS-B DB | Aviation (enrichment) | 0 | Monthly | CSV |
| 26 | CISA KEV | Cyber Security | 0 | 3600s (1h) | JSON |
| 27 | OTX AlienVault | Cyber Security | 0 | 3600s (1h) | JSON |
| 28 | Ukraine Alerts | Conflict | 0 | 60s | JSON |
| 29 | Pikud HaOref | Conflict | 0 | 5s | JSON |
| 30 | UKMTO | Maritime Security | 0 | 3600s (1h) | RSS/XML |
| 31 | ADS-B Exchange | Aviation | 1 | 60s | JSON |
| 32 | AISStream | Maritime | 1 | 30s | WebSocket/JSON |
| 33 | ACLED | Conflict | 1 | 3600s (1h) | JSON |
| 34 | OpenWeatherMap | Weather | 1 | 600s (10m) | JSON |
| 35 | OpenSanctions | Financial/Security | 1 | 3600s (1h) | JSON |
| 36 | Global Fishing Watch | Maritime/Environmental | 1 | 3600s (1h) | JSON |
| 37 | NASA FIRMS RT | Environmental | 1 | 600s (10m) | JSON |

---

## Tier 0 Providers (Zero Key Required)

### USGS Earthquake Hazards

- **Name:** `usgs`
- **Category:** Natural Disaster
- **Endpoint:** `https://earthquake.usgs.gov/earthquakes/feed/v1.0/summary/all_hour.geojson`
- **Format:** GeoJSON FeatureCollection
- **Events Produced:** Earthquakes with magnitude, depth, location, felt reports
- **Rate Limits:** No key required. Fair use -- do not poll more than once per minute.
- **Notes:** Official USGS feed. Covers all earthquakes globally within the last hour.

### GDACS (Global Disaster Alerting Coordination System)

- **Name:** `gdacs`
- **Category:** Natural Disaster
- **Endpoint:** `https://www.gdacs.org/gdacsapi/api/events/geteventlist/SEARCH`
- **Format:** JSON
- **Events Produced:** Multi-hazard alerts (earthquakes, floods, cyclones, droughts, volcanoes)
- **Rate Limits:** Public API, no key required.
- **Notes:** UN/EC joint initiative. Events include alert level (Green/Orange/Red).

### NOAA CAP (Common Alerting Protocol)

- **Name:** `noaa_cap`
- **Category:** Natural Disaster
- **Endpoint:** `https://alerts.weather.gov/cap/us.php`
- **Format:** XML/CAP
- **Events Produced:** Weather warnings, watches, advisories for the United States
- **Rate Limits:** Public API.

### NOAA NWS (National Weather Service)

- **Name:** `noaa_nws`
- **Category:** Weather
- **Endpoint:** `https://api.weather.gov/alerts/active`
- **Format:** GeoJSON
- **Events Produced:** Active weather alerts (tornado, flood, hurricane, winter storm, etc.)
- **Rate Limits:** Requires User-Agent header. No key needed.

### Tsunami (NOAA National Tsunami Warning Center)

- **Name:** `tsunami`
- **Category:** Natural Disaster
- **Endpoint:** `https://www.tsunami.gov/` (parsed feed)
- **Format:** JSON/RSS
- **Events Produced:** Tsunami warnings, watches, advisories
- **Rate Limits:** Public feed.

### Volcano (Smithsonian GVP)

- **Name:** `volcano`
- **Category:** Natural Disaster
- **Endpoint:** Smithsonian Global Volcanism Program feed
- **Format:** JSON
- **Events Produced:** Volcanic eruptions, activity reports, alert levels
- **Rate Limits:** Public feed.

### ReliefWeb (UN OCHA)

- **Name:** `reliefweb`
- **Category:** Humanitarian
- **Endpoint:** `https://api.reliefweb.int/v1/reports`
- **Format:** JSON
- **Events Produced:** Humanitarian reports, disaster updates, situation reports
- **Rate Limits:** Public API with fair-use policy.

### OpenSky Network

- **Name:** `opensky`
- **Category:** Aviation
- **Endpoint:** `https://opensky-network.org/api`
- **Format:** JSON
- **Events Produced:** Real-time aircraft positions, callsigns, velocities, altitudes
- **Rate Limits:** Anonymous: 100 req/day. Registered (free): 4000 req/day.
- **Notes:** Enhanced with Bellingcat aircraft database for identification and military flagging.

### Airplanes.live

- **Name:** `airplanes_live`
- **Category:** Aviation
- **Endpoint:** `https://api.airplanes.live/v2/point/{lat}/{lon}/{radius}`
- **Format:** JSON
- **Events Produced:** Real-time ADS-B aircraft data
- **Rate Limits:** Public API, fair use.

### ADSB.one

- **Name:** `adsbone`
- **Category:** Aviation
- **Endpoint:** `https://api.adsb.one/v2/point/{lat}/{lon}/{radius}`
- **Format:** JSON
- **Events Produced:** Real-time ADS-B aircraft data (fallback for Airplanes.live)
- **Rate Limits:** Public API, fair use.

### Open-Meteo

- **Name:** `openmeteo`
- **Category:** Weather
- **Endpoint:** `https://api.open-meteo.com/v1/forecast`
- **Format:** JSON
- **Events Produced:** Weather forecasts, severe weather indicators, temperature extremes
- **Rate Limits:** 10,000 req/day (non-commercial).

### GDELT (Global Database of Events, Language, and Tone)

- **Name:** `gdelt`
- **Category:** Conflict/OSINT
- **Endpoint:** `https://api.gdeltproject.org/api/v2/doc/doc`
- **Format:** JSON
- **Events Produced:** Global news events, conflict reports, media tone analysis
- **Rate Limits:** Public API.

### LiveUAMap

- **Name:** `liveuamap`
- **Category:** Conflict/OSINT
- **Endpoint:** LiveUAMap public data feed
- **Format:** JSON
- **Events Produced:** Conflict events (Ukraine, Middle East, global hotspots)
- **Rate Limits:** Public feed, respect robots.txt.

### Iran Conflict OSINT

- **Name:** `iranconflict`
- **Category:** Conflict/OSINT
- **Endpoint:** `https://raw.githubusercontent.com/danielrosehill/Iran-Israel-War-2026-OSINT-Data/main/data/waves.json`
- **Format:** JSON
- **Events Produced:** Strike waves with operation names, weapons, targets, coordinates, interception rates
- **Rate Limits:** GitHub raw file, no limit.

### ISW (Institute for the Study of War)

- **Name:** `isw`
- **Category:** Conflict/OSINT
- **Endpoint:** `https://understandingwar.org/rss.xml`
- **Format:** RSS/XML
- **Events Produced:** Conflict analysis reports, filtered for Iran/Israel/Middle East keywords
- **Rate Limits:** Public RSS feed.

### CelesTrak

- **Name:** `celestrak`
- **Category:** Space/Satellite
- **Endpoint:** `https://celestrak.org/NORAD/elements/`
- **Format:** TLE (Two-Line Element) / JSON
- **Events Produced:** Satellite orbital data, ISS position, space debris tracking
- **Rate Limits:** Public data.

### SWPC (NOAA Space Weather Prediction Center)

- **Name:** `swpc`
- **Category:** Space Weather
- **Endpoint:** `https://services.swpc.noaa.gov/products/`
- **Format:** JSON
- **Events Produced:** Solar flares, geomagnetic storms, solar wind, Kp index
- **Rate Limits:** Public API.

### WHO Disease Outbreak News

- **Name:** `who`
- **Category:** Health
- **Endpoint:** WHO DON RSS feed
- **Format:** RSS/XML
- **Events Produced:** Disease outbreaks, pandemic alerts, health emergencies
- **Rate Limits:** Public RSS feed.

### ProMED

- **Name:** `promed`
- **Category:** Health
- **Endpoint:** ProMED RSS feed
- **Format:** RSS/XML
- **Events Produced:** Infectious disease reports, unusual health events, epidemiological alerts
- **Rate Limits:** Public RSS feed.

### NASA FIRMS (Fire Information for Resource Management System)

- **Name:** `nasa_firms`
- **Category:** Environmental
- **Endpoint:** `https://firms.modaps.eosdis.nasa.gov/api/`
- **Format:** CSV/JSON
- **Events Produced:** Active fire detections, thermal anomalies (VIIRS/MODIS satellite data)
- **Rate Limits:** Public API for recent data. MAP_KEY needed for archive.

### Piracy IMB (International Maritime Bureau)

- **Name:** `piracy_imb`
- **Category:** Maritime Security
- **Endpoint:** IMB Piracy Reporting Centre RSS feed
- **Format:** RSS/XML
- **Events Produced:** Piracy incidents, armed robbery at sea, suspicious approaches
- **Rate Limits:** Public RSS feed.

### Financial Markets

- **Name:** `financial_markets`
- **Category:** Financial
- **Endpoint:** Multiple public APIs (Yahoo Finance, CoinGecko, etc.)
- **Format:** JSON
- **Events Produced:** VIX changes, crypto price alerts, oil price spikes, yield curve inversions
- **Rate Limits:** Varies by sub-source. Public tickers.

### Global Forest Watch

- **Name:** `globalforestwatch`
- **Category:** Environmental
- **Endpoint:** Global Forest Watch public API
- **Format:** JSON
- **Events Produced:** Deforestation alerts, forest fire detections
- **Rate Limits:** Public API.

### SEC EDGAR

- **Name:** `sec_edgar`
- **Category:** Financial
- **Endpoint:** `https://efts.sec.gov/LATEST/search-index`
- **Format:** JSON/RSS
- **Events Produced:** SEC filings (8-K material events, 10-K annual reports, insider trading)
- **Rate Limits:** 10 req/sec with User-Agent header.

### Bellingcat ADS-B Database

- **Name:** `bellingcat` (enrichment provider)
- **Category:** Aviation (enrichment)
- **Endpoint:** `https://raw.githubusercontent.com/bellingcat/adsb-history/main/backend-data-loading/modes.csv`
- **Format:** CSV
- **Events Produced:** Aircraft identification data (used to enrich flight tracking events)
- **Rate Limits:** GitHub raw file. Updated monthly.
- **Notes:** ~500,000 aircraft registrations. Enables military aircraft detection.

### CISA KEV (Known Exploited Vulnerabilities)

- **Name:** `cisa_kev`
- **Category:** Cyber Security
- **Endpoint:** `https://www.cisa.gov/sites/default/files/feeds/known_exploited_vulnerabilities.json`
- **Format:** JSON
- **Events Produced:** Newly added exploited vulnerabilities, CVE details, remediation deadlines
- **Rate Limits:** Public feed.

### OTX AlienVault

- **Name:** `otx_alienvault`
- **Category:** Cyber Security
- **Endpoint:** `https://otx.alienvault.com/api/v1/pulses/subscribed`
- **Format:** JSON
- **Events Produced:** Threat intelligence pulses, indicators of compromise (IoCs)
- **Rate Limits:** Public API with optional key for higher limits.

### Ukraine Alerts

- **Name:** `ukraine_alerts`
- **Category:** Conflict
- **Endpoint:** Ukraine air raid alert API
- **Format:** JSON
- **Events Produced:** Air raid alerts by region, all-clear signals
- **Rate Limits:** Public API.

### Pikud HaOref (Israel Home Front Command)

- **Name:** `pikud_haoref`
- **Category:** Conflict
- **Endpoint:** Pikud HaOref public alert API
- **Format:** JSON
- **Events Produced:** Rocket/missile alerts by region
- **Rate Limits:** Public API. Polls every 5 seconds during active alerts.

### UKMTO (United Kingdom Maritime Trade Operations)

- **Name:** `ukmto`
- **Category:** Maritime Security
- **Endpoint:** UKMTO advisory RSS feed
- **Format:** RSS/XML
- **Events Produced:** Maritime security advisories, suspicious activity reports
- **Rate Limits:** Public RSS feed.

---

## Tier 1 Providers (Free Key Required)

### ADS-B Exchange

- **Name:** `adsbexchange`
- **Category:** Aviation
- **Endpoint:** `https://adsbexchange.com/api/` (RapidAPI)
- **Format:** JSON
- **Events Produced:** Real-time global ADS-B aircraft data, military aircraft
- **Required Key:** `keys.adsbexchange` in config
- **Signup:** https://www.adsbexchange.com/data/
- **Rate Limits:** Depends on plan (free tier available via RapidAPI)

### AISStream

- **Name:** `aisstream`
- **Category:** Maritime
- **Endpoint:** `wss://stream.aisstream.io/v0/stream`
- **Format:** WebSocket JSON
- **Events Produced:** Real-time AIS vessel positions, vessel details, port activity
- **Required Key:** `keys.aisstream` in config
- **Signup:** https://aisstream.io/
- **Rate Limits:** Free tier: 1 connection, 1 bounding box

### ACLED (Armed Conflict Location & Event Data)

- **Name:** `acled`
- **Category:** Conflict
- **Endpoint:** `https://api.acleddata.com/acled/read`
- **Format:** JSON
- **Events Produced:** Political violence, protest events, conflict data by country/region
- **Required Key:** `keys.acled` in config
- **Signup:** https://acleddata.com/register/
- **Rate Limits:** Free academic/media access

### OpenWeatherMap

- **Name:** `openweather`
- **Category:** Weather
- **Endpoint:** `https://api.openweathermap.org/data/3.0/`
- **Format:** JSON
- **Events Produced:** Current weather, severe weather alerts, forecasts
- **Required Key:** `keys.openweather` in config
- **Signup:** https://openweathermap.org/appid
- **Rate Limits:** Free: 60 req/min, 1M req/month

### OpenSanctions

- **Name:** `opensanctions`
- **Category:** Financial/Security
- **Endpoint:** `https://api.opensanctions.org/`
- **Format:** JSON
- **Events Produced:** Sanctions list changes, PEP updates, entity matches
- **Required Key:** API key in constructor
- **Signup:** https://www.opensanctions.org/api/
- **Rate Limits:** Free tier available

### Global Fishing Watch

- **Name:** `globalfishingwatch`
- **Category:** Maritime/Environmental
- **Endpoint:** `https://gateway.api.globalfishingwatch.org/`
- **Format:** JSON
- **Events Produced:** Illegal fishing activity, vessel tracking, dark vessel detection
- **Required Key:** API key in constructor
- **Signup:** https://globalfishingwatch.org/our-apis/
- **Rate Limits:** Free for research

### NASA FIRMS (Real-Time, with MAP_KEY)

- **Name:** `nasa_firms_rt`
- **Category:** Environmental
- **Endpoint:** `https://firms.modaps.eosdis.nasa.gov/api/area/`
- **Format:** JSON
- **Events Produced:** Near real-time fire data with higher resolution and archive access
- **Required Key:** `keys.nasa` (MAP_KEY) in config
- **Signup:** https://firms.modaps.eosdis.nasa.gov/api/area/
- **Rate Limits:** Free MAP_KEY: 100 transactions/day

---

## Provider Configuration

Each provider can be enabled/disabled and have its polling interval adjusted in `config.json`:

```json
{
  "providers": {
    "usgs": {
      "enabled": true,
      "interval_seconds": 60
    },
    "gdacs": {
      "enabled": true,
      "interval_seconds": 60
    }
  }
}
```

API keys for Tier 1 providers go in the `keys` section:

```json
{
  "keys": {
    "adsbexchange": "your-api-key",
    "aisstream": "your-api-key",
    "acled": "your-api-key",
    "openweather": "your-api-key"
  }
}
```

---

## Adding Custom Providers

To add a new provider, implement the `Provider` interface in `internal/provider/`:

```go
type Provider interface {
    Fetch(ctx context.Context) ([]*model.Event, error)
    Name() string
    Interval() time.Duration
    Enabled() bool
}
```

Then register it in `cmd/sentinel/main.go` inside `initializePoller()`.
