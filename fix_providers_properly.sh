#!/bin/bash
# Properly fix provider interface by adding Enabled() and Interval() methods

echo "Fixing provider interface properly..."

# Providers that need both methods
PROVIDERS="airplanes_live celestrak gdelt globalfishingwatch globalforestwatch iranconflict liveuamap noaa_cap noaa_nws opensanctions piracy_imb promed swpc"

for provider in $PROVIDERS; do
    file="internal/provider/${provider}.go"
    echo "Processing $file..."
    
    # Get the provider struct name (capitalized, remove underscores)
    struct_name=$(echo "$provider" | sed 's/^./\U&/; s/_\(.\)/\U\1/g')
    
    # Check if file exists and has the provider struct
    if grep -q "type ${struct_name}Provider struct" "$file"; then
        # Find the line with the Fetch method
        fetch_line=$(grep -n "func (p \*${struct_name}Provider) Fetch" "$file" | head -1 | cut -d: -f1)
        
        if [ -n "$fetch_line" ]; then
            # Insert Enabled() method before Fetch
            sed -i "${fetch_line}i \\
// Enabled returns whether the provider is enabled\\
func (p *${struct_name}Provider) Enabled() bool {\\
\tif p.config != nil {\\
\t\treturn p.config.Enabled\\
\t}\\
\treturn true\\
}\\
\\
// Interval returns the polling interval\\
func (p *${struct_name}Provider) Interval() time.Duration {\\
\tif p.config != nil \&\& p.config.PollInterval > 0 {\\
\t\treturn p.config.PollInterval\\
\t}\\
\treturn 5 * time.Minute // Default interval\\
}\\
" "$file"
            echo "  ✓ Added methods to ${struct_name}Provider"
        else
            echo "  ✗ Could not find Fetch method in $file"
        fi
    else
        echo "  ✗ Could not find ${struct_name}Provider struct in $file"
    fi
done

echo "Done!"