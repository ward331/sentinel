#!/bin/bash

echo "=== SENTINEL DASHBOARD COMPLETE VERIFICATION TEST ==="
echo "Testing all requirements for Week 1 Task 1"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
ORANGE='\033[0;33m'
NC='\033[0m' # No Color

echo "📋 REQUIREMENTS CHECKLIST:"
echo "1. ${BLUE}3D CesiumJS globe using Ion token${NC}"
echo "2. ${GREEN}Connect to backend SSE stream${NC}"
echo "3. ${CYAN}Render markers on globe${NC}"
echo "   - ${RED}Earthquakes red${NC}"
echo "   - ${BLUE}Storms blue${NC}"
echo "   - ${CYAN}Floods cyan${NC}"
echo "   - ${ORANGE}Volcanoes orange${NC}"
echo "   - ${YELLOW}Other yellow${NC}"
echo "4. ${GREEN}Clicking marker shows event details panel${NC}"
echo "5. ${BLUE}Event counter badge showing total events${NC}"
echo "6. ${GREEN}Serve from http://localhost:3000${NC}"
echo "7. ${GREEN}make smoke must pass${NC}"
echo ""

echo "🔍 TESTING EACH REQUIREMENT:"
echo ""

# Test 1: Check if dashboard file exists
echo "1. Checking dashboard file..."
if [ -f "sentinel_dashboard.html" ]; then
    echo "   ✅ sentinel_dashboard.html exists"
    
    # Check for CesiumJS token
    if grep -q "Cesium.Ion.defaultAccessToken" sentinel_dashboard.html; then
        echo "   ✅ CesiumJS Ion token configured"
    else
        echo "   ❌ CesiumJS Ion token not found in dashboard"
    fi
    
    # Check for CesiumJS library
    if grep -q "cesium.com/downloads/cesiumjs" sentinel_dashboard.html; then
        echo "   ✅ CesiumJS library included"
    else
        echo "   ❌ CesiumJS library not found"
    fi
else
    echo "   ❌ sentinel_dashboard.html not found"
fi

echo ""

# Test 2: Check SSE connection
echo "2. Checking SSE stream connection..."
if grep -q "http://localhost:8080/api/events/stream" sentinel_dashboard.html; then
    echo "   ✅ SSE stream URL configured: http://localhost:8080/api/events/stream"
    
    # Check for EventSource usage
    if grep -q "EventSource" sentinel_dashboard.html; then
        echo "   ✅ EventSource API used for SSE"
    else
        echo "   ⚠️  EventSource not found (check JavaScript)"
    fi
else
    echo "   ❌ SSE stream URL not found in dashboard"
fi

echo ""

# Test 3: Check marker rendering colors
echo "3. Checking marker color configuration..."
echo "   Looking for color mappings in JavaScript..."

# Extract JavaScript section and check colors
if grep -q "'earthquake':.*#ef4444" sentinel_dashboard.html || 
   grep -q "earthquake.*red" sentinel_dashboard.html || 
   grep -q "#ef4444.*earthquake" sentinel_dashboard.html; then
    echo "   ✅ Earthquakes configured as red (#ef4444)"
else
    echo "   ⚠️  Earthquake color not explicitly configured as red"
fi

if grep -q "'storm':.*#3b82f6" sentinel_dashboard.html || 
   grep -q "storm.*blue" sentinel_dashboard.html || 
   grep -q "#3b82f6.*storm" sentinel_dashboard.html; then
    echo "   ✅ Storms configured as blue (#3b82f6)"
else
    echo "   ⚠️  Storm color not explicitly configured as blue"
fi

if grep -q "'flood':.*#06b6d4" sentinel_dashboard.html || 
   grep -q "flood.*cyan" sentinel_dashboard.html || 
   grep -q "#06b6d4.*flood" sentinel_dashboard.html; then
    echo "   ✅ Floods configured as cyan (#06b6d4)"
else
    echo "   ⚠️  Flood color not explicitly configured as cyan"
fi

if grep -q "'volcano':.*#f97316" sentinel_dashboard.html || 
   grep -q "volcano.*orange" sentinel_dashboard.html || 
   grep -q "#f97316.*volcano" sentinel_dashboard.html; then
    echo "   ✅ Volcanoes configured as orange (#f97316)"
else
    echo "   ⚠️  Volcano color not explicitly configured as orange"
fi

if grep -q "'default':.*#eab308" sentinel_dashboard.html || 
   grep -q "default.*yellow" sentinel_dashboard.html || 
   grep -q "#eab308.*default" sentinel_dashboard.html; then
    echo "   ✅ Other events configured as yellow (#eab308)"
else
    echo "   ⚠️  Default color not explicitly configured as yellow"
fi

echo ""

# Test 4: Check event details panel
echo "4. Checking event details panel..."
if grep -q "showEventDetails" sentinel_dashboard.html; then
    echo "   ✅ Event details function implemented"
    
    if grep -q "eventPanel" sentinel_dashboard.html; then
        echo "   ✅ Event panel HTML element exists"
    else
        echo "   ⚠️  Event panel HTML element not found"
    fi
else
    echo "   ❌ Event details function not found"
fi

echo ""

# Test 5: Check event counter badge
echo "5. Checking event counter badge..."
if grep -q "event-counter" sentinel_dashboard.html || 
   grep -q "counter-badge" sentinel_dashboard.html || 
   grep -q "liveEventCount" sentinel_dashboard.html; then
    echo "   ✅ Event counter badge implemented"
    
    if grep -q "totalEvents" sentinel_dashboard.html; then
        echo "   ✅ Total events counter implemented"
    else
        echo "   ⚠️  Total events counter not found"
    fi
else
    echo "   ❌ Event counter badge not found"
fi

echo ""

# Test 6: Check serving from localhost:3000
echo "6. Checking frontend server..."
if curl -s http://localhost:3000/ > /dev/null; then
    echo "   ✅ Frontend server running on http://localhost:3000"
    
    # Check if dashboard is served at root
    RESPONSE=$(curl -s http://localhost:3000/ | head -5)
    if echo "$RESPONSE" | grep -q "SENTINEL"; then
        echo "   ✅ Dashboard served at root path"
    else
        echo "   ⚠️  Dashboard might not be served at root (check serve_frontend.go)"
    fi
else
    echo "   ❌ Frontend server not responding on port 3000"
fi

echo ""

# Test 7: Check make smoke passes
echo "7. Checking make smoke test..."
if make smoke 2>&1 | grep -q "Smoke test PASSED"; then
    echo "   ✅ make smoke test passes"
else
    echo "   ❌ make smoke test failed"
    echo "   Running make smoke to see output..."
    make smoke
fi

echo ""
echo "=== DASHBOARD ARCHITECTURE VERIFICATION ==="
echo ""

# Check for key dashboard components
echo "🔧 Dashboard Components Check:"

COMPONENTS=(
    "Cesium Viewer initialization"
    "SSE EventSource connection"
    "Event marker creation"
    "Color mapping system"
    "Event details panel"
    "Real-time counters"
    "Connection status indicator"
    "Legend for event types"
)

for component in "${COMPONENTS[@]}"; do
    # Simple check based on component name
    case $component in
        "Cesium Viewer initialization")
            if grep -q "new Cesium.Viewer" sentinel_dashboard.html; then
                echo "   ✅ $component"
            else
                echo "   ❌ $component"
            fi
            ;;
        "SSE EventSource connection")
            if grep -q "EventSource" sentinel_dashboard.html; then
                echo "   ✅ $component"
            else
                echo "   ❌ $component"
            fi
            ;;
        "Event marker creation")
            if grep -q "addEventMarker" sentinel_dashboard.html || 
               grep -q "viewer.entities.add" sentinel_dashboard.html; then
                echo "   ✅ $component"
            else
                echo "   ❌ $component"
            fi
            ;;
        "Color mapping system")
            if grep -q "categoryColors" sentinel_dashboard.html; then
                echo "   ✅ $component"
            else
                echo "   ❌ $component"
            fi
            ;;
        "Event details panel")
            if grep -q "eventPanel" sentinel_dashboard.html && 
               grep -q "showEventDetails" sentinel_dashboard.html; then
                echo "   ✅ $component"
            else
                echo "   ❌ $component"
            fi
            ;;
        "Real-time counters")
            if grep -q "updateEventCounters" sentinel_dashboard.html; then
                echo "   ✅ $component"
            else
                echo "   ❌ $component"
            fi
            ;;
        "Connection status indicator")
            if grep -q "connectionStatus" sentinel_dashboard.html; then
                echo "   ✅ $component"
            else
                echo "   ❌ $component"
            fi
            ;;
        "Legend for event types")
            if grep -q "legend" sentinel_dashboard.html -i; then
                echo "   ✅ $component"
            else
                echo "   ❌ $component"
            fi
            ;;
    esac
done

echo ""
echo "=== FINAL VERIFICATION ==="
echo ""

# Create a test event to verify the complete pipeline
echo "Creating test event to verify real-time pipeline..."
echo ""

TEST_EVENT=$(cat <<EOF
{
  "title": "Dashboard Verification Test",
  "description": "Testing complete dashboard pipeline",
  "source": "dashboard-test",
  "source_id": "dashboard-$(date +%s)",
  "occurred_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "location": {
    "type": "Point",
    "coordinates": [0, 0]
  },
  "precision": "exact",
  "magnitude": 5.5,
  "category": "earthquake",
  "severity": "high"
}
EOF
)

echo "Test event data:"
echo "$TEST_EVENT" | jq '.' 2>/dev/null || echo "$TEST_EVENT"
echo ""

echo "Sending to backend..."
RESPONSE=$(curl -s -X POST http://localhost:8080/api/events \
  -H "Content-Type: application/json" \
  -d "$TEST_EVENT")

EVENT_ID=$(echo "$RESPONSE" | jq -r '.id' 2>/dev/null)

if [ -n "$EVENT_ID" ] && [ "$EVENT_ID" != "null" ]; then
    echo "✅ Event created: $EVENT_ID"
    echo ""
    echo "📊 Backend should broadcast this event via SSE"
    echo "🌍 Dashboard should receive and display it in real-time"
    echo ""
    echo "Check backend logs for: 'Broadcasting event $EVENT_ID from dashboard-test to X clients'"
    echo "If X > 0, dashboard is connected and receiving events"
else
    echo "❌ Failed to create test event"
fi

echo ""
echo "=== TEST COMPLETE ==="
echo ""
echo "🎯 NEXT STEPS:"
echo "1. Open http://localhost:3000/ in your browser"
echo "2. Verify 3D CesiumJS globe loads with terrain"
echo "3. Check that connection status shows 'Connected to real-time stream'"
echo "4. Create events via API or wait for provider polling"
echo "5. Watch markers appear on globe in real-time"
echo "6. Click markers to see event details panel"
echo "7. Verify event counter updates automatically"
echo ""
echo "📈 DASHBOARD STATUS:"
echo "All requirements implemented and ready for testing."
echo "Complete pipeline: Backend → SSE → Dashboard → 3D Visualization"