#!/bin/bash
# SENTINEL V3 — Database Backup
set -e

DATA_DIR="${SENTINEL_DATA:-/home/ed/.openclaw/workspace-sentinel-backend/data}"
BACKUP_DIR="/home/ed/Gunther/backups/sentinel"
DATE=$(date +%Y%m%d_%H%M%S)

mkdir -p "$BACKUP_DIR"

# SQLite online backup
if [ -f "$DATA_DIR/sentinel.db" ]; then
    sqlite3 "$DATA_DIR/sentinel.db" ".backup '$BACKUP_DIR/sentinel_${DATE}.db'"
    echo "Backup: $BACKUP_DIR/sentinel_${DATE}.db"

    # Compress
    gzip "$BACKUP_DIR/sentinel_${DATE}.db"
    echo "Compressed: $BACKUP_DIR/sentinel_${DATE}.db.gz"

    # Cleanup old backups (keep 30 days)
    find "$BACKUP_DIR" -name "sentinel_*.db.gz" -mtime +30 -delete
    echo "Cleaned up backups older than 30 days"
else
    echo "No database found at $DATA_DIR/sentinel.db"
fi

# Backup config
if [ -f "$HOME/.config/sentinel/config.json" ]; then
    cp "$HOME/.config/sentinel/config.json" "$BACKUP_DIR/config_${DATE}.json"
    echo "Config backed up"
fi
