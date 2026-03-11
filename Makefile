VERSION ?= 3.0.0
BINARY = sentinel
GO = /usr/local/go/bin/go
LDFLAGS = -ldflags "-X main.Version=$(VERSION) -s -w"

.PHONY: build build-all build-linux build-linux-arm build-mac build-mac-arm build-windows test smoke clean docker deps fmt help

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o bin/$(BINARY)-linux-amd64 ./cmd/sentinel/

build-linux: build

build-linux-arm:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build $(LDFLAGS) -o bin/$(BINARY)-linux-arm64 ./cmd/sentinel/

build-mac:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GO) build $(LDFLAGS) -o bin/$(BINARY)-darwin-amd64 ./cmd/sentinel/

build-mac-arm:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GO) build $(LDFLAGS) -o bin/$(BINARY)-darwin-arm64 ./cmd/sentinel/

build-windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o bin/$(BINARY)-windows-amd64.exe ./cmd/sentinel/

build-all: build build-linux-arm build-mac build-mac-arm build-windows

test:
	$(GO) test ./...

smoke:
	@echo "Starting smoke test..."
	@bin/$(BINARY)-linux-amd64 --port 19999 &
	@sleep 2
	@curl -sf http://localhost:19999/api/health > /dev/null && echo "PASS: /api/health" || echo "FAIL: /api/health"
	@kill %1 2>/dev/null || true

clean:
	rm -rf bin/
	rm -f sentinel

docker:
	docker build -t sentinel:$(VERSION) .

deps:
	$(GO) mod download
	$(GO) mod verify

fmt:
	$(GO) fmt ./...

help:
	@echo "SENTINEL V3 Makefile"
	@echo ""
	@echo "  build            Build linux/amd64 binary"
	@echo "  build-all        Build all platforms"
	@echo "  test             Run tests"
	@echo "  smoke            Quick health-check smoke test"
	@echo "  clean            Remove build artifacts"
	@echo "  docker           Build Docker image"
	@echo "  deps             Download dependencies"
	@echo "  fmt              Format code"
