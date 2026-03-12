package model

import "time"

// OSINTResource represents an OSINT tool or data source
type OSINTResource struct {
	ID           int       `json:"id"`
	Platform     string    `json:"platform"`      // web, api, dataset, tool
	Category     string    `json:"category"`      // transport, conflict, aviation, maritime, etc.
	DisplayName  string    `json:"display_name"`  // Human-readable name
	ProfileURL   string    `json:"profile_url"`   // URL to resource
	Description  string    `json:"description"`   // Brief description
	Credibility  string    `json:"credibility"`  // verified_osint, community, official
	IsBuiltin    bool      `json:"is_builtin"`   // Pre-loaded in SENTINEL
	LastUpdated  time.Time `json:"last_updated"`
	CreatedAt    time.Time `json:"created_at"`
	Tags         []string  `json:"tags"`         // Search tags
	APIKeyRequired bool    `json:"api_key_required"`
	FreeTier      bool     `json:"free_tier"`
	Notes         string   `json:"notes"`        // Additional notes
}

// OSINTResourceInput is used for creating/updating resources
type OSINTResourceInput struct {
	Platform     string   `json:"platform"`
	Category     string   `json:"category"`
	DisplayName  string   `json:"display_name"`
	ProfileURL   string   `json:"profile_url"`
	Description  string   `json:"description"`
	Credibility  string   `json:"credibility"`
	IsBuiltin    bool     `json:"is_builtin"`
	Tags         []string `json:"tags"`
	APIKeyRequired bool   `json:"api_key_required"`
	FreeTier      bool    `json:"free_tier"`
	Notes         string  `json:"notes"`
}

// Credibility levels for OSINT resources
const (
	CredibilityVerifiedOSINT = "verified_osint"
	CredibilityOfficial      = "official"
	CredibilityCommunity     = "community"
	CredibilityExperimental  = "experimental"
)

// Platform types
const (
	PlatformWeb     = "web"
	PlatformAPI     = "api"
	PlatformDataset = "dataset"
	PlatformTool    = "tool"
	PlatformRSS     = "rss"
	PlatformMap     = "map"
)

// Category types
const (
	CategoryTransport    = "transport"
	CategoryConflict     = "conflict"
	CategoryAviation     = "aviation"
	CategoryMaritime     = "maritime"
	CategoryWeather      = "weather"
	CategoryDisaster     = "disaster"
	CategoryFinancial    = "financial"
	CategorySocial       = "social"
	CategorySatellite    = "satellite"
	CategoryInfrastructure = "infrastructure"
	CategoryGovernment   = "government"
	CategoryResearch     = "research"
	CategorySocialMediaOSINT = "social_media_osint"
)