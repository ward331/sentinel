package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/openclaw/sentinel-backend/internal/model"
)

// EventRow is a lightweight event representation for engine queries.
type EventRow struct {
	ID          string
	Title       string
	Description string
	Source      string
	SourceID    string
	OccurredAt  time.Time
	Lat         float64
	Lon         float64
	Category    string
	Severity    string
	Magnitude   float64
	Metadata    map[string]string
}

// GetRecentEvents returns events that occurred within the last N minutes.
func (s *Storage) GetRecentEvents(ctx context.Context, minutes int) ([]EventRow, error) {
	cutoff := time.Now().UTC().Add(-time.Duration(minutes) * time.Minute)
	query := `
		SELECT id, title, description, source, source_id, occurred_at,
		       coordinates_json, category, severity, magnitude, metadata_json
		FROM events
		WHERE occurred_at >= ?
		ORDER BY occurred_at DESC
	`
	return s.scanEventRows(ctx, query, cutoff)
}

// GetEventsByTimeRange returns events in a time range.
func (s *Storage) GetEventsByTimeRange(ctx context.Context, start, end time.Time) ([]EventRow, error) {
	query := `
		SELECT id, title, description, source, source_id, occurred_at,
		       coordinates_json, category, severity, magnitude, metadata_json
		FROM events
		WHERE occurred_at >= ? AND occurred_at <= ?
		ORDER BY occurred_at DESC
	`
	return s.scanEventRows(ctx, query, start.UTC(), end.UTC())
}

// GetEventsByCategoryAndTimeRange returns events matching category within a time range.
func (s *Storage) GetEventsByCategoryAndTimeRange(ctx context.Context, category string, start, end time.Time) ([]EventRow, error) {
	query := `
		SELECT id, title, description, source, source_id, occurred_at,
		       coordinates_json, category, severity, magnitude, metadata_json
		FROM events
		WHERE category = ? AND occurred_at >= ? AND occurred_at <= ?
		ORDER BY occurred_at DESC
	`
	return s.scanEventRows(ctx, query, category, start.UTC(), end.UTC())
}

// GetEventsBySeverityAndTimeRange returns events with severity >= given level within a time range.
func (s *Storage) GetEventsBySeverityAndTimeRange(ctx context.Context, severities []string, start, end time.Time) ([]EventRow, error) {
	if len(severities) == 0 {
		return nil, nil
	}
	placeholders := make([]string, len(severities))
	args := make([]interface{}, 0, len(severities)+2)
	for i, sev := range severities {
		placeholders[i] = "?"
		args = append(args, sev)
	}
	args = append(args, start.UTC(), end.UTC())
	query := fmt.Sprintf(`
		SELECT id, title, description, source, source_id, occurred_at,
		       coordinates_json, category, severity, magnitude, metadata_json
		FROM events
		WHERE severity IN (%s) AND occurred_at >= ? AND occurred_at <= ?
		ORDER BY occurred_at DESC
	`, strings.Join(placeholders, ","))
	return s.scanEventRows(ctx, query, args...)
}

// GetEventCountBySourceAndRegion returns the count of events per source per country_code
// in the given time window.
type SourceRegionCount struct {
	Source string
	Region string
	Count  int
}

func (s *Storage) GetEventCountBySourceAndHour(ctx context.Context, start, end time.Time) ([]SourceRegionCount, error) {
	// Region is derived from metadata_json country_code, or "unknown"
	query := `
		SELECT source,
		       COALESCE(json_extract(metadata_json, '$.country_code'), 'unknown') AS region,
		       COUNT(*) AS cnt
		FROM events
		WHERE occurred_at >= ? AND occurred_at <= ?
		GROUP BY source, region
	`
	rows, err := s.db.QueryContext(ctx, query, start.UTC(), end.UTC())
	if err != nil {
		return nil, fmt.Errorf("GetEventCountBySourceAndHour: %w", err)
	}
	defer rows.Close()

	var results []SourceRegionCount
	for rows.Next() {
		var r SourceRegionCount
		if err := rows.Scan(&r.Source, &r.Region, &r.Count); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, nil
}

// GetEventsBySourceAndTimeRange returns events for a specific source within a time range.
func (s *Storage) GetEventsBySourceAndTimeRange(ctx context.Context, source string, start, end time.Time) ([]EventRow, error) {
	query := `
		SELECT id, title, description, source, source_id, occurred_at,
		       coordinates_json, category, severity, magnitude, metadata_json
		FROM events
		WHERE source = ? AND occurred_at >= ? AND occurred_at <= ?
		ORDER BY occurred_at DESC
	`
	return s.scanEventRows(ctx, query, source, start.UTC(), end.UTC())
}

// UpdateTruthScore updates the truth_score for an event.
func (s *Storage) UpdateTruthScore(ctx context.Context, eventID string, score int) error {
	_, err := s.db.ExecContext(ctx, "UPDATE events SET truth_score = ? WHERE id = ?", score, eventID)
	return err
}

// InsertTruthConfirmation records a cross-source confirmation.
func (s *Storage) InsertTruthConfirmation(ctx context.Context, primaryEventID, confirmingSource string, confirmingEventID string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO truth_confirmations (primary_event_id, confirming_source, confirming_event_id, confirmed_at)
		VALUES (?, ?, ?, ?)
	`, primaryEventID, confirmingSource, confirmingEventID, time.Now().UTC())
	return err
}

// GetTruthConfirmationCount returns the number of unique confirming sources for an event.
func (s *Storage) GetTruthConfirmationCount(ctx context.Context, eventID string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT confirming_source)
		FROM truth_confirmations WHERE primary_event_id = ?
	`, eventID).Scan(&count)
	return count, err
}

// InsertCorrelation stores a correlation flash record.
func (s *Storage) InsertCorrelation(ctx context.Context, regionName string, lat, lon, radiusKm float64,
	eventCount, sourceCount int, startedAt, lastEventAt time.Time, eventIDs []string) (int64, error) {
	eventsJSON, _ := json.Marshal(eventIDs)
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO correlations (region_name, lat, lon, radius_km, event_count, source_count,
		                          started_at, last_event_at, events_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, regionName, lat, lon, radiusKm, eventCount, sourceCount,
		startedAt.UTC(), lastEventAt.UTC(), string(eventsJSON))
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// GetRecentCorrelations returns correlations from the last N minutes.
func (s *Storage) GetRecentCorrelations(ctx context.Context, minutes int) ([]CorrelationRow, error) {
	cutoff := time.Now().UTC().Add(-time.Duration(minutes) * time.Minute)
	query := `
		SELECT id, region_name, lat, lon, radius_km, event_count, source_count,
		       started_at, last_event_at, events_json
		FROM correlations
		WHERE last_event_at >= ?
		ORDER BY last_event_at DESC
	`
	rows, err := s.db.QueryContext(ctx, query, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []CorrelationRow
	for rows.Next() {
		var r CorrelationRow
		var eventsJSON sql.NullString
		if err := rows.Scan(&r.ID, &r.RegionName, &r.Lat, &r.Lon, &r.RadiusKm,
			&r.EventCount, &r.SourceCount, &r.StartedAt, &r.LastEventAt, &eventsJSON); err != nil {
			return nil, err
		}
		if eventsJSON.Valid {
			json.Unmarshal([]byte(eventsJSON.String), &r.EventIDs)
		}
		results = append(results, r)
	}
	return results, nil
}

// CorrelationRow is a stored correlation.
type CorrelationRow struct {
	ID          int64
	RegionName  string
	Lat         float64
	Lon         float64
	RadiusKm    float64
	EventCount  int
	SourceCount int
	StartedAt   time.Time
	LastEventAt time.Time
	EventIDs    []string
}

// InsertAnomaly stores an anomaly record.
func (s *Storage) InsertAnomaly(ctx context.Context, provider, region string, expected, actual, spike float64) (int64, error) {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO anomalies (provider_name, region, expected_rate, actual_rate, spike_factor, detected_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, provider, region, expected, actual, spike, time.Now().UTC())
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// ResolveAnomaly marks an anomaly as resolved.
func (s *Storage) ResolveAnomaly(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE anomalies SET resolved_at = ? WHERE id = ?
	`, time.Now().UTC(), id)
	return err
}

// GetActiveAnomalies returns unresolved anomalies.
func (s *Storage) GetActiveAnomalies(ctx context.Context) ([]AnomalyRow, error) {
	query := `
		SELECT id, provider_name, region, expected_rate, actual_rate, spike_factor, detected_at
		FROM anomalies
		WHERE resolved_at IS NULL
		ORDER BY detected_at DESC
	`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []AnomalyRow
	for rows.Next() {
		var r AnomalyRow
		if err := rows.Scan(&r.ID, &r.Provider, &r.Region, &r.ExpectedRate, &r.ActualRate, &r.SpikeFactor, &r.DetectedAt); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, nil
}

// AnomalyRow is a stored anomaly.
type AnomalyRow struct {
	ID           int64
	Provider     string
	Region       string
	ExpectedRate float64
	ActualRate   float64
	SpikeFactor  float64
	DetectedAt   time.Time
}

// InsertSignalBoardEntry stores a signal board snapshot.
func (s *Storage) InsertSignalBoardEntry(ctx context.Context, military, cyber, financial, natural, health int) (int64, error) {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO signal_board_log (military, cyber, financial, natural, health, calculated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, military, cyber, financial, natural, health, time.Now().UTC())
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// GetLatestSignalBoard returns the most recent signal board entry.
func (s *Storage) GetLatestSignalBoard(ctx context.Context) (*SignalBoardRow, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, military, cyber, financial, natural, health, calculated_at
		FROM signal_board_log
		ORDER BY calculated_at DESC
		LIMIT 1
	`)
	var r SignalBoardRow
	if err := row.Scan(&r.ID, &r.Military, &r.Cyber, &r.Financial, &r.Natural, &r.Health, &r.CalculatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &r, nil
}

// SignalBoardRow is a stored signal board snapshot.
type SignalBoardRow struct {
	ID           int64
	Military     int
	Cyber        int
	Financial    int
	Natural      int
	Health       int
	CalculatedAt time.Time
}

// GetSimilarEvents finds events with the same category within radiusKm and timeWindow of a given event.
func (s *Storage) GetSimilarEvents(ctx context.Context, eventID, category string, lat, lon float64, radiusKm float64, since time.Time) ([]EventRow, error) {
	// Get all events in the category within the time range, then filter by distance in Go
	query := `
		SELECT id, title, description, source, source_id, occurred_at,
		       coordinates_json, category, severity, magnitude, metadata_json
		FROM events
		WHERE category = ? AND occurred_at >= ? AND id != ?
		ORDER BY occurred_at DESC
	`
	return s.scanEventRows(ctx, query, category, since.UTC(), eventID)
}

// GetEventsByTextSearch finds events matching text in title or description within a time range.
func (s *Storage) GetEventsByTextSearch(ctx context.Context, searchTerms []string, start, end time.Time) ([]EventRow, error) {
	if len(searchTerms) == 0 {
		return nil, nil
	}
	var conditions []string
	var args []interface{}
	for _, term := range searchTerms {
		conditions = append(conditions, "(LOWER(title) LIKE ? OR LOWER(description) LIKE ?)")
		likeTerm := "%" + strings.ToLower(term) + "%"
		args = append(args, likeTerm, likeTerm)
	}
	args = append(args, start.UTC(), end.UTC())
	query := fmt.Sprintf(`
		SELECT id, title, description, source, source_id, occurred_at,
		       coordinates_json, category, severity, magnitude, metadata_json
		FROM events
		WHERE (%s) AND occurred_at >= ? AND occurred_at <= ?
		ORDER BY occurred_at DESC
	`, strings.Join(conditions, " OR "))
	return s.scanEventRows(ctx, query, args...)
}

// scanEventRows executes a query and returns EventRow results.
func (s *Storage) scanEventRows(ctx context.Context, query string, args ...interface{}) ([]EventRow, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("scanEventRows: %w", err)
	}
	defer rows.Close()

	var results []EventRow
	for rows.Next() {
		var r EventRow
		var coordsJSON string
		var category, severity sql.NullString
		var magnitude sql.NullFloat64
		var metadataJSON sql.NullString
		if err := rows.Scan(&r.ID, &r.Title, &r.Description, &r.Source, &r.SourceID,
			&r.OccurredAt, &coordsJSON, &category, &severity, &magnitude, &metadataJSON); err != nil {
			return nil, err
		}
		r.Category = category.String
		r.Severity = severity.String
		r.Magnitude = magnitude.Float64

		// Parse coordinates [lon, lat]
		var coords []float64
		if err := json.Unmarshal([]byte(coordsJSON), &coords); err == nil && len(coords) >= 2 {
			r.Lon = coords[0]
			r.Lat = coords[1]
		}

		// Parse metadata
		if metadataJSON.Valid && metadataJSON.String != "" {
			json.Unmarshal([]byte(metadataJSON.String), &r.Metadata)
		}

		results = append(results, r)
	}
	return results, nil
}

// GetEventRowByID returns a lightweight EventRow by ID.
func (s *Storage) GetEventRowByID(ctx context.Context, id string) (*EventRow, error) {
	query := `
		SELECT id, title, description, source, source_id, occurred_at,
		       coordinates_json, category, severity, magnitude, metadata_json
		FROM events
		WHERE id = ?
	`
	rows, err := s.scanEventRows(ctx, query, id)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return &rows[0], nil
}

// Ensure model import is used
var _ = model.Event{}
