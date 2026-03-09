# MANIFEST 05 — OSINT RESOURCES: PROFILES, PLATFORMS & RADIO
# =============================================================
# Covers: Stage 7 — OSINT profile suggestions system, platform guides,
#         ham/military radio integration, WebSDR embed, signal library.
#
# IMPORTANT FOR AI: These are SUGGESTIONS displayed to users.
# They are stored as manageable lists in SQLite, not hardcoded.
# Users can mark profiles as "followed", add custom ones, and dismiss ones
# they don't want. Handle account renames/deletions gracefully (links just open).

════════════════════════════════════════════════════════════════
OSINT PROFILE SUGGESTION SYSTEM — ARCHITECTURE
════════════════════════════════════════════════════════════════

New page: web/osint-resources.html
Accessible: header nav → [🔍 OSINT Resources] OR Settings → OSINT Resources tab

This page is an INTELLIGENCE RESOURCE GUIDE, not a tracker.
It helps users build their own OSINT information network.

The page has 3 main sections:
  1. Platform Profiles  — WHO to follow on which platform
  2. Ham & Signal Radio — WHAT frequencies to monitor and where to listen
  3. Live Resources     — aggregated links organized by topic

All lists stored in SQLite: osint_resources table
User can: mark as followed, add custom entries, hide/dismiss, add notes

════════════════════════════════════════════════════════════════
SECTION 1 — PLATFORM PROFILE SUGGESTIONS
════════════════════════════════════════════════════════════════

DISPLAY FORMAT per profile suggestion:
┌─────────────────────────────────────────────────────────────┐
│ 👤 [Handle/Name]     📍 Platform    🏷️ Category             │
│ Brief description of what they cover and why they're useful │
│ Specialty: [tags]                                           │
│ [Open Profile →]  [✅ Mark as Following]  [Dismiss]         │
└─────────────────────────────────────────────────────────────┘

WHEN USER CONFIGURES X/TWITTER IN SETUP WIZARD:
  Show "Suggested accounts to follow" step (dismissable)
  Organized by their subscription interests
  Only show categories matching their subscriptions

PLATFORM SETUP NOTES (shown on first visit to OSINT Resources):
  X/Twitter: "Creating a private list of OSINT accounts is recommended
              over following — keeps your main feed clean."
  Mastodon: "infosec.exchange and kolektiva.social have active OSINT communities"
  Telegram: "Use separate account or dedicated SENTINEL Telegram for channels"
  Reddit: "Multireddits let you monitor multiple communities as a single feed"

── X/TWITTER SUGGESTIONS ────────────────────────────────────

Store as JSON in SQLite. Category = subscription interest.

MILITARY / CONFLICT OSINT:
  @OSINTdefender     — Real-time conflict monitoring, geolocation verification
  @IntelCrab         — Military intelligence aggregation, open-source analysis
  @Osinttechnical    — Technical military analysis, weapons identification
  @GeoConfirmed      — Geolocation of conflict imagery, verification
  @oryxspioenkop     — Equipment loss tracking, vehicle identification (Oryx)
  @RALee85           — Defense policy analyst, Russia/Ukraine expert (Rob Lee)
  @PhillipsPOBrien   — Strategic analysis, naval warfare
  @WarMonitor3       — Conflict monitoring aggregator
  @Conflicts         — Aggregated conflict reporting
  @bradyafr          — African security and conflict
  @SICO_INT          — Spanish-language OSINT collective
  @noclador          — Flight tracking OSINT, military aviation

NAVAL / MARITIME:
  @HI_Sutton         — Naval analyst, submarine tracking, underwater systems
  @navalanalyst      — Naval news and analysis
  @CovertShores      — Covert naval systems analyst
  @MaritimeForum     — Maritime security incidents
  @vesseltracker     — AIS anomalies and tracking

AVIATION OSINT:
  @CivMilAir         — Civil and military aviation monitoring
  @AircraftSpots     — Aviation spotting and military identification
  @JoeLieber_        — Aviation incidents and tracking
  @TaiwanADIZ        — Taiwan airspace intrusions
  @Scramble_NL       — Military aviation news (Scramble Magazine)

SPACE / SATELLITES:
  @TSKelso           — CelesTrak author, orbital mechanics
  @planet            — Planet Labs satellite imagery
  @Maxar             — Commercial satellite imagery
  @AstroViz          — Orbital visualization and analysis
  @TLE_Updates       — TLE data updates and launches
  @johnkrausphotos   — Space/launch photographer, OSINT-relevant

SPACE WEATHER:
  @SpaceWxCenter     — NOAA Space Weather Prediction Center
  @TamithaSkov       — Space weather forecaster, aurora predictions
  @sdo               — NASA Solar Dynamics Observatory

GEOPOLITICS / INTELLIGENCE:
  @Bellingcat        — Investigative OSINT collective
  @Benjamin_Strick   — Bellingcat investigator, open source investigation
  @christogrozev     — Bellingcat, Russia/aerospace
  @Nrg8000           — Intelligence analysis
  @KamilGaleev       — Russian politics and military analysis
  @JominiW           — Geopolitical analysis, operational art
  @DCourtney05       — Intelligence community analysis
  @TheStudyofWar     — ISW posts (Institute for the Study of War)

CYBER / INFOSEC:
  @MalwareHunterTeam — Malware tracking, threat intelligence
  @VK_Intel          — Vulnerability tracking
  @threat_intel      — Threat intelligence aggregator
  @SwiftOnSecurity   — Security analysis, incident response
  @GossiTheDog       — Ransomware and cyber incident tracking
  @campuscodi        — Ransomware, data breach tracker
  @vxunderground     — Malware intelligence (follow for awareness, not endorsement)

FINANCIAL / ECONOMIC:
  @zerohedge         — Market alerts, financial instability (verify claims)
  @LynAldenContact   — Macro economics, monetary policy
  @RaoulGMI          — Global macro investor (paid but public tweets)
  @elerianm          — Mohamed El-Erian, monetary policy analysis
  @IMFNews           — Official IMF announcements
  @FedReserve        — Official Federal Reserve
  @USTreasury        — OFAC sanctions announcements (official)

DISASTER / EMERGENCY:
  @NWStweets         — NOAA National Weather Service (official)
  @USGS_Quakes       — USGS earthquake alerts (official)
  @PDC_Alerts        — Pacific Disaster Center alerts
  @ReliefWeb         — Humanitarian emergency updates
  @OFCOMwatch        — UK telecoms outage monitoring

── MASTODON / FEDIVERSE SUGGESTIONS ─────────────────────────

Note: Search these on mastodon.social or specific instance

  infosec.exchange/@hacks4pancakes  — Infosec and threat intel
  infosec.exchange/@GossiTheDog     — Cyber incidents (also on X)
  kolektiva.social                  — Activist journalism, conflict zones
  social.coop                       — Cooperative journalism

── TELEGRAM CHANNELS (public, read-only) ───────────────────

Display as LINK ONLY — open in user's Telegram app.
Note: Telegram channel content is user-verified, not AI-curated.
Display credibility warning: "Verify all claims independently."

CONFLICT MONITORING:
  t.me/intelslava           — Conflict monitoring aggregator
  t.me/wartranslated        — Ukraine war translation/analysis
  t.me/osintukraine         — Ukraine-specific OSINT
  t.me/mod_russia_en        — Russian MoD English (official Kremlin — note: state media)
  t.me/UkraineNow           — Ukraine updates
  t.me/militarylandnet      — Military analysis

MIDDLE EAST:
  t.me/IsraeliPM            — Israeli PM office (official)
  t.me/idfofficial          — IDF English (official)
  t.me/AlMayadeenEnglish    — Al Mayadeen English (note: Hezbollah-aligned)

SPACE / AVIATION:
  t.me/nasaspaceflight      — NSF community
  t.me/RocketLaunchLive     — Launch coverage

FINANCIAL:
  t.me/durov                — Pavel Durov / Telegram news
  t.me/cryptosignals        — WARN users: crypto channels are often scams, use critically

── SUBSTACK / NEWSLETTERS ───────────────────────────────────

Display as RSS-subscribable OR manual follow link:
  Phillips O'Brien:        https://www.phillipspobrien.com/
  Rob Lee (War on Rocks):  https://warontherocks.com/
  Bellingcat:              https://www.bellingcat.com/
  Perun (defence policy):  https://perun.substack.com/
  The Intel Drop:          https://intelbrief.com/
  NAFO Field Post:         Various — no single URL

── REDDIT COMMUNITIES ───────────────────────────────────────

Display as links with feed preview (Reddit RSS works):
  r/worldnews:             News + major discussion
  r/geopolitics:           Deep analysis, academic quality
  r/CredibleDefense:       High-quality military analysis, strict sourcing
  r/UkraineWarVideoReport: Conflict footage, high verification standards noted
  r/OSINT:                 Techniques and active investigations
  r/flightradar24:         Aviation OSINT community
  r/conspiracy:            NOTE: low reliability, show with warning label
  r/collapse:              Environmental/systemic risk monitoring
  r/cybersecurity:         Cyber threat discussion
  r/CryptoCurrency:        Crypto market movements

── YOUTUBE CHANNELS (non-live, analysis) ────────────────────

Shown in OSINT Resources page AND in media wall stream picker:

MILITARY ANALYSIS:
  Perun              — Defence economics and strategy (deep, reliable)
  Binkov's Battlegrounds — Military scenario analysis
  Task & Purpose     — US military news and analysis
  War College        — Military history and doctrine
  The Duran          — NOTE: Russia-sympathetic, mark accordingly
  Military Aviation History — Historical context, aircraft ID

NAVAL:
  Naval News         — Naval developments, ship identification
  ThinkDefence       — UK defence analysis

SPACE:
  Scott Manley       — Space science, orbital mechanics, launch analysis
  Everyday Astronaut — Space launches, technical detail
  NASASpaceflight    — NSF launch coverage

GEOPOLITICS:
  Foreign Policy Association — Academic analysis
  Council on Foreign Relations — CFR (mainstream Western IR)
  CGTN (China Global TV Network) — NOTE: Chinese state media, mark accordingly
  RT (Russia Today)  — NOTE: Russian state media, mark accordingly
  Al Jazeera English — Qatar-funded, Middle East focus

OSINT TECHNIQUE:
  Trace Labs         — OSINT techniques and competitions
  The OSINT Curious Project — Methodology training
  Bellingcat        — Investigative methodology

════════════════════════════════════════════════════════════════
SECTION 2 — HAM RADIO & SIGNAL MONITORING
════════════════════════════════════════════════════════════════

Philosophy: Radio signals are a real-time intelligence layer.
Military, emergency services, aviation ATC, and maritime distress
calls often precede or accompany major events visible in SENTINEL.

SENTINEL integrates radio as LINKS and EMBEDS — it does not
receive radio signals directly. It surfaces relevant streaming
receivers (WebSDR/KiwiSDR) based on the event region.

── ONLINE SDR RECEIVER NETWORKS (free, no account) ─────────

  WebSDR Network:
    URL: http://websdr.org/
    Description: 600+ online receivers worldwide, browser-based
    Embed: http://websdr.hb9drp.ch:8901/ (example HF receiver)
    API: No formal API — link to websdr.org and let user browse by map

  KiwiSDR Network:
    URL: https://map.kiwisdr.com/
    Description: ~500 receivers, 0-30 MHz HF coverage worldwide
    Best for: HF military, aviation HFDL, shortwave
    Link to nearest receiver based on event lat/lon

  OpenWebRX (public instances):
    URL: https://sdr.hu/ (directory of public OpenWebRX instances)
    Description: SDR software, many public instances
    Some cover VHF/UHF as well as HF

  Broadcastify (scanner streams):
    URL: https://www.broadcastify.com/listen/
    Description: Police, fire, EMS, aviation, military (unclassified)
    Embed: <iframe src="https://www.broadcastify.com/listen/feed/{ID}" ...>
    Free streaming, no account needed to listen
    Paid: archive playback requires premium

  LiveATC.net (aviation ATC):
    URL: https://www.liveatc.net/
    Description: Aviation ATC feeds worldwide, free to stream
    Embed: <iframe src="https://www.liveatc.net/play/{airport}.pls" ...>
    Coverage: Major airports worldwide including military adjacent

  RadioReference.com:
    URL: https://www.radioreference.com/
    Description: Frequency database + some live streams (Broadcastify partnership)
    Free: frequency lookup | Paid: full database access

  Global Tuners:
    URL: https://www.globaltuners.com/
    Description: Remote receiver network

── FREQUENCY REFERENCE GUIDE ───────────────────────────────

Store in SQLite: radio_frequencies table
Display in OSINT Resources → Radio tab

AVIATION (all monitored by default in WebSDR/LiveATC):
  121.500 MHz  VHF-AM  International Aeronautical Emergency (GUARD)
  123.100 MHz  VHF-AM  Search and Rescue
  225.500 MHz  UHF-AM  US Military Aviation Primary
  243.000 MHz  UHF-AM  Military Emergency/GUARD (UHF equivalent)
  282.800 MHz  UHF-AM  US Navy Tactical
  311.000 MHz  UHF-AM  US Air Force Tanker/Receiver
  340.200 MHz  UHF-AM  USAF Command Post Common
  364.200 MHz  UHF-AM  USAF Logistics
  413.925 MHz  UHF-AM  NATO Combined/Joint
  8.992 MHz    HF-USB  USAF Global High Frequency (GHF) primary
  11.175 MHz   HF-USB  USAF Mystic Star primary (presidential/SECDEF)
  13.200 MHz   HF-USB  USAF GHF secondary
  15.016 MHz   HF-USB  USAF Volmet (aviation weather)
  4.724 MHz    HF-USB  Andrews AFB HF
  6.739 MHz    HF-USB  NATO HF primary

MARITIME (monitored via WebSDR/LiveATC maritime feeds):
  156.800 MHz  VHF-FM  Channel 16 — International Maritime Distress
  156.300 MHz  VHF-FM  Channel 6  — Intership Safety
  156.600 MHz  VHF-FM  Channel 12 — Port Operations
  4.125 MHz    HF-USB  Maritime Distress (HF)
  6.215 MHz    HF-USB  Maritime Distress (HF)
  8.291 MHz    HF-USB  Maritime Distress (HF)
  500.000 kHz  CW      Maritime emergency (historically significant)

MILITARY HF NETWORKS:
  3.000-30 MHz  HF      Military voice + data (STANAG 4285, 4539)
  5.371 MHz    HF-USB  NATO Common HF
  6.861 MHz    HF-USB  NATO primary HF
  8.968 MHz    HF-USB  NATO secondary
  11.175 MHz   HF-USB  USAF Command (as above)
  14.993 MHz   HF-USB  USAF secondary
  10.016 MHz   HF-USB  RAF Volmet

NUMBERS STATIONS (HF — historical curiosity, occasionally active):
  Various HF   HF-USB/AM  Numbers stations (espionage communication)
  Monitor via: priyom.org schedule
  HF Underground logging: hfunderground.com

DATA SIGNALS ON HF:
  HFDL (HF Data Link — aircraft position reports on HF):
    Multiple frequencies: 2.998, 4.681, 5.652, 6.532, 8.825, 10.081, 11.384, 13.312, 17.919 MHz
    Software: PC-HFDL (free) at hfdl.net
    Online decoder: airframes.io/hfdl (aggregates HFDL worldwide)
    What it shows: commercial aircraft positions on HF — useful when ADS-B absent

  AIS (Automatic Identification System — ships):
    161.975 MHz and 162.025 MHz VHF
    Most WebSDR receivers in coastal areas can decode AIS
    Online aggregators: marinetraffic.com, vesseltracker.com

  ACARS (Aircraft Communications Addressing and Reporting):
    129.125 MHz, 130.025 MHz, 130.450 MHz VHF-AM
    Decode with: acarsdec (free), JAERO (free for SATCOM ACARS)
    Online: airframes.io/acars

── EVENT-TO-RADIO MAPPING ──────────────────────────────────

When SENTINEL fires a geopolitical event, suggest relevant radio monitoring:

Missile alert (any):
  Suggest: 121.5 MHz guard, 243.0 MHz military guard
  Broadcastify: feeds for affected region emergency services

Military aircraft event:
  Suggest: 121.5 MHz, 225.500 MHz military aviation
  LiveATC: nearest major ATC feed to event location
  Broadcastify: military aviation feeds if available

Naval/maritime event:
  Suggest: 156.800 MHz Channel 16
  Broadcastify: Coast Guard feeds for event region

Earthquake/disaster:
  Suggest: Broadcastify feeds for affected region
           (emergency management, fire, EMS)
  FEMA radio: check for emergency declarations

Space weather (high Kp):
  Suggest: HF propagation will be degraded — note to users
  Ionospheric disturbance affects 2-30 MHz HF communications

Wildfire:
  Suggest: Broadcastify CalFire / state forestry feeds
  Local fire departments for affected counties

Financial crisis:
  No specific radio recommendation (financial comms are encrypted)
  Suggest: news monitoring instead

── WEBSDR INTEGRATION ───────────────────────────────────────

SENTINEL suggests nearest WebSDR/KiwiSDR receiver for event region.

Logic:
  1. Get event lat/lon
  2. Query our embedded list of ~50 major WebSDR receivers with their coordinates
  3. Find 3 nearest receivers within 5,000km
  4. Display as clickable links: "📡 Nearest receiver: [City, Country] →"
  5. Link format: http://websdr.org → user browses/selects
     OR direct link to specific known receivers (maintain list in code)

Known WebSDR receivers list (maintain in internal/radio/receivers.go):
  Store: name, location, lat, lon, URL, frequency_range, notes
  Include ~30 well-known stable receivers globally
  Users can add custom WebSDR URLs in settings

WebSDR embed (when user clicks):
  Open in modal/panel within SENTINEL
  iframe: <iframe src="{websdr_url}" width="100%" height="600px">
  Some WebSDRs support deep links with frequency preset

Broadcastify region mapping:
  Map event country/state → Broadcastify feed IDs
  Store mapping in internal/radio/broadcastify.go
  Include ~100 major metro feeds + national emergency feeds
  US: Emergency Alert feeds by state, major city PD/Fire
  UK: Metropolitan Police, NHS Ambulance regional
  Israel: IDF adjacent (if available on Broadcastify)
  Others: link to broadcastify.com/listen/country/{code}

── HAM RADIO RESOURCES PAGE ───────────────────────────────

web/osint-resources.html — Radio tab:

LAYOUT:
  [Region selector dropdown: based on active SENTINEL events or user location]
  
  LIVE RECEIVERS:
    [WebSDR receiver cards — nearest to selected region]
    [Broadcastify feed cards — relevant to selected region]
    [LiveATC feeds — nearest airport ATC]
  
  FREQUENCY GUIDE:
    Filterable table: frequency | mode | use | who transmits | notes
    Filter by: Aviation / Maritime / Military / Emergency / Data Signals
    [Copy frequency] button (copy to clipboard for manual SDR tuning)
  
  ONLINE DECODERS:
    airframes.io    — ACARS, HFDL, VDL2 aircraft data
    flightradar24   — ADS-B tracking
    marinetraffic   — AIS vessels
    satflare.com    — Satellite pass prediction
    sondehub.org    — Weather balloon tracking (stratospheric OSINT)
    priyom.org      — Numbers station schedule
    hfunderground.com — HF signal logging database
    sigidwiki.com   — Signal identification guide (identify unknown signals)
  
  SOFTWARE (free/open-source):
    SDR++ (Windows/Linux/Mac, free): https://github.com/AlexandreRouma/SDRPlusPlus
    GQRX (Linux/Mac, free): https://gqrx.dk/
    SDR# (Windows, free): https://airspy.com/download/
    Unitrunker (Windows, free): http://www.unitrunker.com/
    RTL-SDR hardware: rtl-sdr.com ($25 USB dongle — note this is hardware, not included)
    Note: RTL-SDR hardware required to receive radio locally.
          WebSDR/KiwiSDR requires NO hardware — browser only.

════════════════════════════════════════════════════════════════
DATABASE SCHEMA (OSINT RESOURCES)
════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS osint_profiles (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  platform TEXT NOT NULL,     -- twitter, telegram, mastodon, reddit, youtube, substack
  handle TEXT,
  display_name TEXT,
  profile_url TEXT,
  category TEXT,              -- military, naval, space, cyber, financial, disaster, geopolitics
  description TEXT,
  tags TEXT,                  -- comma-separated
  credibility TEXT,           -- official, verified_osint, analyst, news, state_media
  state_media_warning INTEGER DEFAULT 0,  -- 1 if state media, show warning
  is_followed INTEGER DEFAULT 0,
  is_dismissed INTEGER DEFAULT 0,
  user_notes TEXT,
  is_builtin INTEGER DEFAULT 1,
  added_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS radio_frequencies (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  frequency_mhz REAL NOT NULL,
  mode TEXT,                  -- AM, FM, USB, LSB, CW, HFDL, AIS, ACARS
  use_category TEXT,          -- aviation, maritime, military, emergency, data
  description TEXT,
  who_transmits TEXT,
  notes TEXT,
  is_builtin INTEGER DEFAULT 1
);

CREATE TABLE IF NOT EXISTS websdr_receivers (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  city TEXT,
  country TEXT,
  lat REAL,
  lon REAL,
  url TEXT NOT NULL,
  freq_range_low_mhz REAL,
  freq_range_high_mhz REAL,
  notes TEXT,
  is_builtin INTEGER DEFAULT 1,
  user_added INTEGER DEFAULT 0
);

CREATE TABLE IF NOT EXISTS broadcastify_feeds (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  feed_id INTEGER NOT NULL,
  name TEXT,
  description TEXT,
  country TEXT,
  region TEXT,                -- state/province
  category TEXT,              -- police, fire, ems, military, aviation
  lat REAL,
  lon REAL,
  embed_url TEXT
);

════════════════════════════════════════════════════════════════
NEW API ENDPOINTS (OSINT RESOURCES)
════════════════════════════════════════════════════════════════

GET  /api/osint/profiles              — all profile suggestions (filterable by platform/category)
POST /api/osint/profiles/{id}/follow  — mark as followed
POST /api/osint/profiles/{id}/dismiss — dismiss suggestion
POST /api/osint/profiles              — add custom profile
GET  /api/radio/receivers             — WebSDR/KiwiSDR receivers (filterable by lat/lon/range)
GET  /api/radio/frequencies           — frequency reference table
GET  /api/radio/broadcastify          — Broadcastify feeds (filterable by region)
GET  /api/radio/suggest?event_id={id} — suggest radio sources for specific event
POST /api/radio/receivers             — add custom WebSDR receiver
GET  /api/osint/resources/export      — export full OSINT resource list as PDF/JSON

════════════════════════════════════════════════════════════════
SETTINGS — OSINT RESOURCES TAB
════════════════════════════════════════════════════════════════

Section: Platform Profiles
  Filter by: platform / category / followed/unfollowed
  [+ Add custom profile] — name, handle, URL, category, notes
  [Export followed list] — download as CSV

Section: Radio Monitoring
  Home region: [set location for nearest receiver suggestions]
  [+ Add custom WebSDR receiver] — name, URL, lat/lon, notes
  [+ Add Broadcastify feed] — feed ID, name, region
  Frequency guide: [filter by category] [show/hide data signals]
  SDR software: download links shown per platform

Section: Auto-suggest on Events
  ☑ Suggest radio feeds when events fire in nearby region
  ☑ Suggest OSINT profiles when new event type detected
  Suggestion min severity: [Watch ▾]

════════════════════════════════════════════════════════════════
SETUP WIZARD ADDITION (STAGE 7 — add to existing wizard)
════════════════════════════════════════════════════════════════

After Step 6 (Optional API Keys), add:

STEP 6b — OSINT NETWORK SETUP (optional, dismissable)
  "Would you like suggestions for building your OSINT monitoring network?"
  ○ Yes — show me recommended accounts and resources
  ○ No — I'll set this up later in Settings

  If Yes, show platform multi-select:
    ☐ X/Twitter      ☐ Telegram       ☐ Reddit
    ☐ YouTube        ☐ Mastodon       ☐ Substack

  "After setup, visit OSINT Resources to see personalized recommendations
   based on your alert subscriptions."

STEP 6c — HAM RADIO (optional, dismissable)
  "Do you have access to radio monitoring?"
  ○ I have an SDR receiver (RTL-SDR, AirSpy, etc.) — I'll tune manually
  ○ I'll use online WebSDR receivers in my browser (no hardware needed)
  ○ I'll use Broadcastify for scanner audio only
  ○ Skip radio monitoring

  "Based on your subscriptions, SENTINEL will suggest relevant frequencies
   and online receivers when events occur in regions you're watching."
