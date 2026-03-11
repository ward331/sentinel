package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

// FinancialOverviewResponse represents financial market indicators.
// Pointer fields allow null in JSON when data is unavailable.
type FinancialOverviewResponse struct {
	VIX           *float64  `json:"vix"`
	BTCUSD        *float64  `json:"btc_usd"`
	ETHUSD        *float64  `json:"eth_usd"`
	OilWTI        *float64  `json:"oil_wti"`
	Gold          *float64  `json:"gold"`
	Yield10Y      *float64  `json:"yield_10y"`
	Yield2Y       *float64  `json:"yield_2y"`
	CurveInverted *bool     `json:"curve_inverted"`
	FearGreed     *int      `json:"fear_greed"`
	Timestamp     time.Time `json:"timestamp"`
}

// GetFinancialOverview handles GET /api/financial/overview
func (h *Handler) GetFinancialOverview(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	response := FinancialOverviewResponse{
		Timestamp: time.Now().UTC(),
	}

	// Query recent financial-category events (last 24h)
	events, err := h.storage.GetRecentEventsByCategory("financial", 24, 200)
	if err == nil && len(events) > 0 {
		// Build a source → most-recent-event metadata map.
		// Events are returned newest-first, so first match per source wins.
		sourceMetadata := make(map[string]map[string]string)
		for _, ev := range events {
			if _, seen := sourceMetadata[ev.Source]; !seen && len(ev.Metadata) > 0 {
				sourceMetadata[ev.Source] = ev.Metadata
			}
		}

		// VIX data (source "financial_vix", metadata key "value")
		if meta, ok := sourceMetadata["financial_vix"]; ok {
			if v, err := strconv.ParseFloat(meta["value"], 64); err == nil {
				response.VIX = &v
			}
		}

		// Oil data (source "financial_oil", metadata key "price")
		if meta, ok := sourceMetadata["financial_oil"]; ok {
			if v, err := strconv.ParseFloat(meta["price"], 64); err == nil {
				response.OilWTI = &v
			}
		}

		// Crypto data (source "financial_crypto", metadata key "price")
		if meta, ok := sourceMetadata["financial_crypto"]; ok {
			if v, err := strconv.ParseFloat(meta["price"], 64); err == nil {
				// Determine asset from metadata symbol field
				switch meta["symbol"] {
				case "ETH":
					response.ETHUSD = &v
				default: // BTC or unspecified
					response.BTCUSD = &v
				}
			}
		}

		// Treasury data (source "financial_treasury")
		if meta, ok := sourceMetadata["financial_treasury"]; ok {
			if v, err := strconv.ParseFloat(meta["yield"], 64); err == nil {
				response.Yield10Y = &v
			}
			if v, err := strconv.ParseFloat(meta["yield_2yr"], 64); err == nil {
				response.Yield2Y = &v
			}
			if meta["inverted"] == "true" {
				inv := true
				response.CurveInverted = &inv
			} else if meta["inverted"] == "false" {
				inv := false
				response.CurveInverted = &inv
			}
		}

		// Gold data (source "financial_gold", metadata key "price")
		if meta, ok := sourceMetadata["financial_gold"]; ok {
			if v, err := strconv.ParseFloat(meta["price"], 64); err == nil {
				response.Gold = &v
			}
		}

		// Use the most recent event's occurred_at as the response timestamp
		response.Timestamp = events[0].OccurredAt
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	if h.metrics != nil {
		h.metrics.RecordAPIRequest("/api/financial/overview", time.Since(startTime))
	}
}
