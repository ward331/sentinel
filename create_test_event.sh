#!/bin/bash

echo "Creating test event for SSE verification..."
echo ""

# Create a unique test event
TEST_EVENT=$(cat <<EOF
{
  "title": "SSE Broadcast Test $(date +%H:%M:%S)",
  "description": "Testing real-time SSE broadcast to frontend",
  "source": "sse-test",
  "source_id": "sse-$(date +%s)",
  "occurred_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "location": {
    "type": "Point",
    "coordinates": [$(echo "scale=2; $RANDOM/32767*360-180" | bc), $(echo "scale=2; $RANDOM/32767*180-90" | bc)]
  },
  "precision": "exact",
  "magnitude": $(echo "scale=1; $RANDOM/32767*5+2" | bc),
  "category": "earthquake",
  "severity": "medium",
  "metadata": {
    "test": "true",
    "purpose": "SSE broadcast verification"
  }
}
EOF
)

echo "Event data:"
echo "$TEST_EVENT" | jq '.' 2>/dev/null || echo "$TEST_EVENT"
echo ""

echo "Sending to backend..."
RESPONSE=$(curl -s -X POST http://localhost:8080/api/events \
  -H "Content-Type: application/json" \
  -d "$TEST_EVENT")

echo "Response:"
echo "$RESPONSE" | jq '.' 2>/dev/null || echo "$RESPONSE"
echo ""

echo "Check backend logs for: 'Broadcasted new event from sse-test to SSE clients'"
echo ""
echo "If SSE frontend is connected, it should receive this event in real-time!"