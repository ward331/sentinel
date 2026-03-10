#!/bin/bash
echo "Fixing missing provider methods..."

for file in internal/provider/*.go; do
    if [[ "$file" == *interface.go ]] || [[ "$file" == *common.go ]]; then
        continue
    fi
    
    provider_name=$(basename "$file" .go)
    struct_name=$(echo "$provider_name" | sed 's/^./\U&/; s/_\(.\)/\U\1/g')
    
    echo "Processing $provider_name ($struct_name)..."
    
    # Check if has config field
    has_config=$(grep -q "config.*\*Config" "$file" && echo "yes" || echo "no")
    
    # Add Name() method if missing
    if ! grep -q "func.*Name() string" "$file"; then
        echo "  Adding Name() method..."
        # Find line with Fetch method
        fetch_line=$(grep -n "func.*Fetch" "$file" | head -1 | cut -d: -f1)
        if [ -n "$fetch_line" ]; then
            sed -i "${fetch_line}i \\
// Name returns the provider identifier\\
func (p *${struct_name}Provider) Name() string {\\
\treturn \"$provider_name\"\\
}\\
" "$file"
        fi
    fi
    
    # Add Enabled() method if missing and has config
    if ! grep -q "func.*Enabled() bool" "$file" && [ "$has_config" = "yes" ]; then
        echo "  Adding Enabled() method..."
        fetch_line=$(grep -n "func.*Fetch" "$file" | head -1 | cut -d: -f1)
        if [ -n "$fetch_line" ]; then
            sed -i "${fetch_line}i \\
// Enabled returns whether the provider is enabled\\
func (p *${struct_name}Provider) Enabled() bool {\\
\tif p.config != nil {\\
\t\treturn p.config.Enabled\\
\t}\\
\treturn true\\
}\\
" "$file"
        fi
    fi
    
    # Add Interval() method if missing and has config
    if ! grep -q "func.*Interval() time.Duration" "$file" && [ "$has_config" = "yes" ]; then
        echo "  Adding Interval() method..."
        fetch_line=$(grep -n "func.*Fetch" "$file" | head -1 | cut -d: -f1)
        if [ -n "$fetch_line" ]; then
            sed -i "${fetch_line}i \\
// Interval returns the polling interval\\
func (p *${struct_name}Provider) Interval() time.Duration {\\
\tif p.config != nil \&\& p.config.PollInterval > 0 {\\
\t\treturn p.config.PollInterval\\
\t}\\
\treturn 5 * time.Minute // Default interval\\
}\\
" "$file"
        fi
    fi
    
    # For providers without config, add simple methods
    if [ "$has_config" = "no" ]; then
        # Add simple Enabled() if missing
        if ! grep -q "func.*Enabled() bool" "$file"; then
            echo "  Adding simple Enabled() method..."
            fetch_line=$(grep -n "func.*Fetch" "$file" | head -1 | cut -d: -f1)
            if [ -n "$fetch_line" ]; then
                sed -i "${fetch_line}i \\
// Enabled returns whether the provider is enabled\\
func (p *${struct_name}Provider) Enabled() bool {\\
\treturn true\\
}\\
" "$file"
            fi
        fi
        
        # Add simple Interval() if missing
        if ! grep -q "func.*Interval() time.Duration" "$file"; then
            echo "  Adding simple Interval() method..."
            fetch_line=$(grep -n "func.*Fetch" "$file" | head -1 | cut -d: -f1)
            if [ -n "$fetch_line" ]; then
                sed -i "${fetch_line}i \\
// Interval returns the polling interval\\
func (p *${struct_name}Provider) Interval() time.Duration {\\
\treturn 5 * time.Minute // Default interval\\
}\\
" "$file"
            fi
        fi
    fi
done

echo "Done!"