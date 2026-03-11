"""
AIS vessel stream handler for SENTINEL V4.
Connects to AISStream.io WebSocket for real-time ship tracking.
Falls back to mock data if no API key is configured.
"""

import asyncio
import json
import logging
import math
import os
import random
import time
from typing import Optional

logger = logging.getLogger("sentinel.ais")

# Vessel storage: mmsi -> vessel dict
_vessels: dict[str, dict] = {}
_stream_task: Optional[asyncio.Task] = None
_running = False

# Ship type classification by AIS type codes
SHIP_TYPE_MAP = {
    range(60, 70): "passenger",
    range(70, 80): "cargo",
    range(80, 90): "tanker",
    range(30, 36): "fishing",
    range(36, 40): "pleasure",
    range(50, 56): "military",
}


def classify_ship_type(ais_type: int) -> str:
    """Classify AIS ship type code into a category."""
    for type_range, category in SHIP_TYPE_MAP.items():
        if ais_type in type_range:
            return category
    if ais_type == 35:
        return "military"
    return "cargo"  # default


def get_vessels() -> list[dict]:
    """Return current vessel list."""
    return list(_vessels.values())


async def start_ais_stream() -> None:
    """Start the AIS stream (real or mock)."""
    global _stream_task, _running
    _running = True

    api_key = os.environ.get("AIS_API_KEY", "").strip()
    if api_key:
        logger.info("Starting real AIS WebSocket stream")
        _stream_task = asyncio.create_task(_real_ais_stream(api_key))
    else:
        logger.info("No AIS_API_KEY set, generating mock vessel data")
        _generate_mock_vessels()
        _stream_task = asyncio.create_task(_mock_vessel_updater())


async def stop_ais_stream() -> None:
    """Stop the AIS stream."""
    global _running, _stream_task
    _running = False
    if _stream_task and not _stream_task.done():
        _stream_task.cancel()
        try:
            await _stream_task
        except asyncio.CancelledError:
            pass
    _stream_task = None
    logger.info("AIS stream stopped")


async def _real_ais_stream(api_key: str) -> None:
    """Connect to AISStream.io WebSocket and process messages."""
    import websockets

    url = "wss://stream.aisstream.io/v0/stream"
    subscribe_msg = json.dumps({
        "APIKey": api_key,
        "BoundingBoxes": [[[-90, -180], [90, 180]]],
        "FilterMessageTypes": ["PositionReport", "ShipStaticData"],
    })

    while _running:
        try:
            async with websockets.connect(url, ping_interval=30) as ws:
                await ws.send(subscribe_msg)
                logger.info("Connected to AISStream.io")

                async for message in ws:
                    if not _running:
                        break
                    try:
                        _process_ais_message(json.loads(message))
                    except Exception as e:
                        logger.debug("Error processing AIS message: %s", e)

                    # Cap vessel count
                    if len(_vessels) > 10000:
                        _prune_stale_vessels()

        except asyncio.CancelledError:
            return
        except Exception as e:
            logger.error("AIS WebSocket error: %s, reconnecting in 30s", e)
            await asyncio.sleep(30)


def _process_ais_message(msg: dict) -> None:
    """Process a single AIS message from the stream."""
    msg_type = msg.get("MessageType", "")
    meta = msg.get("MetaData", {})
    mmsi = str(meta.get("MMSI", ""))
    if not mmsi:
        return

    if mmsi not in _vessels:
        _vessels[mmsi] = {
            "mmsi": mmsi,
            "name": meta.get("ShipName", "").strip() or f"VESSEL-{mmsi}",
            "lat": None,
            "lon": None,
            "speed": 0,
            "course": 0,
            "ship_type": "cargo",
            "destination": "",
            "flag": meta.get("country_code", ""),
            "last_update": time.time(),
        }

    vessel = _vessels[mmsi]
    vessel["last_update"] = time.time()

    if "ShipName" in meta and meta["ShipName"].strip():
        vessel["name"] = meta["ShipName"].strip()

    if msg_type == "PositionReport":
        report = msg.get("Message", {}).get("PositionReport", {})
        if report:
            lat = report.get("Latitude")
            lon = report.get("Longitude")
            if lat is not None and -90 <= lat <= 90 and lon is not None and -180 <= lon <= 180:
                vessel["lat"] = round(lat, 5)
                vessel["lon"] = round(lon, 5)
            vessel["speed"] = round(report.get("Sog", 0), 1)
            vessel["course"] = round(report.get("Cog", 0), 1)

    elif msg_type == "ShipStaticData":
        static = msg.get("Message", {}).get("ShipStaticData", {})
        if static:
            ais_type = static.get("Type", 0)
            vessel["ship_type"] = classify_ship_type(ais_type)
            dest = static.get("Destination", "")
            if dest:
                vessel["destination"] = dest.strip()


def _prune_stale_vessels() -> None:
    """Remove vessels not updated in the last 30 minutes."""
    cutoff = time.time() - 1800
    stale = [k for k, v in _vessels.items() if v.get("last_update", 0) < cutoff]
    for k in stale:
        del _vessels[k]
    logger.info("Pruned %d stale vessels, %d remaining", len(stale), len(_vessels))


# ---------------------------------------------------------------------------
# Mock vessel generation for when no API key is available
# ---------------------------------------------------------------------------

# Major shipping lanes defined as (lat, lon, heading, spread) waypoints
SHIPPING_LANES = [
    # Trans-Pacific
    {"lat_range": (30, 40), "lon_range": (-170, -120), "heading": 90, "count": 60},
    # Trans-Atlantic
    {"lat_range": (35, 55), "lon_range": (-60, -10), "heading": 75, "count": 50},
    # Suez approach
    {"lat_range": (12, 32), "lon_range": (32, 45), "heading": 340, "count": 40},
    # Malacca Strait
    {"lat_range": (-2, 6), "lon_range": (98, 106), "heading": 315, "count": 40},
    # South China Sea
    {"lat_range": (5, 22), "lon_range": (108, 120), "heading": 30, "count": 50},
    # English Channel
    {"lat_range": (49, 52), "lon_range": (-3, 3), "heading": 60, "count": 30},
    # Gulf of Mexico
    {"lat_range": (24, 30), "lon_range": (-95, -85), "heading": 180, "count": 30},
    # Med
    {"lat_range": (33, 38), "lon_range": (0, 30), "heading": 90, "count": 40},
    # Indian Ocean
    {"lat_range": (-10, 10), "lon_range": (55, 80), "heading": 90, "count": 35},
    # Cape of Good Hope
    {"lat_range": (-36, -30), "lon_range": (15, 25), "heading": 45, "count": 25},
    # East China Sea
    {"lat_range": (25, 35), "lon_range": (120, 132), "heading": 30, "count": 40},
    # Persian Gulf
    {"lat_range": (24, 28), "lon_range": (49, 56), "heading": 135, "count": 35},
    # North Sea
    {"lat_range": (52, 58), "lon_range": (0, 8), "heading": 0, "count": 25},
]

SHIP_NAMES_PREFIX = [
    "EVER", "MAERSK", "MSC", "CMA CGM", "COSCO", "OOCL", "NYK", "MOL",
    "HANJIN", "ZIM", "PIL", "YANG MING", "HYUNDAI", "HAPAG", "ATLANTIC",
    "PACIFIC", "ORIENT", "NORDIC", "GLOBAL", "STAR", "SEA", "OCEAN",
]

SHIP_NAMES_SUFFIX = [
    "FORTUNE", "GLORY", "HARMONY", "SPIRIT", "CHAMPION", "PRIDE",
    "DIAMOND", "EXPRESS", "TRADER", "NAVIGATOR", "PIONEER", "VOYAGER",
    "EAGLE", "PHOENIX", "EMERALD", "TITAN", "LIBERTY", "SOVEREIGN",
]

DESTINATIONS = [
    "SINGAPORE", "ROTTERDAM", "SHANGHAI", "LOS ANGELES", "HAMBURG",
    "DUBAI", "HONG KONG", "BUSAN", "ANTWERP", "YOKOHAMA", "FELIXSTOWE",
    "SANTOS", "MUMBAI", "COLOMBO", "PORT SAID", "PIRAEUS", "ISTANBUL",
]

FLAGS = ["PA", "LR", "MH", "HK", "SG", "BS", "MT", "CY", "GB", "NO", "GR", "JP", "CN", "KR", "DE"]

MOCK_SHIP_TYPES = ["cargo"] * 40 + ["tanker"] * 25 + ["passenger"] * 10 + ["fishing"] * 10 + ["military"] * 5 + ["pleasure"] * 10


def _generate_mock_vessels() -> None:
    """Generate 500 mock vessels along major shipping lanes."""
    _vessels.clear()
    vessel_id = 200000000

    for lane in SHIPPING_LANES:
        for _ in range(lane["count"]):
            vessel_id += random.randint(1, 100)
            mmsi = str(vessel_id)
            lat = random.uniform(*lane["lat_range"])
            lon = random.uniform(*lane["lon_range"])
            heading = lane["heading"] + random.uniform(-30, 30)

            name_prefix = random.choice(SHIP_NAMES_PREFIX)
            name_suffix = random.choice(SHIP_NAMES_SUFFIX)
            name = f"{name_prefix} {name_suffix}"

            _vessels[mmsi] = {
                "mmsi": mmsi,
                "name": name,
                "lat": round(lat, 5),
                "lon": round(lon, 5),
                "speed": round(random.uniform(2, 22), 1),
                "course": round(heading % 360, 1),
                "ship_type": random.choice(MOCK_SHIP_TYPES),
                "destination": random.choice(DESTINATIONS),
                "flag": random.choice(FLAGS),
                "last_update": time.time(),
            }

    logger.info("Generated %d mock vessels", len(_vessels))


async def _mock_vessel_updater() -> None:
    """Periodically update mock vessel positions to simulate movement."""
    while _running:
        try:
            await asyncio.sleep(10)
            now = time.time()
            for vessel in _vessels.values():
                if vessel["lat"] is None or vessel["lon"] is None:
                    continue
                speed_kts = vessel["speed"]
                course_rad = math.radians(vessel["course"])
                # Move ~10 seconds of travel
                nm_traveled = speed_kts * (10 / 3600)
                dlat = nm_traveled * math.cos(course_rad) / 60
                dlon = nm_traveled * math.sin(course_rad) / (60 * max(math.cos(math.radians(vessel["lat"])), 0.01))
                vessel["lat"] = round(vessel["lat"] + dlat, 5)
                vessel["lon"] = round(((vessel["lon"] + dlon + 180) % 360) - 180, 5)
                # Random minor course and speed variations
                vessel["course"] = round((vessel["course"] + random.uniform(-1, 1)) % 360, 1)
                vessel["speed"] = round(max(0.5, vessel["speed"] + random.uniform(-0.3, 0.3)), 1)
                vessel["last_update"] = now
        except asyncio.CancelledError:
            return
        except Exception as e:
            logger.error("Mock updater error: %s", e)
