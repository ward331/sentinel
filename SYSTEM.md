# GUNTHER SERVER — SYSTEM KNOWLEDGE

## ARCHITECTURE
You run on Ed's Ubuntu server "Gunther". Here's everything you need to know to be effective.

## SERVICES & PORTS

| Port  | Service                  | Tech        | Manages with                                          |
|-------|--------------------------|-------------|-------------------------------------------------------|
| 4000  | Mission Control          | Next.js+SQLite | `systemctl --user restart mission-control-kanban`    |
| 4080  | Legacy Dashboard         | Node.js     | `systemctl --user restart critical-site-4080`         |
| 4085  | Gunther Telegram Bot     | Node.js     | `systemctl --user restart gunther-telegram-4085`      |
| 4090  | Enhanced Governor Server | Node.js     | `systemctl --user restart gunther-governor-enhanced-4090` |
| 4095  | Card Shark Arena         | Node.js+SQLite | `systemctl --user restart card-shark-arena`        |
| 8317  | CLIProxyAPI              | Go          | `systemctl --user restart openclaw-cliproxy`          |
| 18890 | Governor LLM             | Node.js     | `systemctl --user restart openclaw-governor`          |

## COMMON SYSADMIN COMMANDS

### Service Management
```bash
systemctl --user status SERVICE          # Check service health
systemctl --user restart SERVICE         # Restart
systemctl --user stop SERVICE            # Stop
systemctl --user start SERVICE           # Start
systemctl --user list-units --type=service  # List all services
journalctl --user -u SERVICE -n 50       # Recent logs
journalctl --user -u SERVICE -f          # Follow logs live
journalctl --user -u SERVICE --since "10 min ago"  # Time-filtered
```

### Process & Network
```bash
ss -tlnp | grep PORT         # What's listening on a port
ps aux | grep PROCESS         # Find process
kill PID                      # Stop process
lsof -i :PORT                 # Who's using a port
free -h                       # Memory usage
df -h                         # Disk usage
top -bn1 | head -20           # CPU/memory snapshot
uptime                        # System uptime and load
```

### Git Operations
```bash
git status                    # Working tree status
git add FILE                  # Stage file
git add -A                    # Stage everything
git commit -m "message"       # Commit
git push origin main          # Push to remote
git pull origin main          # Pull latest
git log --oneline -10         # Recent commits
git diff                      # Unstaged changes
git diff --cached             # Staged changes
git stash / git stash pop     # Temporary storage
```

### Build & Deploy
```bash
# Go projects
go build ./cmd/sentinel/                    # Build
go vet ./...                                # Check for issues
go test ./...                               # Run tests
GOOS=windows GOARCH=amd64 go build -o bin/NAME.exe ./cmd/sentinel/  # Cross-compile

# Node.js / Next.js projects
npm install                                 # Install deps
npx next build                              # Build MC (required before restart)
node server.js                              # Run directly

# Python
python3 script.py                           # Run script
pip3 install PACKAGE --break-system-packages  # Install package (no venv)
```

## MISSION CONTROL API (port 4000)
All endpoints accept JSON. Internal requests (localhost) need no auth.

### Tickets
```bash
# List open tickets
curl -s 'http://localhost:4000/api/tickets?status=new,assigned,in_progress,pending,testing,review&limit=50'

# Search tickets
curl -s 'http://localhost:4000/api/tickets?search=QUERY&limit=50'

# Get single ticket
curl -s http://localhost:4000/api/tickets/TICKET_ID

# Create ticket
curl -s -X POST http://localhost:4000/api/tickets \
  -H 'Content-Type: application/json' \
  -d '{"title":"...","description":"...","priority":"medium"}'

# Update ticket (NOT for closing)
curl -s -X PATCH http://localhost:4000/api/tickets/TICKET_ID \
  -H 'Content-Type: application/json' \
  -d '{"status":"in_progress","priority":"high"}'

# Close/resolve a ticket (MUST use this endpoint, not PATCH)
curl -s -X POST http://localhost:4000/api/tickets/TICKET_ID/resolve \
  -H 'Content-Type: application/json' \
  -d '{"resolution":"Fixed the issue.","actor":"sentinel-bot"}'

# Add comment to ticket
curl -s -X POST http://localhost:4000/api/tickets/TICKET_ID/comments \
  -H 'Content-Type: application/json' \
  -d '{"author":"sentinel-bot","content":"Working on this..."}'
```

Valid statuses: new, assigned, in_progress, pending, testing, review, resolved, closed
There is NO "open" status. Use the comma-separated list for all open tickets.

### Agents & Teams
```bash
curl -s http://localhost:4000/api/agents           # List agents
curl -s http://localhost:4000/api/teams            # List teams
```

### Jobs (21,311 in universe)
```bash
curl -s 'http://localhost:4000/api/jobs?search=QUERY'     # Search
curl -s 'http://localhost:4000/api/jobs?section=stats'     # Stats
```

### Oracle (Astrology/HD/Numerology)
```bash
curl -s 'http://localhost:4000/api/oracle?section=gate&number=51'   # Gate detail
curl -s -X POST http://localhost:4000/api/oracle \
  -H 'Content-Type: application/json' \
  -d '{"action":"ask","question":"..."}'                            # Ask oracle
```

### Library (1,803+ items)
```bash
curl -s 'http://localhost:4000/api/proxy/library?search=QUERY'     # Search
curl -s 'http://localhost:4000/api/proxy/library?section=stats'    # Stats
```

### Mail (Agent-to-Agent)
```bash
curl -s 'http://localhost:4000/api/mail'                           # List mail
curl -s -X POST http://localhost:4000/api/mail \
  -H 'Content-Type: application/json' \
  -d '{"from":"sentinel-bot","to":"ops-radar","subject":"...","body":"..."}'
```

## GOVERNOR LLM (port 18890)
Routes LLM requests across 31+ providers.
```bash
curl -s http://localhost:18890/health              # Provider status
curl -s -X POST http://localhost:18890/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{"model":"auto","messages":[{"role":"user","content":"hello"}]}'
```

Key providers: Ollama (local NAS 172.31.5.58:11434, always available), Groq, Gemini, Anthropic

## CARD SHARK ARENA (port 4095)
```bash
curl -s http://localhost:4095/api/sessions         # List game sessions
curl -s -X POST http://localhost:4095/api/sessions \
  -d '{"game_type":"blackjack","starting_bankroll":10000}'
```

## KEY FILE PATHS
```
/home/ed/.openclaw/workspace-sentinel-backend/     # THIS workspace (SENTINEL Go project)
/home/ed/.openclaw/workspace/mission-control-kanban/ # Mission Control source
/home/ed/.openclaw/workspace/governor/bin/governor.js # Governor source
/home/ed/.openclaw/workspace-telegram/gunther-telegram-bot.js # Gunther bot source
/home/ed/Gunther/projects/card-shark-arena/        # Card Shark source
/home/ed/Gunther/projects/sentinel/                # SENTINEL project docs
/home/ed/Gunther/Books/                            # PDF library (2,468 files)
/home/ed/.openclaw/workspace/mission-control-kanban/mission-control.db # Main DB
```

## PROGRAMMING BEST PRACTICES

### Go
- Always `go vet ./...` and `go build` after changes
- Cross-compile: `GOOS=os GOARCH=arch go build -o bin/name ./cmd/entry/`
- Use `context.Context` for cancellation, not bare goroutines
- Handle errors explicitly, don't ignore them
- Use `defer` for cleanup (file handles, mutexes, DB connections)
- Prefer `fmt.Errorf("context: %w", err)` for error wrapping

### Node.js
- For MC changes: edit → `npx next build` → `systemctl --user restart mission-control-kanban`
- For standalone services: edit → `systemctl --user restart SERVICE`
- No build step needed for plain Node.js services (telegram bot, governor, card shark)

### Python
- Use `--break-system-packages` with pip (no venv on this server)
- Scripts in `/home/ed/.openclaw/workspace/bin/`

### General
- Read before editing — always check current file content first
- Test after changes — run builds, check logs
- Don't break running services — build first, restart second
- Commit meaningful changes with descriptive messages

## DATABASE
SQLite at `/home/ed/.openclaw/workspace/mission-control-kanban/mission-control.db`
19 migrations. Key tables: agents, tickets, teams, mail_messages, library_items, jobs_universe, oracle_* (13 tables), events

## SENTINEL PROJECT STRUCTURE
```
cmd/sentinel/main.go          # Entry point, v2.0.0
internal/
  api/                         # HTTP handlers, SSE streaming, middleware
  alert/                       # Alert rules engine (email, Slack, Discord, Telegram, webhook)
  config/                      # V1/V2 config management
  core/                        # Legacy poller (superseded by internal/poller)
  filter/                      # Advanced filtering (30+ operators, geofencing)
  model/                       # Event, Location, Badge structs
  poller/                      # Production poller with stats and deduplication
  provider/                    # 24 data providers (USGS, GDACS, OpenSky, NOAA, etc.)
  storage/                     # SQLite with FTS5 and R*Tree spatial indexing
  health/                      # Health check registry
  metrics/                     # API metrics collection
```
24 providers: earthquakes, aviation, weather, conflicts, markets, health alerts, satellites, piracy, OSINT

## SLA POLICY
- Critical: 4h response, 2h check-ins
- High: 12h response, 4h check-ins
- Medium: 48h response, 12h check-ins
- Low: 72h response, 24h check-ins
