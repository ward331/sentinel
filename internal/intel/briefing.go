package intel

import (
	"time"
)

// Briefing is an AI-generated intelligence summary.
type Briefing struct {
	ID                int64     `json:"id"`
	Content           string    `json:"content"`
	GeneratedAt       time.Time `json:"generated_at"`
	DeliveredChannels string    `json:"delivered_channels,omitempty"`
}

// BriefingGenerator produces daily/weekly intelligence briefings using an LLM.
type BriefingGenerator struct {
	llmEndpoint string
}

// NewBriefingGenerator creates a new generator.
func NewBriefingGenerator(llmEndpoint string) *BriefingGenerator {
	return &BriefingGenerator{llmEndpoint: llmEndpoint}
}

// GenerateMorning creates a morning briefing from recent events.
// Stub — LLM integration in Stage G5.
func (g *BriefingGenerator) GenerateMorning() (*Briefing, error) {
	// TODO: query last 24h events, send to LLM, format response
	return &Briefing{
		Content:     "Morning briefing placeholder",
		GeneratedAt: time.Now().UTC(),
	}, nil
}

// GenerateWeekly creates a weekly digest.
// Stub — LLM integration in Stage G5.
func (g *BriefingGenerator) GenerateWeekly() (*Briefing, error) {
	return &Briefing{
		Content:     "Weekly digest placeholder",
		GeneratedAt: time.Now().UTC(),
	}, nil
}
