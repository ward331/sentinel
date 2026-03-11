"""
Region dossier service for SENTINEL V4.
Reverse-geocodes coordinates to a country, then fetches a Wikipedia summary.
Results are cached for 1 hour.
"""

import logging
import time
from typing import Optional

from services.network_utils import fetch_json

logger = logging.getLogger("sentinel.dossier")

# In-memory cache: key = "lat,lng" rounded to 1 decimal -> (timestamp, result)
_cache: dict[str, tuple[float, dict]] = {}
CACHE_TTL = 3600  # 1 hour


def _cache_key(lat: float, lng: float) -> str:
    return f"{round(lat, 1)},{round(lng, 1)}"


def get_region_dossier(lat: float, lng: float) -> dict:
    """
    Build a dossier for the region at (lat, lng):
    1. Reverse geocode via Nominatim
    2. Fetch Wikipedia summary for the country
    Returns a dict with country info, summary, and thumbnail.
    """
    key = _cache_key(lat, lng)
    now = time.time()

    # Check cache
    if key in _cache:
        cached_time, cached_result = _cache[key]
        if now - cached_time < CACHE_TTL:
            return cached_result

    result = _build_dossier(lat, lng)

    # Cache the result
    _cache[key] = (now, result)

    # Evict old entries if cache grows too large
    if len(_cache) > 500:
        _evict_old_entries(now)

    return result


def _build_dossier(lat: float, lng: float) -> dict:
    """Reverse geocode + Wikipedia lookup."""
    dossier = {
        "lat": lat,
        "lng": lng,
        "country": None,
        "country_code": None,
        "display_name": None,
        "capital": None,
        "population": None,
        "leader": None,
        "wikipedia_summary": None,
        "thumbnail_url": None,
    }

    # Step 1: Reverse geocode via Nominatim
    geo = _reverse_geocode(lat, lng)
    if geo:
        dossier["country"] = geo.get("country")
        dossier["country_code"] = geo.get("country_code", "").upper()
        dossier["display_name"] = geo.get("display_name")

    # Step 2: Wikipedia summary for the country
    if dossier["country"]:
        wiki = _fetch_wikipedia_summary(dossier["country"])
        if wiki:
            dossier["wikipedia_summary"] = wiki.get("summary")
            dossier["thumbnail_url"] = wiki.get("thumbnail")
            dossier["capital"] = wiki.get("capital")
            dossier["population"] = wiki.get("population")

    return dossier


def _reverse_geocode(lat: float, lng: float) -> Optional[dict]:
    """Use Nominatim to reverse-geocode lat/lng to a location."""
    url = "https://nominatim.openstreetmap.org/reverse"
    data = fetch_json(
        url,
        timeout=10,
        params={
            "lat": str(lat),
            "lon": str(lng),
            "format": "json",
            "zoom": 3,
            "accept-language": "en",
        },
        headers={"User-Agent": "SENTINEL-Watchtower/4.0 (research project)"},
    )
    if not data or "error" in data:
        logger.warning("Reverse geocode failed for %s,%s", lat, lng)
        return None

    address = data.get("address", {})
    return {
        "country": address.get("country"),
        "country_code": address.get("country_code", ""),
        "display_name": data.get("display_name"),
    }


def _fetch_wikipedia_summary(country_name: str) -> Optional[dict]:
    """Fetch Wikipedia summary and extract key facts about a country."""
    url = f"https://en.wikipedia.org/api/rest_v1/page/summary/{country_name}"
    data = fetch_json(url, timeout=10)
    if not data or data.get("type") == "not_found":
        logger.warning("Wikipedia lookup failed for %s", country_name)
        return None

    summary = data.get("extract", "")
    thumbnail = None
    if "thumbnail" in data:
        thumbnail = data["thumbnail"].get("source")

    # Try to extract capital and population from the summary text
    capital = _extract_fact(summary, "capital")
    population = _extract_fact(summary, "population")

    return {
        "summary": summary[:2000],  # Cap length
        "thumbnail": thumbnail,
        "capital": capital,
        "population": population,
    }


def _extract_fact(text: str, fact_type: str) -> Optional[str]:
    """
    Simple keyword extraction from Wikipedia summary text.
    Not perfect but good enough for display purposes.
    """
    import re

    text_lower = text.lower()

    if fact_type == "capital":
        # Common patterns: "capital is X", "capital, X,"
        patterns = [
            r"capital(?:\s+city)?\s+(?:is|of)\s+([A-Z][a-zA-Z\s]+?)(?:\.|,|\s+and)",
            r"capital\s*,\s*([A-Z][a-zA-Z\s]+?)(?:\.|,)",
        ]
        for pat in patterns:
            m = re.search(pat, text)
            if m:
                return m.group(1).strip()

    elif fact_type == "population":
        patterns = [
            r"population\s+of\s+(?:about\s+|approximately\s+|over\s+)?([0-9,.]+\s*(?:million|billion)?)",
            r"([0-9,.]+\s*(?:million|billion)?)\s+(?:people|inhabitants|residents)",
        ]
        for pat in patterns:
            m = re.search(pat, text, re.IGNORECASE)
            if m:
                return m.group(1).strip()

    return None


def _evict_old_entries(now: float) -> None:
    """Remove expired cache entries."""
    expired = [k for k, (t, _) in _cache.items() if now - t > CACHE_TTL]
    for k in expired:
        del _cache[k]
