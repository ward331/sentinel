package intel

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// Briefing is an AI-generated intelligence summary.
type Briefing struct {
	ID                int64     `json:"id"`
	Content           string    `json:"content"`
	GeneratedAt       time.Time `json:"generated_at"`
	DeliveredChannels string    `json:"delivered_channels,omitempty"`
}

// EventSummary is a lightweight event representation used for briefing generation.
type EventSummary struct {
	Title      string    `json:"title"`
	Category   string    `json:"category"`
	Severity   string    `json:"severity"`
	Source     string    `json:"source"`
	OccurredAt time.Time `json:"occurred_at"`
	Magnitude  float64   `json:"magnitude,omitempty"`
}

// EventFetcher retrieves recent events for briefing generation.
// Implementations should query the storage layer.
type EventFetcher interface {
	RecentEvents(ctx context.Context, since time.Duration) ([]EventSummary, error)
}

// BriefingGenerator produces situational awareness briefings using an LLM or templates.
type BriefingGenerator struct {
	llmEndpoint  string
	eventFetcher EventFetcher
	client       *http.Client

	// Cache: briefing valid for 30 minutes
	mu          sync.Mutex
	cachedBrief *Briefing
	cacheExpiry time.Time
}

// NewBriefingGenerator creates a new generator.
// llmEndpoint should be the Governor LLM address, e.g. "http://127.0.0.1:18890".
func NewBriefingGenerator(llmEndpoint string, fetcher EventFetcher) *BriefingGenerator {
	return &BriefingGenerator{
		llmEndpoint:  strings.TrimRight(llmEndpoint, "/"),
		eventFetcher: fetcher,
		client:       &http.Client{Timeout: 30 * time.Second},
	}
}

// GenerateBriefing creates a situational awareness briefing from events in the last 6 hours.
// Results are cached for 30 minutes.
func (g *BriefingGenerator) GenerateBriefing(ctx context.Context) (*Briefing, error) {
	g.mu.Lock()
	if g.cachedBrief != nil && time.Now().Before(g.cacheExpiry) {
		b := g.cachedBrief
		g.mu.Unlock()
		return b, nil
	}
	g.mu.Unlock()

	events, err := g.fetchEvents(ctx)
	if err != nil {
		return nil, fmt.Errorf("briefing: fetch events: %w", err)
	}

	var content string
	if g.llmEndpoint != "" {
		content, err = g.generateWithLLM(ctx, events)
		if err != nil {
			log.Printf("[briefing] LLM generation failed, falling back to template: %v", err)
			content = g.generateTemplate(events)
		}
	} else {
		content = g.generateTemplate(events)
	}

	briefing := &Briefing{
		Content:     content,
		GeneratedAt: time.Now().UTC(),
	}

	g.mu.Lock()
	g.cachedBrief = briefing
	g.cacheExpiry = time.Now().Add(30 * time.Minute)
	g.mu.Unlock()

	return briefing, nil
}

// GenerateMorning creates a morning briefing (alias for GenerateBriefing).
func (g *BriefingGenerator) GenerateMorning(ctx context.Context) (*Briefing, error) {
	return g.GenerateBriefing(ctx)
}

// GenerateWeekly creates a weekly digest placeholder.
func (g *BriefingGenerator) GenerateWeekly(ctx context.Context) (*Briefing, error) {
	// For weekly, look back 7 days
	events, err := g.fetchEventsWindow(ctx, 7*24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("weekly briefing: fetch events: %w", err)
	}
	content := g.generateTemplate(events)
	return &Briefing{
		Content:     "WEEKLY DIGEST\n\n" + content,
		GeneratedAt: time.Now().UTC(),
	}, nil
}

func (g *BriefingGenerator) fetchEvents(ctx context.Context) ([]EventSummary, error) {
	return g.fetchEventsWindow(ctx, 6*time.Hour)
}

func (g *BriefingGenerator) fetchEventsWindow(ctx context.Context, window time.Duration) ([]EventSummary, error) {
	if g.eventFetcher != nil {
		return g.eventFetcher.RecentEvents(ctx, window)
	}
	return nil, nil
}

// generateWithLLM sends an events summary to the Governor LLM for analysis.
func (g *BriefingGenerator) generateWithLLM(ctx context.Context, events []EventSummary) (string, error) {
	if len(events) == 0 {
		return "SENTINEL BRIEFING — No significant events in the last 6 hours. All clear.", nil
	}

	// Build a concise events summary for the LLM prompt.
	var sb strings.Builder
	sb.WriteString("You are SENTINEL, an AI world monitoring system. Generate a concise situational awareness briefing from these recent events. ")
	sb.WriteString("Group by category, highlight critical items first. Use plain text suitable for text-to-speech.\n\n")
	sb.WriteString("EVENTS (last 6 hours):\n")

	for _, e := range events {
		line := fmt.Sprintf("- [%s] %s | %s | %s", e.Severity, e.Title, e.Category, e.Source)
		if e.Magnitude > 0 {
			line += fmt.Sprintf(" | mag %.1f", e.Magnitude)
		}
		sb.WriteString(line + "\n")
	}

	// Build Governor-compatible chat completion request
	payload := map[string]interface{}{
		"model": "auto",
		"messages": []map[string]string{
			{"role": "user", "content": sb.String()},
		},
		"max_tokens": 1000,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal LLM request: %w", err)
	}

	url := g.llmEndpoint + "/v1/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create LLM request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("LLM request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("LLM returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode LLM response: %w", err)
	}

	if len(result.Choices) == 0 || result.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("empty LLM response")
	}

	return result.Choices[0].Message.Content, nil
}

// generateTemplate produces a structured plain-text briefing without LLM.
func (g *BriefingGenerator) generateTemplate(events []EventSummary) string {
	var sb strings.Builder
	now := time.Now().UTC()
	sb.WriteString(fmt.Sprintf("SENTINEL SITUATION BRIEFING — %s UTC\n", now.Format("2006-01-02 15:04")))
	sb.WriteString(strings.Repeat("=", 50) + "\n\n")

	if len(events) == 0 {
		sb.WriteString("No significant events in the reporting period. All clear.\n")
		return sb.String()
	}

	// Group by category
	groups := make(map[string][]EventSummary)
	for _, e := range events {
		cat := e.Category
		if cat == "" {
			cat = "other"
		}
		groups[cat] = append(groups[cat], e)
	}

	// Sort categories for deterministic output
	cats := make([]string, 0, len(groups))
	for c := range groups {
		cats = append(cats, c)
	}
	sort.Strings(cats)

	// Count critical/alert items
	var critCount int
	for _, e := range events {
		if e.Severity == "critical" || e.Severity == "alert" {
			critCount++
		}
	}

	sb.WriteString(fmt.Sprintf("Total events: %d | Critical/Alert: %d | Categories: %d\n\n",
		len(events), critCount, len(cats)))

	for _, cat := range cats {
		items := groups[cat]
		sb.WriteString(fmt.Sprintf("--- %s (%d events) ---\n", strings.ToUpper(cat), len(items)))
		// Sort by severity (critical first)
		sort.Slice(items, func(i, j int) bool {
			return severityOrder(items[i].Severity) > severityOrder(items[j].Severity)
		})
		for _, e := range items {
			line := fmt.Sprintf("  [%s] %s", strings.ToUpper(e.Severity), e.Title)
			if e.Magnitude > 0 {
				line += fmt.Sprintf(" (M%.1f)", e.Magnitude)
			}
			sb.WriteString(line + "\n")
		}
		sb.WriteString("\n")
	}

	sb.WriteString("— End of briefing —\n")
	return sb.String()
}

func severityOrder(s string) int {
	switch s {
	case "critical":
		return 5
	case "alert":
		return 4
	case "warning":
		return 3
	case "watch":
		return 2
	case "info":
		return 1
	default:
		return 0
	}
}
