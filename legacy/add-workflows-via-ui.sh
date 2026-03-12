#!/bin/bash
# Script to help add GitHub Actions workflows via GitHub UI
# Since the token lacks 'workflow' scope, we can't push them directly

echo "=== GitHub Actions Workflow Setup via UI ==="
echo ""
echo "Since your GitHub token lacks 'workflow' scope, you need to:"
echo "1. Generate a new token with 'repo' and 'workflow' scopes"
echo "2. OR add these files via GitHub UI"
echo ""
echo "=== OPTION 1: Generate New Token ==="
echo "1. Go to: https://github.com/settings/tokens"
echo "2. Click 'Generate new token (classic)'"
echo "3. Name: 'SENTINEL-CI-CD'"
echo "4. Select scopes: 'repo' and 'workflow'"
echo "5. Generate and copy token"
echo "6. Update git remote:"
echo "   git remote set-url origin https://ward331:<NEW_TOKEN>@github.com/ward331/sentinel.git"
echo "7. Push workflow files"
echo ""
echo "=== OPTION 2: Add via GitHub UI ==="
echo ""
echo "File 1: .github/workflows/ci.yml"
echo "---"
cat << 'EOF'
name: CI

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest
    
    steps:
    - name: Checkout code
      uses: actions/checkout@v4

    - name: Set up Go
      uses: actions/setup-go@v5
      with:
        go-version: '1.21'
        cache: true

    - name: Verify Go version
      run: go version

    - name: Get dependencies
      run: go mod download

    - name: Run tests
      run: go test ./... -v

    - name: Build binary
      run: go build -o sentinel ./cmd/sentinel

    - name: Run smoke test
      run: |
        chmod +x sentinel
        ./sentinel --version
        ./sentinel --help
        echo "✅ Build and basic tests passed"

  lint:
    runs-on: ubuntu-latest
    
    steps:
    - name: Checkout code
      uses: actions/checkout@v4

    - name: Set up Go
      uses: actions/setup-go@v5
      with:
        go-version: '1.21'

    - name: Run gofmt
      run: |
        if [ -n "$(gofmt -l .)" ]; then
          echo "❌ Code is not formatted properly"
          gofmt -l .
          exit 1
        fi
        echo "✅ Code formatting is correct"

    - name: Run go vet
      run: go vet ./...

  cross-build:
    runs-on: ubuntu-latest
    needs: [test, lint]
    
    steps:
    - name: Checkout code
      uses: actions/checkout@v4

    - name: Set up Go
      uses: actions/setup-go@v5
      with:
        go-version: '1.21'

    - name: Cross-compile for all platforms
      run: |
        PLATFORMS="linux/amd64 linux/arm64 windows/amd64 darwin/amd64 darwin/arm64"
        
        for PLATFORM in $PLATFORMS; do
          GOOS=${PLATFORM%/*}
          GOARCH=${PLATFORM#*/}
          OUTPUT="sentinel-$GOOS-$GOARCH"
          if [ "$GOOS" = "windows" ]; then
            OUTPUT="$OUTPUT.exe"
          fi
          
          echo "Building for $GOOS/$GOARCH..."
          GOOS=$GOOS GOARCH=$GOARCH go build -o $OUTPUT ./cmd/sentinel
          
          if [ -f "$OUTPUT" ]; then
            echo "✅ $GOOS/$GOARCH: $(du -h $OUTPUT | cut -f1)"
            rm $OUTPUT
          else
            echo "❌ Failed to build for $GOOS/$GOARCH"
            exit 1
          fi
        done
        
        echo "✅ All platforms compile successfully"
EOF

echo ""
echo "---"
echo ""
echo "File 2: .github/workflows/release.yml"
echo "---"
cat << 'EOF'
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  build-and-release:
    runs-on: ubuntu-latest
    permissions:
      contents: write
    
    steps:
    - name: Checkout code
      uses: actions/checkout@v4
      with:
        fetch-depth: 0

    - name: Set up Go
      uses: actions/setup-go@v5
      with:
        go-version: '1.21'
        cache: true

    - name: Build for Linux (amd64)
      run: |
        GOOS=linux GOARCH=amd64 go build \
          -ldflags="-X main.Version=${{ github.ref_name }}" \
          -o sentinel-linux-amd64 \
          ./cmd/sentinel
        sha256sum sentinel-linux-amd64 > sentinel-linux-amd64.sha256

    - name: Build for Linux (arm64)
      run: |
        GOOS=linux GOARCH=arm64 go build \
          -ldflags="-X main.Version=${{ github.ref_name }}" \
          -o sentinel-linux-arm64 \
          ./cmd/sentinel
        sha256sum sentinel-linux-arm64 > sentinel-linux-arm64.sha256

    - name: Build for Windows (amd64)
      run: |
        GOOS=windows GOARCH=amd64 go build \
          -ldflags="-X main.Version=${{ github.ref_name }}" \
          -o sentinel-windows-amd64.exe \
          ./cmd/sentinel
        sha256sum sentinel-windows-amd64.exe > sentinel-windows-amd64.exe.sha256

    - name: Build for macOS (amd64)
      run: |
        GOOS=darwin GOARCH=amd64 go build \
          -ldflags="-X main.Version=${{ github.ref_name }}" \
          -o sentinel-darwin-amd64 \
          ./cmd/sentinel
        sha256sum sentinel-darwin-amd64 > sentinel-darwin-amd64.sha256

    - name: Build for macOS (arm64)
      run: |
        GOOS=darwin GOARCH=arm64 go build \
          -ldflags="-X main.Version=${{ github.ref_name }}" \
          -o sentinel-darwin-arm64 \
          ./cmd/sentinel
        sha256sum sentinel-darwin-arm64 > sentinel-darwin-arm64.sha256

    - name: Create archives
      run: |
        tar -czf sentinel-${{ github.ref_name }}-linux-amd64.tar.gz sentinel-linux-amd64 sentinel-linux-amd64.sha256
        tar -czf sentinel-${{ github.ref_name }}-linux-arm64.tar.gz sentinel-linux-arm64 sentinel-linux-arm64.sha256
        zip -q sentinel-${{ github.ref_name }}-windows-amd64.zip sentinel-windows-amd64.exe sentinel-windows-amd64.exe.sha256
        tar -czf sentinel-${{ github.ref_name }}-darwin-amd64.tar.gz sentinel-darwin-amd64 sentinel-darwin-amd64.sha256
        tar -czf sentinel-${{ github.ref_name }}-darwin-arm64.tar.gz sentinel-darwin-arm64 sentinel-darwin-arm64.sha256

    - name: Create Release
      uses: softprops/action-gh-release@v1
      with:
        name: Release ${{ github.ref_name }}
        body: |
          # SENTINEL ${{ github.ref_name }}
          
          ## 🚀 Real-time OSINT Data Collection System
          
          ### Downloads
          **Linux:** `sentinel-${{ github.ref_name }}-linux-amd64.tar.gz` (x86_64)
          **Linux:** `sentinel-${{ github.ref_name }}-linux-arm64.tar.gz` (ARM64)
          **Windows:** `sentinel-${{ github.ref_name }}-windows-amd64.zip` (x86_64)
          **macOS:** `sentinel-${{ github.ref_name }}-darwin-amd64.tar.gz` (Intel)
          **macOS:** `sentinel-${{ github.ref_name }}-darwin-arm64.tar.gz` (Apple Silicon)
          
          ### Installation
          ```bash
          # Linux/macOS
          tar -xzf sentinel-*.tar.gz
          chmod +x sentinel-*
          ./sentinel --help
          ```
          
          ### Verification
          ```bash
          sha256sum -c *.sha256
          ```
          
          ### Quick Start
          ```bash
          ./sentinel --data-dir=/tmp/sentinel-data --port=8080
          ```
        draft: false
        prerelease: false
        files: |
          sentinel-${{ github.ref_name }}-linux-amd64.tar.gz
          sentinel-${{ github.ref_name }}-linux-arm64.tar.gz
          sentinel-${{ github.ref_name }}-windows-amd64.zip
          sentinel-${{ github.ref_name }}-darwin-amd64.tar.gz
          sentinel-${{ github.ref_name }}-darwin-arm64.tar.gz
EOF

echo ""
echo "=== How to Add via GitHub UI ==="
echo "1. Go to: https://github.com/ward331/sentinel"
echo "2. Click 'Add file' → 'Create new file'"
echo "3. Enter: '.github/workflows/ci.yml'"
echo "4. Paste the first file content above"
echo "5. Commit directly to main branch"
echo "6. Repeat for '.github/workflows/release.yml'"
echo ""
echo "=== After Adding Files ==="
echo "1. Create release tag:"
echo "   git tag -a v2.0.0 -m 'Release v2.0.0'"
echo "   git push origin v2.0.0"
echo "2. Check Actions tab: Workflows should run automatically"
echo "3. Check Releases tab: Binaries should be published"