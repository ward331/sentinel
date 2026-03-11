package api

import (
	"encoding/json"
	"net/http"
	"time"
)

// GetCorrelations handles GET /api/correlations
func (h *Handler) GetCorrelations(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// Query the correlations table for active (unconfirmed) flashes.
	// Falls back to empty array if DB query fails or table is empty.
	type correlationFlash struct {
		ID          int64     `json:"id"`
		RegionName  string    `json:"region_name"`
		Lat         float64   `json:"lat"`
		Lon         float64   `json:"lon"`
		RadiusKm    float64   `json:"radius_km"`
		EventCount  int       `json:"event_count"`
		SourceCount int       `json:"source_count"`
		StartedAt   time.Time `json:"started_at"`
		LastEventAt time.Time `json:"last_event_at"`
		Confirmed   bool      `json:"confirmed"`
	}

	var correlations []correlationFlash

	rows, err := h.storage.DB().QueryContext(r.Context(),
		`SELECT id, COALESCE(region_name,''), COALESCE(lat,0), COALESCE(lon,0),
		        COALESCE(radius_km,0), COALESCE(event_count,0), COALESCE(source_count,0),
		        COALESCE(started_at, CURRENT_TIMESTAMP), COALESCE(last_event_at, CURRENT_TIMESTAMP),
		        COALESCE(confirmed,0)
		 FROM correlations
		 ORDER BY last_event_at DESC
		 LIMIT 50`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var c correlationFlash
			var confirmed int
			if err := rows.Scan(&c.ID, &c.RegionName, &c.Lat, &c.Lon, &c.RadiusKm,
				&c.EventCount, &c.SourceCount, &c.StartedAt, &c.LastEventAt, &confirmed); err != nil {
				continue
			}
			c.Confirmed = confirmed != 0
			correlations = append(correlations, c)
		}
	}

	if correlations == nil {
		correlations = []correlationFlash{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"correlations": correlations,
		"total":        len(correlations),
	})

	if h.metrics != nil {
		h.metrics.RecordAPIRequest("/api/correlations", time.Since(startTime))
	}
}
