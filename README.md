# Watchtower

Real-time web frontend for [SENTINEL V2](https://github.com/ward331/sentinel) — global event monitoring dashboard.

## Features

- **Live Map** — Events plotted on dark-themed OpenStreetMap via Leaflet (no API key needed)
- **Real-time Feed** — SSE streaming from SENTINEL backend, events appear instantly
- **Advanced Filters** — Source, category, severity, magnitude, full-text search, spatial queries
- **Provider Health** — Live status of all 24+ data providers
- **Alert Rules** — View and manage alert rule configurations
- **Setup Wizard** — First-run prompt asks for SENTINEL server URL, verifies connectivity

## Quick Start

```bash
git clone https://github.com/ward331/sentinel-watchtower.git
cd sentinel-watchtower
npm install
npm run dev
```

Open `http://localhost:5173` — the setup wizard will ask for your SENTINEL V2 server URL.

## Production Deploy

```bash
npm run build
npx serve dist
```

The `dist/` folder contains static files you can serve from any web server (nginx, Apache, S3, etc).

## Requirements

- Node.js 18+
- A running SENTINEL V2 backend (default: `http://localhost:8080`)

## Tech Stack

- React 18 + TypeScript + Vite
- Leaflet + OpenStreetMap (free, no API key)
- Tailwind CSS (dark theme)
- Native EventSource for SSE streaming
- Lucide icons

## SENTINEL V2 API Endpoints Used

| Endpoint | Purpose |
|----------|---------|
| `GET /api/events` | Query events with filters |
| `GET /api/events/stream` | SSE real-time event stream |
| `GET /api/health` | Server health check |
| `GET /api/metrics` | System metrics |
| `GET /api/providers` | List registered providers |
| `GET /api/providers/health` | Provider health status |
| `GET /api/alerts/rules` | Alert rule management |

## License

MIT
