package engine

import (
	"time"
)

// TruthConfirmation records a cross-source confirmation of an event.
type TruthConfirmation struct {
	ID                int64     `json:"id"`
	PrimaryEventID    int64     `json:"primary_event_id"`
	ConfirmingSource  string    `json:"confirming_source"`
	ConfirmingEventID int64     `json:"confirming_event_id"`
	ConfirmedAt       time.Time `json:"confirmed_at"`
}

// TruthScoreCalculator computes a 1–5 truth score based on cross-source confirmation.
//
//	1 = single source only
//	2 = two independent sources
//	3 = three+ sources agree
//	4 = confirmed by an official/authoritative source
//	5 = confirmed by multiple authoritative sources
type TruthScoreCalculator struct{}

// NewTruthScoreCalculator creates a new calculator.
func NewTruthScoreCalculator() *TruthScoreCalculator {
	return &TruthScoreCalculator{}
}

// Score returns the truth score for a given event.
// Placeholder — full implementation in G3.
func (t *TruthScoreCalculator) Score(eventID string) (int, error) {
	// TODO: count confirmations, check source authority, return 1-5
	return 1, nil
}
