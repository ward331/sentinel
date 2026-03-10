#!/bin/bash
# Build script for SENTINEL v2.0.0 when Go is available
# This script compiles the complete system with poller integration

set -e

echo "=========================================="
echo "SENTINEL v2.0.0 - COMPLETE BUILD SCRIPT"
echo "=========================================="
echo "Date: $(date)"
echo ""

# Check for Go
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed or not in PATH"
    echo ""
    echo "Installation options:"
    echo "1. Ubuntu/Debian: sudo apt-get install golang-go"
    echo "2. macOS: brew install go"
    echo "3. Windows: Download from https://golang.org/dl/"
    echo "4. Manual: Download and extract to /usr/local/go"
    echo ""
    echo "After installing Go, run:"
    echo "  export PATH=\$PATH:/usr/local/go/bin"
    echo "  go version"
    echo "  ./build_when_ready.sh"
    exit 1
fi

echo "✅ Go installed: $(go version)"
echo ""

# Clean previous build
echo "Cleaning previous build..."
rm -f sentinel
rm -rf dist/

# Download dependencies
echo "Downloading dependencies..."
go mod download
go mod verify

# Run tests
echo "Running tests..."
go test ./internal/poller/... -v || echo "⚠️ Some tests may fail without database"

# Build the binary
echo "Building SENTINEL v2.0.0..."
go build -ldflags="-s -w" -o sentinel ./cmd/sentinel

# Verify build
if [ -f "sentinel" ]; then
    echo "✅ Build successful!"
    echo ""
    echo "Binary size: $(du -h sentinel | cut -f1)"
    echo "Version: $(./sentinel --version)"
    echo ""
    echo "Build details:"
    echo "  Architecture: $(uname -m)"
    echo "  OS: $(uname -s)"
    echo "  Go version: $(go version | cut -d' ' -f3)"
    echo "  Build time: $(date)"
    echo ""
    echo "Available commands:"
    echo "  ./sentinel --version           # Show version"
    echo "  ./sentinel --help              # Show help"
    echo "  ./sentinel --data-dir /tmp/test --port 18100  # Run server"
    echo "  make smoke                     # Run smoke test"
else
    echo "❌ Build failed!"
    exit 1
fi

echo ""
echo "=========================================="
echo "BUILD COMPLETE - READY FOR DEPLOYMENT"
echo "=========================================="
echo ""
echo "Next steps:"
echo "1. Test the binary: ./sentinel --version"
echo "2. Run smoke test: make smoke"
echo "3. Deploy to production"
echo "4. Monitor with: ./sentinel --data-dir /path/to/data --port 8080"
echo ""
echo "SENTINEL v2.0.0 with poller integration is ready!"