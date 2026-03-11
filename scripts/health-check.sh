#!/bin/bash
# SENTINEL V3 Health Check — suitable for cron or monitoring
PORT=${SENTINEL_PORT:-8080}
URL="http://localhost:${PORT}/api/health"

response=$(curl -sf -w "\n%{http_code}" "$URL" 2>/dev/null)
http_code=$(echo "$response" | tail -1)
body=$(echo "$response" | head -1)

if [ "$http_code" = "200" ]; then
    echo "OK: SENTINEL healthy"
    exit 0
else
    echo "CRITICAL: SENTINEL unhealthy (HTTP $http_code)"
    # Optional: send alert
    if [ -f ~/.openclaw/workspace-sentinel-backend/telegram-bot/sentinel-alerts.py ]; then
        python3 ~/.openclaw/workspace-sentinel-backend/telegram-bot/sentinel-alerts.py \
            --level critical \
            --title "SENTINEL V3 Down" \
            --body "Health check failed. HTTP code: $http_code"
    fi
    exit 2
fi
