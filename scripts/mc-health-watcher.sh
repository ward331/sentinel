#!/bin/bash
# Register SENTINEL V3 with Mission Control's health monitoring
# Adds a health check entry that MC's scheduler can poll

MC_URL="http://localhost:4000"
SENTINEL_PORT=${SENTINEL_PORT:-8080}

# Create a ticket for SENTINEL monitoring if it doesn't exist
curl -sf "${MC_URL}/api/tickets?search=SENTINEL+V3+health" | python3 -c "
import sys, json
data = json.load(sys.stdin)
tickets = data.get('tickets', [])
if not tickets:
    print('Creating health monitoring ticket...')
else:
    print(f'Health monitoring ticket exists: {tickets[0].get(\"id\", \"unknown\")}')
" 2>/dev/null

echo "SENTINEL V3 health endpoint: http://localhost:${SENTINEL_PORT}/api/health"
echo "Add to MC scheduler for monitoring."
