#!/bin/bash

echo "=== Testing SENTINEL Data Infrastructure ==="
echo

# Test 1: Health endpoint
echo "1. Testing health endpoint..."
curl -s http://localhost:8080/api/health | jq -r '.status' 2>/dev/null || curl -s http://localhost:8080/api/health
echo

# Test 2: Provider health
echo "2. Testing provider health reporter..."
curl -s http://localhost:8080/api/providers/health | jq -r 'keys[]' 2>/dev/null || echo "Providers: $(curl -s http://localhost:8080/api/providers/health | grep -o '\"[a-z]*\"' | tr -d '\"' | tr '\n' ' ')"
echo

# Test 3: Healthy providers
echo "3. Testing healthy providers endpoint..."
curl -s http://localhost:8080/api/providers/healthy | jq -r '.[]' 2>/dev/null || curl -s http://localhost:8080/api/providers/healthy
echo

# Test 4: Event log info
echo "4. Testing NDJSON event log..."
curl -s http://localhost:8080/api/event-log/info | jq -r '.size_bytes' 2>/dev/null || curl -s http://localhost:8080/api/event-log/info
echo

# Test 5: Check backup directory
echo "5. Checking backup system..."
if [ -d "/tmp/sentinel-backups" ]; then
    echo "Backup directory exists with $(ls -1 /tmp/sentinel-backups | wc -l) backups"
    ls -lh /tmp/sentinel-backups/ | tail -5
else
    echo "Backup directory not found"
fi
echo

# Test 6: Check NDJSON log file
echo "6. Checking NDJSON log file..."
if [ -f "/tmp/sentinel-events.ndjson" ]; then
    echo "NDJSON log exists: $(ls -lh /tmp/sentinel-events.ndjson)"
    echo "Sample event count: $(wc -l /tmp/sentinel-events.ndjson | cut -d' ' -f1) lines"
else
    echo "NDJSON log not found"
fi
echo

# Test 7: Test OpenSky provider integration
echo "7. Testing OpenSky provider integration..."
curl -s http://localhost:8080/api/providers/health | grep -q opensky && echo "✓ OpenSky provider is integrated and reporting health" || echo "✗ OpenSky provider not found"
echo

echo "=== Data Infrastructure Test Complete ==="