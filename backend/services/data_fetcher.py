"""
SENTINEL V4 — Main data aggregation module.
Fetches from 12+ OSINT sources on independent schedules using APScheduler.
Each source writes into `latest_data` and timestamps into `source_timestamps`.
"""

import json
import logging
import math
import os
import re
import time
import xml.etree.ElementTree as ET
from concurrent.futures import ThreadPoolExecutor
from datetime import datetime, timezone
from pathlib import Path
from typing import Optional

import feedparser
from apscheduler.schedulers.background import BackgroundScheduler
from sgp4.api import Satrec, WGS72
from sgp4 import exporter

from services.network_utils import fetch_json, fetch_text

logger = logging.getLogger("sentinel.fetcher")

# ---------------------------------------------------------------------------
# Shared state
# ---------------------------------------------------------------------------
latest_data: dict[str, list] = {
    "flights": [],
    "satellites": [],
    "earthquakes": [],
    "fires": [],
    "gdelt": [],
    "space_weather": [],
    "internet_outages": [],
    "news": [],
    "kiwisdr": [],
    "datacenters": [],
    "financial": [],
}

source_timestamps: dict[str, str] = {}

_executor = ThreadPoolExecutor(max_workers=6)
_scheduler: Optional[BackgroundScheduler] = None

# ---------------------------------------------------------------------------
# Config
# ---------------------------------------------------------------------------
CONFIG_DIR = Path(__file__).parent.parent / "config"
NEWS_FEEDS_FILE = CONFIG_DIR / "news_feeds.json"

# Country name -> (lat, lon) for simple geocoding of news articles
COUNTRY_COORDS: dict[str, tuple[float, float]] = {
    "United States": (39.8, -98.6), "USA": (39.8, -98.6), "US": (39.8, -98.6),
    "Russia": (61.5, 105.3), "Ukraine": (48.4, 31.2), "China": (35.9, 104.2),
    "Iran": (32.4, 53.7), "Israel": (31.0, 34.9), "Palestine": (31.9, 35.2),
    "Syria": (35.0, 38.5), "Iraq": (33.2, 43.7), "Afghanistan": (33.9, 67.7),
    "Pakistan": (30.4, 69.3), "India": (20.6, 79.0), "Japan": (36.2, 138.3),
    "North Korea": (40.3, 127.5), "South Korea": (35.9, 127.8),
    "Taiwan": (23.7, 121.0), "Philippines": (12.9, 121.8),
    "Germany": (51.2, 10.5), "France": (46.2, 2.2), "UK": (55.4, -3.4),
    "United Kingdom": (55.4, -3.4), "Britain": (55.4, -3.4),
    "Turkey": (39.9, 32.9), "Saudi Arabia": (23.9, 45.1), "Yemen": (15.6, 48.5),
    "Somalia": (5.2, 46.2), "Nigeria": (9.1, 8.7), "Ethiopia": (9.1, 40.5),
    "Sudan": (12.9, 30.2), "Libya": (26.3, 17.2), "Egypt": (26.8, 30.8),
    "Brazil": (-14.2, -51.9), "Mexico": (23.6, -102.6), "Canada": (56.1, -106.3),
    "Australia": (-25.3, 133.8), "Indonesia": (-0.8, 113.9), "Malaysia": (4.2, 101.9),
    "Singapore": (1.35, 103.8), "Thailand": (15.9, 100.9), "Vietnam": (14.1, 108.3),
    "Myanmar": (21.9, 95.9), "Bangladesh": (23.7, 90.4), "Sri Lanka": (7.9, 80.8),
    "Nepal": (28.4, 84.1), "Poland": (51.9, 19.1), "Romania": (45.9, 24.9),
    "Italy": (41.9, 12.6), "Spain": (40.5, -3.7), "Portugal": (39.4, -8.2),
    "Greece": (39.1, 21.8), "Netherlands": (52.1, 5.3), "Belgium": (50.5, 4.5),
    "Sweden": (60.1, 18.6), "Norway": (60.5, 8.5), "Finland": (61.9, 25.7),
    "Denmark": (56.3, 9.5), "Switzerland": (46.8, 8.2), "Austria": (47.5, 14.6),
    "Czech Republic": (49.8, 15.5), "Hungary": (47.2, 19.5),
    "Colombia": (4.6, -74.3), "Venezuela": (6.4, -66.6), "Peru": (-9.2, -75.0),
    "Argentina": (-38.4, -63.6), "Chile": (-35.7, -71.5),
    "South Africa": (-30.6, 22.9), "Kenya": (-0.02, 37.9), "Congo": (-4.0, 21.8),
    "Morocco": (31.8, -7.1), "Algeria": (28.0, 1.7), "Tunisia": (34.0, 9.5),
    "Lebanon": (33.9, 35.9), "Jordan": (30.6, 36.2), "Kuwait": (29.3, 47.5),
    "UAE": (23.4, 53.8), "Qatar": (25.4, 51.2), "Bahrain": (26.0, 50.6),
    "Oman": (21.5, 55.9), "Gaza": (31.4, 34.4), "West Bank": (31.9, 35.3),
    "Arctic": (82.0, 0.0), "Antarctica": (-80.0, 0.0),
    "NATO": (50.9, 4.4), "EU": (50.8, 4.4), "Pentagon": (38.9, -77.1),
    "Kremlin": (55.8, 37.6), "Beijing": (39.9, 116.4), "Moscow": (55.8, 37.6),
    "Washington": (38.9, -77.0), "London": (51.5, -0.1), "Paris": (48.9, 2.4),
    "Berlin": (52.5, 13.4), "Tokyo": (35.7, 139.7), "Seoul": (37.6, 127.0),
    "Taipei": (25.0, 121.5), "Tehran": (35.7, 51.4), "Kyiv": (50.5, 30.5),
    "Kiev": (50.5, 30.5),
}

# Satellite classification by name keywords
SAT_CLASSIFY = {
    "military_recon": ["USA ", "NROL", "KH-", "KEYHOLE", "LACROSSE", "ONYX", "MISTY",
                       "TOPAZ", "CRYSTAL", "MENTOR", "ORION", "TRUMPET", "VORTEX",
                       "YAOGAN", "COSMOS", "KONDOR", "PERSONA", "BARS-M",
                       "OFEK", "SAR-LUPE", "HELIOS", "CSO-", "PLEIADES NEO"],
    "sar": ["RADARSAT", "SENTINEL-1", "ICEYE", "CAPELLA", "SAR", "COSMO-SKYMED",
            "TERRASAR", "TANDEM", "KOMPSAT-5", "ALOS-2", "SAOCOM"],
    "sigint": ["INTRUDER", "ELISA", "CERES", "LOTOS", "PION", "LIANA",
               "NEMESIS", "SHARP", "SIGINT", "MERCURY"],
    "navigation": ["GPS ", "NAVSTAR", "GLONASS", "GALILEO", "BEIDOU", "IRNSS", "QZSS"],
    "early_warning": ["SBIRS", "DSP", "MIDAS", "TUNDRA", "EKS", "OKO",
                      "IMEWS", "STSS", "PTSS"],
    "iss": ["ISS (ZARYA)", "ISS"],
}


# ---------------------------------------------------------------------------
# Scheduler management
# ---------------------------------------------------------------------------

def start_scheduler() -> None:
    """Initialize and start the APScheduler with all fetch jobs."""
    global _scheduler
    _scheduler = BackgroundScheduler(
        executors={"default": {"type": "threadpool", "max_workers": 6}},
        job_defaults={"coalesce": True, "max_instances": 1, "misfire_grace_time": 60},
    )

    # Register jobs with staggered start to avoid thundering herd
    jobs = [
        (fetch_flights,         "flights",          60),
        (fetch_satellites,      "satellites",       120),
        (fetch_earthquakes,     "earthquakes",      300),
        (fetch_fires,           "fires",            300),
        (fetch_gdelt,           "gdelt",            120),
        (fetch_space_weather,   "space_weather",    600),
        (fetch_internet_outages,"internet_outages", 600),
        (fetch_news,            "news",             300),
        (fetch_kiwisdr,         "kiwisdr",         3600),
        (fetch_financial,       "financial",        120),
    ]

    for i, (func, name, interval) in enumerate(jobs):
        _scheduler.add_job(
            func,
            "interval",
            seconds=interval,
            id=name,
            name=name,
            next_run_time=datetime.now(timezone.utc),  # run immediately
        )

    _scheduler.start()
    logger.info("Scheduler started with %d jobs", len(jobs))

    # Datacenters: one-time load
    _executor.submit(fetch_datacenters)


def stop_scheduler() -> None:
    """Shut down the scheduler gracefully."""
    global _scheduler
    if _scheduler and _scheduler.running:
        _scheduler.shutdown(wait=False)
        logger.info("Scheduler stopped")
    _scheduler = None


def force_refresh() -> None:
    """Trigger an immediate refresh of all sources in background threads."""
    funcs = [
        fetch_flights, fetch_satellites, fetch_earthquakes, fetch_fires,
        fetch_gdelt, fetch_space_weather, fetch_internet_outages,
        fetch_news, fetch_financial,
    ]
    for fn in funcs:
        _executor.submit(fn)
    logger.info("Force refresh triggered for %d sources", len(funcs))


# ---------------------------------------------------------------------------
# Fetch functions
# ---------------------------------------------------------------------------

def fetch_flights() -> None:
    """Fetch military + commercial aircraft positions."""
    try:
        aircraft = []
        seen_icao: set[str] = set()

        # Military aircraft from adsb.lol LADD feed
        mil_data = fetch_json("https://api.adsb.lol/v2/ladd", timeout=12)
        if mil_data and "ac" in mil_data:
            for ac in mil_data["ac"][:2500]:
                lat = ac.get("lat")
                lon = ac.get("lon")
                if lat is None or lon is None:
                    continue
                icao = ac.get("hex", "")
                if icao:
                    seen_icao.add(icao)
                aircraft.append({
                    "icao": icao,
                    "callsign": (ac.get("flight") or "").strip(),
                    "lat": round(float(lat), 4),
                    "lon": round(float(lon), 4),
                    "alt_ft": ac.get("alt_baro", 0),
                    "speed_kts": ac.get("gs", 0),
                    "heading": ac.get("track", 0),
                    "on_ground": ac.get("alt_baro") == "ground",
                    "category": "military",
                    "squawk": ac.get("squawk", ""),
                })

        # Commercial/private aircraft from adsb.lol regional queries (no auth needed)
        # Cover major regions for global-ish coverage
        _REGIONS = [
            (40, -74, 500),   # US East
            (37, -122, 500),  # US West
            (41, -87, 500),   # US Central
            (51, 0, 500),     # Europe West
            (50, 15, 500),    # Europe Central
            (55, 37, 500),    # Russia/Moscow
            (35, 139, 500),   # Japan/East Asia
            (1, 104, 500),    # SE Asia
            (25, 55, 500),    # Middle East
            (-33, 151, 500),  # Australia
        ]
        for rlat, rlon, rdist in _REGIONS:
            try:
                region_data = fetch_json(
                    f"https://api.adsb.lol/v2/lat/{rlat}/lon/{rlon}/dist/{rdist}",
                    timeout=10,
                )
                if not region_data or "ac" not in region_data:
                    continue
                for ac in region_data["ac"]:
                    lat = ac.get("lat")
                    lon = ac.get("lon")
                    if lat is None or lon is None:
                        continue
                    icao = ac.get("hex", "")
                    if icao in seen_icao:
                        continue
                    seen_icao.add(icao)
                    callsign = (ac.get("flight") or "").strip()
                    cat = "commercial"
                    if callsign:
                        cs_upper = callsign.upper()
                        if any(p in cs_upper for p in ["RCH", "DUKE", "EVAC", "NAVY", "AIR FORCE",
                                                        "FORTE", "JAKE", "HOMER", "TEAL", "ANVIL",
                                                        "DRACO", "VIPER"]):
                            cat = "military"
                        elif any(p in cs_upper for p in ["N1", "N2", "N3", "N4", "N5", "N6", "N7", "N8", "N9"]):
                            if len(callsign) <= 6:
                                cat = "private"
                    aircraft.append({
                        "icao": icao,
                        "callsign": callsign,
                        "lat": round(float(lat), 4),
                        "lon": round(float(lon), 4),
                        "alt_ft": ac.get("alt_baro", 0) if ac.get("alt_baro") != "ground" else 0,
                        "speed_kts": ac.get("gs", 0),
                        "heading": ac.get("track", 0),
                        "on_ground": ac.get("alt_baro") == "ground",
                        "category": cat,
                        "squawk": ac.get("squawk", ""),
                    })
            except Exception:
                continue  # skip failed regions

        # Deduplicate by ICAO (seen_icao already tracks military + regional)
        deduped = []
        final_seen: set[str] = set()
        for ac in aircraft:
            if ac["icao"] and ac["icao"] not in final_seen:
                final_seen.add(ac["icao"])
                deduped.append(ac)
            elif not ac["icao"]:
                deduped.append(ac)

        latest_data["flights"] = deduped[:5000]
        source_timestamps["flights"] = datetime.now(timezone.utc).isoformat()
        logger.info("Flights: %d aircraft", len(latest_data["flights"]))

    except Exception as e:
        logger.error("fetch_flights error: %s", e, exc_info=True)


def fetch_satellites() -> None:
    """Fetch TLE data from CelesTrak and propagate current positions with SGP4."""
    try:
        data = fetch_json(
            "https://celestrak.org/NORAD/elements/gp.php",
            params={"GROUP": "active", "FORMAT": "json"},
            timeout=30,
        )
        if not data or not isinstance(data, list):
            logger.warning("No satellite TLE data received")
            return

        now = datetime.now(timezone.utc)
        # SGP4 uses Julian date
        jd_base = _datetime_to_jd(now)
        jd_frac = 0.0

        satellites = []
        for entry in data:
            try:
                name = entry.get("OBJECT_NAME", "UNKNOWN")
                norad_id = entry.get("NORAD_CAT_ID", 0)

                # Extract TLE parameters for SGP4
                tle_line1 = entry.get("TLE_LINE1")
                tle_line2 = entry.get("TLE_LINE2")

                if tle_line1 and tle_line2:
                    sat = Satrec.twoline2rv(tle_line1, tle_line2, WGS72)
                else:
                    # Build from GP elements
                    epoch_yr = entry.get("EPOCH", "")
                    mean_motion = float(entry.get("MEAN_MOTION", 0))
                    eccentricity = float(entry.get("ECCENTRICITY", 0))
                    inclination = float(entry.get("INCLINATION", 0))
                    ra_of_asc = float(entry.get("RA_OF_ASC_NODE", 0))
                    arg_of_pericenter = float(entry.get("ARG_OF_PERICENTER", 0))
                    mean_anomaly = float(entry.get("MEAN_ANOMALY", 0))
                    bstar = float(entry.get("BSTAR", 0))
                    mean_motion_dot = float(entry.get("MEAN_MOTION_DOT", 0))
                    mean_motion_ddot = float(entry.get("MEAN_MOTION_DDOT", 0))

                    if mean_motion <= 0:
                        continue

                    # Parse epoch
                    epoch_dt = datetime.fromisoformat(epoch_yr.replace("Z", "+00:00")) if epoch_yr else now
                    epoch_jd = _datetime_to_jd(epoch_dt)

                    sat = Satrec()
                    sat.sgp4init(
                        WGS72,
                        'i',  # improved mode
                        int(norad_id),
                        epoch_jd - 2433281.5,  # epoch in days since 1949 Dec 31
                        bstar,
                        mean_motion_dot,
                        mean_motion_ddot,
                        eccentricity,
                        math.radians(arg_of_pericenter),
                        math.radians(inclination),
                        math.radians(mean_anomaly),
                        mean_motion * (2 * math.pi / 1440),  # rev/day to rad/min
                        math.radians(ra_of_asc),
                    )

                # Propagate to current time
                e, r, v = sat.sgp4(jd_base, jd_frac)
                if e != 0:
                    continue  # propagation error

                # Convert ECI to lat/lon/alt
                x, y, z = r  # km
                vx, vy, vz = v  # km/s

                # Calculate lat/lon from ECI
                alt_km = math.sqrt(x**2 + y**2 + z**2) - 6371.0
                if alt_km < 100 or alt_km > 50000:
                    continue  # filter unreasonable altitudes

                lat = math.degrees(math.atan2(z, math.sqrt(x**2 + y**2)))
                # Account for Earth's rotation (GMST)
                gmst = _gmst_from_jd(jd_base + jd_frac)
                lon = math.degrees(math.atan2(y, x)) - gmst
                lon = ((lon + 180) % 360) - 180  # normalize to [-180, 180]

                speed_kph = math.sqrt(vx**2 + vy**2 + vz**2) * 3600  # km/s to km/h

                # Classify mission type
                mission_type = _classify_satellite(name)
                country = _satellite_country(name, entry.get("OWNER", ""))

                satellites.append({
                    "name": name,
                    "norad_id": int(norad_id),
                    "lat": round(lat, 3),
                    "lon": round(lon, 3),
                    "alt_km": round(alt_km, 1),
                    "speed_kph": round(speed_kph, 1),
                    "mission_type": mission_type,
                    "country": country,
                })

            except Exception as e:
                continue  # skip malformed entries

        latest_data["satellites"] = satellites
        source_timestamps["satellites"] = datetime.now(timezone.utc).isoformat()
        logger.info("Satellites: %d tracked", len(satellites))

    except Exception as e:
        logger.error("fetch_satellites error: %s", e, exc_info=True)


def _datetime_to_jd(dt: datetime) -> float:
    """Convert a datetime to Julian Date."""
    a = (14 - dt.month) // 12
    y = dt.year + 4800 - a
    m = dt.month + 12 * a - 3
    jdn = dt.day + (153 * m + 2) // 5 + 365 * y + y // 4 - y // 100 + y // 400 - 32045
    jd = jdn + (dt.hour - 12) / 24 + dt.minute / 1440 + dt.second / 86400
    return jd


def _gmst_from_jd(jd: float) -> float:
    """Compute Greenwich Mean Sidereal Time in degrees from Julian Date."""
    t = (jd - 2451545.0) / 36525.0
    gmst = 280.46061837 + 360.98564736629 * (jd - 2451545.0) + 0.000387933 * t**2 - t**3 / 38710000.0
    return gmst % 360


def _classify_satellite(name: str) -> str:
    """Classify satellite mission type from its name."""
    name_upper = name.upper()
    for mission_type, keywords in SAT_CLASSIFY.items():
        for kw in keywords:
            if kw in name_upper:
                return mission_type
    # Additional heuristic rules
    if "STARLINK" in name_upper or "ONEWEB" in name_upper or "IRIDIUM" in name_upper:
        return "commercial"
    if "GOES" in name_upper or "METEOSAT" in name_upper or "NOAA" in name_upper:
        return "commercial"
    return "commercial"


def _satellite_country(name: str, owner: str) -> str:
    """Infer satellite operating country."""
    combined = f"{name} {owner}".upper()
    country_hints = {
        "US": ["USA ", "NROL", "GPS ", "NAVSTAR", "TDRS", "GOES ", "DSP", "SBIRS",
               "STARLINK", "SPACEX", "PLANET", "CAPELLA"],
        "RU": ["COSMOS", "GLONASS", "KONDOR", "PERSONA", "BARS", "LOTOS", "PION",
               "LIANA", "TUNDRA", "EKS", "METEOR", "RESURS"],
        "CN": ["BEIDOU", "YAOGAN", "GAOFEN", "FENGYUN", "TIANLIAN", "SHIJIAN",
               "CZ-", "CHANG ZHENG", "TIANWEN"],
        "IN": ["IRNSS", "CARTOSAT", "RESOURCESAT", "RISAT", "ASTROSAT"],
        "JP": ["QZSS", "HIMAWARI", "ALOS", "MICHIBIKI"],
        "EU": ["GALILEO", "SENTINEL", "METEOSAT", "COPERNICUS", "CERES", "CSO"],
        "KR": ["KOMPSAT", "ARIRANG"],
        "IL": ["OFEK", "EROS"],
        "INT": ["ISS"],
    }
    for country, keywords in country_hints.items():
        for kw in keywords:
            if kw in combined:
                return country
    # Use owner field if available
    if owner:
        return owner[:4].strip()
    return "UNK"


def fetch_earthquakes() -> None:
    """Fetch recent earthquakes from USGS."""
    try:
        data = fetch_json(
            "https://earthquake.usgs.gov/earthquakes/feed/v1.0/summary/all_day.geojson",
            timeout=15,
        )
        if not data or "features" not in data:
            return

        quakes = []
        for f in data["features"]:
            props = f.get("properties", {})
            coords = f.get("geometry", {}).get("coordinates", [])
            if len(coords) < 3:
                continue
            quakes.append({
                "id": f.get("id", ""),
                "mag": props.get("mag", 0),
                "place": props.get("place", ""),
                "lat": round(coords[1], 4),
                "lon": round(coords[0], 4),
                "depth_km": round(coords[2], 1),
                "time": props.get("time", 0),
                "url": props.get("url", ""),
            })

        latest_data["earthquakes"] = quakes
        source_timestamps["earthquakes"] = datetime.now(timezone.utc).isoformat()
        logger.info("Earthquakes: %d events", len(quakes))

    except Exception as e:
        logger.error("fetch_earthquakes error: %s", e, exc_info=True)


def fetch_fires() -> None:
    """Fetch active fire data from NASA FIRMS."""
    try:
        firms_key = os.environ.get("NASA_FIRMS_KEY", "")
        if firms_key:
            url = f"https://firms.modaps.eosdis.nasa.gov/api/area/csv/{firms_key}/VIIRS_SNPP_NRT/world/1"
        else:
            # Use the open data feed
            url = "https://firms.modaps.eosdis.nasa.gov/data/active_fire/suomi-npp-viirs-c2/csv/SUOMI_VIIRS_C2_Global_24h.csv"

        text = fetch_text(url, timeout=30)
        if not text:
            return

        fires = []
        lines = text.strip().split("\n")
        if len(lines) < 2:
            return

        header = lines[0].split(",")
        # Find column indices
        col_map = {col.strip().lower(): i for i, col in enumerate(header)}
        lat_idx = col_map.get("latitude", col_map.get("lat"))
        lon_idx = col_map.get("longitude", col_map.get("lon"))
        bright_idx = col_map.get("bright_ti4", col_map.get("brightness"))
        frp_idx = col_map.get("frp")
        date_idx = col_map.get("acq_date")
        conf_idx = col_map.get("confidence")

        if lat_idx is None or lon_idx is None:
            logger.warning("FIRMS CSV: could not find lat/lon columns in: %s", header)
            return

        for line in lines[1:5001]:  # Cap at 5000
            parts = line.split(",")
            try:
                fires.append({
                    "lat": round(float(parts[lat_idx]), 3),
                    "lon": round(float(parts[lon_idx]), 3),
                    "brightness": float(parts[bright_idx]) if bright_idx is not None and bright_idx < len(parts) else 0,
                    "frp": float(parts[frp_idx]) if frp_idx is not None and frp_idx < len(parts) else 0,
                    "acq_date": parts[date_idx] if date_idx is not None and date_idx < len(parts) else "",
                    "confidence": parts[conf_idx] if conf_idx is not None and conf_idx < len(parts) else "",
                })
            except (ValueError, IndexError):
                continue

        latest_data["fires"] = fires
        source_timestamps["fires"] = datetime.now(timezone.utc).isoformat()
        logger.info("Fires: %d hotspots", len(fires))

    except Exception as e:
        logger.error("fetch_fires error: %s", e, exc_info=True)


def fetch_gdelt() -> None:
    """Fetch conflict/military events from GDELT v2 Article API (geo endpoint is dead)."""
    try:
        # Use v2 doc API — artlist mode returns articles with metadata
        data = fetch_json(
            "https://api.gdeltproject.org/api/v2/doc/doc",
            params={
                "query": "(conflict OR military OR attack OR airstrike OR missile)",
                "mode": "artlist",
                "maxrecords": "250",
                "format": "json",
                "timespan": "24h",
                "sourcelang": "English",
            },
            timeout=20,
        )
        if not data or "articles" not in data:
            return

        events = []
        for art in data["articles"]:
            title = art.get("title", "")
            if not title:
                continue
            # Geocode from title text since article API has no coordinates
            lat, lon = _geocode_text(title)
            if lat is None:
                lat, lon = 0, 0
            events.append({
                "title": title[:300],
                "lat": lat,
                "lon": lon,
                "tone": 0,
                "url": art.get("url", ""),
                "domain": art.get("domain", ""),
                "date": art.get("seendate", ""),
                "source_country": art.get("sourcecountry", ""),
                "language": art.get("language", ""),
            })

        latest_data["gdelt"] = events
        source_timestamps["gdelt"] = datetime.now(timezone.utc).isoformat()
        logger.info("GDELT: %d conflict articles", len(events))

    except Exception as e:
        logger.error("fetch_gdelt error: %s", e, exc_info=True)


def fetch_space_weather() -> None:
    """Fetch planetary K-index from NOAA SWPC."""
    try:
        data = fetch_json(
            "https://services.swpc.noaa.gov/products/noaa-planetary-k-index.json",
            timeout=10,
        )
        if not data or len(data) < 2:
            return

        # Data format: [["time_tag", "Kp", "Kp_fraction", ...], [...], ...]
        # First row is header, last row is latest
        latest_row = data[-1]
        try:
            kp_value = float(latest_row[1])
        except (ValueError, IndexError):
            kp_value = 0

        if kp_value < 4:
            storm_level = "quiet"
        elif kp_value < 6:
            storm_level = "active"
        else:
            storm_level = "storm"

        latest_data["space_weather"] = [{
            "kp_index": kp_value,
            "timestamp": latest_row[0] if latest_row else "",
            "storm_level": storm_level,
        }]
        source_timestamps["space_weather"] = datetime.now(timezone.utc).isoformat()
        logger.info("Space weather: Kp=%.1f (%s)", kp_value, storm_level)

    except Exception as e:
        logger.error("fetch_space_weather error: %s", e, exc_info=True)


def fetch_internet_outages() -> None:
    """Fetch internet outage signals from IODA."""
    try:
        data = fetch_json(
            "https://ioda.inetintel.cc.gatech.edu/api/v2/signals/raw/country",
            params={"from": "-3600"},
            timeout=15,
        )
        if not data:
            # Fallback: generate summary from known monitoring
            latest_data["internet_outages"] = []
            source_timestamps["internet_outages"] = datetime.now(timezone.utc).isoformat()
            return

        # Extract top outages by severity
        outages = []
        results = data if isinstance(data, list) else data.get("data", data.get("results", []))
        if isinstance(results, list):
            for entry in results[:100]:
                if isinstance(entry, dict):
                    outages.append({
                        "country": entry.get("entity", {}).get("name", entry.get("country", "")),
                        "country_code": entry.get("entity", {}).get("code", entry.get("code", "")),
                        "severity": entry.get("severity", entry.get("score", 0)),
                        "datasource": entry.get("datasource", ""),
                        "timestamp": entry.get("from", entry.get("timestamp", "")),
                    })

        latest_data["internet_outages"] = outages
        source_timestamps["internet_outages"] = datetime.now(timezone.utc).isoformat()
        logger.info("Internet outages: %d signals", len(outages))

    except Exception as e:
        logger.error("fetch_internet_outages error: %s", e, exc_info=True)


def fetch_news() -> None:
    """Fetch and geocode articles from OSINT/security RSS feeds."""
    try:
        feeds = _load_news_feeds()
        articles = []

        for feed_conf in feeds:
            try:
                parsed = feedparser.parse(feed_conf["url"])
                for entry in parsed.entries[:20]:
                    title = entry.get("title", "")
                    link = entry.get("link", "")
                    summary = entry.get("summary", entry.get("description", ""))
                    # Strip HTML tags from summary
                    summary = re.sub(r"<[^>]+>", "", summary)[:500]
                    published = entry.get("published", entry.get("updated", ""))

                    # Geocode by keyword matching
                    lat, lon = _geocode_text(f"{title} {summary}")

                    articles.append({
                        "title": title[:300],
                        "link": link,
                        "source": feed_conf["name"],
                        "published": published,
                        "summary": summary,
                        "weight": feed_conf.get("weight", 3),
                        "lat": lat,
                        "lon": lon,
                    })
            except Exception as e:
                logger.debug("Error parsing feed %s: %s", feed_conf["name"], e)
                continue

        # Sort by weight descending, then by published date
        articles.sort(key=lambda a: a.get("weight", 0), reverse=True)

        latest_data["news"] = articles[:500]
        source_timestamps["news"] = datetime.now(timezone.utc).isoformat()
        logger.info("News: %d articles from %d feeds", len(latest_data["news"]), len(feeds))

    except Exception as e:
        logger.error("fetch_news error: %s", e, exc_info=True)


def _load_news_feeds() -> list[dict]:
    """Load RSS feed configuration."""
    try:
        with open(NEWS_FEEDS_FILE) as f:
            return json.load(f)
    except Exception:
        # Fallback defaults
        return [
            {"name": "BBC World", "url": "http://feeds.bbci.co.uk/news/world/rss.xml", "weight": 3},
            {"name": "The Hacker News", "url": "https://feeds.feedburner.com/TheHackersNews", "weight": 5},
        ]


def _geocode_text(text: str) -> tuple[Optional[float], Optional[float]]:
    """Simple keyword-based geocoding. Returns (lat, lon) or (None, None)."""
    text_upper = text.upper()
    # Check longest names first to avoid partial matches
    sorted_countries = sorted(COUNTRY_COORDS.keys(), key=len, reverse=True)
    for country in sorted_countries:
        if country.upper() in text_upper:
            lat, lon = COUNTRY_COORDS[country]
            return lat, lon
    return None, None


def fetch_kiwisdr() -> None:
    """Fetch KiwiSDR receiver locations from the public listing."""
    try:
        text = fetch_text("http://rx.linkfanel.net/kiwisdr_com.js", timeout=15)
        if not text:
            text = fetch_text("http://kiwisdr.com/public/", timeout=15)
        if not text:
            return

        receivers = []

        # The kiwisdr_com.js file is JavaScript with JSON array of objects.
        # Each entry has: "name":"...", "gps":"(lat, lon)", "url":"...", "users":"N", "users_max":"N"
        # Strip the JS wrapper to get the JSON array
        json_start = text.find("[")
        json_end = text.rfind("]")
        if json_start >= 0 and json_end > json_start:
            try:
                json_str = text[json_start:json_end + 1]
                # Remove trailing commas before ] (JS allows them, JSON doesn't)
                json_str = re.sub(r",\s*]", "]", json_str)
                entries = json.loads(json_str)
                for entry in entries:
                    if not isinstance(entry, dict):
                        continue
                    gps_str = entry.get("gps", "")
                    name = entry.get("name", "")
                    url = entry.get("url", "")
                    if not gps_str or not name:
                        continue
                    # Parse "(lat, lon)" format
                    gps_match = re.match(r"\((-?\d+\.?\d*),\s*(-?\d+\.?\d*)\)", gps_str)
                    if not gps_match:
                        continue
                    try:
                        lat = float(gps_match.group(1))
                        lon = float(gps_match.group(2))
                    except ValueError:
                        continue
                    users = 0
                    try:
                        users = int(entry.get("users", 0))
                    except (ValueError, TypeError):
                        pass
                    # Only include active receivers
                    if entry.get("offline") == "yes":
                        continue
                    receivers.append({
                        "name": name[:120],
                        "lat": round(lat, 3),
                        "lon": round(lon, 3),
                        "url": url,
                        "bands": "0-30 MHz",
                        "users_active": users,
                    })
            except json.JSONDecodeError:
                logger.warning("KiwiSDR: failed to parse JSON from JS file")

        if receivers:
            latest_data["kiwisdr"] = receivers[:1000]
            source_timestamps["kiwisdr"] = datetime.now(timezone.utc).isoformat()
            logger.info("KiwiSDR: %d receivers", len(receivers))
        else:
            logger.warning("KiwiSDR: no receivers parsed from response (%d bytes)", len(text))

    except Exception as e:
        logger.error("fetch_kiwisdr error: %s", e, exc_info=True)


def fetch_datacenters() -> None:
    """Load static data center locations (runs once at startup)."""
    try:
        datacenters = [
            # AWS
            {"name": "AWS US-East-1", "operator": "AWS", "lat": 39.04, "lon": -77.49, "region": "us-east-1"},
            {"name": "AWS US-West-2", "operator": "AWS", "lat": 45.59, "lon": -122.60, "region": "us-west-2"},
            {"name": "AWS EU-West-1", "operator": "AWS", "lat": 53.35, "lon": -6.26, "region": "eu-west-1"},
            {"name": "AWS EU-Central-1", "operator": "AWS", "lat": 50.11, "lon": 8.68, "region": "eu-central-1"},
            {"name": "AWS AP-Southeast-1", "operator": "AWS", "lat": 1.35, "lon": 103.82, "region": "ap-southeast-1"},
            {"name": "AWS AP-Northeast-1", "operator": "AWS", "lat": 35.68, "lon": 139.77, "region": "ap-northeast-1"},
            {"name": "AWS SA-East-1", "operator": "AWS", "lat": -23.55, "lon": -46.63, "region": "sa-east-1"},
            {"name": "AWS AP-South-1", "operator": "AWS", "lat": 19.08, "lon": 72.88, "region": "ap-south-1"},
            {"name": "AWS ME-South-1", "operator": "AWS", "lat": 26.07, "lon": 50.56, "region": "me-south-1"},
            {"name": "AWS AF-South-1", "operator": "AWS", "lat": -33.93, "lon": 18.42, "region": "af-south-1"},
            # Google Cloud
            {"name": "Google US-Central1", "operator": "Google", "lat": 41.26, "lon": -95.86, "region": "us-central1"},
            {"name": "Google US-East1", "operator": "Google", "lat": 33.21, "lon": -80.01, "region": "us-east1"},
            {"name": "Google Europe-West1", "operator": "Google", "lat": 50.45, "lon": 3.82, "region": "europe-west1"},
            {"name": "Google Europe-West4", "operator": "Google", "lat": 53.45, "lon": 6.73, "region": "europe-west4"},
            {"name": "Google Asia-East1", "operator": "Google", "lat": 24.05, "lon": 120.52, "region": "asia-east1"},
            {"name": "Google Asia-Southeast1", "operator": "Google", "lat": 1.37, "lon": 103.98, "region": "asia-southeast1"},
            {"name": "Google Australia-SE1", "operator": "Google", "lat": -33.86, "lon": 151.21, "region": "australia-southeast1"},
            # Azure
            {"name": "Azure East US", "operator": "Azure", "lat": 37.38, "lon": -79.44, "region": "eastus"},
            {"name": "Azure West US", "operator": "Azure", "lat": 37.78, "lon": -122.42, "region": "westus"},
            {"name": "Azure West Europe", "operator": "Azure", "lat": 52.37, "lon": 4.90, "region": "westeurope"},
            {"name": "Azure North Europe", "operator": "Azure", "lat": 53.35, "lon": -6.26, "region": "northeurope"},
            {"name": "Azure Southeast Asia", "operator": "Azure", "lat": 1.28, "lon": 103.84, "region": "southeastasia"},
            {"name": "Azure Japan East", "operator": "Azure", "lat": 35.68, "lon": 139.77, "region": "japaneast"},
            {"name": "Azure Brazil South", "operator": "Azure", "lat": -23.55, "lon": -46.63, "region": "brazilsouth"},
            # Equinix
            {"name": "Equinix SV5 Silicon Valley", "operator": "Equinix", "lat": 37.39, "lon": -121.98, "region": "sv"},
            {"name": "Equinix NY5 New York", "operator": "Equinix", "lat": 40.77, "lon": -74.07, "region": "ny"},
            {"name": "Equinix LD8 London", "operator": "Equinix", "lat": 51.52, "lon": -0.03, "region": "ld"},
            {"name": "Equinix FR5 Frankfurt", "operator": "Equinix", "lat": 50.10, "lon": 8.63, "region": "fr"},
            {"name": "Equinix SG3 Singapore", "operator": "Equinix", "lat": 1.32, "lon": 103.82, "region": "sg"},
            {"name": "Equinix TY2 Tokyo", "operator": "Equinix", "lat": 35.63, "lon": 139.75, "region": "ty"},
            {"name": "Equinix HK1 Hong Kong", "operator": "Equinix", "lat": 22.37, "lon": 114.12, "region": "hk"},
            {"name": "Equinix SY4 Sydney", "operator": "Equinix", "lat": -33.93, "lon": 151.19, "region": "sy"},
            {"name": "Equinix AM5 Amsterdam", "operator": "Equinix", "lat": 52.29, "lon": 4.94, "region": "am"},
            # Other major
            {"name": "Digital Realty Ashburn", "operator": "Digital Realty", "lat": 39.04, "lon": -77.49, "region": "ash"},
            {"name": "CyrusOne Dallas", "operator": "CyrusOne", "lat": 32.90, "lon": -96.99, "region": "dfw"},
            {"name": "CoreSite Los Angeles", "operator": "CoreSite", "lat": 34.05, "lon": -118.26, "region": "lax"},
            {"name": "NTT Tokyo", "operator": "NTT", "lat": 35.69, "lon": 139.69, "region": "nrt"},
            {"name": "Interxion Marseille", "operator": "Interxion", "lat": 43.30, "lon": 5.37, "region": "mrs"},
        ]

        latest_data["datacenters"] = datacenters
        source_timestamps["datacenters"] = datetime.now(timezone.utc).isoformat()
        logger.info("Datacenters: %d facilities loaded", len(datacenters))

    except Exception as e:
        logger.error("fetch_datacenters error: %s", e, exc_info=True)


def fetch_financial() -> None:
    """Fetch Fear & Greed Index and crypto prices."""
    try:
        result = {
            "fear_greed_index": None,
            "fear_greed_label": None,
            "btc_usd": None,
            "eth_usd": None,
            "timestamp": datetime.now(timezone.utc).isoformat(),
        }

        # Fear & Greed Index
        fg_data = fetch_json("https://api.alternative.me/fng/?limit=1", timeout=10)
        if fg_data and "data" in fg_data and fg_data["data"]:
            entry = fg_data["data"][0]
            result["fear_greed_index"] = int(entry.get("value", 0))
            result["fear_greed_label"] = entry.get("value_classification", "")

        # Crypto prices from CoinGecko
        crypto_data = fetch_json(
            "https://api.coingecko.com/api/v3/simple/price",
            params={"ids": "bitcoin,ethereum", "vs_currencies": "usd"},
            timeout=10,
        )
        if crypto_data:
            if "bitcoin" in crypto_data:
                result["btc_usd"] = crypto_data["bitcoin"].get("usd")
            if "ethereum" in crypto_data:
                result["eth_usd"] = crypto_data["ethereum"].get("usd")

        latest_data["financial"] = [result]
        source_timestamps["financial"] = datetime.now(timezone.utc).isoformat()
        logger.info(
            "Financial: F&G=%s, BTC=$%s, ETH=$%s",
            result["fear_greed_index"],
            result["btc_usd"],
            result["eth_usd"],
        )

    except Exception as e:
        logger.error("fetch_financial error: %s", e, exc_info=True)
