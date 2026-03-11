# Contributing to SENTINEL

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR_USER/sentinel.git`
3. Create a branch: `git checkout -b feature/my-feature`
4. Make your changes
5. Test: `make test`
6. Build: `make build`
7. Commit: `git commit -m "Add my feature"`
8. Push: `git push origin feature/my-feature`
9. Open a Pull Request

## Development Setup

### Prerequisites
- Go 1.26+
- Make
- SQLite3 (for debugging)

### Quick Start
```bash
git clone https://github.com/ward331/sentinel.git
cd sentinel
make build
bin/sentinel-linux-amd64 --wizard
```

### Project Structure
```
cmd/sentinel/     — Entry point
internal/
  api/            — HTTP handlers and SSE
  config/         — Configuration management
  engine/         — Intelligence (correlation, truth, anomaly, signal board)
  intel/          — AI briefing and news aggregation
  model/          — Data structures
  notify/         — Notification channels
  poller/         — Provider polling loop
  provider/       — Data source adapters (33 tier-0, 11 tier-1)
  storage/        — SQLite database layer
web/              — Frontend (vanilla HTML/CSS/JS)
scripts/          — Deployment and ops scripts
docs/             — Documentation
```

## Coding Standards

### Go Code
- `go vet ./...` must pass
- `go test ./...` must pass
- CGO_ENABLED=0 always — no C dependencies
- `filepath.Join()` for paths — never string concatenation
- `context.Context` for cancellation
- Timeouts on all HTTP clients (10s default)
- `recover()` in all goroutines
- Parameterized SQL queries only — zero string formatting
- Error wrapping: `fmt.Errorf("context: %w", err)`

### Frontend Code
- Vanilla JS only — no npm, no build step, no frameworks
- Mobile-first responsive design
- Touch targets >= 44px
- Dark theme default
- Works offline (IndexedDB cache)

### Adding a New Provider
1. Create `internal/provider/your_source.go`
2. Implement the `Provider` interface
3. Register in provider registry
4. Add to `docs/PROVIDERS.md`
5. Add test in `internal/provider/your_source_test.go`

### Adding a Notification Channel
1. Create `internal/notify/your_channel.go`
2. Implement the `Channel` interface
3. Register in dispatcher
4. Add config fields
5. Add to `docs/CONFIGURATION.md`

## Commit Messages
- Use present tense: "Add feature" not "Added feature"
- Keep first line under 72 characters
- Reference issues: "Fix #123 — description"
- Prefix with area: "providers: Add ACLED conflict data"

## Pull Requests
- One feature/fix per PR
- Include tests for new functionality
- Update docs if behavior changes
- Screenshots for UI changes
- `make test` and `make build` must pass

## License
By contributing, you agree that your contributions will be licensed under the same license as the project.
