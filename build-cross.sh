#!/bin/bash
# Cross-compilation script for SENTINEL backend

set -e

BINARY_NAME="sentinel"
VERSION="v2.0.0"
BUILD_DIR="dist"

echo "🔨 Building SENTINEL $VERSION for multiple platforms..."

# Create build directory
mkdir -p $BUILD_DIR

# Function to build for a specific platform
build_for() {
    local os=$1
    local arch=$2
    local output_name=$3
    
    echo "Building for $os/$arch..."
    
    GOOS=$os GOARCH=$arch /usr/local/go/bin/go build \
        -ldflags="-X main.Version=$VERSION" \
        -o "$BUILD_DIR/$output_name" \
        ./cmd/sentinel
    
    # Create checksum
    if [ -f "$BUILD_DIR/$output_name" ]; then
        sha256sum "$BUILD_DIR/$output_name" > "$BUILD_DIR/$output_name.sha256"
        echo "  ✅ Built: $output_name"
        echo "  📊 Size: $(du -h "$BUILD_DIR/$output_name" | cut -f1)"
    else
        echo "  ❌ Failed to build: $output_name"
    fi
}

# Build for different platforms
echo "=== Building Linux ==="
build_for "linux" "amd64" "sentinel-linux-amd64"
build_for "linux" "arm64" "sentinel-linux-arm64"

echo "=== Building Windows ==="
build_for "windows" "amd64" "sentinel-windows-amd64.exe"
build_for "windows" "arm64" "sentinel-windows-arm64.exe"

echo "=== Building macOS ==="
build_for "darwin" "amd64" "sentinel-darwin-amd64"
build_for "darwin" "arm64" "sentinel-darwin-arm64"

echo "=== Creating archive ==="
cd $BUILD_DIR
tar -czf "sentinel-$VERSION-linux-amd64.tar.gz" "sentinel-linux-amd64" "sentinel-linux-amd64.sha256"
tar -czf "sentinel-$VERSION-linux-arm64.tar.gz" "sentinel-linux-arm64" "sentinel-linux-arm64.sha256"
zip -q "sentinel-$VERSION-windows-amd64.zip" "sentinel-windows-amd64.exe" "sentinel-windows-amd64.exe.sha256"
zip -q "sentinel-$VERSION-windows-arm64.zip" "sentinel-windows-arm64.exe" "sentinel-windows-arm64.exe.sha256"
tar -czf "sentinel-$VERSION-darwin-amd64.tar.gz" "sentinel-darwin-amd64" "sentinel-darwin-arm64.sha256"
tar -czf "sentinel-$VERSION-darwin-arm64.tar.gz" "sentinel-darwin-arm64" "sentinel-darwin-arm64.sha256"
cd ..

echo ""
echo "📦 Build complete! Files in $BUILD_DIR/:"
ls -lh $BUILD_DIR/*.gz $BUILD_DIR/*.zip 2>/dev/null || true
echo ""
echo "📋 Summary:"
echo "  Linux (amd64):   $BUILD_DIR/sentinel-$VERSION-linux-amd64.tar.gz"
echo "  Linux (arm64):   $BUILD_DIR/sentinel-$VERSION-linux-arm64.tar.gz"
echo "  Windows (amd64): $BUILD_DIR/sentinel-$VERSION-windows-amd64.zip"
echo "  Windows (arm64): $BUILD_DIR/sentinel-$VERSION-windows-arm64.zip"
echo "  macOS (amd64):   $BUILD_DIR/sentinel-$VERSION-darwin-amd64.tar.gz"
echo "  macOS (arm64):   $BUILD_DIR/sentinel-$VERSION-darwin-arm64.tar.gz"
echo ""
echo "🚀 Ready for distribution!"