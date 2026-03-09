package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
	"github.com/openclaw/sentinel-backend/internal/model"
)

// Storage handles database operations for events
type Storage struct {
	db *sql.DB
}

// New creates a new Storage instance
func New(dbPath string) (*Storage, error) {
	return NewWithConfig(dbPath, false, 5) // Default: no pooling, 5 max connections
}

// NewWithConfig creates a new Storage instance with configuration
func NewWithConfig(dbPath string, usePool bool, maxConnections int) (*Storage, error) {
	if usePool && maxConnections > 1 {
		// Use optimized storage with connection pooling
		optStorage, err := NewOptimizedStorage(dbPath)
		if err != nil {
			return nil, err
		}
		return optStorage.Storage, nil
	}
	
	// Use simple storage without pooling (for SQLite compatibility)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(1) // SQLite limitation
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Initialize schema
	if err := initSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return &Storage{db: db}, nil
}

// Close closes the database connection
func (s *Storage) Close() error {
	return s.db.Close()
}

// DB returns the underlying database connection (for health checks)
func (s *Storage) DB() *sql.DB {
	return s.db
}

// initSchema creates tables and indexes if they don't exist
func initSchema(db *sql.DB) error {
	_, err := db.Exec(defaultSchema)
	return err
}

// StoreEvent stores a new event in the database
func (s *Storage) StoreEvent(ctx context.Context, event *model.Event) error {
	// Generate ID if not provided
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	event.IngestedAt = time.Now().UTC()

	// Marshal location coordinates and bbox
	coordsJSON, err := json.Marshal(event.Location.Coordinates)
	if err != nil {
		return fmt.Errorf("failed to marshal coordinates: %w", err)
	}

	// Handle bbox - marshal if exists, otherwise use NULL
	var bboxJSON interface{} = nil
	if event.Location.BBox != nil && len(event.Location.BBox) == 4 {
		bboxJSON, err = json.Marshal(event.Location.BBox)
		if err != nil {
			return fmt.Errorf("failed to marshal bbox: %w", err)
		}
	}

	// Handle metadata - marshal if exists, otherwise use NULL
	var metadataJSON interface{} = nil
	if event.Metadata != nil && len(event.Metadata) > 0 {
		metadataJSON, err = json.Marshal(event.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert event
	query := `
		INSERT INTO events (
			id, title, description, source, source_id, occurred_at, ingested_at,
			location_type, coordinates_json, bbox_json, precision, magnitude,
			category, severity, metadata_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	// Convert empty values to NULL for database
	severityValue := interface{}(nil)
	if event.Severity != "" {
		severityValue = string(event.Severity)
	}

	categoryValue := interface{}(nil)
	if event.Category != "" {
		categoryValue = event.Category
	}

	magnitudeValue := interface{}(nil)
	if event.Magnitude != 0 {
		magnitudeValue = event.Magnitude
	}

	// Execute insert and get last insert ID
	result, err := tx.ExecContext(ctx, query,
		event.ID,
		event.Title,
		event.Description,
		event.Source,
		event.SourceID,
		event.OccurredAt.UTC(),
		event.IngestedAt.UTC(),
		event.Location.Type,
		string(coordsJSON),
		bboxJSON,
		string(event.Precision),
		magnitudeValue,
		categoryValue,
		severityValue,
		metadataJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to insert event: %w", err)
	}

	// Get the rowid (SQLite's internal row identifier)
	rowid, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	// Insert badges
	for _, badge := range event.Badges {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO badges (event_id, label, type, timestamp)
			VALUES (?, ?, ?, ?)
		`, event.ID, badge.Label, string(badge.Type), badge.Timestamp.UTC())
		if err != nil {
			return fmt.Errorf("failed to insert badge: %w", err)
		}
	}

	// Insert into FTS5 table for full-text search
	ftsQuery := "INSERT INTO events_fts (rowid, title, description) VALUES (?, ?, ?)"
	if _, err := tx.ExecContext(ctx, ftsQuery, rowid, event.Title, event.Description); err != nil {
		// Log error but don't fail - FTS5 is optional
		fmt.Printf("Warning: failed to insert into FTS5 table: %v\n", err)
	}

	// Insert into R*Tree table for spatial indexing
	// Parse coordinates to get lat/lon
	var coords []float64
	if err := json.Unmarshal([]byte(coordsJSON), &coords); err == nil && len(coords) >= 2 {
		lon := coords[0]
		lat := coords[1]
		rtreeQuery := "INSERT INTO events_rtree (id, min_lat, max_lat, min_lon, max_lon) VALUES (?, ?, ?, ?, ?)"
		if _, err := tx.ExecContext(ctx, rtreeQuery, rowid, lat, lat, lon, lon); err != nil {
			// Log error but don't fail - R*Tree is optional
			fmt.Printf("Warning: failed to insert into R*Tree table: %v\n", err)
		}
	}

	return tx.Commit()
}

// GetEvent retrieves an event by ID
func (s *Storage) GetEvent(ctx context.Context, id string) (*model.Event, error) {
	query := `
		SELECT 
			e.id, e.title, e.description, e.source, e.source_id, e.occurred_at, e.ingested_at,
			e.location_type, e.coordinates_json, e.bbox_json, e.precision, e.magnitude,
			e.category, e.severity, e.metadata_json,
			b.label, b.type, b.timestamp
		FROM events e
		LEFT JOIN badges b ON e.id = b.event_id
		WHERE e.id = ?
		ORDER BY b.timestamp DESC
	`

	rows, err := s.db.QueryContext(ctx, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to query event: %w", err)
	}
	defer rows.Close()

	var event *model.Event
	var badges []model.Badge

	for rows.Next() {
		var (
			eventID, title, description, source, sourceID string
			occurredAt, ingestedAt                        time.Time
			locationType, coordsJSON string
			bboxJSON                                      sql.NullString
			precisionStr                                  string
			magnitude                                     sql.NullFloat64
			category, severityStr                         sql.NullString
			metadataJSON                                  sql.NullString
			badgeLabel, badgeTypeStr                     sql.NullString
			badgeTimestamp                                sql.NullTime
		)

		err := rows.Scan(
			&eventID, &title, &description, &source, &sourceID,
			&occurredAt, &ingestedAt, &locationType, &coordsJSON, &bboxJSON,
			&precisionStr, &magnitude, &category, &severityStr, &metadataJSON,
			&badgeLabel, &badgeTypeStr, &badgeTimestamp,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event row: %w", err)
		}

		if event == nil {
			// Parse coordinates
			var coords interface{}
			if err := json.Unmarshal([]byte(coordsJSON), &coords); err != nil {
				return nil, fmt.Errorf("failed to unmarshal coordinates: %w", err)
			}

			// Parse bbox
			var bbox []float64
			if bboxJSON.Valid && bboxJSON.String != "" {
				if err := json.Unmarshal([]byte(bboxJSON.String), &bbox); err != nil {
					return nil, fmt.Errorf("failed to unmarshal bbox: %w", err)
				}
			}

			// Parse metadata
			var metadata map[string]string
			if metadataJSON.Valid && metadataJSON.String != "" {
				if err := json.Unmarshal([]byte(metadataJSON.String), &metadata); err != nil {
					return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
				}
			}

			event = &model.Event{
				ID:          eventID,
				Title:       title,
				Description: description,
				Source:      source,
				SourceID:    sourceID,
				OccurredAt:  occurredAt,
				IngestedAt:  ingestedAt,
				Location: model.Location{
					Type:        locationType,
					Coordinates: coords,
					BBox:        bbox,
				},
				Precision: model.Precision(precisionStr),
				Magnitude: magnitude.Float64,
				Category:  category.String,
				Severity:  model.Severity(severityStr.String),
				Metadata:  metadata,
			}
		}

		// Add badge if present
		if badgeLabel.Valid && badgeTypeStr.Valid && badgeTimestamp.Valid {
			badges = append(badges, model.Badge{
				Label:     badgeLabel.String,
				Type:      model.BadgeType(badgeTypeStr.String),
				Timestamp: badgeTimestamp.Time,
			})
		}
	}

	if event == nil {
		return nil, sql.ErrNoRows
	}

	event.Badges = badges
	return event, nil
}

// ListEvents retrieves events with pagination and filtering
func (s *Storage) ListEvents(ctx context.Context, filter ListFilter) ([]model.Event, int, error) {
	var query string
	var countQuery string
	var args []interface{}
	var total int

	if filter.Query != "" {
		// Use FTS5 for full-text search
		ftsWhere, ftsArgs := buildFTSWhereClause(filter)
		countQuery = fmt.Sprintf(`
			SELECT COUNT(*) 
			FROM events e
			INNER JOIN events_fts fts ON e.rowid = fts.rowid
			%s
		`, ftsWhere)
		
		query = fmt.Sprintf(`
			SELECT 
				e.id, e.title, e.description, e.source, e.source_id, e.occurred_at, e.ingested_at,
				e.location_type, e.coordinates_json, e.bbox_json, e.precision, e.magnitude,
				e.category, e.severity, e.metadata_json
			FROM events e
			INNER JOIN events_fts fts ON e.rowid = fts.rowid
			%s
			ORDER BY e.occurred_at DESC
			LIMIT ? OFFSET ?
		`, ftsWhere)
		
		args = ftsArgs
	} else {
		// Regular query without full-text search
		whereClause, whereArgs := buildWhereClause(filter)
		countQuery = fmt.Sprintf("SELECT COUNT(*) FROM events e %s", whereClause)
		
		query = fmt.Sprintf(`
			SELECT 
				e.id, e.title, e.description, e.source, e.source_id, e.occurred_at, e.ingested_at,
				e.location_type, e.coordinates_json, e.bbox_json, e.precision, e.magnitude,
				e.category, e.severity, e.metadata_json
			FROM events e
			%s
			ORDER BY e.occurred_at DESC
			LIMIT ? OFFSET ?
		`, whereClause)
		
		args = whereArgs
	}
	
	// Count total matching events
	err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count events: %w", err)
	}

	args = append(args, filter.Limit, filter.Offset)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []model.Event
	for rows.Next() {
		var event model.Event
		var (
			coordsJSON string
			bboxJSON   sql.NullString
			magnitude            sql.NullFloat64
			category, severityStr sql.NullString
			metadataJSON         sql.NullString
		)

		var precisionStr string
		err := rows.Scan(
			&event.ID, &event.Title, &event.Description, &event.Source, &event.SourceID,
			&event.OccurredAt, &event.IngestedAt, &event.Location.Type, &coordsJSON, &bboxJSON,
			&precisionStr, &magnitude, &category, &severityStr, &metadataJSON,
		)
		event.Precision = model.Precision(precisionStr)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan event: %w", err)
		}

		// Parse coordinates
		if err := json.Unmarshal([]byte(coordsJSON), &event.Location.Coordinates); err != nil {
			return nil, 0, fmt.Errorf("failed to unmarshal coordinates: %w", err)
		}

		// Parse bbox
		if bboxJSON.Valid && bboxJSON.String != "" {
			if err := json.Unmarshal([]byte(bboxJSON.String), &event.Location.BBox); err != nil {
				return nil, 0, fmt.Errorf("failed to unmarshal bbox: %w", err)
			}
		}

		// Parse optional fields
		event.Magnitude = magnitude.Float64
		event.Category = category.String
		event.Severity = model.Severity(severityStr.String)

		// Parse metadata
		if metadataJSON.Valid && metadataJSON.String != "" {
			if err := json.Unmarshal([]byte(metadataJSON.String), &event.Metadata); err != nil {
				return nil, 0, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		events = append(events, event)
	}

	return events, total, nil
}

// ListFilter defines filtering options for listing events
type ListFilter struct {
	Source       string
	Category     string
	Severity     string
	MinMagnitude float64
	MaxMagnitude float64
	StartTime    time.Time
	EndTime      time.Time
	BBox         []float64 // [min_lon, min_lat, max_lon, max_lat]
	Query        string    // Full-text search query
	Limit        int
	Offset       int
}

func buildWhereClause(filter ListFilter) (string, []interface{}) {
	var conditions []string
	var args []interface{}

	if filter.Source != "" {
		conditions = append(conditions, "e.source = ?")
		args = append(args, filter.Source)
	}

	if filter.Category != "" {
		conditions = append(conditions, "e.category = ?")
		args = append(args, filter.Category)
	}

	if filter.Severity != "" {
		conditions = append(conditions, "e.severity = ?")
		args = append(args, filter.Severity)
	}

	if filter.MinMagnitude > 0 {
		conditions = append(conditions, "e.magnitude >= ?")
		args = append(args, filter.MinMagnitude)
	}

	if filter.MaxMagnitude > 0 {
		conditions = append(conditions, "e.magnitude <= ?")
		args = append(args, filter.MaxMagnitude)
	}

	if !filter.StartTime.IsZero() {
		conditions = append(conditions, "e.occurred_at >= ?")
		args = append(args, filter.StartTime.UTC())
	}

	if !filter.EndTime.IsZero() {
		conditions = append(conditions, "e.occurred_at <= ?")
		args = append(args, filter.EndTime.UTC())
	}

	if len(filter.BBox) == 4 {
		// Use R*Tree for spatial queries
		conditions = append(conditions, `
			EXISTS (
				SELECT 1 FROM events_rtree r 
				WHERE r.id = e.rowid 
				AND r.min_lon <= ? AND r.max_lon >= ?
				AND r.min_lat <= ? AND r.max_lat >= ?
			)
		`)
		args = append(args, filter.BBox[2], filter.BBox[0], filter.BBox[3], filter.BBox[1])
	}

	if len(conditions) == 0 {
		return "", args
	}

	return "WHERE " + joinConditions(conditions, " AND "), args
}

func joinConditions(conditions []string, sep string) string {
	if len(conditions) == 0 {
		return ""
	}
	if len(conditions) == 1 {
		return conditions[0]
	}
	result := conditions[0]
	for _, cond := range conditions[1:] {
		result += sep + cond
	}
	return result
}

// buildFTSWhereClause builds WHERE clause for FTS5 search with additional filters
func buildFTSWhereClause(filter ListFilter) (string, []interface{}) {
	var conditions []string
	var args []interface{}

	// FTS5 search condition
	if filter.Query != "" {
		conditions = append(conditions, "fts.events_fts MATCH ?")
		args = append(args, filter.Query)
	}

	// Add other filters
	if filter.Source != "" {
		conditions = append(conditions, "e.source = ?")
		args = append(args, filter.Source)
	}

	if filter.Category != "" {
		conditions = append(conditions, "e.category = ?")
		args = append(args, filter.Category)
	}

	if filter.Severity != "" {
		conditions = append(conditions, "e.severity = ?")
		args = append(args, filter.Severity)
	}

	if filter.MinMagnitude > 0 {
		conditions = append(conditions, "e.magnitude >= ?")
		args = append(args, filter.MinMagnitude)
	}

	if filter.MaxMagnitude > 0 {
		conditions = append(conditions, "e.magnitude <= ?")
		args = append(args, filter.MaxMagnitude)
	}

	if !filter.StartTime.IsZero() {
		conditions = append(conditions, "e.occurred_at >= ?")
		args = append(args, filter.StartTime.UTC())
	}

	if !filter.EndTime.IsZero() {
		conditions = append(conditions, "e.occurred_at <= ?")
		args = append(args, filter.EndTime.UTC())
	}

	if len(filter.BBox) == 4 {
		// Use R*Tree for spatial queries
		conditions = append(conditions, `
			EXISTS (
				SELECT 1 FROM events_rtree r 
				WHERE r.id = e.rowid 
				AND r.min_lon <= ? AND r.max_lon >= ?
				AND r.min_lat <= ? AND r.max_lat >= ?
			)
		`)
		args = append(args, filter.BBox[2], filter.BBox[0], filter.BBox[3], filter.BBox[1])
	}

	if len(conditions) == 0 {
		return "", args
	}

	return "WHERE " + joinConditions(conditions, " AND "), args
}

// defaultSchema is the fallback schema if embedded assets fail
const defaultSchema = `
-- Simplified SQLite schema for SENTINEL with WAL, FTS5, and R*Tree
-- No triggers, just the essential tables
-- WAL mode causes disk I/O error with modernc.org/sqlite, using DELETE mode instead
PRAGMA journal_mode = DELETE;
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
    location_type TEXT NOT NULL,
    coordinates_json TEXT NOT NULL,
    bbox_json TEXT,
    precision TEXT NOT NULL,
    magnitude REAL,
    category TEXT,
    severity TEXT,
    metadata_json TEXT
);

-- Badges table
CREATE TABLE IF NOT EXISTS badges (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_id TEXT NOT NULL,
    label TEXT NOT NULL,
    type TEXT NOT NULL,
    timestamp DATETIME NOT NULL
);

-- Full-text search table (simple version without content= parameter)
CREATE VIRTUAL TABLE IF NOT EXISTS events_fts USING fts5(
    title,
    description
);

-- Spatial index table
CREATE VIRTUAL TABLE IF NOT EXISTS events_rtree USING rtree(
    id,
    min_lat, max_lat,
    min_lon, max_lon
);
`

// GetEventBySourceID retrieves an event by source and source_id
func (s *Storage) GetEventBySourceID(ctx context.Context, source, sourceID string) (*model.Event, error) {
	query := `
		SELECT 
			e.id, e.title, e.description, e.source, e.source_id, e.occurred_at, e.ingested_at,
			e.location_type, e.coordinates_json, e.bbox_json, e.precision, e.magnitude,
			e.category, e.severity, e.metadata_json,
			b.label, b.type, b.timestamp
		FROM events e
		LEFT JOIN badges b ON e.id = b.event_id
		WHERE e.source = ? AND e.source_id = ?
		ORDER BY b.timestamp DESC
	`

	rows, err := s.db.QueryContext(ctx, query, source, sourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query event by source_id: %w", err)
	}
	defer rows.Close()

	var event *model.Event
	var badges []model.Badge

	for rows.Next() {
		var (
			eventID, title, description, source, sourceID string
			occurredAt, ingestedAt                        time.Time
			locationType, coordsJSON string
			bboxJSON                                      sql.NullString
			precisionStr                                  string
			magnitude                                     sql.NullFloat64
			category, severityStr                         sql.NullString
			metadataJSON                                  sql.NullString
			badgeLabel, badgeTypeStr                     sql.NullString
			badgeTimestamp                                sql.NullTime
		)

		err := rows.Scan(
			&eventID, &title, &description, &source, &sourceID,
			&occurredAt, &ingestedAt, &locationType, &coordsJSON, &bboxJSON,
			&precisionStr, &magnitude, &category, &severityStr, &metadataJSON,
			&badgeLabel, &badgeTypeStr, &badgeTimestamp,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event row: %w", err)
		}

		if event == nil {
			// Parse coordinates
			var coords interface{}
			if err := json.Unmarshal([]byte(coordsJSON), &coords); err != nil {
				return nil, fmt.Errorf("failed to unmarshal coordinates: %w", err)
			}

			// Parse bbox
			var bbox []float64
			if bboxJSON.Valid && bboxJSON.String != "" {
				if err := json.Unmarshal([]byte(bboxJSON.String), &bbox); err != nil {
					return nil, fmt.Errorf("failed to unmarshal bbox: %w", err)
				}
			}

			// Parse metadata
			var metadata map[string]string
			if metadataJSON.Valid && metadataJSON.String != "" {
				if err := json.Unmarshal([]byte(metadataJSON.String), &metadata); err != nil {
					return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
				}
			}

			event = &model.Event{
				ID:          eventID,
				Title:       title,
				Description: description,
				Source:      source,
				SourceID:    sourceID,
				OccurredAt:  occurredAt,
				IngestedAt:  ingestedAt,
				Location: model.Location{
					Type:        locationType,
					Coordinates: coords,
					BBox:        bbox,
				},
				Precision: model.Precision(precisionStr),
				Magnitude: magnitude.Float64,
				Category:  category.String,
				Severity:  model.Severity(severityStr.String),
				Metadata:  metadata,
			}
		}

		// Add badge if present
		if badgeLabel.Valid && badgeTypeStr.Valid && badgeTimestamp.Valid {
			badges = append(badges, model.Badge{
				Label:     badgeLabel.String,
				Type:      model.BadgeType(badgeTypeStr.String),
				Timestamp: badgeTimestamp.Time,
			})
		}
	}

	if event == nil {
		return nil, nil // Not an error, just not found
	}

	event.Badges = badges
	return event, nil
}