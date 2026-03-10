package filter

import (
	"context"
	"time"

	"github.com/openclaw/sentinel-backend/internal/model"
)

// Filter represents a rule for filtering events
type Filter struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	
	// Rule conditions
	Conditions []Condition `json:"conditions"`
	
	// Actions when filter matches
	Actions []Action `json:"actions,omitempty"`
}

// Condition represents a single filtering condition
type Condition struct {
	Field     string      `json:"field"`     // Field to check (category, severity, location, etc.)
	Operator  string      `json:"operator"`  // Comparison operator (eq, ne, gt, lt, contains, in, etc.)
	Value     interface{} `json:"value"`     // Value to compare against
	Negate    bool        `json:"negate"`    // Whether to negate the condition
}

// Action represents an action to take when filter matches
type Action struct {
	Type    string                 `json:"type"`    // Action type (notify, tag, route, etc.)
	Config  map[string]interface{} `json:"config"`  // Action configuration
	Enabled bool                   `json:"enabled"` // Whether action is enabled
}

// Geofence represents a geographic boundary for filtering
type Geofence struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"` // point_radius, polygon, bbox
	Geometry  GeoJSON   `json:"geometry"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

// GeoJSON represents a geographic feature (simplified)
type GeoJSON struct {
	Type        string                 `json:"type"`
	Coordinates interface{}            `json:"coordinates"`
	Properties  map[string]interface{} `json:"properties,omitempty"`
}

// FilterEngine evaluates filters against events
type FilterEngine interface {
	// Evaluate evaluates a single event against all active filters
	Evaluate(ctx context.Context, event *model.Event) ([]string, error)
	
	// AddFilter adds a new filter
	AddFilter(ctx context.Context, filter *Filter) error
	
	// UpdateFilter updates an existing filter
	UpdateFilter(ctx context.Context, id string, filter *Filter) error
	
	// RemoveFilter removes a filter
	RemoveFilter(ctx context.Context, id string) error
	
	// GetFilter retrieves a filter by ID
	GetFilter(ctx context.Context, id string) (*Filter, error)
	
	// ListFilters lists all filters
	ListFilters(ctx context.Context) ([]*Filter, error)
	
	// EnableFilter enables a filter
	EnableFilter(ctx context.Context, id string) error
	
	// DisableFilter disables a filter
	DisableFilter(ctx context.Context, id string) error
	
	// MatchCount returns statistics about filter matches
	MatchCount(ctx context.Context, filterID string) (int64, error)
}

// RuleEvaluator evaluates individual conditions
type RuleEvaluator interface {
	// EvaluateCondition evaluates a single condition against an event
	EvaluateCondition(ctx context.Context, event *model.Event, condition *Condition) (bool, error)
	
	// ValidateCondition validates a condition configuration
	ValidateCondition(condition *Condition) error
	
	// SupportedFields returns list of fields that can be filtered
	SupportedFields() []string
	
	// SupportedOperators returns operators supported for each field
	SupportedOperators(field string) []string
}

// GeofenceEngine manages and evaluates geofences
type GeofenceEngine interface {
	// AddGeofence adds a new geofence
	AddGeofence(ctx context.Context, geofence *Geofence) error
	
	// UpdateGeofence updates an existing geofence
	UpdateGeofence(ctx context.Context, id string, geofence *Geofence) error
	
	// RemoveGeofence removes a geofence
	RemoveGeofence(ctx context.Context, id string) error
	
	// GetGeofence retrieves a geofence by ID
	GetGeofence(ctx context.Context, id string) (*Geofence, error)
	
	// ListGeofences lists all geofences
	ListGeofences(ctx context.Context) ([]*Geofence, error)
	
	// PointInGeofence checks if a point is inside any geofence
	PointInGeofence(ctx context.Context, lat, lon float64) ([]string, error)
	
	// EventInGeofence checks if an event is inside any geofence
	EventInGeofence(ctx context.Context, event *model.Event) ([]string, error)
	
	// ValidateGeometry validates geofence geometry
	ValidateGeometry(geometry GeoJSON) error
}