-- SQLite schema for SENTINEL events database
-- Enable WAL mode for better concurrency
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA foreign_keys = ON;

-- Main events table
CREATE TABLE IF NOT EXISTS events (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    source TEXT NOT NULL,
    source_id TEXT,
    occurred_at DATETIME NOT NULL,
    ingested_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    location_type TEXT NOT NULL CHECK (location_type IN ('Point', 'Polygon')),
    coordinates_json TEXT NOT NULL, -- Store GeoJSON coordinates as JSON
    bbox_json TEXT, -- Store bounding box as JSON array [min_lon, min_lat, max_lon, max_lat]
    precision TEXT NOT NULL CHECK (precision IN ('exact', 'polygon_area', 'approximate', 'text_inferred', 'unknown')),
    magnitude REAL,
    category TEXT,
    severity TEXT CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    metadata_json TEXT, -- Store metadata as JSON
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Badges table (many-to-one relationship with events)
CREATE TABLE IF NOT EXISTS badges (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_id TEXT NOT NULL,
    label TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('source', 'precision', 'freshness')),
    timestamp DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_events_source ON events(source);
CREATE INDEX IF NOT EXISTS idx_events_occurred_at ON events(occurred_at);
CREATE INDEX IF NOT EXISTS idx_events_ingested_at ON events(ingested_at);
CREATE INDEX IF NOT EXISTS idx_events_magnitude ON events(magnitude);
CREATE INDEX IF NOT EXISTS idx_events_category ON events(category);
CREATE INDEX IF NOT EXISTS idx_events_severity ON events(severity);
CREATE INDEX IF NOT EXISTS idx_badges_event_id ON badges(event_id);

-- Full-text search table for event titles and descriptions
CREATE VIRTUAL TABLE IF NOT EXISTS events_fts USING fts5(
    title,
    description,
    content='events',
    content_rowid='rowid'
);

-- Spatial index for bounding box queries using R*Tree
CREATE VIRTUAL TABLE IF NOT EXISTS events_rtree USING rtree(
    id,              -- Integer primary key matching events.rowid
    min_lat, max_lat, -- Latitude bounds
    min_lon, max_lon  -- Longitude bounds
);

-- Triggers to maintain FTS5 index
CREATE TRIGGER IF NOT EXISTS events_ai AFTER INSERT ON events BEGIN
    INSERT INTO events_fts(rowid, title, description) VALUES (new.rowid, new.title, new.description);
END;

CREATE TRIGGER IF NOT EXISTS events_ad AFTER DELETE ON events BEGIN
    INSERT INTO events_fts(events_fts, rowid, title, description) VALUES('delete', old.rowid, old.title, old.description);
END;

CREATE TRIGGER IF NOT EXISTS events_au AFTER UPDATE ON events BEGIN
    INSERT INTO events_fts(events_fts, rowid, title, description) VALUES('delete', old.rowid, old.title, old.description);
    INSERT INTO events_fts(rowid, title, description) VALUES (new.rowid, new.title, new.description);
END;

-- Trigger to maintain R*Tree index
CREATE TRIGGER IF NOT EXISTS events_rtree_insert AFTER INSERT ON events BEGIN
    INSERT INTO events_rtree(id, min_lat, max_lat, min_lon, max_lon)
    SELECT 
        new.rowid,
        json_extract(new.bbox_json, '$[1]'), -- min_lat
        json_extract(new.bbox_json, '$[3]'), -- max_lat (corrected from [2] to [3])
        json_extract(new.bbox_json, '$[0]'), -- min_lon
        json_extract(new.bbox_json, '$[2]')  -- max_lon
    WHERE new.bbox_json IS NOT NULL;
END;

CREATE TRIGGER IF NOT EXISTS events_rtree_update AFTER UPDATE ON events BEGIN
    DELETE FROM events_rtree WHERE id = old.rowid;
    INSERT INTO events_rtree(id, min_lat, max_lat, min_lon, max_lon)
    SELECT 
        new.rowid,
        json_extract(new.bbox_json, '$[1]'), -- min_lat
        json_extract(new.bbox_json, '$[3]'), -- max_lat
        json_extract(new.bbox_json, '$[0]'), -- min_lon
        json_extract(new.bbox_json, '$[2]')  -- max_lon
    WHERE new.bbox_json IS NOT NULL;
END;

CREATE TRIGGER IF NOT EXISTS events_rtree_delete AFTER DELETE ON events BEGIN
    DELETE FROM events_rtree WHERE id = old.rowid;
END;