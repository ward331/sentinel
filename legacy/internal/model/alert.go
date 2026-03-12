package model

import "encoding/json"

// AlertRule defines a user-configured alert trigger.
type AlertRule struct {
	ID         int64           `json:"id"`
	Name       string          `json:"name"`
	Conditions json.RawMessage `json:"conditions"`
	Actions    json.RawMessage `json:"actions"`
	Enabled    bool            `json:"enabled"`
	CreatedAt  string          `json:"created_at"`
}
