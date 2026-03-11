package engine

import (
	"time"
)

// Domain represents a threat domain tracked on the signal board.
type Domain string

const (
	DomainMilitary  Domain = "military"
	DomainCyber     Domain = "cyber"
	DomainFinancial Domain = "financial"
	DomainNatural   Domain = "natural"
	DomainHealth    Domain = "health"
)

// SignalBoardEntry is a point-in-time snapshot of all domain threat levels.
type SignalBoardEntry struct {
	ID           int64     `json:"id"`
	Military     int       `json:"military"`  // 0-5
	Cyber        int       `json:"cyber"`     // 0-5
	Financial    int       `json:"financial"` // 0-5
	Natural      int       `json:"natural"`   // 0-5
	Health       int       `json:"health"`    // 0-5
	CalculatedAt time.Time `json:"calculated_at"`
}

// SignalBoard calculates domain threat levels from recent event data.
type SignalBoard struct {
	enabled bool
}

// NewSignalBoard creates a new signal board calculator.
func NewSignalBoard(enabled bool) *SignalBoard {
	return &SignalBoard{enabled: enabled}
}

// Calculate produces a new SignalBoardEntry based on current event data.
// Placeholder — full implementation in G3.
func (sb *SignalBoard) Calculate() (*SignalBoardEntry, error) {
	if !sb.enabled {
		return nil, nil
	}
	// TODO: aggregate event severity per category, map to domain levels
	return &SignalBoardEntry{
		CalculatedAt: time.Now().UTC(),
	}, nil
}
