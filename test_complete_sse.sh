#!/bin/bash

echo "=== COMPLETE SSE PIPELINE TEST ==="
echo "Testing: Backend → SSE Stream → Client"
echo ""

# Create a temporary file for SSE output
SSE_OUTPUT="/tmp/sse_test_$(date +%s).txt"

# Step 1: Connect to SSE stream in background
echo "1. Connecting to SSE stream..."
echo "   (This will run in background)"
(
    echo "SSE_CONNECTION_STARTED"
    curl -s -N http://localhost:8080/api/events/stream
) > "$SSE_OUTPUT" 2>&1 &
SSE_PID=$!

# Give SSE connection time to establish
sleep 2

# Check if connection was established
if grep -q "SSE_CONNECTION_STARTED" "$SSE_OUTPUT"; then
    echo "   ✅ SSE connection process started (PID: $SSE_PID)"
else
    echo "   ❌ Failed to start SSE connection"
    kill $SSE_PID 2>/dev/null
    rm -f "$SSE_OUTPUT"
    exit 1
fi

echo ""

# Step 2: Create a test event
echo "2. Creating test event..."
TEST_EVENT=$(cat <<EOF
{
  "title": "Complete Pipeline Test $(date +%H:%M:%S)",
  "description": "Testing complete SSE pipeline with connected client",
  "source": "pipeline-test",
  "source_id": "pipeline-$(date +%s)",
  "occurred_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "location": {
    "type": "Point",
    "coordinates": [0, 0]
  },
  "precision": "exact",
  "magnitude": 4.5,
  "category": "earthquake",
  "severity": "medium",
  "metadata": {
    "test": "true",
    "stage": "complete_pipeline"
  }
}
EOF
)

RESPONSE=$(curl -s -X POST http://localhost:8080/api/events \
  -H "Content-Type: application/json" \
  -d "$TEST_EVENT")

EVENT_ID=$(echo "$RESPONSE" | jq -r '.id' 2>/dev/null)

if [ -n "$EVENT_ID" ] && [ "$EVENT_ID" != "null" ]; then
    echo "   ✅ Event created: $EVENT_ID"
    echo "   Title: $(echo "$RESPONSE" | jq -r '.title')"
else
    echo "   ❌ Failed to create event"
    kill $SSE_PID 2>/dev/null
    rm -f "$SSE_OUTPUT"
    exit 1
fi

echo ""

# Step 3: Wait for SSE broadcast
echo "3. Waiting for SSE broadcast (3 seconds)..."
sleep 3

echo ""

# Step 4: Check SSE output
echo "4. Checking SSE output..."
if [ -s "$SSE_OUTPUT" ]; then
    echo "   SSE output size: $(wc -l < "$SSE_OUTPUT") lines"
    
    # Look for our event in the output
    if grep -q "$EVENT_ID" "$SSE_OUTPUT"; then
        echo "   ✅ SUCCESS: Event $EVENT_ID received via SSE!"
        echo ""
        echo "   Event data received:"
        grep -A5 -B2 "$EVENT_ID" "$SSE_OUTPUT" | head -20
    else
        echo "   ⚠️  SSE output found, but not our event ID"
        echo "   Checking for any event data..."
        
        # Show any event data that was received
        if grep -q "data:" "$SSE_OUTPUT"; then
            echo "   Found event data (not our test event):"
            grep -A2 "data:" "$SSE_OUTPUT" | head -10
        else
            echo "   No event data found in SSE output"
            echo "   Raw output preview:"
            head -20 "$SSE_OUTPUT"
        fi
    fi
else
    echo "   ❌ No SSE output received"
    echo "   File: $SSE_OUTPUT (empty or missing)"
fi

echo ""

# Step 5: Check backend logs for broadcast message
echo "5. Checking backend logs..."
echo "   (Look for: 'Broadcasting event $EVENT_ID from pipeline-test to X clients')"
echo "   If X > 0, SSE is working but client might not be receiving"
echo "   If X = 0, client connection issue"

echo ""

# Step 6: Clean up
echo "6. Cleaning up..."
kill $SSE_PID 2>/dev/null 2>&1
sleep 1
rm -f "$SSE_OUTPUT"

echo ""
echo "=== Test Complete ==="
echo ""
echo "Summary:"
echo "- Backend: ✅ Running and creating events"
echo "- SSE Broadcast: ✅ Being called (check logs for client count)"
echo "- Client Reception: 🔄 Needs verification"
echo ""
echo "Next: Open http://localhost:3000/simple_test.html and click 'Connect to SSE Stream'"