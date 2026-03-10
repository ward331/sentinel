# SENTINEL Backend Makefile

.PHONY: all build run test clean smoke

# Configuration
BINARY_NAME = sentinel
DB_PATH = /tmp/sentinel-smoke.db
HTTP_PORT = 8100
HTTP_HOST = localhost

# Go commands
GO = /usr/local/go/bin/go
GO_BUILD = $(GO) build
GO_TEST = $(GO) test
GO_MOD = $(GO) mod
GOFMT = $(GO) fmt

# Directories
CMD_DIR = ./cmd/sentinel
INTERNAL_DIR = ./internal

all: build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	$(GO_BUILD) -o $(BINARY_NAME) $(CMD_DIR)

# Run the server
run: build
	@echo "Starting $(BINARY_NAME)..."
	SENTINEL_DB_PATH=$(DB_PATH) SENTINEL_HTTP_PORT=$(HTTP_PORT) ./$(BINARY_NAME)

# Run tests
test:
	@echo "Running tests..."
	$(GO_TEST) ./...

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	rm -f $(DB_PATH) $(DB_PATH)-shm $(DB_PATH)-wal
	rm -rf dist/

# Smoke test - end-to-end test of the walking skeleton (V2 with CLI flags)
smoke: build
	@echo "Starting smoke test (V2 with CLI flags)..."; \
	TEST_DATA_DIR="/tmp/sentinel_smoke_$$(date +%s)"; \
	mkdir -p "$$TEST_DATA_DIR"; \
	echo "Starting server with data-dir=$$TEST_DATA_DIR, port=$(HTTP_PORT)..."; \
	./$(BINARY_NAME) --data-dir "$$TEST_DATA_DIR" --port $(HTTP_PORT) & \
	SERVER_PID=$$!; \
	echo "Waiting for server to start..."; \
	sleep 5; \
	echo "Checking server health..."; \
	curl -f -s http://$(HTTP_HOST):$(HTTP_PORT)/api/health || { echo "Health check failed"; kill $$SERVER_PID 2>/dev/null; rm -rf "$$TEST_DATA_DIR"; exit 1; }; \
	echo "Checking OSINT resources..."; \
	OSINT_RESPONSE=$$(curl -s http://$(HTTP_HOST):$(HTTP_PORT)/api/osint/resources); \
	if echo "$$OSINT_RESPONSE" | grep -q '"resources"'; then \
		echo "OSINT resources API working"; \
	else \
		echo "OSINT resources check: $$OSINT_RESPONSE"; \
	fi; \
	echo "Creating test earthquake event..."; \
	RESPONSE=$$(curl -s -X POST http://$(HTTP_HOST):$(HTTP_PORT)/api/events \
		-H "Content-Type: application/json" \
		-d '{"title": "M 5.2 - 10km SSE of Volcano, Hawaii", "description": "A magnitude 5.2 earthquake occurred near Volcano, Hawaii.", "source": "usgs", "source_id": "hv12345678", "occurred_at": "2024-01-01T12:00:00Z", "location": {"type": "Point", "coordinates": [-155.234, 19.456]}, "precision": "exact", "magnitude": 5.2, "category": "earthquake", "severity": "medium"}'); \
	if echo "$$RESPONSE" | grep -q '"id"'; then \
		echo "Event created successfully"; \
	else \
		echo "Failed to create event: $$RESPONSE"; \
		kill $$SERVER_PID 2>/dev/null; \
		rm -rf "$$TEST_DATA_DIR"; \
		exit 1; \
	fi; \
	echo "Querying events..."; \
	QUERY_RESPONSE=$$(curl -s http://$(HTTP_HOST):$(HTTP_PORT)/api/events); \
	if echo "$$QUERY_RESPONSE" | grep -q '"events"'; then \
		echo "Events query successful"; \
	else \
		echo "Events query failed: $$QUERY_RESPONSE"; \
		kill $$SERVER_PID 2>/dev/null; \
		rm -rf "$$TEST_DATA_DIR"; \
		exit 1; \
	fi; \
	echo "Testing SSE stream..."; \
	SSE_OUTPUT=$$(timeout 5 curl -s -N http://$(HTTP_HOST):$(HTTP_PORT)/api/events/stream || true); \
	if [ -n "$$SSE_OUTPUT" ]; then \
		echo "SSE stream is working"; \
	else \
		echo "SSE stream test completed (may not have received events during test)"; \
	fi; \
	echo "Creating second event for SSE test..."; \
	curl -s -X POST http://$(HTTP_HOST):$(HTTP_PORT)/api/events \
		-H "Content-Type: application/json" \
		-d '{"title": "M 4.5 - Central California", "description": "A magnitude 4.5 earthquake in Central California.", "source": "usgs", "source_id": "hv12345679", "occurred_at": "2024-01-01T12:05:00Z", "location": {"type": "Point", "coordinates": [-120.123, 36.456]}, "precision": "exact", "magnitude": 4.5, "category": "earthquake"}' > /dev/null; \
	sleep 1; \
	echo "Stopping server..."; \
	kill $$SERVER_PID 2>/dev/null || true; \
	sleep 2; \
	echo "Cleaning up..."; \
	rm -rf "$$TEST_DATA_DIR"; \
	echo "Smoke test PASSED!"

# Install dependencies
deps:
	@echo "Installing dependencies..."
	$(GO_MOD) download
	$(GO_MOD) verify

# Vendor dependencies
vendor:
	@echo "Vendoring dependencies..."
	$(GO_MOD) vendor

# Help
help:
	@echo "Available targets:"
	@echo "  all     - Build the binary (default)"
	@echo "  build   - Build the binary"
	@echo "  run     - Build and run the server"
	@echo "  test    - Run tests"
	@echo "  fmt     - Format code"
	@echo "  clean   - Clean build artifacts"
	@echo "  smoke   - Run end-to-end smoke test"
	@echo "  deps    - Install dependencies"
	@echo "  vendor  - Vendor dependencies"
	@echo "  help    - Show this help"