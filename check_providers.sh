#!/bin/bash
echo "Checking provider interface compliance..."

for file in internal/provider/*.go; do
    if [[ "$file" == *interface.go ]] || [[ "$file" == *common.go ]]; then
        continue
    fi
    
    provider_name=$(basename "$file" .go)
    echo -n "$provider_name: "
    
    # Check for each required method
    has_fetch=$(grep -q "func.*Fetch.*context.Context.*\[\]\*model.Event" "$file" && echo "✓" || echo "✗")
    has_name=$(grep -q "func.*Name() string" "$file" && echo "✓" || echo "✗")
    has_interval=$(grep -q "func.*Interval() time.Duration" "$file" && echo "✓" || echo "✗")
    has_enabled=$(grep -q "func.*Enabled() bool" "$file" && echo "✓" || echo "✗")
    
    echo "Fetch:$has_fetch Name:$has_name Interval:$has_interval Enabled:$has_enabled"
    
    # List missing methods
    if [[ "$has_fetch" == "✗" ]] || [[ "$has_name" == "✗" ]] || [[ "$has_interval" == "✗" ]] || [[ "$has_enabled" == "✗" ]]; then
        echo "  Missing:"
        [[ "$has_fetch" == "✗" ]] && echo "    - Fetch(ctx context.Context) ([]*model.Event, error)"
        [[ "$has_name" == "✗" ]] && echo "    - Name() string"
        [[ "$has_interval" == "✗" ]] && echo "    - Interval() time.Duration"
        [[ "$has_enabled" == "✗" ]] && echo "    - Enabled() bool"
    fi
done