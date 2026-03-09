package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/openclaw/sentinel-backend/internal/model"
)

// OSINTStorage handles OSINT resource database operations
type OSINTStorage struct {
	db *sql.DB
}

// NewOSINTStorage creates a new OSINT storage instance
func NewOSINTStorage(db *sql.DB) *OSINTStorage {
	return &OSINTStorage{db: db}
}

// CreateTable creates the osint_resources table if it doesn't exist
func (s *OSINTStorage) CreateTable(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS osint_resources (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		platform TEXT NOT NULL,
		category TEXT NOT NULL,
		display_name TEXT NOT NULL,
		profile_url TEXT NOT NULL,
		description TEXT,
		credibility TEXT NOT NULL DEFAULT 'community',
		is_builtin BOOLEAN NOT NULL DEFAULT 0,
		last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		tags TEXT, -- JSON array of strings
		api_key_required BOOLEAN NOT NULL DEFAULT 0,
		free_tier BOOLEAN NOT NULL DEFAULT 1,
		notes TEXT,
		UNIQUE(profile_url)
	);

	CREATE INDEX IF NOT EXISTS idx_osint_platform ON osint_resources(platform);
	CREATE INDEX IF NOT EXISTS idx_osint_category ON osint_resources(category);
	CREATE INDEX IF NOT EXISTS idx_osint_credibility ON osint_resources(credibility);
	CREATE INDEX IF NOT EXISTS idx_osint_builtin ON osint_resources(is_builtin);
	`

	_, err := s.db.ExecContext(ctx, query)
	return err
}

// Insert inserts a new OSINT resource
func (s *OSINTStorage) Insert(ctx context.Context, resource *model.OSINTResource) error {
	tagsJSON, err := json.Marshal(resource.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	query := `
	INSERT INTO osint_resources (
		platform, category, display_name, profile_url, description,
		credibility, is_builtin, tags, api_key_required, free_tier, notes
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := s.db.ExecContext(ctx, query,
		resource.Platform,
		resource.Category,
		resource.DisplayName,
		resource.ProfileURL,
		resource.Description,
		resource.Credibility,
		resource.IsBuiltin,
		string(tagsJSON),
		resource.APIKeyRequired,
		resource.FreeTier,
		resource.Notes,
	)
	if err != nil {
		return fmt.Errorf("failed to insert OSINT resource: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}

	resource.ID = int(id)
	resource.CreatedAt = time.Now()
	resource.LastUpdated = time.Now()

	return nil
}

// GetByID retrieves an OSINT resource by ID
func (s *OSINTStorage) GetByID(ctx context.Context, id int) (*model.OSINTResource, error) {
	query := `
	SELECT id, platform, category, display_name, profile_url, description,
	       credibility, is_builtin, last_updated, created_at, tags,
	       api_key_required, free_tier, notes
	FROM osint_resources
	WHERE id = ?
	`

	row := s.db.QueryRowContext(ctx, query, id)
	return s.scanRow(row)
}

// GetByURL retrieves an OSINT resource by URL
func (s *OSINTStorage) GetByURL(ctx context.Context, url string) (*model.OSINTResource, error) {
	query := `
	SELECT id, platform, category, display_name, profile_url, description,
	       credibility, is_builtin, last_updated, created_at, tags,
	       api_key_required, free_tier, notes
	FROM osint_resources
	WHERE profile_url = ?
	`

	row := s.db.QueryRowContext(ctx, query, url)
	return s.scanRow(row)
}

// List retrieves OSINT resources with optional filtering
func (s *OSINTStorage) List(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*model.OSINTResource, error) {
	query := `
	SELECT id, platform, category, display_name, profile_url, description,
	       credibility, is_builtin, last_updated, created_at, tags,
	       api_key_required, free_tier, notes
	FROM osint_resources
	WHERE 1=1
	`
	args := []interface{}{}

	// Apply filters
	if platform, ok := filters["platform"].(string); ok && platform != "" {
		query += " AND platform = ?"
		args = append(args, platform)
	}
	if category, ok := filters["category"].(string); ok && category != "" {
		query += " AND category = ?"
		args = append(args, category)
	}
	if credibility, ok := filters["credibility"].(string); ok && credibility != "" {
		query += " AND credibility = ?"
		args = append(args, credibility)
	}
	if builtin, ok := filters["is_builtin"].(bool); ok {
		query += " AND is_builtin = ?"
		args = append(args, builtin)
	}
	if freeTier, ok := filters["free_tier"].(bool); ok {
		query += " AND free_tier = ?"
		args = append(args, freeTier)
	}

	// Search by tag
	if tag, ok := filters["tag"].(string); ok && tag != "" {
		query += " AND tags LIKE ?"
		args = append(args, "%"+tag+"%")
	}

	// Search by name/description
	if search, ok := filters["search"].(string); ok && search != "" {
		query += " AND (display_name LIKE ? OR description LIKE ?)"
		args = append(args, "%"+search+"%", "%"+search+"%")
	}

	// Order and limit
	query += " ORDER BY is_builtin DESC, credibility DESC, last_updated DESC"
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}
	if offset > 0 {
		query += " OFFSET ?"
		args = append(args, offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query OSINT resources: %w", err)
	}
	defer rows.Close()

	var resources []*model.OSINTResource
	for rows.Next() {
		resource, err := s.scanRows(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

// Update updates an OSINT resource
func (s *OSINTStorage) Update(ctx context.Context, resource *model.OSINTResource) error {
	tagsJSON, err := json.Marshal(resource.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	query := `
	UPDATE osint_resources SET
		platform = ?,
		category = ?,
		display_name = ?,
		profile_url = ?,
		description = ?,
		credibility = ?,
		is_builtin = ?,
		tags = ?,
		api_key_required = ?,
		free_tier = ?,
		notes = ?,
		last_updated = CURRENT_TIMESTAMP
	WHERE id = ?
	`

	_, err = s.db.ExecContext(ctx, query,
		resource.Platform,
		resource.Category,
		resource.DisplayName,
		resource.ProfileURL,
		resource.Description,
		resource.Credibility,
		resource.IsBuiltin,
		string(tagsJSON),
		resource.APIKeyRequired,
		resource.FreeTier,
		resource.Notes,
		resource.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update OSINT resource: %w", err)
	}

	return nil
}

// Delete deletes an OSINT resource by ID
func (s *OSINTStorage) Delete(ctx context.Context, id int) error {
	query := "DELETE FROM osint_resources WHERE id = ?"
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// Count returns the total number of OSINT resources
func (s *OSINTStorage) Count(ctx context.Context, filters map[string]interface{}) (int, error) {
	query := "SELECT COUNT(*) FROM osint_resources WHERE 1=1"
	args := []interface{}{}

	// Apply filters
	if platform, ok := filters["platform"].(string); ok && platform != "" {
		query += " AND platform = ?"
		args = append(args, platform)
	}
	if category, ok := filters["category"].(string); ok && category != "" {
		query += " AND category = ?"
		args = append(args, category)
	}
	if credibility, ok := filters["credibility"].(string); ok && credibility != "" {
		query += " AND credibility = ?"
		args = append(args, credibility)
	}
	if builtin, ok := filters["is_builtin"].(bool); ok {
		query += " AND is_builtin = ?"
		args = append(args, builtin)
	}

	var count int
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&count)
	return count, err
}

// scanRow scans a single row from the database
func (s *OSINTStorage) scanRow(row *sql.Row) (*model.OSINTResource, error) {
	var resource model.OSINTResource
	var tagsJSON string
	var lastUpdated, createdAt string

	err := row.Scan(
		&resource.ID,
		&resource.Platform,
		&resource.Category,
		&resource.DisplayName,
		&resource.ProfileURL,
		&resource.Description,
		&resource.Credibility,
		&resource.IsBuiltin,
		&lastUpdated,
		&createdAt,
		&tagsJSON,
		&resource.APIKeyRequired,
		&resource.FreeTier,
		&resource.Notes,
	)
	if err != nil {
		return nil, err
	}

	// Parse timestamps
	if lastUpdated != "" {
		resource.LastUpdated, _ = time.Parse(time.RFC3339, lastUpdated)
	}
	if createdAt != "" {
		resource.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	}

	// Parse tags JSON
	if tagsJSON != "" {
		json.Unmarshal([]byte(tagsJSON), &resource.Tags)
	}

	return &resource, nil
}

// scanRows scans rows from a query result
func (s *OSINTStorage) scanRows(rows *sql.Rows) (*model.OSINTResource, error) {
	var resource model.OSINTResource
	var tagsJSON string
	var lastUpdated, createdAt string

	err := rows.Scan(
		&resource.ID,
		&resource.Platform,
		&resource.Category,
		&resource.DisplayName,
		&resource.ProfileURL,
		&resource.Description,
		&resource.Credibility,
		&resource.IsBuiltin,
		&lastUpdated,
		&createdAt,
		&tagsJSON,
		&resource.APIKeyRequired,
		&resource.FreeTier,
		&resource.Notes,
	)
	if err != nil {
		return nil, err
	}

	// Parse timestamps
	if lastUpdated != "" {
		resource.LastUpdated, _ = time.Parse(time.RFC3339, lastUpdated)
	}
	if createdAt != "" {
		resource.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	}

	// Parse tags JSON
	if tagsJSON != "" {
		json.Unmarshal([]byte(tagsJSON), &resource.Tags)
	}

	return &resource, nil
}

// SeedBuiltinResources seeds the database with built-in OSINT resources
func (s *OSINTStorage) SeedBuiltinResources(ctx context.Context) error {
	builtinResources := []*model.OSINTResource{
		{
			Platform:    model.PlatformWeb,
			Category:    model.CategoryAviation,
			DisplayName: "Bellingcat ADS-B History",
			ProfileURL:  "https://github.com/bellingcat/adsb-history",
			Description: "Historical aircraft tracking and investigation tool with comprehensive aircraft identification database",
			Credibility: model.CredibilityVerifiedOSINT,
			IsBuiltin:   true,
			Tags:        []string{"aviation", "aircraft", "tracking", "investigation", "osint"},
			APIKeyRequired: false,
			FreeTier:    true,
			Notes:       "Integrated into SENTINEL for aircraft identification and military detection",
		},
		{
			Platform:    model.PlatformWeb,
			Category:    model.CategoryConflict,
			DisplayName: "Iran Strike Map",
			ProfileURL:  "https://www.iranstrikemap.com",
			Description: "Interactive map of Iran-Israel conflict events with real-time updates",
			Credibility: model.CredibilityCommunity,
			IsBuiltin:   true,
			Tags:        []string{"conflict", "iran", "israel", "middle-east", "map", "real-time"},
			APIKeyRequired: false,
			FreeTier:    true,
			Notes:       "Embedded as iframe in SENTINEL media wall",
		},
		{
			Platform:    model.PlatformDataset,
			Category:    model.CategoryConflict,
			DisplayName: "Iran-Israel War 2026 OSINT Data",
			ProfileURL:  "https://github.com/danielrosehill/Iran-Israel-War-2026-OSINT-Data",
			Description: "Comprehensive OSINT dataset on Iran-Israel conflict including waves.json with operation details",
			Credibility: model.CredibilityVerifiedOSINT,
			IsBuiltin:   true,
			Tags:        []string{"conflict", "osint", "dataset", "github", "waves"},
			APIKeyRequired: false,
			FreeTier:    true,
			Notes:       "Integrated into SENTINEL IranConflictProvider with 15-minute polling",
		},
		{
			Platform:    model.PlatformRSS,
			Category:    model.CategoryConflict,
			DisplayName: "ISW RSS Feed",
			ProfileURL:  "https://understandingwar.org/rss.xml",
			Description: "Institute for the Study of War analysis and conflict reporting",
			Credibility: model.CredibilityOfficial,
			IsBuiltin:   true,
			Tags:        []string{"conflict", "analysis", "rss", "middle-east", "ukraine"},
			APIKeyRequired: false,
			FreeTier:    true,
			Notes:       "Integrated into SENTINEL with 30-minute polling and keyword filtering",
		},
		{
			Platform:    model.PlatformWeb,
			Category:    "social_media_osint",
			DisplayName: "BirdHunt (Twitter/X Geotagged Search)",
			ProfileURL:  "https://birdhunt.huntintel.io/",
			Description: "Search for geotagged tweets near specific coordinates. Opens with event location pre-filled for TIER 2+ events.",
			Credibility: model.CredibilityVerifiedOSINT,
			IsBuiltin:   true,
			Tags:        []string{"twitter", "x", "social_media", "geotagged", "location_search", "osint"},
			APIKeyRequired: false,
			FreeTier:    true,
			Notes:       "Contextual link for TIER 2+ events. Auto-populates coordinates when user clicks [OSINT Sources].",
		},
		{
			Platform:    model.PlatformWeb,
			Category:    "social_media_osint",
			DisplayName: "InstaHunt (Instagram Location Search)",
			ProfileURL:  "https://instahunt.huntintel.io/",
			Description: "Search for Instagram posts near specific coordinates. Opens with event location pre-filled for conflict, disaster, and military events.",
			Credibility: model.CredibilityVerifiedOSINT,
			IsBuiltin:   true,
			Tags:        []string{"instagram", "social_media", "geotagged", "location_search", "osint", "visual_intel"},
			APIKeyRequired: false,
			FreeTier:    true,
			Notes:       "Contextual link for conflict, disaster, military events. Auto-populates coordinates when user clicks [OSINT Sources].",
		},
		{
			Platform:    model.PlatformWeb,
			Category:    model.CategoryConflict,
			DisplayName: "OpenSanctions",
			ProfileURL:  "https://opensanctions.org/",
			Description: "Comprehensive sanctions and PEP database covering 245+ sources (UN, EU, UK, US, Switzerland, Australia). Updated daily with structured entity data.",
			Credibility: model.CredibilityVerifiedOSINT,
			IsBuiltin:   true,
			Tags:        []string{"sanctions", "pep", "financial_intelligence", "compliance", "geopolitical_risk"},
			APIKeyRequired: false,
			FreeTier:    true,
			Notes:       "Replaces OFAC-only provider. Bulk data downloads free, API 10k req/month free. Integrated into SENTINEL financial alerts.",
		},
		{
			Platform:    model.PlatformAPI,
			Category:    model.CategoryWeather,
			DisplayName: "Global Forest Watch",
			ProfileURL:  "https://data-api.globalforestwatch.org/dataset/nasa_viirs_fire_alerts/latest/query",
			Description: "NASA VIIRS fire alerts via Global Forest Watch API. Higher accuracy than NASA FIRMS alone, better detection of smaller fires and improved nighttime detection.",
			Credibility: model.CredibilityOfficial,
			IsBuiltin:   true,
			Tags:        []string{"wildfire", "satellite", "nasa", "viirs", "environmental", "disaster"},
			APIKeyRequired: false,
			FreeTier:    true,
			Notes:       "Free, no API key required. 30-minute polling interval. Integrated into SENTINEL wildfire alerts.",
		},
		{
			Platform:    model.PlatformAPI,
			Category:    model.CategoryMaritime,
			DisplayName: "Global Fishing Watch",
			ProfileURL:  "https://gateway.api.globalfishingwatch.org/v3/events",
			Description: "Vessel activity tracking with AIS data. Detects fishing, transshipment, loitering, port visits. Includes dark vessel detection (flag = 'UNK').",
			Credibility: model.CredibilityOfficial,
			IsBuiltin:   true,
			Tags:        []string{"maritime", "vessel_tracking", "fishing", "transshipment", "ais", "ocean_intel"},
			APIKeyRequired: true,
			FreeTier:    true,
			Notes:       "Free tier available (API key optional). 1-hour polling interval. Integrated into SENTINEL maritime alerts.",
		},
		{
			Platform:    model.PlatformRSS,
			Category:    model.CategoryConflict,
			DisplayName: "LiveUAMap RSS",
			ProfileURL:  "https://liveuamap.com/rss",
			Description: "Crowdsourced conflict reporting with geotagged events. Community-verified OSINT from conflict zones worldwide.",
			Credibility: model.CredibilityCommunity,
			IsBuiltin:   true,
			Tags:        []string{"conflict", "crowdsourced", "geotagged", "real_time", "osint", "war_reporting"},
			APIKeyRequired: false,
			FreeTier:    true,
			Notes:       "Free, no API key required. 15-minute polling interval. Integrated into SENTINEL conflict alerts.",
		},
	}

	for _, resource := range builtinResources {
		// Check if resource already exists
		existing, err := s.GetByURL(ctx, resource.ProfileURL)
		if err == nil && existing != nil {
			// Update existing resource
			resource.ID = existing.ID
			if err := s.Update(ctx, resource); err != nil {
				fmt.Printf("Failed to update OSINT resource %s: %v\n", resource.DisplayName, err)
			}
		} else {
			// Insert new resource
			if err := s.Insert(ctx, resource); err != nil {
				fmt.Printf("Failed to insert OSINT resource %s: %v\n", resource.DisplayName, err)
			}
		}
	}

	return nil
}