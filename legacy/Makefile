VERSION ?= 3.0.0
BINARY = sentinel
LDFLAGS = -ldflags "-X main.Version=$(VERSION) -s -w"
PLATFORMS = linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

.PHONY: build build-all build-linux build-linux-arm build-mac build-mac-arm build-windows
.PHONY: test smoke clean docker release checksum

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY)-linux-amd64 ./cmd/sentinel/

build-linux: build

build-linux-arm:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY)-linux-arm64 ./cmd/sentinel/

build-mac:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY)-darwin-amd64 ./cmd/sentinel/

build-mac-arm:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY)-darwin-arm64 ./cmd/sentinel/

build-windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY)-windows-amd64.exe ./cmd/sentinel/

build-all: build build-linux-arm build-mac build-mac-arm build-windows
	@echo "All platforms built successfully"
	@ls -lh bin/

test:
	go test -v -race ./...

vet:
	go vet ./...

smoke: build
	@echo "=== Smoke Test ==="
	@bin/$(BINARY)-linux-amd64 --port 19999 --data-dir /tmp/sentinel-smoke &
	@SMOKE_PID=$$!; sleep 3; \
	 curl -sf http://localhost:19999/api/health > /dev/null && echo "PASS: /api/health" || echo "FAIL: /api/health"; \
	 curl -sf http://localhost:19999/api/providers > /dev/null && echo "PASS: /api/providers" || echo "FAIL: /api/providers"; \
	 curl -sf http://localhost:19999/api/signal-board > /dev/null && echo "PASS: /api/signal-board" || echo "FAIL: /api/signal-board"; \
	 kill $$SMOKE_PID 2>/dev/null; \
	 rm -rf /tmp/sentinel-smoke; \
	 echo "=== Smoke Test Complete ==="

clean:
	rm -rf bin/ dist/

docker:
	docker build -t sentinel:$(VERSION) -t sentinel:latest .

# Package for distribution
dist: build-all
	@mkdir -p dist
	@cd bin && for f in sentinel-*; do \
		if echo "$$f" | grep -q windows; then \
			zip ../dist/$$f.zip $$f; \
		else \
			tar czf ../dist/$$f.tar.gz $$f; \
		fi; \
	done
	@echo "Distribution packages:"
	@ls -lh dist/

checksum: dist
	@cd dist && sha256sum * > SHA256SUMS.txt
	@echo "Checksums generated:"
	@cat dist/SHA256SUMS.txt

release: checksum
	@echo "Release artifacts ready in dist/"
	@echo "To create GitHub release: gh release create v$(VERSION) dist/*"

install-linux: build
	@mkdir -p $(HOME)/.local/bin
	@cp bin/$(BINARY)-linux-amd64 $(HOME)/.local/bin/sentinel
	@chmod +x $(HOME)/.local/bin/sentinel
	@echo "Installed to $(HOME)/.local/bin/sentinel"

version:
	@echo $(VERSION)
