package intel

import (
	"time"
)

// NewsItem represents an ingested news article.
type NewsItem struct {
	ID             int64      `json:"id"`
	Title          string     `json:"title"`
	URL            string     `json:"url"`
	Description    string     `json:"description,omitempty"`
	SourceName     string     `json:"source_name,omitempty"`
	SourceCategory string     `json:"source_category,omitempty"`
	PubDate        *time.Time `json:"pub_date,omitempty"`
	IngestedAt     time.Time  `json:"ingested_at"`
	RelevanceScore int        `json:"relevance_score"`
	Lat            *float64   `json:"lat,omitempty"`
	Lon            *float64   `json:"lon,omitempty"`
	MatchedEventID *int64     `json:"matched_event_id,omitempty"`
	TruthScore     int        `json:"truth_score"`
}

// NewsAggregator ingests RSS feeds and matches articles to events.
type NewsAggregator struct {
	feeds []string
}

// NewNewsAggregator creates a new aggregator with the given RSS feed URLs.
func NewNewsAggregator(feeds []string) *NewsAggregator {
	return &NewsAggregator{feeds: feeds}
}

// Fetch retrieves and parses all configured RSS feeds.
// Stub — actual RSS parsing in Stage G4.
func (a *NewsAggregator) Fetch() ([]NewsItem, error) {
	// TODO: iterate feeds, parse XML, deduplicate by URL, geo-tag
	return nil, nil
}

// AddFeed registers an additional RSS feed URL.
func (a *NewsAggregator) AddFeed(url string) {
	a.feeds = append(a.feeds, url)
}

// FeedCount returns the number of configured feeds.
func (a *NewsAggregator) FeedCount() int {
	return len(a.feeds)
}
