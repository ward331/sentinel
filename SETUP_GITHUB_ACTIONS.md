# Complete GitHub Actions Setup Guide

## The Problem
Your current GitHub token only has `repo` scope, but needs `workflow` scope to create/update workflow files.

## Solution Options

### Option 1: Generate New Token (Easiest)
1. **Go to**: https://github.com/settings/tokens
2. **Click**: "Generate new token (classic)"
3. **Name**: `SENTINEL-CI-CD`
4. **Select scopes**:
   - ✅ `repo` (Full control of private repositories)
   - ✅ `workflow` (Update GitHub Action workflows)
5. **Generate token** and copy it
6. **Update git remote**:
   ```bash
   git remote set-url origin https://ward331:<NEW_TOKEN>@github.com/ward331/sentinel.git
   ```
7. **Create workflow files**:
   ```bash
   mkdir -p .github/workflows
   # Create ci.yml (content below)
   # Create release.yml (content below)
   git add .github/
   git commit -m "Add GitHub Actions workflows"
   git push origin main
   ```

### Option 2: Add via GitHub UI
1. **Go to**: https://github.com/ward331/sentinel
2. **Click**: "Add file" → "Create new file"
3. **Path**: `.github/workflows/ci.yml`
4. **Content**: [Paste ci.yml content below]
5. **Commit**: Directly to main branch
6. **Repeat** for `.github/workflows/release.yml`

### Option 3: Use SSH (if SSH key is added to GitHub)
1. **Add SSH key to GitHub**:
   - Go to: https://github.com/settings/keys
   - Click "New SSH key"
   - Title: `gunther`
   - Key: `ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIJ73nmMYpGTYN8A539gels+rMYfx06wXdl4aRGPcsNap ed@gunther`
2. **Update remote**:
   ```bash
   git remote set-url origin git@github.com:ward331/sentinel.git
   ```
3. **Push workflow files**

## Workflow Files

### File 1: `.github/workflows/ci.yml`
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

### File 2: `.github/workflows/release.yml`
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

## After Setup

### Test the Workflow
1. **Push a test commit** to trigger CI
2. **Check Actions tab**: Should show running workflows
3. **Verify**: Tests pass, linting passes, cross-compilation works

### Create First Release
```bash
# Tag the release
git tag -a v2.0.0 -m "SENTINEL v2.0.0 - Complete real-time OSINT system"

# Push tag (triggers release workflow)
git push origin v2.0.0
```

### Verify Release
1. **Go to Releases**: https://github.com/ward331/sentinel/releases
2. **Check**: v2.0.0 release with 5 binaries
3. **Download**: Test a binary on your system

## Troubleshooting

### "workflow scope required"
- **Cause**: Token doesn't have `workflow` scope
- **Fix**: Generate new token with correct scopes (Option 1)

### SSH Permission Denied
- **Cause**: SSH key not added to GitHub account
- **Fix**: Add SSH key at https://github.com/settings/keys

### Workflow Not Triggering
- **Check**: Files are in `.github/workflows/` directory
- **Check**: YAML syntax is correct
- **Check**: No errors in Actions tab

## Quick Fix Script
Run `./add-workflows-via-ui.sh` for interactive setup help.

## Status
- ✅ Workflow files created locally
- ✅ Documentation complete
- ✅ Ready for token/SSH setup
- ✅ Release pipeline designed
- ⚠️ Need token with `workflow` scope or SSH setup

**Once workflow files are added, SENTINEL will have full CI/CD automation!** 🚀