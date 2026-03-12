"""
Network utilities for SENTINEL V4 data fetchers.
Provides resilient HTTP fetch helpers with timeout, retry, and logging.
"""

import logging
import time
from typing import Optional, Any

import httpx

logger = logging.getLogger("sentinel.network")

_client: Optional[httpx.Client] = None


def get_client() -> httpx.Client:
    """Lazy-init a shared httpx client with connection pooling."""
    global _client
    if _client is None or _client.is_closed:
        _client = httpx.Client(
            timeout=httpx.Timeout(20.0, connect=10.0),
            follow_redirects=True,
            limits=httpx.Limits(max_connections=50, max_keepalive_connections=20),
            headers={
                "User-Agent": "SENTINEL-Watchtower/4.0 (research; +https://github.com/sentinel)"
            },
        )
    return _client


def close_client() -> None:
    """Close the shared client (call on shutdown)."""
    global _client
    if _client is not None and not _client.is_closed:
        _client.close()
        _client = None


def fetch_json(
    url: str,
    timeout: float = 15.0,
    retries: int = 2,
    headers: Optional[dict] = None,
    params: Optional[dict] = None,
) -> Optional[Any]:
    """
    GET a URL and return parsed JSON, or None on failure.
    Retries on transient errors with exponential backoff.
    """
    client = get_client()
    last_err = None
    for attempt in range(1, retries + 1):
        try:
            resp = client.get(
                url,
                timeout=timeout,
                headers=headers or {},
                params=params or {},
            )
            resp.raise_for_status()
            return resp.json()
        except httpx.TimeoutException as e:
            last_err = e
            logger.warning("Timeout fetching %s (attempt %d/%d)", url, attempt, retries)
        except httpx.HTTPStatusError as e:
            last_err = e
            status = e.response.status_code
            if status in (429, 500, 502, 503, 504) and attempt < retries:
                logger.warning("HTTP %d from %s, retrying...", status, url)
            else:
                logger.error("HTTP %d from %s: %s", status, url, str(e)[:200])
                return None
        except Exception as e:
            last_err = e
            logger.error("Error fetching JSON from %s: %s", url, str(e)[:200])
            if attempt >= retries:
                return None
        if attempt < retries:
            time.sleep(min(2 ** attempt, 8))

    logger.error("All %d retries exhausted for %s: %s", retries, url, last_err)
    return None


def fetch_text(
    url: str,
    timeout: float = 15.0,
    retries: int = 2,
    headers: Optional[dict] = None,
) -> Optional[str]:
    """
    GET a URL and return response text, or None on failure.
    """
    client = get_client()
    last_err = None
    for attempt in range(1, retries + 1):
        try:
            resp = client.get(url, timeout=timeout, headers=headers or {})
            resp.raise_for_status()
            return resp.text
        except httpx.TimeoutException as e:
            last_err = e
            logger.warning("Timeout fetching %s (attempt %d/%d)", url, attempt, retries)
        except httpx.HTTPStatusError as e:
            last_err = e
            logger.error("HTTP %d from %s", e.response.status_code, url)
            if attempt >= retries:
                return None
        except Exception as e:
            last_err = e
            logger.error("Error fetching text from %s: %s", url, str(e)[:200])
            if attempt >= retries:
                return None
        if attempt < retries:
            time.sleep(min(2 ** attempt, 8))

    logger.error("All retries exhausted for %s: %s", retries, url, last_err)
    return None
