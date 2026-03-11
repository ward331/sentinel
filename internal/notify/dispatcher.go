package notify

import (
	"fmt"
	"log"
	"time"
)

// Channel is a named notification transport.
type Channel interface {
	// Name returns the channel identifier (e.g. "telegram", "email").
	Name() string
	// Send delivers a notification. Returns an error on failure.
	Send(alert Alert) error
}

// Alert represents a notification payload.
type Alert struct {
	EventID     int64     `json:"event_id"`
	Title       string    `json:"title"`
	Body        string    `json:"body"`
	Severity    string    `json:"severity"`
	URL         string    `json:"url,omitempty"`
	OccurredAt  time.Time `json:"occurred_at"`
}

// LogEntry records a sent notification for audit.
type LogEntry struct {
	ID        int64     `json:"id"`
	Channel   string    `json:"channel"`
	EventID   int64     `json:"event_id"`
	SentAt    time.Time `json:"sent_at"`
	Status    string    `json:"status"` // "ok" | "error"
	Error     string    `json:"error,omitempty"`
}

// Dispatcher routes alerts to one or more configured channels.
type Dispatcher struct {
	channels []Channel
}

// NewDispatcher creates a dispatcher with the given channels.
func NewDispatcher(channels ...Channel) *Dispatcher {
	return &Dispatcher{channels: channels}
}

// Dispatch sends an alert to every registered channel and returns a log entry per channel.
func (d *Dispatcher) Dispatch(alert Alert) []LogEntry {
	var entries []LogEntry
	for _, ch := range d.channels {
		entry := LogEntry{
			Channel: ch.Name(),
			EventID: alert.EventID,
			SentAt:  time.Now().UTC(),
		}
		if err := ch.Send(alert); err != nil {
			entry.Status = "error"
			entry.Error = err.Error()
			log.Printf("[notify] %s send failed: %v", ch.Name(), err)
		} else {
			entry.Status = "ok"
		}
		entries = append(entries, entry)
	}
	return entries
}

// AddChannel registers an additional channel at runtime.
func (d *Dispatcher) AddChannel(ch Channel) {
	d.channels = append(d.channels, ch)
}

// ChannelNames returns the names of all registered channels.
func (d *Dispatcher) ChannelNames() []string {
	names := make([]string, len(d.channels))
	for i, ch := range d.channels {
		names[i] = ch.Name()
	}
	return names
}

// Validate checks that all channels are reachable (placeholder).
func (d *Dispatcher) Validate() error {
	if len(d.channels) == 0 {
		return fmt.Errorf("no notification channels configured")
	}
	return nil
}
