# SENTINEL V3 Dependencies

All dependencies are pure Go (no CGO required). SENTINEL builds as a fully static binary.

## Direct Dependencies

| Module | Version | License | Purpose |
|--------|---------|---------|---------|
| `github.com/google/uuid` | v1.6.0 | BSD-3-Clause | UUID generation for event IDs |
| `golang.org/x/time` | v0.14.0 | BSD-3-Clause | Token bucket rate limiter |
| `modernc.org/sqlite` | v1.29.5 | BSD-3-Clause | Pure Go SQLite database driver (no CGO) |

## Indirect Dependencies

| Module | Version | License | Purpose |
|--------|---------|---------|---------|
| `github.com/dustin/go-humanize` | v1.0.1 | MIT | Human-readable formatting (used by sqlite) |
| `github.com/getlantern/context` | v0.0.0-20190109183933 | Apache-2.0 | Context utilities (systray dependency) |
| `github.com/getlantern/errors` | v0.0.0-20190325191628 | Apache-2.0 | Error handling (systray dependency) |
| `github.com/getlantern/golog` | v0.0.0-20190830074920 | Apache-2.0 | Logging (systray dependency) |
| `github.com/getlantern/hex` | v0.0.0-20190417191902 | Apache-2.0 | Hex encoding (systray dependency) |
| `github.com/getlantern/hidden` | v0.0.0-20190325191715 | Apache-2.0 | Hidden data (systray dependency) |
| `github.com/getlantern/ops` | v0.0.0-20190325191751 | Apache-2.0 | Operations tracking (systray dependency) |
| `github.com/getlantern/systray` | v1.2.2 | Apache-2.0 | Cross-platform system tray icon |
| `github.com/go-stack/stack` | v1.8.0 | MIT | Stack trace utilities (systray dependency) |
| `github.com/golang-jwt/jwt/v5` | v5.3.1 | MIT | JWT token support (auth middleware) |
| `github.com/gorilla/mux` | v1.8.1 | BSD-3-Clause | HTTP router with URL path parameters |
| `github.com/hashicorp/golang-lru/v2` | v2.0.7 | MPL-2.0 | LRU cache (sqlite dependency) |
| `github.com/mattn/go-isatty` | v0.0.16 | MIT | TTY detection (sqlite dependency) |
| `github.com/ncruces/go-strftime` | v0.1.9 | MIT | Strftime formatting (sqlite dependency) |
| `github.com/oxtoacart/bpool` | v0.0.0-20190530202638 | Apache-2.0 | Buffer pool (systray dependency) |
| `github.com/remyoudompheng/bigfft` | v0.0.0-20230129092748 | BSD-3-Clause | FFT (sqlite math dependency) |
| `golang.org/x/sys` | v0.16.0 | BSD-3-Clause | OS-level system calls |
| `modernc.org/gc/v3` | v3.0.0-20240107210532 | BSD-3-Clause | Garbage collector (sqlite dependency) |
| `modernc.org/libc` | v1.41.0 | BSD-3-Clause | C library emulation (sqlite dependency) |
| `modernc.org/mathutil` | v1.6.0 | BSD-3-Clause | Math utilities (sqlite dependency) |
| `modernc.org/memory` | v1.7.2 | BSD-3-Clause | Memory allocation (sqlite dependency) |
| `modernc.org/strutil` | v1.2.0 | BSD-3-Clause | String utilities (sqlite dependency) |
| `modernc.org/token` | v1.1.0 | BSD-3-Clause | Token types (sqlite dependency) |

## Dependency Rules

1. **No CGO dependencies** -- all modules must be pure Go for static cross-compilation
2. **Explicit version pinning** -- all versions are locked in `go.mod` and `go.sum`
3. **Minimal direct dependencies** -- only 3 direct modules; everything else is transitive
4. **License compliance** -- all dependencies use permissive licenses (MIT, BSD, Apache-2.0, MPL-2.0)

## Updating Dependencies

```bash
# Update all to latest compatible versions
go get -u ./...
go mod tidy

# Update a specific dependency
go get github.com/google/uuid@latest
go mod tidy

# Verify checksums
go mod verify
```

## Build Tags

The build uses `CGO_ENABLED=0` to ensure the SQLite driver uses the pure-Go implementation from `modernc.org/sqlite` rather than requiring a C compiler.
