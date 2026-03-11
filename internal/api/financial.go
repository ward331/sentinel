package api

import (
	"encoding/json"
	"net/http"
	"time"
)

// FinancialOverviewResponse represents financial market indicators.
type FinancialOverviewResponse struct {
	VIX           float64   `json:"vix"`
	BTCUSD        float64   `json:"btc_usd"`
	ETHUSD        float64   `json:"eth_usd"`
	OilWTI        float64   `json:"oil_wti"`
	Gold          float64   `json:"gold"`
	Yield10Y      float64   `json:"yield_10y"`
	Yield2Y       float64   `json:"yield_2y"`
	CurveInverted bool      `json:"curve_inverted"`
	FearGreed     int       `json:"fear_greed"`
	Timestamp     time.Time `json:"timestamp"`
}

// GetFinancialOverview handles GET /api/financial/overview
func (h *Handler) GetFinancialOverview(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// Placeholder financial data — will be populated by providers once wired.
	response := FinancialOverviewResponse{
		VIX:           18.5,
		BTCUSD:        67500.00,
		ETHUSD:        3450.00,
		OilWTI:        78.20,
		Gold:          2340.00,
		Yield10Y:      4.25,
		Yield2Y:       4.70,
		CurveInverted: true,
		FearGreed:     55,
		Timestamp:     time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	if h.metrics != nil {
		h.metrics.RecordAPIRequest("/api/financial/overview", time.Since(startTime))
	}
}
