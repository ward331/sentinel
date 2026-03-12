# MANIFEST 04 — FEED DASHBOARD, NEWS INTEL & MEDIA WALL
# ========================================================
# Covers: Stage 5 (view modes, text feed, subscriptions, incident threading)
#         Stage 6 (news aggregation, AI briefing, media wall, YouTube embeds)

════════════════════════════════════════════════════════════════
FIVE VIEW MODES
════════════════════════════════════════════════════════════════

Add global view mode selector to header nav bar. Present in ALL views.

MODES:
  GLOBE ONLY    — current dashboard, full 3D globe. URL: /?view=globe
  FEED ONLY     — text alert feed, zero WebGL. URL: /?view=feed
  SPLIT VIEW    — globe left (default 60%), feed right (40%). URL: /?view=split
  MEDIA WALL    — video grid top 70%, alert ticker bottom 30%. URL: /?view=media
  COMMAND CENTER— globe top-left, feed top-right, news bottom-left,
                  media/financial bottom-right. URL: /?view=command

Header nav: [🌍 Globe] [📋 Feed] [⚡ Split] [📺 Media] [🖥️ Command]

Mode persistence: saved via GET/POST /api/layout/preferences
Mobile (width < 768px): default to Feed view

Mode switching: instant, no reload, SSE connections maintained.
Globe pauses rendering when hidden (viewer.useDefaultRenderLoop = false).
URL updates on switch (browser history.pushState).

════════════════════════════════════════════════════════════════
TEXT ALERT FEED
════════════════════════════════════════════════════════════════

FeedManager class (JavaScript):
  Connects to /api/events/stream (existing SSE)
  In-memory ring buffer: last 500 events
  Applies subscription filters
  Auto-scrolls, pauses on hover/touch

FEED HEADER:
  "SENTINEL FEED" | Live count badge | Unread badge
  Filter chips: [All] [🔴 Critical] [✈️ Military] [🌩️ Weather]
                [💥 Conflict] [🚀 Missiles] [🛰️ Space] [🔥 Wildfire]
                [🚢 Maritime] [🦠 Outbreak] [🌐 Cyber] [📈 Financial]
  Search input (live filter as user types)
  [Mark all read] [Clear] [⚙️ Subscriptions]

EVENT CARD:
  Row 1: Severity badge | Category icon+label | Timestamp UTC | Relative time
  Row 2: Event title (bold)
  Row 3: 📍 Location + key stats (varies by category)
  Row 4: Description (1-2 sentences, "more..." expands)
  Row 5: Source link | [View on Globe] [Watch Asset] [⭐ Save] [📤 Share] [💬 Note] [🔗 OSINT]

  Severity colors: green=info, amber=watch, orange=warning, red=alert, pulsing red=critical

CARD VARIANTS (additional fields by category):
  Earthquake: depth, felt reports, ShakeMap link, aftershock probability
  Military aircraft: type, registration, operator, squawk (red if 7700/7600/7500)
  Missile/air raid: pulsing red background, time-to-impact if known, shelter link
  Space weather: Kp value, flare class, aurora visibility, CME arrival
  Disease: case count, fatalities, countries, WHO level
  Financial: price/change, correlation insight if present, source link
  Wildfire: fire radius, confidence, wind direction, evacuation status if known

INCIDENT THREADING:
  3+ events in same region within 30 minutes → group into expandable incident card
  Incident card stays pinned while active, auto-closes 1hr after last related event

PRIORITY PINNING:
  TIER 3: pinned top, orange left border, until acknowledged
  TIER 4: pinned top, pulsing red border + background, alarm sound, [ACKNOWLEDGE] required

LIVE TICKER:
  Thin strip at bottom of screen (all view modes)
  "🔴 MISSILE ALERT — ISRAEL ••• ⚡ M5.2 EARTHQUAKE — TURKEY •••"
  Toggle in settings, speed: slow/medium/fast, min severity filter

════════════════════════════════════════════════════════════════
SUBSCRIPTION SYSTEM
════════════════════════════════════════════════════════════════

SQLite: user_subscriptions, watchlist_regions, watchlist_keywords, watchlist_assets

Category subscriptions: toggle per category + per-category threshold
Severity filter: all / watch+ / warning+ / alert+

REGION WATCHLIST:
  Add: type country name OR click on mini-map OR select from list
  Events in watched regions: shown regardless of category filter, highlighted border

KEYWORD WATCHLIST:
  Keywords highlighted in yellow in event cards
  Match: case-insensitive, partial word
  Examples: missile, nuclear, carrier, hypersonic, wagner, hamas, hezbollah

ASSET WATCHLIST:
  Track: aircraft (ICAO hex or callsign), vessel (MMSI or name), satellite (NORAD ID)
  Matched events: "👁️ WATCHED ASSET" badge

OSINT SOURCE PANEL (slides out from [🔗 OSINT] button on any card):
  Relevant live streams (auto-suggested by region + category)
  Relevant OSINT YouTube channels for this event type
  Broadcastify scanner feed for event region
  Public Telegram channels (as links, not embedded)
  Relevant subreddits
  Official source links for this event type

════════════════════════════════════════════════════════════════
NEWS INTELLIGENCE AGGREGATOR
════════════════════════════════════════════════════════════════

New provider: NewsAggregator
  Polls all RSS feeds on staggered schedule
  Extracts: title, link, description, pubDate, source
  Geocodes location mentions (built-in place name dictionary, no external API)
  Relevance score: count of OSINT keywords matched
  Cross-references: headline location within 100km of active SENTINEL event
  Deduplicates by URL
  Stores in: news_items table

PRE-LOADED RSS SOURCES:
  OSINT/ANALYSIS:
    Bellingcat:         https://www.bellingcat.com/feed/
    War on the Rocks:   https://warontherocks.com/feed/
    CSIS:               https://www.csis.org/rss/reports.xml
    ISW Ukraine:        https://understandingwar.org/rss.xml
    The War Zone:       https://www.thedrive.com/the-war-zone/rss

  MILITARY/DEFENSE:
    Defense News:       https://www.defensenews.com/arc/outboundfeeds/rss/
    Naval News:         https://www.navalnews.com/feed/
    Breaking Defense:   https://breakingdefense.com/feed/
    USNI News:          https://news.usni.org/feed

  SPACE:
    SpaceFlightNow:     https://spaceflightnow.com/feed/
    NASASpaceFlight:    https://www.nasaspaceflight.com/feed/
    Space.com:          https://www.space.com/feeds/all

  GEOPOLITICAL:
    Reuters World:      https://feeds.reuters.com/reuters/worldNews
    Al Jazeera:         https://www.aljazeera.com/xml/rss/all.xml
    BBC World:          https://feeds.bbci.co.uk/news/world/rss.xml
    Foreign Policy:     https://foreignpolicy.com/feed/
    The Diplomat:       https://thediplomat.com/feed/

  CYBER:
    Krebs on Security:  https://krebsonsecurity.com/feed/
    Schneier:           https://www.schneier.com/feed/atom/
    Dark Reading:       https://www.darkreading.com/rss.xml

  FINANCIAL/MACRO:
    Reuters Business:   https://feeds.reuters.com/reuters/businessNews
    FT (free):          https://www.ft.com/rss/home
    MarketWatch:        https://feeds.marketwatch.com/marketwatch/topstories/
    Investopedia:       https://www.investopedia.com/feedbuilder/feed/getfeed?feedName=rss_headline

  DISASTER/HEALTH:
    ReliefWeb:          https://reliefweb.int/updates/rss.xml
    UN News:            https://news.un.org/feed/subscribe/en/news/all/rss.xml
    WHO:                https://www.who.int/rss-feeds/news-english.xml

NEWS CARD:
  Relevance score badge | Category | Source | Time
  Headline (bold)
  📍 Matched active event badge (if cross-ref found)
  Excerpt (2 sentences)
  [Read Article] [⭐ Save] [📤 Share] [🗺️ Show on Globe]

AI BRIEFING PANEL (top of news panel):
  Updates every 30 minutes via GET /api/intel/news-briefing
  Takes top 20 news + top 5 SENTINEL events → DeepSeek summary
  Output: 3 bullet points, max 2 sentences each, specific/factual
  [Refresh] button (rate-limited 1/5min)
  Cached 30 minutes, served to all clients

════════════════════════════════════════════════════════════════
OSINT MEDIA WALL
════════════════════════════════════════════════════════════════

Used in Media Wall view and Command Center.

GRID LAYOUTS: [1×1] [2×1] [2×2] [3×2] [4×2]

Each slot: stream selector + controls
  [🔊 Unmute] solo this stream, mute others
  [📌 Pin] prevent auto-swap
  [⛶ Fullscreen] expand slot
  [↗ PiP] browser Picture-in-Picture API
  [✕] clear slot

STREAM SOURCES (pre-loaded):
  Al Jazeera English Live:  UCNye-wNBqNL5ZzHSJj3l8Bg
  France24 English:         UCQfwfsi5VrQ8yKZ-UWmAoBw
  DW News:                  UCknLrEdhRCp1aegoMqRaCZg
  Sky News:                 UCoMdktPbSTixAyNGwb-UYkQ
  AP Live:                  UCkUQnvUpgWsYkNf_I5gWkpg
  WION:                     UCrQMZwctQFCQCV80bWkbHOw
  NHK World:                UC6KMYvqFHrBQP3hNPbFxAkA
  NASA ISS Live:            UCLA_DiR1FfKNvjuUpBHmylQ
  NASA TV:                  UCLA_DiR1FfKNvjuUpBHmylQ
  [Enter custom YouTube URL or HLS stream URL]

YouTube embed: ?autoplay=1&mute=1&controls=1&rel=0&modestbranding=1

Stream health: green dot if loaded, gray + "OFFLINE" if failed, retry 30s

Auto-suggest on TIER 3+ alert:
  Middle East event → suggest Al Jazeera, i24 (if available)
  Ukraine event → DW, France24
  Space event → NASA TV
  Hurricane → NHK, AP

Media presets saved in SQLite: media_presets table
Pre-loaded presets: "Breaking News", "Space Ops", "Conflict Watch"

════════════════════════════════════════════════════════════════
DATABASE SCHEMA (FEED/NEWS/MEDIA)
════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS news_items (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  title TEXT NOT NULL,
  url TEXT UNIQUE NOT NULL,
  description TEXT,
  source_name TEXT,
  source_category TEXT,
  pub_date DATETIME,
  ingested_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  relevance_score INTEGER DEFAULT 0,
  location_lat REAL,
  location_lon REAL,
  matched_event_id INTEGER,
  is_read INTEGER DEFAULT 0,
  is_saved INTEGER DEFAULT 0
);

CREATE TABLE IF NOT EXISTS user_feeds (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  url TEXT UNIQUE NOT NULL,
  category TEXT,
  credibility_tag TEXT,  -- "official", "verified_osint", "news", "unverified"
  enabled INTEGER DEFAULT 1,
  is_builtin INTEGER DEFAULT 0,
  last_fetched DATETIME,
  fetch_interval_seconds INTEGER DEFAULT 1800,
  error_count INTEGER DEFAULT 0
);

CREATE TABLE IF NOT EXISTS user_subscriptions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  category TEXT,
  severity_min TEXT,
  enabled INTEGER DEFAULT 1,
  notify_telegram INTEGER DEFAULT 0,
  notify_email INTEGER DEFAULT 0,
  notify_ntfy INTEGER DEFAULT 0,
  threshold_value TEXT
);

CREATE TABLE IF NOT EXISTS watchlist_regions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  country_code TEXT,
  lat REAL,
  lon REAL,
  radius_km INTEGER DEFAULT 200,
  notify_all INTEGER DEFAULT 1
);

CREATE TABLE IF NOT EXISTS watchlist_keywords (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  keyword TEXT NOT NULL,
  case_sensitive INTEGER DEFAULT 0,
  notify_telegram INTEGER DEFAULT 1,
  notify_email INTEGER DEFAULT 0
);

CREATE TABLE IF NOT EXISTS watchlist_assets (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  asset_type TEXT,
  identifier TEXT NOT NULL,
  display_name TEXT,
  notify INTEGER DEFAULT 1
);

CREATE TABLE IF NOT EXISTS event_saves (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  event_id INTEGER,
  note TEXT,
  saved_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS media_presets (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  layout TEXT,
  slots_json TEXT,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS layout_preferences (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  device_hint TEXT,
  view_mode TEXT DEFAULT 'globe',
  split_ratio INTEGER DEFAULT 60,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

════════════════════════════════════════════════════════════════
NEW API ENDPOINTS (FEED/NEWS/MEDIA)
════════════════════════════════════════════════════════════════

GET  /api/feed                        — paginated feed (applies subscriptions)
GET  /api/feed/subscriptions          — subscription settings
POST /api/feed/subscriptions          — update subscriptions
GET  /api/feed/watchlist              — all watchlist items
POST /api/feed/watchlist              — add item
DEL  /api/feed/watchlist/{id}         — remove item
GET  /api/news                        — paginated news
POST /api/news/sources                — add custom RSS
GET  /api/news/sources                — list all sources
DEL  /api/news/sources/{id}           — remove source
POST /api/news/{id}/save              — bookmark article
GET  /api/intel/news-briefing         — AI news summary (cached 30min)
GET  /api/events/{id}/osint-sources   — suggested OSINT sources for event
POST /api/events/{id}/save            — bookmark event
POST /api/events/{id}/note            — add note
GET  /api/media/presets               — media presets
POST /api/media/presets               — save preset
DEL  /api/media/presets/{id}          — delete preset
GET  /api/layout/preferences          — view mode prefs
POST /api/layout/preferences          — save prefs
