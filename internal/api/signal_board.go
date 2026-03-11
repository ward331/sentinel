package api

import (
	"encoding/json"
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

	// Return placeholder signal board data.
	// The real engine.SignalBoard.Calculate() is a stub; we return sensible defaults.
	response := SignalBoardResponse{
		Military:     1,
		Cyber:        2,
		Financial:    1,
		Natural:      1,
		Health:       0,
		CalculatedAt: time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	if h.metrics != nil {
		h.metrics.RecordAPIRequest("/api/signal-board", time.Since(startTime))
	}
}
