#!/bin/bash
echo "Testing SSE stream connection..."
echo "Press Ctrl+C after 5 seconds to stop"

timeout 5s curl -s -N http://localhost:8080/api/events/stream | \
  while IFS= read -r line; do
    if [[ $line == data:* ]]; then
      echo "Received event: ${line:5}"
      echo "---"
    fi
  done

echo "SSE test complete"