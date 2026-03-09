# MANIFEST 03 — FINANCIAL ALERTS
# ================================
# Covers: Stage 4 — Financial data providers, alert types,
#         geopolitical correlation engine, financial feed panel.
#
# Philosophy: Financial markets are geopolitical sensors.
# A sudden oil spike, gold rush, or currency crash often precedes
# or accompanies conflict escalation. SENTINEL treats financial
# alerts as OSINT, not as trading signals.

════════════════════════════════════════════════════════════════
FINANCIAL DATA PROVIDERS — FREE SOURCES
════════════════════════════════════════════════════════════════

All implemented in internal/providers/financial/ package.

── ZERO-KEY SOURCES (always available) ────────────────────────

1. CBOE VIX (Volatility Index — "Fear Gauge")
   URL: https://cdn.cboe.com/api/global/us_indices/daily_prices/VIX_History.csv
   Also: https://cdn.cboe.com/api/global/us_indices/daily_prices/VIX_Current.json
   Interval: 60s
   Fields: date, open, high, low, close
   Alerts:
     VIX > 20:  WATCH    — elevated fear/uncertainty
     VIX > 30:  WARNING  — high fear (2008/2020 levels)
     VIX > 40:  ALERT    — extreme fear / market stress
     VIX > 50:  CRITICAL — crisis level (rare)
     VIX spike: if VIX moves +5 points in 1 hour → WARNING

2. CoinGecko (Cryptocurrency — no key ever)
   URL: https://api.coingecko.com/api/v3/simple/price?ids=bitcoin,ethereum,tether&vs_currencies=usd&include_24hr_change=true&include_market_cap=true
   URL: https://api.coingecko.com/api/v3/global (market cap totals)
   Interval: 60s
   Rate limit: 10-30 req/min (free, no key)
   Alerts per asset (configurable threshold, default 10% / 1hr):
     Flash crash (>10% drop in 1hr): WARNING
     Flash crash (>20% drop in 1hr): ALERT
     Flash pump (>20% rise in 1hr):  WATCH
   Also track: total crypto market cap change

3. CoinCap.io (backup crypto, WebSocket available)
   REST: https://api.coincap.io/v2/assets?ids=bitcoin,ethereum&limit=10
   WebSocket: wss://ws.coincap.io/prices?assets=bitcoin,ethereum
   Interval: 60s REST or real-time WebSocket
   Use WebSocket if available for real-time price feed

4. US Treasury Yields (free XML feed)
   URL: https://home.treasury.gov/resource-center/data-chart-center/interest-rates/pages/xml?data=daily_treasury_yield_curve&field_tdr_date_value_month={YYYYMM}
   Interval: 3600s
   Fields: 1-month, 3-month, 6-month, 1-year, 2-year, 5-year, 10-year, 30-year
   Alerts:
     2yr/10yr inversion deepens > 50bp: WARNING (recession signal)
     2yr/10yr inversion first appears: WATCH
     10-year yield spikes > 0.2% in one day: WATCH

5. OFAC Sanctions List (US Treasury — free, public)
   URL: https://www.treasury.gov/ofac/downloads/sdnlist.txt
   Also XML: https://www.treasury.gov/ofac/downloads/sdn.xml
   Interval: 3600s
   Detect new entries since last check (compare entry count + names)
   Alert on new SDN additions: WATCH (always geopolitically significant)
   Parse: name, type, program (RUSSIA, IRAN, DPRK, CYBER, etc.)
   Include SDN program name in alert body

6. SEC EDGAR Full-Text Search (free API, no key)
   URL: https://efts.sec.gov/LATEST/search-index?q="material+adverse"&dateRange=custom&startdt={TODAY}&enddt={TODAY}&forms=8-K
   URL: https://www.sec.gov/cgi-bin/browse-edgar?action=getcurrent&type=8-K&dateb=&owner=include&count=40&output=atom
   Interval: 900s
   Filter 8-K filings for keywords: "material adverse", "cybersecurity incident",
     "ransomware", "force majeure", "sanctions", "military conflict"
   Alert on matching 8-K: WATCH

7. World Bank Commodity Prices (free)
   URL: https://api.worldbank.org/v2/en/indicator/PNRGASIA?format=json&mrv=2
   Commodity IDs to track:
     CRUDE_WTI: Crude Oil WTI
     PNRGASIA: Energy index
     PMAIZMMT: Maize (food security)
     PWHEAMTUS: Wheat (food security — geopolitical)
     PGOLDMT: Gold (safe haven)
   Interval: 3600s
   Used for baseline — daily data, not real-time

8. CNN Fear & Greed Index (free endpoint)
   URL: https://production.dataviz.cnn.io/index/fearandgreed/graphdata
   Interval: 3600s
   Fields: score (0-100), rating (Extreme Fear / Fear / Neutral / Greed / Extreme Greed)
   Alert: score < 20 (Extreme Fear): WATCH

9. IMF Data API (free)
   URL: https://www.imf.org/external/datamapper/api/v1/PCPIPCH (inflation)
   URL: https://www.imf.org/external/datamapper/api/v1/NGDP_RPCH (GDP growth)
   Interval: 86400s (daily, data changes slowly)
   Used for context in morning briefing

10. BLS Economic Indicators (free, key recommended)
    No-key URL: https://api.bls.gov/publicAPI/v1/timeseries/data/CUUR0000SA0 (CPI)
    Key URL:    https://api.bls.gov/publicAPI/v2/timeseries/data/
    Key cfg: cfg.Keys["bls"] (optional — free signup at bls.gov/developers)
    Interval: 3600s
    Alert on new CPI/Jobs data release: WATCH with value

── OPTIONAL KEY SOURCES ────────────────────────────────────────

11. Alpha Vantage (25 req/day free)
    Key: cfg.Keys["alpha_vantage"]
    Base URL: https://www.alphavantage.co/query?apikey={KEY}
    Use for:
      GLOBAL_QUOTE: stock prices for watchlist tickers
      CURRENCY_EXCHANGE_RATE: forex rates (USD/RUB, USD/CNY, USD/IRR, etc.)
      COMMODITY: WTI oil, Brent oil, natural gas, copper, wheat, corn, gold, silver
    Interval: conservative — 25/day → poll every ~1hr for 20 symbols max
    Alert on commodity: +/-5% in 24hr → WATCH

12. Finnhub (60 req/min free)
    Key: cfg.Keys["finnhub"]
    Base URL: https://finnhub.io/api/v1/
    Use for:
      /quote: real-time quotes (US stocks, ETFs)
      /forex/rates: forex rates
      /news: market news with category filter (general, forex, crypto, merger)
      /stock/market-status: market open/closed status
    Interval: 60s for watchlist items
    Note: WebSocket available for real-time: wss://ws.finnhub.io

13. FRED (Federal Reserve Economic Data — free, unlimited)
    Key: cfg.Keys["fred"]
    Base URL: https://api.stlouisfed.org/fred/series/observations?api_key={KEY}&file_type=json
    Series to track:
      FEDFUNDS: Federal Funds Rate
      T10Y2Y:   10yr-2yr Treasury spread (inverted = recession signal)
      DEXUSEU:  USD/EUR exchange rate
      DEXCHUS:  USD/CNY exchange rate
      DEXUSUK:  USD/GBP
      DEXRNUS:  RUB/USD (geopolitical indicator)
      WTISPLC:  WTI Oil price
      GOLDAMGBD228NLBM: Gold price
    Interval: 3600s (most FRED data updates daily)

14. Polygon.io (free tier: EOD data only)
    Key: cfg.Keys["polygon"]
    Base URL: https://api.polygon.io/v2/
    Use for: snapshot of market indices (SPY, QQQ, GLD, USO, UNG)
    Interval: 3600s (respects free tier limits)

════════════════════════════════════════════════════════════════
FINANCIAL ALERT TYPES
════════════════════════════════════════════════════════════════

All financial events use category: "FINANCIAL"
Subcategories for filtering: MARKET | COMMODITY | CRYPTO | FOREX |
                              BONDS | SANCTIONS | REGULATORY | MACRO

MARKET ALERTS:
  VIX Spike          — VIX > threshold OR spike > 5pts/hr
  Circuit Breaker    — Detect from news/SEC (hard to get free real-time)
                       Monitor for news keywords instead
  Market Status      — Exchange open/closed (Finnhub /stock/market-status)
  S&P 500 Move       — Daily move > configured % (via Polygon or Alpha Vantage)

COMMODITY ALERTS (geopolitically significant):
  Oil Spike (WTI/Brent)  — > configured % in configured timeframe
  Oil Crash              — < configured % (demand collapse signal)
  Gold Spike             — > configured % (safe haven flight)
  Wheat/Grain Spike      — > configured % (food security / conflict signal)
  Natural Gas Spike      — > configured % (energy crisis signal)
  Copper Crash           — > configured % (global recession signal "Dr. Copper")

CRYPTOCURRENCY:
  Flash Crash            — > configured % drop in 1hr
  Flash Pump             — > configured % rise in 1hr
  Stablecoin depeg       — USDT/USDC depegs from $1 by > 2% (systemic risk)
  Market cap total drop  — > configured % in 24hr

FOREX ALERTS:
  USD/RUB spike          — rubble weakening (Russia sanctions/conflict signal)
  USD/CNY move           — China economic signal
  USD/TRY spike          — Turkish lira crisis
  USD/IRR note           — Iranian rial (sanctions signal)
  DXY (dollar index)     — significant strengthening/weakening
  Any EM currency crash  — > 5% single day

BONDS / RATES:
  Yield curve inversion   — 2yr > 10yr (recession signal)
  Inversion deepens       — spread worsens by 10bp+
  Emergency rate change   — detect from FRED FEDFUNDS update
  10-year yield breakout  — > 5% or < 1% (extreme levels)

SANCTIONS / REGULATORY:
  OFAC new SDN entries    — always notable, always WATCH minimum
  EU sanctions (via news) — monitor news feed for "EU sanctions" keywords
  SEC enforcement action  — 8-K with "SEC" + action keywords
  FINRA action            — monitor via news feed

MACRO EVENTS:
  CPI data release        — detected from BLS API new data
  Jobs report             — BLS nonfarm payrolls
  Fed decision            — FOMC meeting dates (pre-populate calendar)
  GDP revision            — World Bank/IMF data change

════════════════════════════════════════════════════════════════
GEOPOLITICAL CORRELATION ENGINE
════════════════════════════════════════════════════════════════

New component: internal/correlation/correlation.go

Runs every 15 minutes. Examines last 2 hours of events.
Finds correlations between financial and geopolitical events.
Generates "correlation insights" shown in feed.

CORRELATION RULES:

  IF oil_price_change > 3% AND (
       military_event EXISTS IN ["Middle East", "Persian Gulf", "Strait of Hormuz"]
       OR missile_alert EXISTS
       OR conflict_event EXISTS IN ["Iraq", "Iran", "Saudi Arabia", "Yemen", "Libya"])
  THEN insight: "⚡ OIL CORRELATION — Oil up {pct}% may correlate with {event_title}"

  IF gold_price_change > 2% AND (
       any TIER 3+ event EXISTS
       OR OFAC_new_sanctions EXISTS
       OR conflict_escalation EXISTS)
  THEN insight: "🥇 GOLD CORRELATION — Safe haven buying detected (+{pct}%)"

  IF ruble_change > 3% AND (
       conflict_event EXISTS IN ["Ukraine", "Russia"]
       OR OFAC_sanctions CONTAINS "RUSSIA")
  THEN insight: "💱 RUBLE CORRELATION — RUB/USD moved {pct}%"

  IF wheat_change > 5% AND (
       conflict_event EXISTS IN ["Ukraine", "Russia", "Black Sea"]
       OR weather_event EXISTS IN ["Ukraine", "Kansas", "Argentina"])
  THEN insight: "🌾 FOOD SECURITY SIGNAL — Wheat +{pct}%"

  IF VIX_spike AND conflict_TIER3_EXISTS
  THEN insight: "📊 FEAR INDEX — VIX at {value}, elevated alongside {event_count} TIER 3+ events"

  IF crypto_flash_crash AND (
       regulatory_action EXISTS
       OR sanctions_action EXISTS)
  THEN insight: "₿ CRYPTO SIGNAL — Flash crash may correlate with regulatory action"

Insights stored in SQLite: correlation_insights table
Displayed in feed as: [⚡ CORRELATION INSIGHT] card with special styling
Also included in morning briefing under "Market Intelligence" section

════════════════════════════════════════════════════════════════
FINANCIAL WATCHLIST
════════════════════════════════════════════════════════════════

Users can add custom tickers/symbols to track:
  Stored in SQLite: financial_watchlist table
  Fields: symbol, type (stock/forex/crypto/commodity), threshold_pct, enabled

Default watchlist (pre-populated):
  BTC       crypto     threshold: 10%
  ETH       crypto     threshold: 10%
  OIL_WTI   commodity  threshold: 5%
  GOLD      commodity  threshold: 3%
  WHEAT     commodity  threshold: 5%
  VIX       index      threshold: 5 points
  RUB/USD   forex      threshold: 3%
  CNY/USD   forex      threshold: 2%
  EUR/USD   forex      threshold: 1.5%

Manage via Settings → Financial → Watchlist tab
Add/edit/delete symbols, set per-symbol thresholds

════════════════════════════════════════════════════════════════
FINANCIAL FEED PANEL
════════════════════════════════════════════════════════════════

Financial events appear in main alert feed with 📈 / 📉 icons.
Also available as dedicated panel in Command Center view (right-side column).

FINANCIAL TICKER (addition to main live ticker):
  Shows: [📉 OIL -3.2%] [📈 GOLD +1.8%] [₿ BTC -12.4%] [⚡ VIX 28.4]
  Cycles alongside geopolitical events in bottom ticker strip
  Updates every 60s for real-time prices

FINANCIAL CARD FORMAT:
┌─────────────────────────────────────────────────────────────┐
│ 📉 WARNING  📊 MARKET    14:22 UTC  •  8 min ago            │
│                                                              │
│ WTI CRUDE OIL — FLASH SPIKE +6.2%                           │
│ 📍 New York Mercantile Exchange                              │
│                                                              │
│ West Texas Intermediate crude surged 6.2% in 45 minutes,   │
│ currently at $89.40/bbl. Correlated with military activity  │
│ detected in Persian Gulf region (2 events in last 2hrs).    │
│                                                              │
│ ⚡ CORRELATION: Military activity in Middle East (2 events)  │
│ 🔗 CBOE/NYMEX  📍 View correlation  📰 Related news         │
└─────────────────────────────────────────────────────────────┘

MARKET OVERVIEW WIDGET:
  Small collapsible panel in feed sidebar:
  ┌─────────────────────────┐
  │ MARKET OVERVIEW  14:22  │
  │ VIX    24.3  ▲+2.1      │
  │ Oil    89.4  ▲+6.2%     │
  │ Gold  1923   ▲+1.8%     │
  │ BTC   43,210 ▼-2.1%     │
  │ RUB   89.4   ▼-3.2%     │
  └─────────────────────────┘
  Updates every 60s.
  Green = up, Red = down, Amber = extreme move.
  Click any row → opens financial detail card.

════════════════════════════════════════════════════════════════
FINANCIAL SETTINGS (Settings → [FINANCIAL] tab)
════════════════════════════════════════════════════════════════

Section: Market Subscriptions
  ☑ Equities (VIX, S&P, circuit breakers)
  ☑ Commodities (Oil, Gold, Wheat, Gas, Copper)
  ☑ Cryptocurrency (BTC, ETH, market cap)
  ☑ Forex (major pairs + geopolitical currencies)
  ☑ Bonds (Treasury yields, spread)
  ☑ Sanctions (OFAC SDN updates)
  ☑ Regulatory (SEC 8-K filings)
  ☑ Macro Indicators (CPI, jobs, Fed decisions)

Section: Alert Thresholds
  Crypto flash crash threshold:   [10]% in [1 hour ▾]
  Commodity spike threshold:      [5]% in [24 hours ▾]
  Forex threshold:                [3]% in [24 hours ▾]
  VIX spike threshold:            [5] points in [1 hour ▾]
  VIX level warning:              [30] (absolute level)

Section: Geopolitical Correlation
  ☑ Show correlation insights in feed
  ☑ Include financial alerts in morning briefing
  ☑ Show market overview widget in feed

Section: Watchlist
  [+ Add symbol]  [Import symbols]
  List of watched symbols with threshold sliders

Section: API Keys
  Alpha Vantage: [key input] [Test] [signup link]
  Finnhub:       [key input] [Test] [signup link]
  FRED:          [key input] [Test] [signup link]
  Polygon.io:    [key input] [Test] [signup link]
  BLS:           [key input] [Test] [signup link]

════════════════════════════════════════════════════════════════
NEW API ENDPOINTS (FINANCIAL)
════════════════════════════════════════════════════════════════

GET  /api/financial/overview              — current prices for all watched symbols
GET  /api/financial/watchlist             — user's financial watchlist
POST /api/financial/watchlist             — add symbol
DEL  /api/financial/watchlist/{id}        — remove symbol
GET  /api/financial/alerts                — recent financial alerts (paginated)
GET  /api/financial/correlations          — recent correlation insights
GET  /api/financial/history/{symbol}      — price history for symbol (last 24hr)
GET  /api/financial/sanctions             — recent OFAC SDN changes
GET  /api/financial/sec                   — recent flagged 8-K filings

════════════════════════════════════════════════════════════════
DATABASE SCHEMA (FINANCIAL)
════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS financial_prices (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  symbol TEXT NOT NULL,
  price REAL,
  change_pct_1h REAL,
  change_pct_24h REAL,
  volume REAL,
  source TEXT,
  fetched_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS financial_watchlist (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  symbol TEXT NOT NULL UNIQUE,
  asset_type TEXT,  -- stock, forex, crypto, commodity, index
  display_name TEXT,
  threshold_pct REAL DEFAULT 5.0,
  threshold_window_hours INTEGER DEFAULT 1,
  enabled INTEGER DEFAULT 1,
  notify_telegram INTEGER DEFAULT 0,
  notify_email INTEGER DEFAULT 0
);

CREATE TABLE IF NOT EXISTS correlation_insights (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  insight_text TEXT NOT NULL,
  financial_event_id INTEGER,
  geo_event_id INTEGER,
  confidence REAL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sanctions_entries (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  sdn_name TEXT NOT NULL,
  sdn_type TEXT,
  program TEXT,
  first_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
  is_new INTEGER DEFAULT 1
);

CREATE TABLE IF NOT EXISTS sec_filings (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  company TEXT,
  form_type TEXT,
  filing_url TEXT UNIQUE,
  matched_keyword TEXT,
  filed_at DATETIME,
  ingested_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_financial_prices_symbol ON financial_prices(symbol, fetched_at);
