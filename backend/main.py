"""
SENTINEL V4 — FastAPI Backend
Real-time OSINT data aggregation server.
Fetches from 12+ sources on independent schedules and serves via REST API.
"""

import hashlib
import json
import logging
import sys
import time
from contextlib import asynccontextmanager
from datetime import datetime, timezone

from dotenv import load_dotenv
from fastapi import FastAPI, Query, Request, Response
from fastapi.middleware.cors import CORSMiddleware
from fastapi.middleware.gzip import GZipMiddleware

from services.ais_stream import get_vessels, start_ais_stream, stop_ais_stream
from services.data_fetcher import (
    force_refresh,
    latest_data,
    source_timestamps,
    start_scheduler,
    stop_scheduler,
)
from services.region_dossier import get_region_dossier

# ---------------------------------------------------------------------------
# Logging
# ---------------------------------------------------------------------------
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
    handlers=[logging.StreamHandler(sys.stdout)],
)
logger = logging.getLogger("sentinel.main")

# ---------------------------------------------------------------------------
# Env
# ---------------------------------------------------------------------------
load_dotenv()

# ---------------------------------------------------------------------------
# Startup / shutdown
# ---------------------------------------------------------------------------
_start_time: float = 0.0


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Manage scheduler and AIS stream lifecycle."""
    global _start_time
    _start_time = time.time()

    logger.info("SENTINEL V4 starting up...")
    start_scheduler()
    await start_ais_stream()
    logger.info("All services started")

    yield

    logger.info("SENTINEL V4 shutting down...")
    stop_scheduler()
    await stop_ais_stream()
    from services.network_utils import close_client
    close_client()
    logger.info("Shutdown complete")


# ---------------------------------------------------------------------------
# App
# ---------------------------------------------------------------------------
app = FastAPI(
    title="SENTINEL V4",
    description="Real-time OSINT data aggregation backend",
    version="4.0.0",
    lifespan=lifespan,
)

app.add_middleware(GZipMiddleware, minimum_size=1000)
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
    expose_headers=["ETag"],
)


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
def _compute_etag(data: dict | list) -> str:
    """Compute MD5 ETag for response data."""
    raw = json.dumps(data, separators=(",", ":"), sort_keys=True, default=str)
    return hashlib.md5(raw.encode()).hexdigest()


def _etag_response(request: Request, response: Response, data: dict) -> dict | Response:
    """Return 304 if ETag matches, else set ETag header and return data."""
    etag = _compute_etag(data)
    response.headers["ETag"] = f'"{etag}"'
    response.headers["Cache-Control"] = "no-cache"

    if_none_match = request.headers.get("if-none-match", "").strip('"')
    if if_none_match == etag:
        return Response(status_code=304, headers={"ETag": f'"{etag}"'})

    return data


# ---------------------------------------------------------------------------
# Routes
# ---------------------------------------------------------------------------
@app.get("/api/live-data")
async def get_all_live_data():
    """Return all latest data from every source, plus vessel data."""
    result = dict(latest_data)
    result["ships"] = get_vessels()
    result["_timestamps"] = dict(source_timestamps)
    return result


@app.get("/api/live-data/fast")
async def get_fast_data(request: Request, response: Response):
    """High-frequency data (flights, ships, GPS jamming) with ETag caching."""
    data = {
        "flights": latest_data.get("flights", []),
        "ships": get_vessels(),
        "_timestamps": {
            k: v for k, v in source_timestamps.items()
            if k in ("flights",)
        },
    }
    return _etag_response(request, response, data)


@app.get("/api/live-data/slow")
async def get_slow_data(request: Request, response: Response):
    """Low-frequency data with ETag caching."""
    slow_keys = [
        "satellites", "earthquakes", "fires", "gdelt", "space_weather",
        "internet_outages", "news", "kiwisdr", "datacenters", "financial",
    ]
    data = {k: latest_data.get(k, []) for k in slow_keys}
    data["_timestamps"] = {
        k: v for k, v in source_timestamps.items() if k in slow_keys
    }
    return _etag_response(request, response, data)


@app.get("/api/health")
async def health_check():
    """Health endpoint with source counts and uptime."""
    uptime_s = time.time() - _start_time if _start_time else 0
    hours, remainder = divmod(int(uptime_s), 3600)
    minutes, seconds = divmod(remainder, 60)

    sources = {}
    for key, items in latest_data.items():
        sources[key] = {
            "count": len(items),
            "last_update": source_timestamps.get(key, "never"),
        }
    sources["ships"] = {
        "count": len(get_vessels()),
        "last_update": source_timestamps.get("ships", "never"),
    }

    total = sum(s["count"] for s in sources.values())

    return {
        "status": "operational",
        "uptime": f"{hours}h {minutes}m {seconds}s",
        "uptime_seconds": int(uptime_s),
        "total_data_points": total,
        "sources": sources,
        "server_time": datetime.now(timezone.utc).isoformat(),
    }


@app.get("/api/refresh")
async def trigger_refresh():
    """Force an immediate background refresh of all sources."""
    force_refresh()
    return {
        "status": "refresh_triggered",
        "message": "All sources queued for immediate refresh",
        "server_time": datetime.now(timezone.utc).isoformat(),
    }


@app.get("/api/region-dossier")
async def region_dossier(
    lat: float = Query(..., ge=-90, le=90, description="Latitude"),
    lng: float = Query(..., ge=-180, le=180, description="Longitude"),
):
    """Get a country/region profile with Wikipedia summary for given coordinates."""
    result = get_region_dossier(lat, lng)
    return result


# ---------------------------------------------------------------------------
# Entry point
# ---------------------------------------------------------------------------
if __name__ == "__main__":
    import uvicorn

    uvicorn.run(
        "main:app",
        host="0.0.0.0",
        port=8000,
        log_level="info",
        access_log=True,
    )
