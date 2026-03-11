package model

// ProviderCatalog describes a data provider's current state and metadata.
type ProviderCatalog struct {
	Name           string `json:"name"`
	DisplayName    string `json:"display_name"`
	Category       string `json:"category"`
	Tier           int    `json:"tier"`
	Enabled        bool   `json:"enabled"`
	Status         string `json:"status"`
	LastFetch      string `json:"last_fetch,omitempty"`
	EventsLastHour int    `json:"events_last_hour"`
	ErrorStreak    int    `json:"error_streak"`
	KeyFile        string `json:"key_file,omitempty"`
	SignupURL      string `json:"signup_url,omitempty"`
}
