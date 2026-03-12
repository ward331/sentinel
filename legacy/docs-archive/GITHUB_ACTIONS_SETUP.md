# GitHub Actions Setup for SENTINEL

## Overview
This document contains the GitHub Actions workflows for automated CI/CD and releases. Due to token permission limitations, these files need to be manually added to the repository with a token that has `workflow` scope.

## Workflow Files to Create

### 1. Create Directory Structure
```bash
mkdir -p .github/workflows
```

### 2. Create `.github/workflows/ci.yml`
```yaml
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
```

### 3. Create `.github/workflows/release.yml`
```yaml
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
```

## Setup Instructions

### 1. Generate GitHub Token with Proper Scopes
1. Go to GitHub → Settings → Developer settings → Personal access tokens → Tokens (classic)
2. Click "Generate new token (classic)"
3. Name: `SENTINEL-CI-CD`
4. Select scopes:
   - `repo` (Full control of private repositories)
   - `workflow` (Update GitHub Action workflows)
5. Generate token and copy it

### 2. Add Token to Repository Secrets
1. Go to your repository on GitHub
2. Settings → Secrets and variables → Actions
3. Click "New repository secret"
4. Name: `GH_TOKEN`
5. Value: Paste the token from step 1
6. Click "Add secret"

### 3. Add Workflow Files
```bash
# Create directories
mkdir -p .github/workflows

# Create CI workflow
cat > .github/workflows/ci.yml << 'EOF'
[PASTE THE ci.yml CONTENT FROM ABOVE]
EOF

# Create Release workflow  
cat > .github/workflows/release.yml << 'EOF'
[PASTE THE release.yml CONTENT FROM ABOVE]
EOF

# Commit and push
git add .github/
git commit -m "Add GitHub Actions workflows for CI/CD"
git push origin main
```

### 4. Create First Release
```bash
# Tag the release
git tag -a v2.0.0 -m "Release v2.0.0 - Complete SENTINEL system"

# Push the tag (triggers release workflow)
git push origin v2.0.0
```

## Verification
After setup:
1. Go to GitHub → Your repository → Actions
2. You should see workflows running
3. When you push a tag, a release should be created automatically
4. Check Releases page for binaries

## Troubleshooting

### Token Permission Errors
If you see: `refusing to allow a Personal Access Token to create or update workflow without workflow scope`
- Solution: Regenerate token with `workflow` scope

### Workflow Not Triggering
- Check that files are in `.github/workflows/` directory
- Verify YAML syntax is correct
- Check Actions tab for errors

### Build Failures
- Verify Go version compatibility
- Check for compilation errors in filter package
- Ensure all dependencies are in go.mod

## Benefits
- **Automated testing** on every push
- **Cross-platform builds** for Linux, Windows, macOS
- **Automatic releases** when tags are pushed
- **Professional distribution** via GitHub Releases
- **Quality assurance** with linting and testing

## Notes
- The current GitHub token doesn't have `workflow` scope
- These workflows need to be added manually with a proper token
- Once added, the system will have full CI/CD automation