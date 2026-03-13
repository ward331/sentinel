# SENTINEL Watchtower V4.5.0

Real-time geospatial OSINT intelligence dashboard. Tracks aircraft, satellites, earthquakes, wildfires, weather, carrier strike groups, GPS jamming zones, Ukraine frontline positions, and global CCTV feeds on an interactive map.

---

## Stack

| Layer | Tech |
|-------|------|
| Frontend | React 19, TypeScript 5.9, Vite 7, Tailwind 4, MapLibre GL JS |
| Backend | Python FastAPI, APScheduler, httpx |
| Map tiles | CartoDB Dark Matter (default), Esri World Imagery, NASA GIBS MODIS Terra |

---

## Features

### Map Layers

| Layer | Source | Refresh |
|-------|--------|---------|
| Aircraft | [adsb.lol](https://adsb.lol) ADS-B | 60s |
| Satellites | [CelesTrak](https://celestrak.org) TLE/SGP4 | 5min |
| Earthquakes | [USGS](https://earthquake.usgs.gov) M2.5+ | 5min |
| Wildfires | [NASA FIRMS](https://firms.modaps.eosdis.nasa.gov) | 15min |
| Weather alerts | [NWS](https://api.weather.gov) | 5min |
| GDELT news | [GDELT](https://api.gdeltproject.org) geolocated events | 5min |
| Carrier Strike Groups | GDELT news analysis (11 US Navy carriers) | 12h |
| GPS Jamming Zones | ADS-B NAC-P analysis, 2-degree grid heatmap | 60s |
| Ukraine Frontline | [DeepState Map](https://deepstatemap.live) GeoJSON | 30min |
| CCTV Cameras | [TfL JamCams](https://api.tfl.gov.uk) + [NYC DOT](https://webcams.nyctmc.org) | 5min |

### Tools & Interactions

- **Flight trails** -- Accumulated breadcrumb trails per aircraft
- **Measurement tool** -- Click-to-measure with haversine distance and bearing
- **Region dossier** -- Right-click any point for country info via Wikipedia
- **Basemap switcher** -- CartoDB Dark, Esri Imagery, NASA GIBS MODIS Terra
- **Layer toggles** -- Individual visibility control per data source
- **CCTV viewer** -- Floating panel with auto-refreshing camera feeds
- **Find/Locate bar** -- Search and pan to entities
- **Intel briefing** -- Aggregated OSINT news and signal board
- **Financial dashboard** -- Market data panel

---

## Quick Start

### Backend

```bash
cd backend
python -m venv venv
source venv/bin/activate
pip install -r requirements.txt
uvicorn main:app --host 0.0.0.0 --port 8000
```

The backend runs scheduled fetchers via APScheduler. No external cron needed.

### Frontend

```bash
npm install
npm run dev        # Development server (port 5173)
npm run build      # Production build
npm run preview    # Preview production build
```

The frontend expects the backend API at `http://localhost:8000` (configured in `src/api/client.ts`).

---

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/live-data` | GET | All data sources (full payload) |
| `/api/live-data/fast` | GET | Aircraft + GPS jamming (high-frequency polling) |
| `/api/live-data/carriers` | GET | Carrier strike group positions |
| `/api/live-data/cctv` | GET | CCTV camera feeds |
| `/api/live-data/ukraine` | GET | Ukraine frontline GeoJSON |
| `/api/events` | GET | GDELT news events |
| `/api/intel/news` | GET | Aggregated intel news feed |
| `/api/intel/signals` | GET | Signal board entries |
| `/api/financial/markets` | GET | Market data |
| `/health` | GET | Backend health check |

All data endpoints support `ETag` / `If-None-Match` for conditional responses.

---

## Project Structure

```
sentinel-watchtower/
├── backend/
│   ├── main.py                      # FastAPI app + API endpoints
│   ├── requirements.txt
│   ├── config/
│   │   └── news_feeds.json          # RSS feed configuration
│   └── services/
│       ├── data_fetcher.py          # All data source fetchers + schedulers
│       ├── network_utils.py         # HTTP helpers
│       ├── region_dossier.py        # Wikipedia region lookup
│       └── ais_stream.py           # AIS vessel tracking (experimental)
├── src/
│   ├── App.tsx                      # App shell, data polling, state management
│   ├── api/client.ts                # Backend API configuration
│   ├── types/
│   │   ├── livedata.ts              # Data interfaces (Aircraft, Satellite, CarrierGroup, etc.)
│   │   └── sentinel.ts             # App-level types
│   ├── components/
│   │   ├── Map/
│   │   │   └── MaplibreViewer.tsx   # Map with all layers, interactions, tools
│   │   ├── Panels/
│   │   │   ├── WorldviewLeftPanel.tsx    # Layer toggles + basemap switcher
│   │   │   ├── WorldviewRightPanel.tsx   # Event details sidebar
│   │   │   ├── CCTVPanel.tsx             # Camera feed viewer
│   │   │   ├── StatusBar.tsx             # Bottom status bar
│   │   │   ├── FindLocateBar.tsx         # Entity search bar
│   │   │   ├── MapLegend.tsx             # Map legend overlay
│   │   │   └── MarketsPanel.tsx          # Financial data panel
│   │   ├── Intel/                   # Intelligence briefing views
│   │   ├── Financial/               # Market dashboard
│   │   └── Feed/                    # Event feed components
│   └── hooks/
│       ├── useEvents.ts
│       └── useSSE.ts
├── legacy/v4/                       # Archived V4.0 source
├── package.json
├── vite.config.ts
└── tsconfig.json
```

---

## V4.5.0 Changelog

New features merged from [Shadowbroker](https://github.com/BigBodyCobain/Shadowbroker) concepts + custom additions:

- Carrier Strike Group tracking (11 US Navy carriers via GDELT news analysis)
- Ukraine frontline GeoJSON overlay (DeepState Map)
- CCTV camera feeds with clustered map layer and live viewer panel
- GPS jamming detection via NAC-P analysis with heatmap visualization
- Flight trail breadcrumbs per tracked aircraft
- Measurement tool with haversine distance/bearing
- Right-click region dossier (country, capital, population via Wikipedia)
- Basemap switcher (CartoDB Dark, Esri Imagery, NASA GIBS MODIS Terra)
- 6 new TypeScript interfaces, 4 new Python fetchers, 3 new API endpoints
- V4.0 source archived to `legacy/v4/`

---

## License

Private project.
