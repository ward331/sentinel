#!/bin/bash
# SENTINEL V3 — Setup Script
set -e

echo "🛡 SENTINEL V3 Setup"
echo "===================="

# Build if needed
if [ ! -f bin/sentinel-linux-amd64 ]; then
    echo "Building..."
    make build
fi

# Create data directory
mkdir -p data logs

# Install systemd service (user level)
mkdir -p ~/.config/systemd/user
cp scripts/sentinel-user.service ~/.config/systemd/user/sentinel-v3.service
systemctl --user daemon-reload
systemctl --user enable sentinel-v3

echo ""
echo "✅ SENTINEL V3 installed!"
echo ""
echo "Commands:"
echo "  Start:   systemctl --user start sentinel-v3"
echo "  Stop:    systemctl --user stop sentinel-v3"
echo "  Status:  systemctl --user status sentinel-v3"
echo "  Logs:    journalctl --user -u sentinel-v3 -f"
echo ""
echo "  First run: bin/sentinel-linux-amd64 --wizard"
echo "  Web UI:    http://localhost:8080"
