package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/openclaw/sentinel-backend/internal/model"
)

// ---------------------------------------------------------------------------
// Event queries (V3 API layer)
// ---------------------------------------------------------------------------

// EventQueryParams controls filtering and pagination for GetEvents.
type EventQueryParams struct {
	Category string
	Severity string
	Source   string
	Hours    int
	Limit    int
	Offset   int
}

// V3Event converts a legacy Event into a V3 model.Event-compatible struct.
// The existing Event struct uses string IDs and time.Time; V3 API models use
// int64 IDs and string timestamps. We bridge the gap at the query layer.

// GetV3Events returns events with V3-style pagination and total count.
func (s *Storage) GetV3Events(params EventQueryParams) ([]model.Event, int, error) {
	// Build the existing ListFilter from V3 params
	filter := ListFilter{
		Category: params.Category,
		Severity: params.Severity,
		Source:   params.Source,
		Limit:    params.Limit,
		Offset:   params.Offset,
	}
	if params.Hours > 0 {
		filter.StartTime = time.Now().UTC().Add(-time.Duration(params.Hours) * time.Hour)
	}
	if filter.Limit <= 0 {
		filter.Limit = 50
	}

	// Delegate to existing ListEvents
	return s.ListEvents(nil, filter)
}

// GetEventByRowID retrieves a single event by its SQLite rowid.
func (s *Storage) GetEventByRowID(id int64) (*model.Event, error) {
	query := `
		SELECT id FROM events WHERE rowid = ?
	`
	var textID string
	if err := s.db.QueryRow(query, id).Scan(&textID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("GetEventByRowID: %w", err)
	}
	return s.GetEvent(nil, textID)
}

// SearchEvents performs a full-text search on events.
func (s *Storage) SearchEvents(query string, limit int) ([]model.Event, error) {
	if limit <= 0 {
		limit = 50
	}
	filter := ListFilter{
		Query: query,
		Limit: limit,
	}
	events, _, err := s.ListEvents(nil, filter)
	return events, err
}

// GetEventsByBBox returns events within a geographic bounding box.
func (s *Storage) GetEventsByBBox(minLon, minLat, maxLon, maxLat float64, limit int) ([]model.Event, error) {
	if limit <= 0 {
		limit = 200
	}
	filter := ListFilter{
		BBox:  []float64{minLon, minLat, maxLon, maxLat},
		Limit: limit,
	}
	events, _, err := s.ListEvents(nil, filter)
	return events, err
}

// GetV3EventsByTimeRange returns events between two ISO-8601 timestamps.
func (s *Storage) GetV3EventsByTimeRange(start, end string, limit int) ([]model.Event, error) {
	if limit <= 0 {
		limit = 200
	}
	startT, err := time.Parse(time.RFC3339, start)
	if err != nil {
		return nil, fmt.Errorf("invalid start time: %w", err)
	}
	endT, err := time.Parse(time.RFC3339, end)
	if err != nil {
		return nil, fmt.Errorf("invalid end time: %w", err)
	}
	filter := ListFilter{
		StartTime: startT,
		EndTime:   endT,
		Limit:     limit,
	}
	events, _, err2 := s.ListEvents(nil, filter)
	return events, err2
}

// GetRecentEventsByCategory returns events in a category from the last N hours.
func (s *Storage) GetRecentEventsByCategory(category string, hours int, limit int) ([]model.Event, error) {
	if limit <= 0 {
		limit = 100
	}
	filter := ListFilter{
		Category:  category,
		StartTime: time.Now().UTC().Add(-time.Duration(hours) * time.Hour),
		Limit:     limit,
	}
	events, _, err := s.ListEvents(nil, filter)
	return events, err
}

// GetRecentEventsBySeverity returns events at or above a minimum severity from the last N hours.
func (s *Storage) GetRecentEventsBySeverity(minSeverity string, hours int) ([]model.Event, error) {
	// Map severity to list of matching severities (inclusive upward)
	severityOrder := []string{"low", "medium", "high", "critical"}
	var included []string
	found := false
	for _, s := range severityOrder {
		if s == minSeverity {
			found = true
		}
		if found {
			included = append(included, s)
		}
	}
	if len(included) == 0 {
		included = severityOrder
	}

	cutoff := time.Now().UTC().Add(-time.Duration(hours) * time.Hour)
	placeholders := make([]string, len(included))
	args := make([]interface{}, len(included))
	for i, sev := range included {
		placeholders[i] = "?"
		args[i] = sev
	}
	args = append(args, cutoff)

	query := fmt.Sprintf(`
		SELECT
			e.id, e.title, e.description, e.source, e.source_id, e.occurred_at, e.ingested_at,
			e.location_type, e.coordinates_json, e.bbox_json, e.precision, e.magnitude,
			e.category, e.severity, e.metadata_json
		FROM events e
		WHERE e.severity IN (%s) AND e.occurred_at >= ?
		ORDER BY e.occurred_at DESC
	`, strings.Join(placeholders, ","))

	return s.scanV3Events(query, args...)
}

// AcknowledgeEvent marks an event as acknowledged.
func (s *Storage) AcknowledgeEvent(id int64) error {
	_, err := s.db.Exec("UPDATE events SET acknowledged = 1 WHERE rowid = ?", id)
	return err
}

// UpdateV3TruthScore updates the truth_score for an event by rowid.
func (s *Storage) UpdateV3TruthScore(eventID int64, score int) error {
	_, err := s.db.Exec("UPDATE events SET truth_score = ? WHERE rowid = ?", score, eventID)
	return err
}

// ---------------------------------------------------------------------------
// Correlation queries (V3 model types)
// ---------------------------------------------------------------------------

// InsertV3Correlation stores a correlation flash and returns its ID.
func (s *Storage) InsertV3Correlation(c *model.CorrelationFlash) error {
	eventsJSON, err := json.Marshal(c.Events)
	if err != nil {
		return fmt.Errorf("marshal events: %w", err)
	}
	confirmed := 0
	if c.Confirmed {
		confirmed = 1
	}
	result, err := s.db.Exec(`
		INSERT INTO correlations (region_name, lat, lon, radius_km, event_count, source_count,
		                          started_at, last_event_at, confirmed, incident_name, events_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, c.RegionName, c.Lat, c.Lon, c.RadiusKm, c.EventCount, c.SourceCount,
		c.StartedAt, c.LastEventAt, confirmed, c.IncidentName, string(eventsJSON))
	if err != nil {
		return err
	}
	c.ID, _ = result.LastInsertId()
	return nil
}

// GetActiveCorrelations returns all unconfirmed (active) correlation flashes.
func (s *Storage) GetActiveCorrelations() ([]model.CorrelationFlash, error) {
	rows, err := s.db.Query(`
		SELECT id, region_name, lat, lon, radius_km, event_count, source_count,
		       started_at, last_event_at, confirmed, COALESCE(incident_name,''), events_json
		FROM correlations
		WHERE confirmed = 0
		ORDER BY last_event_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.CorrelationFlash
	for rows.Next() {
		var c model.CorrelationFlash
		var confirmed int
		var eventsJSON sql.NullString
		if err := rows.Scan(&c.ID, &c.RegionName, &c.Lat, &c.Lon, &c.RadiusKm,
			&c.EventCount, &c.SourceCount, &c.StartedAt, &c.LastEventAt,
			&confirmed, &c.IncidentName, &eventsJSON); err != nil {
			return nil, err
		}
		c.Confirmed = confirmed != 0
		if eventsJSON.Valid && eventsJSON.String != "" {
			json.Unmarshal([]byte(eventsJSON.String), &c.Events)
		}
		results = append(results, c)
	}
	return results, nil
}

// ConfirmCorrelation marks a correlation as confirmed with an incident name.
func (s *Storage) ConfirmCorrelation(id int64, incidentName string) error {
	_, err := s.db.Exec(`
		UPDATE correlations SET confirmed = 1, incident_name = ? WHERE id = ?
	`, incidentName, id)
	return err
}

// ---------------------------------------------------------------------------
// Signal board
// ---------------------------------------------------------------------------

// InsertSignalBoardSnapshot stores a signal board snapshot.
func (s *Storage) InsertSignalBoardSnapshot(sb *model.SignalBoard) error {
	calculatedAt := sb.CalculatedAt
	if calculatedAt == "" {
		calculatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	_, err := s.db.Exec(`
		INSERT INTO signal_board_log (military, cyber, financial, natural, health, calculated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, sb.Military, sb.Cyber, sb.Financial, sb.Natural, sb.Health, calculatedAt)
	return err
}

// GetLatestV3SignalBoard returns the most recent signal board entry as a V3 model.
func (s *Storage) GetLatestV3SignalBoard() (*model.SignalBoard, error) {
	row := s.db.QueryRow(`
		SELECT military, cyber, financial, natural, health, calculated_at
		FROM signal_board_log
		ORDER BY calculated_at DESC
		LIMIT 1
	`)
	var sb model.SignalBoard
	if err := row.Scan(&sb.Military, &sb.Cyber, &sb.Financial, &sb.Natural, &sb.Health, &sb.CalculatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Populate active counts
	var alertCount int
	s.db.QueryRow("SELECT COUNT(*) FROM alert_rules WHERE enabled = 1").Scan(&alertCount)
	sb.ActiveAlerts = alertCount

	var corrCount int
	s.db.QueryRow("SELECT COUNT(*) FROM correlations WHERE confirmed = 0").Scan(&corrCount)
	sb.ActiveCorrelations = corrCount

	return &sb, nil
}

// ---------------------------------------------------------------------------
// Anomalies (V3 model types)
// ---------------------------------------------------------------------------

// InsertV3Anomaly stores an anomaly and sets its ID.
func (s *Storage) InsertV3Anomaly(a *model.Anomaly) error {
	detectedAt := a.DetectedAt
	if detectedAt == "" {
		detectedAt = time.Now().UTC().Format(time.RFC3339)
	}
	result, err := s.db.Exec(`
		INSERT INTO anomalies (provider_name, region, expected_rate, actual_rate, spike_factor, detected_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, a.ProviderName, a.Region, a.ExpectedRate, a.ActualRate, a.SpikeFactor, detectedAt)
	if err != nil {
		return err
	}
	a.ID, _ = result.LastInsertId()
	return nil
}

// GetV3ActiveAnomalies returns unresolved anomalies as V3 model types.
func (s *Storage) GetV3ActiveAnomalies() ([]model.Anomaly, error) {
	rows, err := s.db.Query(`
		SELECT id, provider_name, region, expected_rate, actual_rate, spike_factor, detected_at
		FROM anomalies
		WHERE resolved_at IS NULL
		ORDER BY detected_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.Anomaly
	for rows.Next() {
		var a model.Anomaly
		if err := rows.Scan(&a.ID, &a.ProviderName, &a.Region, &a.ExpectedRate,
			&a.ActualRate, &a.SpikeFactor, &a.DetectedAt); err != nil {
			return nil, err
		}
		results = append(results, a)
	}
	return results, nil
}

// ResolveV3Anomaly marks an anomaly as resolved by ID.
func (s *Storage) ResolveV3Anomaly(id int64) error {
	_, err := s.db.Exec(`
		UPDATE anomalies SET resolved_at = ? WHERE id = ?
	`, time.Now().UTC().Format(time.RFC3339), id)
	return err
}

// ---------------------------------------------------------------------------
// Truth confirmations (V3 convenience wrapper)
// ---------------------------------------------------------------------------

// InsertV3TruthConfirmation records a cross-source confirmation using int64 IDs.
func (s *Storage) InsertV3TruthConfirmation(primaryID, confirmingID int64, confirmingSource string) error {
	_, err := s.db.Exec(`
		INSERT INTO truth_confirmations (primary_event_id, confirming_source, confirming_event_id, confirmed_at)
		VALUES (?, ?, ?, ?)
	`, primaryID, confirmingSource, confirmingID, time.Now().UTC().Format(time.RFC3339))
	return err
}

// ---------------------------------------------------------------------------
// Alert rules
// ---------------------------------------------------------------------------

// GetAlertRules returns all alert rules.
func (s *Storage) GetAlertRules() ([]model.AlertRule, error) {
	rows, err := s.db.Query(`
		SELECT id, name, conditions_json, actions_json, enabled, created_at
		FROM alert_rules
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.AlertRule
	for rows.Next() {
		var r model.AlertRule
		var condJSON, actJSON sql.NullString
		var enabled int
		if err := rows.Scan(&r.ID, &r.Name, &condJSON, &actJSON, &enabled, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.Enabled = enabled != 0
		if condJSON.Valid {
			r.Conditions = json.RawMessage(condJSON.String)
		}
		if actJSON.Valid {
			r.Actions = json.RawMessage(actJSON.String)
		}
		results = append(results, r)
	}
	return results, nil
}

// CreateAlertRule inserts a new alert rule.
func (s *Storage) CreateAlertRule(rule *model.AlertRule) error {
	enabled := 0
	if rule.Enabled {
		enabled = 1
	}
	createdAt := rule.CreatedAt
	if createdAt == "" {
		createdAt = time.Now().UTC().Format(time.RFC3339)
	}
	result, err := s.db.Exec(`
		INSERT INTO alert_rules (name, conditions_json, actions_json, enabled, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, rule.Name, string(rule.Conditions), string(rule.Actions), enabled, createdAt)
	if err != nil {
		return err
	}
	rule.ID, _ = result.LastInsertId()
	return nil
}

// UpdateAlertRule updates an existing alert rule.
func (s *Storage) UpdateAlertRule(id int64, rule *model.AlertRule) error {
	enabled := 0
	if rule.Enabled {
		enabled = 1
	}
	_, err := s.db.Exec(`
		UPDATE alert_rules SET name = ?, conditions_json = ?, actions_json = ?, enabled = ?
		WHERE id = ?
	`, rule.Name, string(rule.Conditions), string(rule.Actions), enabled, id)
	return err
}

// DeleteAlertRule removes an alert rule by ID.
func (s *Storage) DeleteAlertRule(id int64) error {
	_, err := s.db.Exec("DELETE FROM alert_rules WHERE id = ?", id)
	return err
}

// ---------------------------------------------------------------------------
// News items
// ---------------------------------------------------------------------------

// InsertNewsItem stores a news article.
func (s *Storage) InsertNewsItem(item *model.NewsItem) error {
	ingestedAt := item.IngestedAt
	if ingestedAt == "" {
		ingestedAt = time.Now().UTC().Format(time.RFC3339)
	}
	result, err := s.db.Exec(`
		INSERT OR IGNORE INTO news_items
			(title, url, description, source_name, source_category, pub_date, ingested_at,
			 relevance_score, lat, lon, matched_event_id, truth_score)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, item.Title, item.URL, item.Description, item.SourceName, item.SourceCategory,
		item.PubDate, ingestedAt, item.RelevanceScore, item.Lat, item.Lon,
		item.MatchedEventID, item.TruthScore)
	if err != nil {
		return err
	}
	item.ID, _ = result.LastInsertId()
	return nil
}

// GetRecentNews returns the most recent news items.
func (s *Storage) GetRecentNews(limit int) ([]model.NewsItem, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(`
		SELECT id, title, url, COALESCE(description,''), source_name, COALESCE(source_category,''),
		       pub_date, ingested_at, relevance_score, COALESCE(lat,0), COALESCE(lon,0),
		       COALESCE(matched_event_id,0), truth_score
		FROM news_items
		ORDER BY pub_date DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.NewsItem
	for rows.Next() {
		var n model.NewsItem
		if err := rows.Scan(&n.ID, &n.Title, &n.URL, &n.Description, &n.SourceName,
			&n.SourceCategory, &n.PubDate, &n.IngestedAt, &n.RelevanceScore,
			&n.Lat, &n.Lon, &n.MatchedEventID, &n.TruthScore); err != nil {
			return nil, err
		}
		results = append(results, n)
	}
	return results, nil
}

// ---------------------------------------------------------------------------
// Notification log
// ---------------------------------------------------------------------------

// LogNotification records a sent notification.
func (s *Storage) LogNotification(channel string, eventID int64, status string, errMsg string) error {
	_, err := s.db.Exec(`
		INSERT INTO notification_log (channel, event_id, sent_at, status, error)
		VALUES (?, ?, ?, ?, ?)
	`, channel, eventID, time.Now().UTC().Format(time.RFC3339), status, errMsg)
	return err
}

// ---------------------------------------------------------------------------
// Provider health
// ---------------------------------------------------------------------------

// UpdateProviderHealth upserts a provider health record.
// Uses the events table to count recent events for the provider.
func (s *Storage) UpdateProviderHealth(name, status string, eventsLastHour int, lastError string) error {
	// Provider health is tracked via the provider_catalog table if it exists,
	// otherwise we just update metadata. For now, this is a no-op that can be
	// extended when the provider_catalog table is added.
	// The data is typically held in-memory by the scheduler; this persists it.
	return nil
}

// GetProviderHealth returns health info for all known providers.
// Derives data from events table grouped by source.
func (s *Storage) GetProviderHealth() ([]model.ProviderCatalog, error) {
	cutoff := time.Now().UTC().Add(-1 * time.Hour)
	rows, err := s.db.Query(`
		SELECT source, COUNT(*) AS cnt,
		       MAX(ingested_at) AS last_fetch
		FROM events
		WHERE ingested_at >= ?
		GROUP BY source
		ORDER BY source
	`, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.ProviderCatalog
	for rows.Next() {
		var p model.ProviderCatalog
		var lastFetch string
		if err := rows.Scan(&p.Name, &p.EventsLastHour, &lastFetch); err != nil {
			return nil, err
		}
		p.DisplayName = p.Name
		p.Status = "ok"
		p.Enabled = true
		p.LastFetch = lastFetch
		results = append(results, p)
	}
	return results, nil
}

// ---------------------------------------------------------------------------
// Financial overview (derived from events with category=financial)
// ---------------------------------------------------------------------------

// GetLatestFinancialData returns the most recent financial event metadata as a FinancialOverview.
func (s *Storage) GetLatestFinancialData() (*model.FinancialOverview, error) {
	row := s.db.QueryRow(`
		SELECT metadata_json, occurred_at
		FROM events
		WHERE category = 'financial'
		ORDER BY occurred_at DESC
		LIMIT 1
	`)
	var metaJSON sql.NullString
	var occurredAt string
	if err := row.Scan(&metaJSON, &occurredAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	fo := &model.FinancialOverview{
		UpdatedAt: occurredAt,
	}
	if metaJSON.Valid && metaJSON.String != "" {
		// Attempt to unmarshal the metadata directly into the overview
		json.Unmarshal([]byte(metaJSON.String), fo)
		fo.UpdatedAt = occurredAt
	}
	return fo, nil
}

// ---------------------------------------------------------------------------
// Briefing log
// ---------------------------------------------------------------------------

// InsertBriefing stores an AI-generated briefing.
func (s *Storage) InsertBriefing(content string, channels string) error {
	_, err := s.db.Exec(`
		INSERT INTO briefing_log (content, generated_at, delivered_channels)
		VALUES (?, ?, ?)
	`, content, time.Now().UTC().Format(time.RFC3339), channels)
	return err
}

// GetLatestBriefing returns the most recent briefing content and timestamp.
func (s *Storage) GetLatestBriefing() (string, string, error) {
	var content, generatedAt string
	err := s.db.QueryRow(`
		SELECT content, generated_at FROM briefing_log
		ORDER BY generated_at DESC
		LIMIT 1
	`).Scan(&content, &generatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", "", nil
		}
		return "", "", err
	}
	return content, generatedAt, nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// scanV3Events runs a raw query and scans results into model.Event slices.
func (s *Storage) scanV3Events(query string, args ...interface{}) ([]model.Event, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("scanV3Events: %w", err)
	}
	defer rows.Close()

	var events []model.Event
	for rows.Next() {
		var event model.Event
		var (
			coordsJSON            string
			bboxJSON              sql.NullString
			magnitude             sql.NullFloat64
			category, severityStr sql.NullString
			metadataJSON          sql.NullString
			precisionStr          string
		)

		if err := rows.Scan(
			&event.ID, &event.Title, &event.Description, &event.Source, &event.SourceID,
			&event.OccurredAt, &event.IngestedAt, &event.Location.Type, &coordsJSON, &bboxJSON,
			&precisionStr, &magnitude, &category, &severityStr, &metadataJSON,
		); err != nil {
			return nil, fmt.Errorf("scanV3Events scan: %w", err)
		}

		event.Precision = model.Precision(precisionStr)
		event.Magnitude = magnitude.Float64
		event.Category = category.String
		event.Severity = model.Severity(severityStr.String)

		// Parse coordinates
		if err := json.Unmarshal([]byte(coordsJSON), &event.Location.Coordinates); err != nil {
			return nil, fmt.Errorf("unmarshal coordinates: %w", err)
		}

		// Parse bbox
		if bboxJSON.Valid && bboxJSON.String != "" {
			json.Unmarshal([]byte(bboxJSON.String), &event.Location.BBox)
		}

		// Parse metadata
		if metadataJSON.Valid && metadataJSON.String != "" {
			json.Unmarshal([]byte(metadataJSON.String), &event.Metadata)
		}

		events = append(events, event)
	}
	return events, nil
}
