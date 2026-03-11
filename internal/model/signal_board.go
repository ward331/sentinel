package model

// SignalBoard represents the threat-level dashboard across domains.
type SignalBoard struct {
	Military           int    `json:"military"`
	Cyber              int    `json:"cyber"`
	Financial          int    `json:"financial"`
	Natural            int    `json:"natural"`
	Health             int    `json:"health"`
	CalculatedAt       string `json:"calculated_at"`
	ActiveAlerts       int    `json:"active_alerts"`
	ActiveCorrelations int    `json:"active_correlations"`
}
