# SENTINEL V2 Dependencies

## Current Dependencies (from V1)

### Go Modules
- `modernc.org/sqlite` - Pure Go SQLite driver (CGO-free)
- `github.com/google/uuid` - UUID generation

### System Dependencies
- Go 1.21+ (for building)
- SQLite 3.x (runtime, via modernc.org/sqlite)

## V2 Planned Dependencies

### Approved for V2 (per master instructions)
- `golang.org/x/sys` - Windows service + platform utilities
- `github.com/getlantern/systray` - System tray (Windows + macOS)
- `github.com/gorilla/websocket` - WebSocket support (if needed)

### Dependency Rules
1. NO CGO dependencies allowed
2. All dependencies must be pure Go
3. New dependencies must be added to this file
4. Version constraints in go.mod must be explicit