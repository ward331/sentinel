#!/bin/bash

echo "=== VERIFYING CORS FIX ==="
echo ""

echo "🔍 Testing Backend CORS Headers..."
echo ""

# Test 1: Health endpoint CORS headers
echo "Test 1: Health endpoint (/api/health)"
echo "-----------------------------------"
curl -s -I http://localhost:8080/api/health | grep -i "access-control" | sed 's/^/  /'
echo ""

# Test 2: Events endpoint CORS headers
echo "Test 2: Events endpoint (/api/events)"
echo "------------------------------------"
curl -s -I http://localhost:8080/api/events | grep -i "access-control" | sed 's/^/  /'
echo ""

# Test 3: Test OPTIONS preflight request
echo "Test 3: OPTIONS preflight request"
echo "---------------------------------"
curl -s -X OPTIONS http://localhost:8080/api/events -H "Origin: http://localhost:3000" \
  -H "Access-Control-Request-Method: POST" \
  -H "Access-Control-Request-Headers: Content-Type" \
  -I | grep -i "access-control\|^HTTP" | sed 's/^/  /'
echo ""

# Test 4: Test actual API call from different origin
echo "Test 4: Actual API call from browser simulation"
echo "-----------------------------------------------"
echo "Simulating browser fetch from http://localhost:3000 to http://localhost:8080..."
RESPONSE=$(curl -s -w "\n%{http_code}" -H "Origin: http://localhost:3000" http://localhost:8080/api/health)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
RESPONSE_BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    echo "  ✅ API call successful (HTTP $HTTP_CODE)"
    echo "  Response: $(echo "$RESPONSE_BODY" | jq -r '.status' 2>/dev/null || echo "$RESPONSE_BODY")"
else
    echo "  ❌ API call failed (HTTP $HTTP_CODE)"
    echo "  Response: $RESPONSE_BODY"
fi
echo ""

# Test 5: Check backend is running and has events
echo "Test 5: Backend status and events"
echo "---------------------------------"
HEALTH=$(curl -s http://localhost:8080/api/health)
STATUS=$(echo "$HEALTH" | jq -r '.status' 2>/dev/null)
UPTIME=$(echo "$HEALTH" | jq -r '.uptime' 2>/dev/null)

if [ "$STATUS" = "ok" ]; then
    echo "  ✅ Backend healthy (uptime: $UPTIME)"
    
    # Check event count
    EVENTS_RESPONSE=$(curl -s "http://localhost:8080/api/events?limit=1")
    TOTAL_EVENTS=$(echo "$EVENTS_RESPONSE" | jq -r '.total' 2>/dev/null)
    if [ -n "$TOTAL_EVENTS" ] && [ "$TOTAL_EVENTS" != "null" ]; then
        echo "  📊 Total events in database: $TOTAL_EVENTS"
    else
        echo "  ⚠️  Could not fetch event count"
    fi
else
    echo "  ❌ Backend not healthy"
fi
echo ""

# Test 6: Frontend server status
echo "Test 6: Frontend server status"
echo "------------------------------"
if curl -s -o /dev/null -w "%{http_code}" http://localhost:3000/test_cors.html | grep -q "200"; then
    echo "  ✅ Frontend server running on http://localhost:3000"
    echo "  Test page: http://localhost:3000/test_cors.html"
else
    echo "  ❌ Frontend server not responding"
fi
echo ""

# Test 7: Dashboard status
echo "Test 7: Main dashboard"
echo "---------------------"
if curl -s -o /dev/null -w "%{http_code}" http://localhost:3000/ | grep -q "200"; then
    echo "  ✅ Dashboard served at http://localhost:3000/"
    
    # Check if it's the correct dashboard
    DASHBOARD_TITLE=$(curl -s http://localhost:3000/ | grep -o "<title>[^<]*</title>" | sed 's/<title>//;s/<\/title>//')
    if echo "$DASHBOARD_TITLE" | grep -q "SENTINEL"; then
        echo "  ✅ Correct dashboard: $DASHBOARD_TITLE"
    else
        echo "  ⚠️  Unexpected dashboard title: $DASHBOARD_TITLE"
    fi
else
    echo "  ❌ Dashboard not accessible"
fi
echo ""

echo "=== FIX SUMMARY ==="
echo ""
echo "Changes made to fix CORS issue:"
echo "1. Created CORS middleware in internal/api/cors.go"
echo "2. Added CORS middleware to middleware chain in main.go"
echo "3. CORS middleware sets:"
echo "   - Access-Control-Allow-Origin: *"
echo "   - Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS"
echo "   - Access-Control-Allow-Headers: Content-Type, Authorization"
echo "   - Access-Control-Allow-Credentials: true"
echo "   - Access-Control-Max-Age: 86400"
echo "4. Handles OPTIONS preflight requests"
echo ""
echo "=== TESTING INSTRUCTIONS ==="
echo ""
echo "To verify complete fix:"
echo "1. Open browser to http://localhost:3000/test_cors.html"
echo "2. Click 'Test Connection' for all 4 tests"
echo "3. All tests should show ✅ Success"
echo "4. SSE stream should connect and show messages"
echo ""
echo "To test main dashboard:"
echo "1. Open browser to http://localhost:3000/"
echo "2. Check browser console for CORS errors"
echo "3. Dashboard should connect to SSE stream"
echo "4. Should show 'Connected to real-time stream'"
echo "5. Events should load and display on 3D globe"
echo ""
echo "=== EXPECTED RESULTS ==="
echo ""
echo "✅ No CORS errors in browser console"
echo "✅ Frontend can fetch data from backend"
echo "✅ SSE stream connects successfully"
echo "✅ Real-time events display on 3D globe"
echo "✅ Dashboard fully functional with backend integration"