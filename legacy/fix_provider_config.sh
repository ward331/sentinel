#!/bin/bash
# Fix providers that don't have config field

echo "Fixing providers without config field..."

# Check each provider file
for file in internal/provider/*.go; do
    if [[ "$file" == *interface.go ]] || [[ "$file" == *common.go ]]; then
        continue
    fi
    
    # Check if provider has config field
    if grep -q "type.*Provider struct" "$file" && ! grep -q "config.*\*Config" "$file"; then
        provider_name=$(basename "$file" .go)
        echo "Fixing $provider_name (no config field)..."
        
        # Fix Enabled() method
        sed -i 's/if p.config != nil {.*return p.config.Enabled.*}.*return true/return true \/\/ '"$provider_name"' is always enabled/' "$file"
        sed -i 's/if p.config != nil {.*return p.config.Enabled.*}.*return true/return true/' "$file"  # Alternative pattern
        
        # Fix Interval() method - check what field it uses
        if grep -q "interval.*time.Duration" "$file"; then
            sed -i 's/if p.config != nil && p.config.PollInterval > 0 {.*return p.config.PollInterval.*}.*return 5 \* time.Minute/return p.interval/' "$file"
        elif grep -q "pollInterval.*time.Duration" "$file"; then
            sed -i 's/if p.config != nil && p.config.PollInterval > 0 {.*return p.config.PollInterval.*}.*return 5 \* time.Minute/return p.pollInterval/' "$file"
        else
            # Use default interval
            sed -i 's/if p.config != nil && p.config.PollInterval > 0 {.*return p.config.PollInterval.*}.*return 5 \* time.Minute/return 5 \* time.Minute \/\/ Default interval/' "$file"
        fi
    fi
done

echo "Done!"