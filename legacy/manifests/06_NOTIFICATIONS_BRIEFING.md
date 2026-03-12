# MANIFEST 06 — NOTIFICATIONS, ALERT RULES & BRIEFINGS
# ======================================================
# Covers: Stage 8 — All notification channels, alert rules engine,
#         geofences, morning briefing, weekly digest.

════════════════════════════════════════════════════════════════
NOTIFICATION CHANNELS
════════════════════════════════════════════════════════════════

internal/notify/ package with per-channel implementations.

All channels:
  - Test button in settings (POST /api/notifications/test/{channel})
  - Per-channel enable/disable
  - Per-channel min severity
  - Rate limiting: max 10 messages/min per channel globally

TELEGRAM:
  Already partially implemented — extend:
  Per-category toggles
  Digest mode: batch events every N minutes instead of per-event
  TIER 4: repeat alert every 30min until acknowledged
  Message format:
    🚨 [SEVERITY] [CATEGORY]
    📍 [Location]
    [Title]
    [Key stats — 1 line]
    [Description — max 200 chars]
    🔗 [Source link]
    SENTINEL v2 | [timestamp UTC]

SLACK:
  Webhook URL → POST JSON to webhook
  Block Kit formatting for rich messages
  Color attachment: green/amber/orange/red by severity
  Include: title, location, description, source link

DISCORD:
  Webhook URL → POST embed JSON
  Embed color by severity
  Include all event fields as embed fields

NTFY.SH:
  POST to https://ntfy.sh/{topic} (or custom server)
  Headers: Title, Priority (by severity), Tags (by category emoji)
  Message: "Location — Key stats — Source"
  Priority mapping: info=2, watch=3, warning=4, alert=5, critical=5

EMAIL:
  Methods (in order of recommendation):
    Gmail OAuth2       — most secure, token auto-refresh
    Gmail App Password — 16-char, requires 2FA
    Generic SMTP       — covers Outlook/Yahoo/custom
    SendGrid           — 100/day free
    Mailgun            — 1000/month free
    Webhook (POST JSON)— Slack/Discord/PagerDuty/custom

  HTML template: dark-themed HTML email, SENTINEL branding
  Plain text fallback always included
  Subject: "SENTINEL [SEVERITY]: [TITLE] — [LOCATION]"
  TIER 1 WATCH: batch 5 events max per 15min email
  TIER 2+: immediate
  TIER 4: immediate + repeat every 30min until ACK

PUSHOVER:
  POST to https://api.pushover.net/1/messages.json
  Priority by severity: info=-1, watch=0, warning=1, alert=1, critical=2
  Critical priority requires user ACK (Pushover feature)
  Sound: selectable per severity

════════════════════════════════════════════════════════════════
ALERT RULES ENGINE
════════════════════════════════════════════════════════════════

Stored in SQLite: notification_rules table

Rule structure:
  Conditions (AND logic within a rule, OR logic across rules):
    category == X
    severity >= X
    location_country == X
    location_region contains X
    title contains keyword X
    magnitude >= X (earthquakes)
    any_keyword_watchlist_match == true
    any_region_watchlist_match == true
    financial_change_pct >= X

  Actions:
    notify_telegram = true/false
    notify_email = true/false
    notify_slack = true/false
    notify_discord = true/false
    notify_ntfy = true/false
    notify_pushover = true/false
    play_sound = true/false
    sound_tier = X

DEFAULT RULES (pre-loaded on first run):
  Rule 1: severity >= critical → all channels enabled
  Rule 2: category == missile_alert → all channels
  Rule 3: category == earthquake AND magnitude >= 6.0 → telegram + email
  Rule 4: category == space_weather AND kp >= 7 → telegram
  Rule 5: category == financial AND change_pct >= 10 → telegram
  Rule 6: category == sanctions → telegram + email
  Rule 7: any_region_watchlist_match → telegram
  Rule 8: any_keyword_watchlist_match → telegram

Rules UI (Settings → Notifications → Alert Rules):
  [+ Add Rule] modal:
    IF: [category ▾] [operator ▾] [value]
    AND (optional): [+ Add condition]
    THEN notify via: [checkboxes]
    Rule name (optional)
    [Save] [Cancel]
  List with: enable/disable toggle | edit | delete

════════════════════════════════════════════════════════════════
GEOFENCE ALERTS
════════════════════════════════════════════════════════════════

User draws polygon on globe → any event inside → alert regardless of severity.
Stored in SQLite: geofences table with GeoJSON polygon.

Geofence card in feed: "📌 [GEOFENCE: Zone Name]" badge at top.

Geofence management (Settings → Geofences):
  List: name, vertex count, notification settings, [Edit] [Delete]
  [Draw New Geofence] → returns to globe with drawing mode
    Click to place vertices, double-click to close
    Name input + notification toggles
    [Save Geofence]
  [Import GeoJSON] → paste or upload
  [Export All] → downloads geofences.geojson

════════════════════════════════════════════════════════════════
MORNING BRIEFING
════════════════════════════════════════════════════════════════

Scheduled goroutine checks time every minute.
When current UTC time matches cfg.MorningBriefing.TimeUTC: generate + send.

GET /api/intel/morning-briefing — generate or serve cached

Collects:
  Top 10 events (last 24hr) by severity
  Active alerts right now
  Top 5 correlation insights (last 12hr)
  Top 5 news items by relevance (last 12hr)
  ISS passes for cfg.Location (if set) — next 24hr
  Space weather forecast
  Financial market overview (last close prices)
  Provider health summary

DeepSeek prompt:
  "Generate a structured morning intelligence briefing. Be specific: name
   actors, locations, and numbers. Use UTC times. Format exactly as:
   
   ## SENTINEL MORNING BRIEFING — {DATE} UTC
   
   ### 🔴 ACTIVE ALERTS ({N})
   {list active alerts — one per line, most critical first}
   
   ### 🌍 OVERNIGHT SUMMARY
   {3 sentences on most significant events}
   
   ### ⚔️ CONFLICT STATUS
   {one line per active conflict zone: UNCHANGED / ESCALATING / DE-ESCALATING}
   
   ### 📊 MARKET INTELLIGENCE
   {VIX level, key commodity moves, notable financial events}
   
   ### 🌐 SPACE & ENVIRONMENT
   {space weather, major natural events}
   
   ### 📰 KEY DEVELOPMENTS
   {3 bullet points from news}
   
   ### 🛰️ ISS PASSES TODAY
   {list if location configured, else omit section}
   
   SENTINEL v2 | Generated {TIMESTAMP} UTC"

Email: HTML version with same structure + SENTINEL header graphic
Telegram: same content with Markdown formatting
ntfy: title "🌅 SENTINEL Morning Briefing" + first 500 chars

Stored in SQLite: morning_briefing_log (full text, delivery status)

════════════════════════════════════════════════════════════════
WEEKLY DIGEST
════════════════════════════════════════════════════════════════

Same structure as morning briefing but covers 7 days.
Sent on configured day at configured time.

Additional sections for weekly:
  Most active regions this week
  Top 5 events by significance
  Trend: escalating/de-escalating conflict zones
  Provider reliability stats for the week
  Financial: weekly % changes for watchlist symbols

════════════════════════════════════════════════════════════════
ACKNOWLEDGE SYSTEM
════════════════════════════════════════════════════════════════

TIER 3+ alerts get pinned panel with [ACKNOWLEDGE] button.
On ACK: unpins card, logs to SQLite (ack_log table), sends Telegram confirmation.
Telegram confirmation: "✅ Alert acknowledged: {title} by operator at {time}"

API:
  POST /api/events/{id}/acknowledge — mark event acknowledged
  GET  /api/events/unacknowledged   — list unacknowledged TIER 3+ events

════════════════════════════════════════════════════════════════
NEW API ENDPOINTS (NOTIFICATIONS)
════════════════════════════════════════════════════════════════

GET  /api/notifications/config            — notification config
POST /api/notifications/config            — update config
POST /api/notifications/test/telegram     — send test
POST /api/notifications/test/email        — send test
POST /api/notifications/test/slack        — send test
POST /api/notifications/test/discord      — send test
POST /api/notifications/test/ntfy         — send test
POST /api/notifications/test/pushover     — send test
GET  /api/notifications/rules             — alert rules list
POST /api/notifications/rules             — add rule
PUT  /api/notifications/rules/{id}        — update rule
DEL  /api/notifications/rules/{id}        — delete rule
GET  /api/notifications/history           — sent notification log
GET  /api/geofences                       — list geofences
POST /api/geofences                       — create geofence
PUT  /api/geofences/{id}                  — update geofence
DEL  /api/geofences/{id}                  — delete geofence
GET  /api/intel/morning-briefing          — get/trigger briefing
GET  /api/intel/weekly-digest             — get/trigger digest
POST /api/events/{id}/acknowledge         — ACK alert
GET  /api/events/unacknowledged           — unACK alerts
