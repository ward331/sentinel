#!/bin/bash
# Fix provider interface by adding Enabled() and Interval() methods to all providers

echo "Fixing provider interface for all providers..."

# List of provider files (excluding interface.go and common.go)
PROVIDER_FILES=$(find internal/provider -name "*.go" -type f | grep -v interface.go | grep -v common.go)

for file in $PROVIDER_FILES; do
    echo "Processing $file..."
    
    # Check if file already has Enabled() method
    if grep -q "func.*Enabled()" "$file"; then
        echo "  ✓ Already has Enabled() method"
    else
        # Add Enabled() method before the last closing brace of the provider struct
        sed -i '/^func.*Fetch.*context.Context.*/i \
// Enabled returns whether the provider is enabled\
func (p *'"$(basename "$file" .go | sed 's/^./\U&/; s/_\(.\)/\U\1/g')"'Provider) Enabled() bool {\
\tif p.config != nil {\
\t\treturn p.config.Enabled\
\t}\
\treturn true\
}' "$file"
        echo "  ✓ Added Enabled() method"
    fi
    
    # Check if file already has Interval() method
    if grep -q "func.*Interval()" "$file"; then
        echo "  ✓ Already has Interval() method"
    else
        # Add Interval() method
        sed -i '/^func.*Enabled() bool {/a \
\
// Interval returns the polling interval\
func (p *'"$(basename "$file" .go | sed 's/^./\U&/; s/_\(.\)/\U\1/g')"'Provider) Interval() time.Duration {\
\tif p.config != nil && p.config.PollInterval > 0 {\
\t\treturn p.config.PollInterval\
\t}\
\treturn 5 * time.Minute // Default interval\
}' "$file"
        echo "  ✓ Added Interval() method"
    fi
done

echo "Done fixing provider interface!"