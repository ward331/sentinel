# SENTINEL V2 — MASTER BUILD INSTRUCTIONS
# =========================================
# Drop this folder into your SENTINEL workspace root.
# Tell the AI agent: "Read SENTINEL_V2_MASTER_INSTRUCTIONS.md and execute it."
# The agent will read each manifest in order and build everything stage by stage.

## BEFORE YOU START

Read these files first (already in your workspace):
- SOUL.md
- USER.md
- Any existing SENTINEL source files

This instruction set upgrades SENTINEL from its current state to V2.
V2 adds: financial market alerts, OSINT platform resource guides,
ham/military radio integration, complete feed dashboard, media wall,
cross-platform distribution, and full documentation.

Do NOT skip stages. Do NOT batch stages together.
Report completion of EVERY stage before proceeding to the next.
make smoke MUST pass after Stages 1, 2, 3, 6, and 9.
After every stage: commit a one-line summary to CHANGELOG.md.

---

## BUILD ORDER — EXECUTE EXACTLY IN THIS SEQUENCE

╔══════════════════════════════════════════════════════════════════╗
║  STAGE 1  │  Read: manifests/01_PORTABILITY_CONFIG.md           ║
║           │  Task: Scrub all hardcoded values. Build unified     ║
║           │        config system. Merge to single binary.        ║
║           │  Gate: make smoke passes. grep for /home/ returns 0. ║
╠══════════════════════════════════════════════════════════════════╣
║  STAGE 2  │  Read: manifests/01_PORTABILITY_CONFIG.md           ║
║           │  Task: CLI flags. Service installer. System tray.    ║
║           │        First-run setup wizard. Config encryption.    ║
║           │  Gate: --install-service works. Wizard opens.        ║
╠══════════════════════════════════════════════════════════════════╣
║  STAGE 3  │  Read: manifests/02_DATA_PROVIDERS.md               ║
║           │  Task: All zero-key data providers. Optional key     ║
║           │        providers. Provider health system.            ║
║           │  Gate: make smoke passes. All zero-key providers     ║
║           │        start on first run.                           ║
╠══════════════════════════════════════════════════════════════════╣
║  STAGE 4  │  Read: manifests/03_FINANCIAL_ALERTS.md             ║
║           │  Task: Financial data providers. Market alert        ║
║           │        types. Geopolitical correlation engine.       ║
║           │        Financial feed panel.                         ║
║           │  Gate: VIX, oil, crypto alerts firing in feed.       ║
╠══════════════════════════════════════════════════════════════════╣
║  STAGE 5  │  Read: manifests/04_FEED_MEDIA_DASHBOARD.md         ║
║           │  Task: All 5 view modes. Text alert feed. Event      ║
║           │        cards. Incident threading. Subscription       ║
║           │        system. Watchlist. Live ticker.               ║
║           │  Gate: All 5 views load. SSE feeds into all.         ║
╠══════════════════════════════════════════════════════════════════╣
║  STAGE 6  │  Read: manifests/04_FEED_MEDIA_DASHBOARD.md         ║
║           │  Task: News intelligence aggregator. RSS feeds.      ║
║           │        AI briefing panel. Cross-referencing.         ║
║           │        Media wall + YouTube embeds.                  ║
║           │  Gate: make smoke passes. RSS feeds in news panel.   ║
╠══════════════════════════════════════════════════════════════════╣
║  STAGE 7  │  Read: manifests/05_OSINT_RESOURCES.md              ║
║           │  Task: OSINT profile suggestions system. Platform    ║
║           │        guides. Ham/military radio integration.        ║
║           │        WebSDR embed. Signal type library.            ║
║           │  Gate: OSINT panel loads. WebSDR links resolve.      ║
╠══════════════════════════════════════════════════════════════════╣
║  STAGE 8  │  Read: manifests/06_NOTIFICATIONS_BRIEFING.md       ║
║           │  Task: All notification channels. Alert rules.       ║
║           │        Geofences. Morning briefing. Weekly digest.   ║
║           │  Gate: Telegram test sends. Email test sends.        ║
╠══════════════════════════════════════════════════════════════════╣
║  STAGE 9  │  Read: manifests/07_DISTRIBUTION_BUILD.md           ║
║           │  Task: Cross-platform Makefile. Windows installer.   ║
║           │        macOS DMG. Linux packages. Docker.            ║
║           │  Gate: make build-all succeeds all 5 targets.        ║
║           │        make smoke passes on Linux binary.            ║
╠══════════════════════════════════════════════════════════════════╣
║  STAGE 10 │  Read: manifests/07_DISTRIBUTION_BUILD.md           ║
║           │  Task: All documentation. Per-platform guides.       ║
║           │        API reference. OSINT resource docs.           ║
║           │  Gate: All docs/*.md files exist and non-empty.      ║
╠══════════════════════════════════════════════════════════════════╣
║  STAGE 11 │  Read: manifests/08_FINAL_REVIEW.md                 ║
║           │  Task: Full audit checklist. All gates verified.     ║
║           │        Final smoke test. Output build manifest.      ║
║           │  Gate: ALL checklist items pass.                     ║
╚══════════════════════════════════════════════════════════════════╝

---

## HOW TO REPORT STAGE COMPLETION

After completing each stage, output EXACTLY this block before continuing:

```
╔══════════════════════════════════╗
║  STAGE {N} COMPLETE              ║
║  Files changed: {list}           ║
║  New endpoints: {list or none}   ║
║  Tests: {make smoke PASS/SKIP}   ║
║  Next: STAGE {N+1}               ║
╚══════════════════════════════════╝
```

Then immediately read the next manifest and begin the next stage.

---

## RULES FOR ALL STAGES

1. NEVER hardcode: paths, IPs, credentials, tokens, user IDs
2. ALWAYS use filepath.Join() — never string concatenation with /
3. ALWAYS add config options for anything user-configurable
4. ALWAYS add the new feature to the setup wizard if user needs to configure it
5. ALWAYS add the new feature to the settings page
6. ALWAYS add documentation to the relevant docs/ file as you build
7. NEVER use CGO — CGO_ENABLED=0 must work for all builds
8. NEVER break existing functionality — make smoke must keep passing
9. ALL new HTML/JS goes in web/ directory, embedded via go:embed
10. ALL new DB tables use auto-migration on startup

---

## DEPENDENCY ORDER (read before writing any code)

The following Go packages are approved for use:
  modernc.org/sqlite          — already in use, pure Go SQLite
  golang.org/x/sys            — windows service + platform utils
  github.com/getlantern/systray — system tray (Windows + macOS)
  github.com/gorilla/websocket — WebSocket if needed
  standard library only for everything else

Do NOT add new dependencies without noting them in DEPENDENCIES.md.

---

## VERSION

SENTINEL V2 build version: 2.0.0
Bake into binary: -ldflags "-X main.Version=2.0.0"
Update CHANGELOG.md with each stage completion.

---

## IF YOU GET STUCK

If a stage fails or is ambiguous:
1. Check existing source files for patterns already established
2. Default to the simplest working implementation
3. Add a TODO comment and move on — do not stop the build
4. Note the TODO in the stage completion report

Begin now. Read SOUL.md, then USER.md, then start Stage 1.
