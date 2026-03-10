#!/bin/bash
echo "Removing duplicate provider methods..."

# Files with known duplicates
FILES="nasa_firms.go noaa_nws.go openmeteo.go opensanctions.go"

for file in $FILES; do
    echo "Processing $file..."
    
    # Create temp file
    tmpfile=$(mktemp)
    
    # Read file and remove duplicate method blocks
    in_duplicate=0
    skip_count=0
    line_num=0
    
    while IFS= read -r line; do
        ((line_num++))
        
        # Check for start of a duplicate method
        if [[ "$line" =~ ^//\ (Name|Enabled|Interval)\ returns ]] && [[ $line_num -gt 50 ]]; then
            # This is likely a duplicate (appears later in file)
            in_duplicate=1
            skip_count=0
            echo "  Found duplicate at line $line_num: $line"
            continue
        fi
        
        if [[ $in_duplicate -eq 1 ]]; then
            if [[ "$line" =~ ^}$ ]]; then
                skip_count=$((skip_count + 1))
                if [[ $skip_count -eq 1 ]]; then
                    # Skip the closing brace
                    continue
                else
                    # End of method
                    in_duplicate=0
                    skip_count=0
                fi
            else
                # Skip lines inside duplicate method
                continue
            fi
        fi
        
        # Write non-duplicate lines
        echo "$line" >> "$tmpfile"
    done < "internal/provider/$file"
    
    # Replace original file
    mv "$tmpfile" "internal/provider/$file"
    echo "  Fixed $file"
done

echo "Done!"