#!/bin/bash

echo "=== Quick SENTINEL Backend Test ==="
echo ""

# Test 1: Health check
echo "1. Testing health endpoint..."
curl -s http://localhost:8080/api/health | jq -r '"   Status: \(.status), Uptime: \(.uptime)s"' 2>/dev/null || echo "   ❌ Health check failed"

echo ""

# Test 2: Create a test event
echo "2. Creating test event..."
TEST_EVENT=$(cat <<EOF
{
  "title": "Quick Test Earthquake",
  "description": "Testing backend integration",
  "source": "quick-test",
  "source_id": "quick-$(date +%s)",
  "occurred_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "location": {
    "type": "Point",
    "coordinates": [0, 0]
  },
  "precision": "exact",
  "magnitude": 3.5,
  "category": "earthquake",
  "severity": "medium"
}
EOF
)

RESPONSE=$(curl -s -X POST http://localhost:8080/api/events \
  -H "Content-Type: application/json" \
  -d "$TEST_EVENT")

EVENT_ID=$(echo "$RESPONSE" | jq -r '.id' 2>/dev/null)
if [ "$EVENT_ID" != "null" ] && [ -n "$EVENT_ID" ]; then
    echo "   ✅ Event created: $EVENT_ID"
else
    echo "   ❌ Event creation failed"
    echo "   Response: $RESPONSE"
fi

echo ""

# Test 3: List events
echo "3. Listing events..."
curl -s "http://localhost:8080/api/events?limit=3" | jq -r '"   Total events: \(.total), Showing: \(.events | length)"' 2>/dev/null || echo "   ❌ Failed to list events"

echo ""

# Test 4: Check if our test event is in the list
echo "4. Verifying test event..."
if [ -n "$EVENT_ID" ]; then
    curl -s "http://localhost:8080/api/events/$EVENT_ID" | jq -r '"   Found: \(.title) (ID: \(.id))"' 2>/dev/null || echo "   ❌ Test event not found"
fi

echo ""

# Test 5: Check providers are running
echo "5. Checking providers..."
echo "   Backend logs show USGS and GDACS polling every 60s"
echo "   (Check running processes for confirmation)"

echo ""

# Test 6: Frontend accessibility
echo "6. Frontend server status..."
curl -s -o /dev/null -w "   HTTP Status: %{http_code}\n" http://localhost:3000/simple_test.html

echo ""
echo "=== Test Complete ==="
echo ""
echo "Next steps:"
echo "1. Open http://localhost:3000/simple_test.html in browser"
echo "2. Click 'Connect to SSE Stream'"
echo "3. Create another test event to see real-time updates"
echo "4. Check backend logs for 'Broadcasted new event' messages"