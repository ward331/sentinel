#!/bin/bash

echo "=== SENTINEL SSE Stream Test with curl ==="
echo ""

# Start SSE connection in background
echo "1. Starting SSE connection in background..."
curl -s -N http://localhost:8080/api/events/stream > /tmp/sse_output.txt 2>&1 &
SSE_PID=$!

echo "   SSE PID: $SSE_PID"
sleep 2

# Create test event
echo ""
echo "2. Creating test event..."
TEST_EVENT=$(cat <<EOF
{
  "title": "Curl SSE Test $(date +%H:%M:%S)",
  "description": "Testing SSE with curl background listener",
  "source": "curl-test",
  "source_id": "curl-$(date +%s)",
  "occurred_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "location": {
    "type": "Point",
    "coordinates": [0, 0]
  },
  "precision": "exact",
  "magnitude": 3.8,
  "category": "earthquake",
  "severity": "medium"
}
EOF
)

RESPONSE=$(curl -s -X POST http://localhost:8080/api/events \
  -H "Content-Type: application/json" \
  -d "$TEST_EVENT")

EVENT_ID=$(echo "$RESPONSE" | jq -r '.id' 2>/dev/null)

if [ -n "$EVENT_ID" ] && [ "$EVENT_ID" != "null" ]; then
    echo "   ✅ Event created: $EVENT_ID"
else
    echo "   ❌ Failed to create event"
    kill $SSE_PID 2>/dev/null
    exit 1
fi

# Wait a bit for SSE broadcast
echo ""
echo "3. Waiting for SSE broadcast (5 seconds)..."
sleep 5

# Check SSE output
echo ""
echo "4. Checking SSE output..."
if [ -s /tmp/sse_output.txt ]; then
    echo "   SSE output found:"
    grep -A2 -B2 "data:" /tmp/sse_output.txt | head -20
    
    # Check if our event ID is in the output
    if grep -q "$EVENT_ID" /tmp/sse_output.txt; then
        echo ""
        echo "   ✅ SUCCESS: Event $EVENT_ID was broadcast via SSE!"
    else
        echo ""
        echo "   ⚠️  SSE output found, but not our event ID"
        echo "   (Might be other events or formatting issue)"
    fi
else
    echo "   ❌ No SSE output received"
    echo "   Possible issues:"
    echo "   - SSE stream not broadcasting"
    echo "   - Backend not calling Broadcast()"
    echo "   - Check backend logs"
fi

# Clean up
echo ""
echo "5. Cleaning up..."
kill $SSE_PID 2>/dev/null
rm -f /tmp/sse_output.txt

echo ""
echo "=== Test Complete ==="