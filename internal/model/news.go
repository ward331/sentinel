package model

// NewsItem represents an ingested news article or RSS entry.
type NewsItem struct {
	ID             int64   `json:"id"`
	Title          string  `json:"title"`
	URL            string  `json:"url"`
	Description    string  `json:"description,omitempty"`
	SourceName     string  `json:"source_name"`
	SourceCategory string  `json:"source_category"`
	PubDate        string  `json:"pub_date"`
	IngestedAt     string  `json:"ingested_at"`
	RelevanceScore int     `json:"relevance_score"`
	Lat            float64 `json:"lat,omitempty"`
	Lon            float64 `json:"lon,omitempty"`
	MatchedEventID int64   `json:"matched_event_id,omitempty"`
	TruthScore     int     `json:"truth_score"`
}
