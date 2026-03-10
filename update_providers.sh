#!/bin/bash
# Update all providers to implement the Provider interface

set -e

PROVIDER_DIR="internal/provider"

echo "Updating providers to implement Provider interface..."

# List of provider files (excluding interface.go)
PROVIDER_FILES=(
    "adsb_one.go"
    "airplanes_live.go"
    "celestrak.go"
    "gdelt.go"
    "global_fishing_watch.go"
    "global_forest_watch.go"
    "iranconflict.go"
    "liveuamap.go"
    "noaa_cap.go"
    "noaa_nws.go"
    "openmeteo.go"
    "opensanctions.go"
    "opensky_enhanced.go"
    "promed.go"
    "reliefweb.go"
    "swpc.go"
    "tsunami.go"
    "usgs.go"
    "volcano.go"
    "who.go"
    "nasa_firms.go"
    "piracy_imb.go"
    "financial_markets.go"
)

# Default intervals for different provider types
declare -A INTERVALS=(
    ["realtime"]="5s"
    ["fast"]="1m"
    ["medium"]="5m"
    ["slow"]="15m"
    ["daily"]="1h"
    ["weekly"]="6h"
)

# Determine interval for each provider
get_interval() {
    local file="$1"
    case "$file" in
        *airplanes*|*adsb*|*opensky*)
            echo "${INTERVALS[realtime]}"
            ;;
        *usgs*|*noaa_nws*|*gdelt*)
            echo "${INTERVALS[fast]}"
            ;;
        *noaa_cap*|*openmeteo*|*iranconflict*|*liveuamap*)
            echo "${INTERVALS[medium]}"
            ;;
        *celestrak*|*swpc*|*nasa_firms*)
            echo "${INTERVALS[slow]}"
            ;;
        *global*|*opensanctions*|*reliefweb*|*tsunami*|*volcano*|*who*|*promed*|*piracy*|*financial*)
            echo "${INTERVALS[daily]}"
            ;;
        *)
            echo "${INTERVALS[medium]}"
            ;;
    esac
}

# Update a single provider file
update_provider() {
    local file="$1"
    local interval="$2"
    
    echo "Updating $file with interval $interval..."
    
    # Create a temporary file
    tmp_file="${file}.tmp"
    
    # Read the file and update it
    awk -v interval="$interval" '
    BEGIN { in_struct=0; struct_name=""; added_methods=0 }
    
    # Find the provider struct definition
    /type.*Provider struct/ {
        in_struct=1
        struct_name=$2
        sub(/Provider.*/, "", struct_name)
        print $0
        next
    }
    
    # End of struct
    in_struct && /^}/ {
        print $0
        if (!added_methods) {
            print ""
            print "// Name returns the provider name"
            print "func (p *" struct_name "Provider) Name() string {"
            print "    return \"" tolower(struct_name) "\""
            print "}"
            print ""
            print "// Interval returns the polling interval"
            print "func (p *" struct_name "Provider) Interval() time.Duration {"
            print "    interval, _ := time.ParseDuration(\"" interval "\")"
            print "    return interval"
            print "}"
            print ""
            print "// Enabled returns whether the provider is enabled"
            print "func (p *" struct_name "Provider) Enabled() bool {"
            print "    return p.config != nil && p.config.Enabled"
            print "}"
            added_methods=1
        }
        in_struct=0
        next
    }
    
    # Preserve other lines
    { print $0 }
    ' "$file" > "$tmp_file"
    
    # Check if we need to add time import
    if ! grep -q "time\.Duration" "$tmp_file" && grep -q "Interval() time\.Duration" "$tmp_file"; then
        sed -i '/^import ($/,/^)/ {
            /^)/i\    "time"
        }' "$tmp_file"
    fi
    
    # Replace the original file
    mv "$tmp_file" "$file"
}

# Update all providers
for file in "${PROVIDER_FILES[@]}"; do
    full_path="$PROVIDER_DIR/$file"
    if [[ -f "$full_path" ]]; then
        interval=$(get_interval "$file")
        update_provider "$full_path" "$interval"
    else
        echo "Warning: $full_path not found"
    fi
done

echo "Provider updates complete!"
echo "Testing compilation..."

# Test compilation
cd /home/ed/.openclaw/workspace-sentinel-backend
if go build ./internal/provider/...; then
    echo "✅ Providers compile successfully"
else
    echo "❌ Compilation failed"
    exit 1
fi