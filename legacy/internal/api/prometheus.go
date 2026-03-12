package api

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

// PrometheusMetrics handles GET /metrics
// Outputs Prometheus text exposition format (no external dependencies).
func (h *Handler) PrometheusMetrics(w http.ResponseWriter, r *http.Request) {
	var b strings.Builder

	// --- sentinel_events_total ---
	eventsTotal, err := h.storage.CountAllEvents()
	if err == nil {
		b.WriteString("# HELP sentinel_events_total Total events in database.\n")
		b.WriteString("# TYPE sentinel_events_total counter\n")
		b.WriteString(fmt.Sprintf("sentinel_events_total %d\n", eventsTotal))
	}

	// --- sentinel_events_by_category ---
	catCounts, err := h.storage.CountEventsByCategory()
	if err == nil {
		b.WriteString("# HELP sentinel_events_by_category Event count per category.\n")
		b.WriteString("# TYPE sentinel_events_by_category gauge\n")
		for _, cat := range sortedKeys(catCounts) {
			b.WriteString(fmt.Sprintf("sentinel_events_by_category{category=%q} %d\n", cat, catCounts[cat]))
		}
	}

	// --- sentinel_events_by_severity ---
	sevCounts, err := h.storage.CountEventsBySeverity()
	if err == nil {
		b.WriteString("# HELP sentinel_events_by_severity Event count per severity.\n")
		b.WriteString("# TYPE sentinel_events_by_severity gauge\n")
		for _, sev := range sortedKeys(sevCounts) {
			b.WriteString(fmt.Sprintf("sentinel_events_by_severity{severity=%q} %d\n", sev, sevCounts[sev]))
		}
	}

	// --- sentinel_providers_total / sentinel_providers_healthy ---
	if h.poller != nil {
		providerNames := h.poller.GetProviderNames()
		b.WriteString("# HELP sentinel_providers_total Number of registered providers.\n")
		b.WriteString("# TYPE sentinel_providers_total gauge\n")
		b.WriteString(fmt.Sprintf("sentinel_providers_total %d\n", len(providerNames)))

		healthyCount := 0
		if h.healthReporter != nil {
			healthyCount = len(h.healthReporter.GetHealthyProviders())
		} else {
			// If no health reporter, assume all providers are healthy
			healthyCount = len(providerNames)
		}
		b.WriteString("# HELP sentinel_providers_healthy Number of healthy providers.\n")
		b.WriteString("# TYPE sentinel_providers_healthy gauge\n")
		b.WriteString(fmt.Sprintf("sentinel_providers_healthy %d\n", healthyCount))
	} else {
		b.WriteString("# HELP sentinel_providers_total Number of registered providers.\n")
		b.WriteString("# TYPE sentinel_providers_total gauge\n")
		b.WriteString("sentinel_providers_total 0\n")
		b.WriteString("# HELP sentinel_providers_healthy Number of healthy providers.\n")
		b.WriteString("# TYPE sentinel_providers_healthy gauge\n")
		b.WriteString("sentinel_providers_healthy 0\n")
	}

	// --- sentinel_signal_board_level ---
	sbRow, err := h.storage.GetLatestSignalBoard(r.Context())
	if err == nil && sbRow != nil {
		b.WriteString("# HELP sentinel_signal_board_level Current signal board threat level per domain.\n")
		b.WriteString("# TYPE sentinel_signal_board_level gauge\n")
		b.WriteString(fmt.Sprintf("sentinel_signal_board_level{domain=\"military\"} %d\n", sbRow.Military))
		b.WriteString(fmt.Sprintf("sentinel_signal_board_level{domain=\"cyber\"} %d\n", sbRow.Cyber))
		b.WriteString(fmt.Sprintf("sentinel_signal_board_level{domain=\"financial\"} %d\n", sbRow.Financial))
		b.WriteString(fmt.Sprintf("sentinel_signal_board_level{domain=\"natural\"} %d\n", sbRow.Natural))
		b.WriteString(fmt.Sprintf("sentinel_signal_board_level{domain=\"health\"} %d\n", sbRow.Health))
	} else {
		b.WriteString("# HELP sentinel_signal_board_level Current signal board threat level per domain.\n")
		b.WriteString("# TYPE sentinel_signal_board_level gauge\n")
		for _, domain := range []string{"military", "cyber", "financial", "natural", "health"} {
			b.WriteString(fmt.Sprintf("sentinel_signal_board_level{domain=%q} 0\n", domain))
		}
	}

	// --- sentinel_correlations_total ---
	corrCount, err := h.storage.CountCorrelations()
	if err == nil {
		b.WriteString("# HELP sentinel_correlations_total Total correlation flashes.\n")
		b.WriteString("# TYPE sentinel_correlations_total counter\n")
		b.WriteString(fmt.Sprintf("sentinel_correlations_total %d\n", corrCount))
	}

	// --- sentinel_anomalies_total ---
	anomCount, err := h.storage.CountAnomalies()
	if err == nil {
		b.WriteString("# HELP sentinel_anomalies_total Total anomalies detected.\n")
		b.WriteString("# TYPE sentinel_anomalies_total counter\n")
		b.WriteString(fmt.Sprintf("sentinel_anomalies_total %d\n", anomCount))
	}

	// --- sentinel_uptime_seconds ---
	uptime := time.Since(h.startTime).Seconds()
	b.WriteString("# HELP sentinel_uptime_seconds Server uptime in seconds.\n")
	b.WriteString("# TYPE sentinel_uptime_seconds gauge\n")
	b.WriteString(fmt.Sprintf("sentinel_uptime_seconds %.2f\n", uptime))

	// --- sentinel_news_items_total ---
	newsCount, err := h.storage.CountNewsItems()
	if err == nil {
		b.WriteString("# HELP sentinel_news_items_total Total news items.\n")
		b.WriteString("# TYPE sentinel_news_items_total counter\n")
		b.WriteString(fmt.Sprintf("sentinel_news_items_total %d\n", newsCount))
	}

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, b.String())
}

// sortedKeys returns the keys of a map[string]int64 in sorted order.
func sortedKeys(m map[string]int64) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
