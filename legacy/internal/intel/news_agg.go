package intel

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
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

// NewsStore persists news items and provides deduplication.
type NewsStore interface {
	HasNewsURL(ctx context.Context, url string) (bool, error)
	StoreNewsItem(ctx context.Context, item *NewsItem) error
}

// Feed describes an RSS/Atom feed source.
type Feed struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Category string `json:"category"`
}

// DefaultFeeds returns the built-in feed list.
func DefaultFeeds() []Feed {
	return []Feed{
		{Name: "Reuters World", URL: "https://feeds.reuters.com/Reuters/worldNews", Category: "world"},
		{Name: "BBC News", URL: "https://feeds.bbci.co.uk/news/world/rss.xml", Category: "world"},
		{Name: "Al Jazeera", URL: "https://www.aljazeera.com/xml/rss/all.xml", Category: "world"},
		{Name: "AP News", URL: "https://rsshub.app/apnews/topics/apf-topnews", Category: "world"},
	}
}

// NewsAggregator ingests RSS feeds, parses them, and stores items.
type NewsAggregator struct {
	feeds    []Feed
	store    NewsStore
	client   *http.Client
	interval time.Duration

	mu     sync.Mutex
	stopCh chan struct{}
}

// NewNewsAggregator creates a new aggregator with the given feeds.
// If feeds is nil, the default feed list is used.
func NewNewsAggregator(feeds []Feed, store NewsStore) *NewsAggregator {
	if len(feeds) == 0 {
		feeds = DefaultFeeds()
	}
	return &NewsAggregator{
		feeds:    feeds,
		store:    store,
		client:   &http.Client{Timeout: 15 * time.Second},
		interval: 15 * time.Minute,
	}
}

// SetInterval sets the polling interval.
func (a *NewsAggregator) SetInterval(d time.Duration) {
	a.interval = d
}

// AddFeed registers an additional RSS feed.
func (a *NewsAggregator) AddFeed(f Feed) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.feeds = append(a.feeds, f)
}

// FeedCount returns the number of configured feeds.
func (a *NewsAggregator) FeedCount() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.feeds)
}

// Fetch retrieves and parses all configured RSS/Atom feeds, returning new items.
func (a *NewsAggregator) Fetch(ctx context.Context) ([]NewsItem, error) {
	a.mu.Lock()
	feeds := make([]Feed, len(a.feeds))
	copy(feeds, a.feeds)
	a.mu.Unlock()

	var allItems []NewsItem
	for _, feed := range feeds {
		items, err := a.fetchFeed(ctx, feed)
		if err != nil {
			log.Printf("[news] Failed to fetch %s: %v", feed.Name, err)
			continue
		}
		allItems = append(allItems, items...)
	}
	return allItems, nil
}

// FetchAndStore fetches all feeds, deduplicates by URL, and stores new items.
func (a *NewsAggregator) FetchAndStore(ctx context.Context) (int, error) {
	items, err := a.Fetch(ctx)
	if err != nil {
		return 0, err
	}

	if a.store == nil {
		return len(items), nil
	}

	var stored int
	for i := range items {
		exists, err := a.store.HasNewsURL(ctx, items[i].URL)
		if err != nil {
			log.Printf("[news] URL check failed: %v", err)
			continue
		}
		if exists {
			continue
		}
		if err := a.store.StoreNewsItem(ctx, &items[i]); err != nil {
			log.Printf("[news] Store failed: %v", err)
			continue
		}
		stored++
	}

	log.Printf("[news] Fetched %d items, stored %d new", len(items), stored)
	return stored, nil
}

// Start begins periodic feed polling in the background.
func (a *NewsAggregator) Start(ctx context.Context) {
	a.mu.Lock()
	if a.stopCh != nil {
		a.mu.Unlock()
		return
	}
	a.stopCh = make(chan struct{})
	a.mu.Unlock()

	go func() {
		// Initial fetch
		if _, err := a.FetchAndStore(ctx); err != nil {
			log.Printf("[news] Initial fetch error: %v", err)
		}

		ticker := time.NewTicker(a.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if _, err := a.FetchAndStore(ctx); err != nil {
					log.Printf("[news] Periodic fetch error: %v", err)
				}
			case <-a.stopCh:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

// Stop halts periodic polling.
func (a *NewsAggregator) Stop() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.stopCh != nil {
		close(a.stopCh)
		a.stopCh = nil
	}
}

// fetchFeed fetches and parses a single RSS/Atom feed.
func (a *NewsAggregator) fetchFeed(ctx context.Context, feed Feed) ([]NewsItem, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feed.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "SENTINEL-NewsAggregator/1.0")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024)) // 5MB limit
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	// Try RSS 2.0 first, then Atom
	items, err := parseRSS(body, feed)
	if err != nil || len(items) == 0 {
		items2, err2 := parseAtom(body, feed)
		if err2 != nil {
			if err != nil {
				return nil, fmt.Errorf("not RSS (%v) or Atom (%v)", err, err2)
			}
			return nil, err2
		}
		items = items2
	}

	return items, nil
}

// --- RSS 2.0 parser ---

type rssRoot struct {
	XMLName xml.Name   `xml:"rss"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Items []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
}

func parseRSS(data []byte, feed Feed) ([]NewsItem, error) {
	var root rssRoot
	if err := xml.Unmarshal(data, &root); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	var items []NewsItem
	for _, ri := range root.Channel.Items {
		url := ri.Link
		if url == "" {
			url = ri.GUID
		}
		if url == "" {
			continue
		}

		item := NewsItem{
			Title:          cleanXML(ri.Title),
			URL:            url,
			Description:    truncate(cleanXML(ri.Description), 500),
			SourceName:     feed.Name,
			SourceCategory: feed.Category,
			IngestedAt:     now,
			TruthScore:     1,
		}

		if t, err := parseRSSDate(ri.PubDate); err == nil {
			item.PubDate = &t
		}

		items = append(items, item)
	}

	return items, nil
}

// --- Atom parser ---

type atomFeed struct {
	XMLName xml.Name   `xml:"feed"`
	Entries []atomEntry `xml:"entry"`
}

type atomEntry struct {
	Title   string     `xml:"title"`
	Links   []atomLink `xml:"link"`
	Summary string     `xml:"summary"`
	Content string     `xml:"content"`
	Updated string     `xml:"updated"`
	ID      string     `xml:"id"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
}

func parseAtom(data []byte, feed Feed) ([]NewsItem, error) {
	var root atomFeed
	if err := xml.Unmarshal(data, &root); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	var items []NewsItem
	for _, ae := range root.Entries {
		url := ""
		for _, l := range ae.Links {
			if l.Rel == "alternate" || l.Rel == "" {
				url = l.Href
				break
			}
		}
		if url == "" {
			url = ae.ID
		}
		if url == "" {
			continue
		}

		desc := ae.Summary
		if desc == "" {
			desc = ae.Content
		}

		item := NewsItem{
			Title:          cleanXML(ae.Title),
			URL:            url,
			Description:    truncate(cleanXML(desc), 500),
			SourceName:     feed.Name,
			SourceCategory: feed.Category,
			IngestedAt:     now,
			TruthScore:     1,
		}

		if t, err := time.Parse(time.RFC3339, ae.Updated); err == nil {
			item.PubDate = &t
		}

		items = append(items, item)
	}

	return items, nil
}

// --- Helpers ---

// parseRSSDate attempts to parse common RSS date formats.
func parseRSSDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty date")
	}
	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC3339,
		"Mon, 02 Jan 2006 15:04:05 MST",
		"Mon, 2 Jan 2006 15:04:05 MST",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized date format: %s", s)
}

func cleanXML(s string) string {
	// Strip CDATA and basic HTML tags
	s = strings.ReplaceAll(s, "<![CDATA[", "")
	s = strings.ReplaceAll(s, "]]>", "")
	// Simple tag stripping
	var result strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(r)
		}
	}
	return strings.TrimSpace(result.String())
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
