package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// SignalBoardResponse represents domain threat levels.
type SignalBoardResponse struct {
	Military     int       `json:"military"`
	Cyber        int       `json:"cyber"`
	Financial    int       `json:"financial"`
	Natural      int       `json:"natural"`
	Health       int       `json:"health"`
	CalculatedAt time.Time `json:"calculated_at"`
}

// GetSignalBoard handles GET /api/signal-board
func (h *Handler) GetSignalBoard(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// Read the latest signal board snapshot from the database.
	row, err := h.storage.GetLatestSignalBoard(r.Context())
	var response SignalBoardResponse
	if err != nil || row == nil {
		if err != nil {
			log.Printf("[api] signal-board query error: %v", err)
		}
		// Fall back to zeroed response if no data yet
		response = SignalBoardResponse{
			CalculatedAt: time.Now().UTC(),
		}
	} else {
		response = SignalBoardResponse{
			Military:     row.Military,
			Cyber:        row.Cyber,
			Financial:    row.Financial,
			Natural:      row.Natural,
			Health:       row.Health,
			CalculatedAt: row.CalculatedAt,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	if h.metrics != nil {
		h.metrics.RecordAPIRequest("/api/signal-board", time.Since(startTime))
	}
}
