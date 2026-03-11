package storage

import (
	"database/sql"
	"log"
)

// v3MigrationSQL adds the V3 tables alongside existing V2 tables.
const v3MigrationSQL = `
-- V3 correlation flash table
CREATE TABLE IF NOT EXISTS correlations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  region_name TEXT,
  lat REAL, lon REAL,
  radius_km REAL,
  event_count INTEGER,
  source_count INTEGER,
  started_at DATETIME,
  last_event_at DATETIME,
  confirmed INTEGER DEFAULT 0,
  incident_name TEXT,
  events_json TEXT
);

-- V3 truth confirmations
CREATE TABLE IF NOT EXISTS truth_confirmations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  primary_event_id INTEGER,
  confirming_source TEXT,
  confirming_event_id INTEGER,
  confirmed_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- V3 anomaly detection log
CREATE TABLE IF NOT EXISTS anomalies (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  provider_name TEXT,
  region TEXT,
  expected_rate REAL,
  actual_rate REAL,
  spike_factor REAL,
  detected_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  resolved_at DATETIME
);

-- V3 signal board history
CREATE TABLE IF NOT EXISTS signal_board_log (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  military INTEGER,
  cyber INTEGER,
  financial INTEGER,
  natural INTEGER,
  health INTEGER,
  calculated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- V3 notification audit log
CREATE TABLE IF NOT EXISTS notification_log (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  channel TEXT,
  event_id INTEGER,
  sent_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  status TEXT,
  error TEXT
);

-- V3 alert rules
CREATE TABLE IF NOT EXISTS alert_rules (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT,
  conditions_json TEXT,
  actions_json TEXT,
  enabled INTEGER DEFAULT 1,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- V3 AI briefing log
CREATE TABLE IF NOT EXISTS briefing_log (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  content TEXT,
  generated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  delivered_channels TEXT
);

-- V3 news items
CREATE TABLE IF NOT EXISTS news_items (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  title TEXT NOT NULL,
  url TEXT UNIQUE NOT NULL,
  description TEXT,
  source_name TEXT,
  source_category TEXT,
  pub_date DATETIME,
  ingested_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  relevance_score INTEGER DEFAULT 0,
  lat REAL, lon REAL,
  matched_event_id INTEGER,
  truth_score INTEGER DEFAULT 1
);
`

// RunV3Migration adds V3 tables and columns to an existing database.
// Safe to call multiple times — all statements use IF NOT EXISTS.
func RunV3Migration(db *sql.DB) error {
	log.Println("[storage] Running V3 schema migration...")

	if _, err := db.Exec(v3MigrationSQL); err != nil {
		return err
	}

	// Add truth_score and acknowledged columns to events if missing.
	// SQLite ALTER TABLE ADD COLUMN is idempotent-safe: it will error if
	// the column exists. We ignore that error.
	addColumnSafe(db, "events", "truth_score", "INTEGER DEFAULT 1")
	addColumnSafe(db, "events", "acknowledged", "INTEGER DEFAULT 0")

	log.Println("[storage] V3 migration complete")
	return nil
}

// addColumnSafe adds a column to a table, ignoring "duplicate column" errors.
func addColumnSafe(db *sql.DB, table, column, colType string) {
	stmt := "ALTER TABLE " + table + " ADD COLUMN " + column + " " + colType
	if _, err := db.Exec(stmt); err != nil {
		// "duplicate column name" is expected if migration already ran
		log.Printf("[storage] Column %s.%s: %v (likely already exists)", table, column, err)
	}
}
