# MANIFEST 08 — FINAL REVIEW & COMPLETION CHECKLIST
# ====================================================
# Covers: Stage 11 — Complete audit, all gates, final smoke test.
# Run this ONLY after all previous stages are complete.

════════════════════════════════════════════════════════════════
PORTABILITY AUDIT
════════════════════════════════════════════════════════════════

Run each command. All must return 0 results (docs/ and manifests/ exempt):

  grep -r "/home/" --include="*.go" --include="*.html" --include="*.js" .
  grep -r "172\.31\." --include="*.go" --include="*.html" .
  grep -r "8177356632" --include="*.go" .
  grep -r "eyJhbGci" --include="*.go" --include="*.html" .
  grep -r "localhost:8080" --include="*.js" --include="*.html" .
  grep -r "127\.0\.0\.1" --include="*.go" --include="*.html" .

Filepath audit (must all use filepath.Join, not string concat):
  grep -rn '+ "/' --include="*.go" .
  grep -rn '"/' --include="*.go" . | grep -v "^//"
  Any path starting with / in Go code = potential hardcode

════════════════════════════════════════════════════════════════
BUILD AUDIT
════════════════════════════════════════════════════════════════

  □ make build-linux     exits 0
  □ make build-linux-arm exits 0
  □ make build-mac       exits 0
  □ make build-mac-arm   exits 0
  □ make build-windows   exits 0

  □ CGO check:
    file dist/sentinel-linux-amd64
    → Must show: "statically linked" (not "dynamically linked")

  □ Binary size check:
    ls -lh dist/
    → All binaries < 80MB (web assets embedded)
    → If > 80MB: check for accidentally embedded large files

  □ make smoke passes on Linux binary

════════════════════════════════════════════════════════════════
FUNCTIONAL AUDIT
════════════════════════════════════════════════════════════════

Start fresh with no config:
  rm -f ~/.config/sentinel/config.json (or platform equivalent)
  ./dist/sentinel-linux-amd64

  □ Setup wizard auto-opens at http://localhost:8080/setup
  □ Step 1-8 all render correctly
  □ Cesium token test validates (use test token from ion.cesium.com)
  □ Notification test buttons function
  □ Setup completes → redirects to /
  □ Dashboard loads with globe

After setup, verify all views:
  □ /?view=globe    — globe renders with Cesium
  □ /?view=feed     — text feed loads, SSE events appear
  □ /?view=split    — both panels render, divider draggable
  □ /?view=media    — media wall renders, stream picker works
  □ /?view=command  — all 4 panels render

Provider checks:
  □ GET /api/providers/health — all zero-key providers show "active" after 2min
  □ USGS events appearing in feed
  □ GDACS events appearing in feed
  □ Airplanes.live events appearing in feed

Feed checks:
  □ Filter chips work (filter to specific category)
  □ Search input filters feed in real time
  □ Event card [View on Globe] switches to globe and flies to location
  □ [Save] bookmarks event
  □ [Note] opens and saves note
  □ [OSINT Sources] panel opens with relevant sources

Financial checks:
  □ GET /api/financial/overview returns prices
  □ VIX value shown in market overview widget
  □ CoinGecko crypto prices appearing
  □ Financial events appear in feed when thresholds met

OSINT Resources checks:
  □ /osint-resources page loads
  □ Platform profiles section populated with suggestions
  □ "Mark as Following" button works
  □ Radio tab shows frequency guide
  □ WebSDR receiver suggestions show for a known event location
  □ Broadcastify feed suggestions populate

News intelligence checks:
  □ GET /api/news returns articles from at least 3 sources
  □ News panel renders in Command Center view
  □ AI briefing generates (GET /api/intel/news-briefing)
  □ Relevance scoring present on articles
  □ [Add custom RSS] works, feed appears in list

Notification checks:
  □ POST /api/notifications/test/telegram → Telegram message received
  □ POST /api/notifications/test/email → email received (if configured)
  □ Alert rules engine fires correctly
  □ TIER 4 event pins to feed top (simulate with test event)

Settings checks:
  □ Settings page loads all tabs
  □ All tabs save changes via PATCH /api/config
  □ Financial tab shows watchlist management
  □ OSINT Resources tab shows profile and radio management
  □ API Keys tab: test buttons function, masking works

CLI flag checks:
  □ --version prints version and exits
  □ --config /path/to/config.json uses that config
  □ --data-dir /path/to/dir uses that directory
  □ --no-browser doesn't open browser
  □ --setup forces wizard

════════════════════════════════════════════════════════════════
DOCUMENTATION AUDIT
════════════════════════════════════════════════════════════════

All files must exist and be > 500 bytes:

  □ README.md
  □ LICENSE
  □ SECURITY.md
  □ DEPENDENCIES.md
  □ CHANGELOG.md
  □ docs/INSTALL-WINDOWS.md
  □ docs/INSTALL-MACOS.md
  □ docs/INSTALL-LINUX.md
  □ docs/INSTALL-DOCKER.md
  □ docs/CONFIGURATION.md
  □ docs/PROVIDERS.md
  □ docs/FINANCIAL-ALERTS.md
  □ docs/NOTIFICATIONS.md
  □ docs/FEED-DASHBOARD.md
  □ docs/OSINT-RESOURCES.md
  □ docs/MEDIA-WALL.md
  □ docs/API.md
  □ docs/TROUBLESHOOTING.md
  □ docs/CONTRIBUTING.md
  □ docs/OSINT-PROFILES-DIRECTORY.md
  □ docs/RADIO-FREQUENCY-GUIDE.md

Check: wc -l docs/*.md | sort -n
Any file < 50 lines is likely incomplete — expand it.

════════════════════════════════════════════════════════════════
SECURITY AUDIT
════════════════════════════════════════════════════════════════

  □ No credentials in any source file (grep -r "password\|secret\|token" --include="*.go")
     Verify any hits are config struct fields, not hardcoded values

  □ Config encryption works:
     Set a password in config, restart, verify it reads "enc:" prefix in JSON

  □ All /api/ endpoints (except /api/setup/* and /api/health):
     If cfg.Server.AuthEnabled: require Authorization: Bearer {token}
     Test: curl without auth → 401
     Test: curl with correct token → 200

  □ No path traversal in file serving:
     embed.FS is safe by design — verify no os.Open calls in HTTP handlers

  □ SQLite injection check:
     All queries use parameterized statements (?, $1) — no string formatting in SQL

════════════════════════════════════════════════════════════════
PERFORMANCE AUDIT
════════════════════════════════════════════════════════════════

  □ Memory with 35,000 satellites (max setting):
     Should not exceed 500MB RAM with full satellite load

  □ Memory with essential satellites (~50 objects):
     Should not exceed 100MB RAM normally

  □ Database size after 24hr of all providers:
     ls -lh {data_dir}/sentinel.db
     Should be < 100MB

  □ API response times (all under 200ms on local machine):
     curl -o /dev/null -s -w "%{time_total}" http://localhost:8080/api/health
     curl -o /dev/null -s -w "%{time_total}" http://localhost:8080/api/events
     curl -o /dev/null -s -w "%{time_total}" http://localhost:8080/api/financial/overview

  □ Provider goroutine leak check:
     After 30min run: runtime.NumGoroutine() stable (not growing unbounded)

════════════════════════════════════════════════════════════════
FINAL COMPLETION REPORT
════════════════════════════════════════════════════════════════

After all checks pass, output this exact block:

╔══════════════════════════════════════════════════════════════════╗
║  SENTINEL V2 — BUILD COMPLETE                                    ║
║  Version: 2.0.0                                                  ║
╠══════════════════════════════════════════════════════════════════╣
║  BINARIES:                                                       ║
║    dist/sentinel-linux-amd64        {size}                       ║
║    dist/sentinel-linux-arm64        {size}                       ║
║    dist/sentinel-darwin-amd64       {size}                       ║
║    dist/sentinel-darwin-arm64       {size}                       ║
║    dist/sentinel-windows-amd64.exe  {size}                       ║
╠══════════════════════════════════════════════════════════════════╣
║  INSTALLERS:                                                     ║
║    dist/SENTINEL-Setup-v2.0.0-windows-x64.exe                   ║
║    dist/SENTINEL-v2.0.0-macos-x64.dmg                           ║
║    dist/SENTINEL-v2.0.0-macos-arm64.dmg                         ║
║    dist/sentinel-v2.0.0-linux-amd64.tar.gz                      ║
║    dist/sentinel-v2.0.0-linux-arm64.tar.gz                      ║
╠══════════════════════════════════════════════════════════════════╣
║  NEW IN V2:                                                      ║
║    ✅ Financial alerts (VIX, oil, crypto, sanctions, SEC)        ║
║    ✅ Geopolitical correlation engine                            ║
║    ✅ 5 view modes (Globe/Feed/Split/Media/Command)              ║
║    ✅ Text alert feed with subscriptions                         ║
║    ✅ OSINT profile suggestions ({N} pre-loaded profiles)        ║
║    ✅ Ham/military radio integration with WebSDR                 ║
║    ✅ News intelligence aggregator ({N} sources)                 ║
║    ✅ Media wall with YouTube embeds                             ║
║    ✅ Morning briefing (AI-generated)                            ║
║    ✅ All notification channels (Telegram/Slack/Discord/         ║
║        Email/ntfy/Pushover)                                      ║
║    ✅ Cross-platform installers (Windows/macOS/Linux/Docker)     ║
║    ✅ Full documentation ({N} docs files)                        ║
║    ✅ Single binary, zero dependencies                           ║
╠══════════════════════════════════════════════════════════════════╣
║  ALL CHECKS: PASSED                                              ║
║  make smoke: PASSED                                              ║
║  Ready for distribution.                                         ║
╚══════════════════════════════════════════════════════════════════╝
