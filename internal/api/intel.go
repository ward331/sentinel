package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/openclaw/sentinel-backend/internal/model"
)

// briefingSection holds one named section of the briefing.
type briefingSection struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// intelBriefingResponse is the structured response for /api/intel/briefing.
type intelBriefingResponse struct {
	Content     string            `json:"content"`
	Sections    []briefingSection `json:"sections"`
	GeneratedAt time.Time         `json:"generated_at"`
	Type        string            `json:"type"`
	EventCount  int               `json:"event_count"`
	WindowHours int               `json:"window_hours"`
}

// GetIntelBriefing handles GET /api/intel/briefing
// Generates a real template-based intelligence briefing from live data.
func (h *Handler) GetIntelBriefing(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	ctx := r.Context()
	now := time.Now().UTC()
	windowHours := 6

	var sections []briefingSection
	var fullText strings.Builder
	fullText.WriteString(fmt.Sprintf("SENTINEL INTELLIGENCE BRIEFING — %s UTC\n", now.Format("2006-01-02 15:04")))
	fullText.WriteString(strings.Repeat("=", 55) + "\n\n")

	// --- 1. Gather recent events (last 6 hours) ---
	events, err := h.storage.GetRecentEvents(ctx, windowHours*60)
	if err != nil {
		log.Printf("[intel/briefing] failed to fetch recent events: %v", err)
	}

	// --- 2. SITUATION OVERVIEW ---
	{
		var sb strings.Builder
		catCounts := map[string]int{}
		sevCounts := map[string]int{}
		for _, e := range events {
			cat := e.Category
			if cat == "" {
				cat = "other"
			}
			catCounts[cat]++
			sev := e.Severity
			if sev == "" {
				sev = "info"
			}
			sevCounts[sev]++
		}

		sb.WriteString(fmt.Sprintf("Reporting window: last %d hours (%s to %s UTC)\n",
			windowHours,
			now.Add(-time.Duration(windowHours)*time.Hour).Format("15:04"),
			now.Format("15:04")))
		sb.WriteString(fmt.Sprintf("Total events ingested: %d\n\n", len(events)))

		if len(events) == 0 {
			sb.WriteString("No events recorded in the reporting window.\n")
		} else {
			// Category breakdown
			sb.WriteString("Events by category:\n")
			cats := sortedMapKeys(catCounts)
			for _, c := range cats {
				sb.WriteString(fmt.Sprintf("  %-20s %d\n", strings.ToUpper(c), catCounts[c]))
			}

			// Severity distribution
			sb.WriteString("\nSeverity distribution:\n")
			for _, sev := range []string{"critical", "high", "medium", "low", "info"} {
				if n, ok := sevCounts[sev]; ok {
					sb.WriteString(fmt.Sprintf("  %-12s %d\n", strings.ToUpper(sev), n))
				}
			}
			// Include any non-standard severities
			for sev, n := range sevCounts {
				switch sev {
				case "critical", "high", "medium", "low", "info":
				default:
					sb.WriteString(fmt.Sprintf("  %-12s %d\n", strings.ToUpper(sev), n))
				}
			}
		}
		section := briefingSection{Title: "SITUATION OVERVIEW", Content: sb.String()}
		sections = append(sections, section)
		fullText.WriteString("1. SITUATION OVERVIEW\n")
		fullText.WriteString(strings.Repeat("-", 40) + "\n")
		fullText.WriteString(sb.String() + "\n")
	}

	// --- 3. SIGNAL BOARD STATUS ---
	{
		var sb strings.Builder
		sbRow, err := h.storage.GetLatestSignalBoard(ctx)
		if err != nil {
			log.Printf("[intel/briefing] failed to fetch signal board: %v", err)
		}
		if sbRow != nil {
			sb.WriteString(fmt.Sprintf("As of %s UTC:\n", sbRow.CalculatedAt.Format("15:04")))
			sb.WriteString(fmt.Sprintf("  Military:   %s (level %d/5)\n", threatLabel(sbRow.Military), sbRow.Military))
			sb.WriteString(fmt.Sprintf("  Cyber:      %s (level %d/5)\n", threatLabel(sbRow.Cyber), sbRow.Cyber))
			sb.WriteString(fmt.Sprintf("  Financial:  %s (level %d/5)\n", threatLabel(sbRow.Financial), sbRow.Financial))
			sb.WriteString(fmt.Sprintf("  Natural:    %s (level %d/5)\n", threatLabel(sbRow.Natural), sbRow.Natural))
			sb.WriteString(fmt.Sprintf("  Health:     %s (level %d/5)\n", threatLabel(sbRow.Health), sbRow.Health))
		} else {
			sb.WriteString("No signal board data available. Defaulting to NOMINAL across all domains.\n")
		}
		section := briefingSection{Title: "SIGNAL BOARD STATUS", Content: sb.String()}
		sections = append(sections, section)
		fullText.WriteString("2. SIGNAL BOARD STATUS\n")
		fullText.WriteString(strings.Repeat("-", 40) + "\n")
		fullText.WriteString(sb.String() + "\n")
	}

	// --- 4. ACTIVE CORRELATIONS ---
	{
		var sb strings.Builder
		corrs, err := h.storage.GetRecentCorrelations(ctx, windowHours*60)
		if err != nil {
			log.Printf("[intel/briefing] failed to fetch correlations: %v", err)
		}
		if len(corrs) == 0 {
			sb.WriteString("No active correlation flashes in the reporting window.\n")
		} else {
			sb.WriteString(fmt.Sprintf("%d correlation flash(es) detected:\n", len(corrs)))
			for i, c := range corrs {
				if i >= 10 {
					sb.WriteString(fmt.Sprintf("  ... and %d more\n", len(corrs)-10))
					break
				}
				region := c.RegionName
				if region == "" {
					region = fmt.Sprintf("%.2f, %.2f", c.Lat, c.Lon)
				}
				sb.WriteString(fmt.Sprintf("  [%d] %s — %d events from %d sources (radius %.0f km)\n",
					i+1, region, c.EventCount, c.SourceCount, c.RadiusKm))
			}
		}
		section := briefingSection{Title: "ACTIVE CORRELATIONS", Content: sb.String()}
		sections = append(sections, section)
		fullText.WriteString("3. ACTIVE CORRELATIONS\n")
		fullText.WriteString(strings.Repeat("-", 40) + "\n")
		fullText.WriteString(sb.String() + "\n")
	}

	// --- 5. KEY EVENTS (top 5 highest severity) ---
	{
		var sb strings.Builder
		if len(events) == 0 {
			sb.WriteString("No events to highlight.\n")
		} else {
			// Sort copy by severity desc, then magnitude desc
			type rankedEvent struct {
				Title      string
				Category   string
				Severity   string
				Source     string
				Magnitude  float64
				OccurredAt time.Time
			}
			ranked := make([]rankedEvent, len(events))
			for i, e := range events {
				ranked[i] = rankedEvent{
					Title:      e.Title,
					Category:   e.Category,
					Severity:   e.Severity,
					Source:     e.Source,
					Magnitude:  e.Magnitude,
					OccurredAt: e.OccurredAt,
				}
			}
			sort.Slice(ranked, func(i, j int) bool {
				si := briefingSeverityRank(ranked[i].Severity)
				sj := briefingSeverityRank(ranked[j].Severity)
				if si != sj {
					return si > sj
				}
				return ranked[i].Magnitude > ranked[j].Magnitude
			})
			limit := 5
			if len(ranked) < limit {
				limit = len(ranked)
			}
			for i := 0; i < limit; i++ {
				e := ranked[i]
				sev := e.Severity
				if sev == "" {
					sev = "info"
				}
				line := fmt.Sprintf("  %d. [%s] %s", i+1, strings.ToUpper(sev), e.Title)
				if e.Magnitude > 0 {
					line += fmt.Sprintf(" (M%.1f)", e.Magnitude)
				}
				line += fmt.Sprintf(" — %s via %s", e.OccurredAt.Format("15:04 UTC"), e.Source)
				sb.WriteString(line + "\n")
			}
			if len(events) > 5 {
				sb.WriteString(fmt.Sprintf("\n  (%d additional events not shown)\n", len(events)-5))
			}
		}
		section := briefingSection{Title: "KEY EVENTS", Content: sb.String()}
		sections = append(sections, section)
		fullText.WriteString("4. KEY EVENTS\n")
		fullText.WriteString(strings.Repeat("-", 40) + "\n")
		fullText.WriteString(sb.String() + "\n")
	}

	// --- 6. ASSESSMENT ---
	{
		var sb strings.Builder
		if len(events) == 0 {
			sb.WriteString("All monitored domains are quiet. No actionable intelligence at this time.\n")
		} else {
			var critHigh int
			for _, e := range events {
				sev := strings.ToLower(e.Severity)
				if sev == "critical" || sev == "high" {
					critHigh++
				}
			}
			if critHigh > 0 {
				sb.WriteString(fmt.Sprintf("ELEVATED POSTURE: %d critical/high-severity event(s) detected in the last %d hours. ",
					critHigh, windowHours))
				sb.WriteString("Recommend increased monitoring of affected domains.\n")
			} else {
				sb.WriteString(fmt.Sprintf("NORMAL POSTURE: %d event(s) recorded, none at critical or high severity. ",
					len(events)))
				sb.WriteString("Situation is stable across all monitored domains.\n")
			}
		}
		section := briefingSection{Title: "ASSESSMENT", Content: sb.String()}
		sections = append(sections, section)
		fullText.WriteString("5. ASSESSMENT\n")
		fullText.WriteString(strings.Repeat("-", 40) + "\n")
		fullText.WriteString(sb.String() + "\n")
	}

	fullText.WriteString("— End of briefing —\n")

	response := intelBriefingResponse{
		Content:     fullText.String(),
		Sections:    sections,
		GeneratedAt: now,
		Type:        "morning",
		EventCount:  len(events),
		WindowHours: windowHours,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	if h.metrics != nil {
		h.metrics.RecordAPIRequest("/api/intel/briefing", time.Since(startTime))
	}
}

// threatLabel returns a human-readable label for a signal board threat level (0-5).
func threatLabel(level int) string {
	switch {
	case level <= 0:
		return "NOMINAL"
	case level == 1:
		return "GUARDED"
	case level == 2:
		return "ELEVATED"
	case level == 3:
		return "HIGH"
	case level == 4:
		return "SEVERE"
	default:
		return "CRITICAL"
	}
}

// briefingSeverityRank returns a numeric rank for sorting (higher = more severe).
func briefingSeverityRank(s string) int {
	switch strings.ToLower(s) {
	case "critical":
		return 5
	case "high":
		return 4
	case "medium", "warning":
		return 3
	case "low", "watch":
		return 2
	case "info":
		return 1
	default:
		return 0
	}
}

// sortedMapKeys returns map keys sorted alphabetically.
func sortedMapKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// GetNews handles GET /api/news
func (h *Handler) GetNews(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 500 {
			limit = v
		}
	}

	items, err := h.storage.GetRecentNews(limit)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to query news: %v"}`, err), http.StatusInternalServerError)
		if h.metrics != nil {
			h.metrics.RecordAPIError("/api/news")
		}
		return
	}
	if items == nil {
		items = []model.NewsItem{}
	}

	response := map[string]interface{}{
		"items": items,
		"total": len(items),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	if h.metrics != nil {
		h.metrics.RecordAPIRequest("/api/news", time.Since(startTime))
	}
}
