# Security Policy

## Supported Versions

| Version | Supported          |
|---------|--------------------|
| 3.x     | :white_check_mark: |
| 2.x     | :x:                |
| 1.x     | :x:                |

## Reporting a Vulnerability

If you discover a security vulnerability in SENTINEL, please report it responsibly.

**Do NOT open a public issue for security vulnerabilities.**

Instead, email: **security@gunther.local** (or contact Ed directly)

### What to include:
- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

### What to expect:
- Acknowledgment within 48 hours
- Assessment and fix timeline within 7 days
- Credit in the changelog (unless you prefer anonymity)

## Security Design

### API
- All API endpoints are localhost-only by default
- External access requires explicit `--host 0.0.0.0` flag
- CORS headers configurable (default: same-origin)
- No authentication on localhost (trusted internal network)
- Bearer token authentication available for external access

### Data Storage
- SQLite database with WAL mode
- API keys encrypted at rest (AES-256-GCM)
- No plaintext secrets in config file after first encryption
- Database files excluded from git via .gitignore

### Providers
- All HTTP requests use timeouts (10s default)
- No credentials sent to third-party APIs beyond documented auth
- Tier 0 sources require zero authentication
- Tier 1+ keys stored encrypted in config

### Notifications
- Bot tokens and webhook URLs encrypted in config
- SMTP passwords encrypted in config
- No notification content is logged (only send status)

### Build
- CGO_ENABLED=0 — no C dependencies
- Minimal Alpine Docker image with non-root user
- Reproducible builds with pinned Go version
- Dependencies audited and documented in DEPENDENCIES.md
