package storage

import (
	"context"
	"time"

	"github.com/openclaw/sentinel-backend/internal/intel"
	"github.com/openclaw/sentinel-backend/internal/model"
)

// NewsStoreAdapter wraps Storage to implement intel.NewsStore.
type NewsStoreAdapter struct {
	s *Storage
}

// NewNewsStoreAdapter returns an adapter that satisfies intel.NewsStore.
func NewNewsStoreAdapter(s *Storage) *NewsStoreAdapter {
	return &NewsStoreAdapter{s: s}
}

// HasNewsURL delegates to Storage.HasNewsURL.
func (a *NewsStoreAdapter) HasNewsURL(ctx context.Context, url string) (bool, error) {
	return a.s.HasNewsURL(ctx, url)
}

// StoreNewsItem converts an intel.NewsItem to model.NewsItem and stores it.
func (a *NewsStoreAdapter) StoreNewsItem(ctx context.Context, item *intel.NewsItem) error {
	mi := &model.NewsItem{
		Title:          item.Title,
		URL:            item.URL,
		Description:    item.Description,
		SourceName:     item.SourceName,
		SourceCategory: item.SourceCategory,
		RelevanceScore: item.RelevanceScore,
		TruthScore:     item.TruthScore,
		IngestedAt:     item.IngestedAt.Format(time.RFC3339),
	}
	if item.PubDate != nil {
		mi.PubDate = item.PubDate.Format(time.RFC3339)
	}
	if item.Lat != nil {
		mi.Lat = *item.Lat
	}
	if item.Lon != nil {
		mi.Lon = *item.Lon
	}
	if item.MatchedEventID != nil {
		mi.MatchedEventID = *item.MatchedEventID
	}
	return a.s.InsertNewsItem(mi)
}
