package model

// EntitySearchResult holds the outcome of an entity lookup across data sources.
type EntitySearchResult struct {
	Query      string                 `json:"query"`
	EntityType string                 `json:"entity_type"`
	Results    map[string]interface{} `json:"results"`
}
