package notify

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// Channel is a named notification transport.
type Channel interface {
	// Name returns the channel identifier (e.g. "telegram", "email").
	Name() string
	// Send delivers a notification. Returns an error on failure.
	Send(ctx context.Context, alert Alert) error
	// Test sends a test message to verify the channel is configured.
	Test(ctx context.Context) error
	// Enabled returns whether this channel is active.
	Enabled() bool
}

// Alert represents a notification payload.
type Alert struct {
	EventID  int64     `json:"event_id"`
	Title    string    `json:"title"`
	Body     string    `json:"body"`
	Severity string    `json:"severity"` // info, watch, warning, alert, critical
	Category string    `json:"category"`
	URL      string    `json:"url,omitempty"`
	OccurredAt time.Time `json:"occurred_at"`
}

// SeverityEmoji returns the emoji for a severity level.
func SeverityEmoji(severity string) string {
	switch severity {
	case "info":
		return "\U0001F7E2" // green circle
	case "watch":
		return "\U0001F7E1" // yellow circle
	case "warning":
		return "\U0001F7E0" // orange circle
	case "alert":
		return "\U0001F534" // red circle
	case "critical":
		return "\U0001F6A8" // rotating light
	default:
		return "\u2754" // question mark
	}
}

// SeverityColor returns a hex color integer for a severity level (for embeds).
func SeverityColor(severity string) int {
	switch severity {
	case "info":
		return 0x00cc00 // green
	case "watch":
		return 0xffcc00 // yellow
	case "warning":
		return 0xff8800 // orange
	case "alert":
		return 0xff0000 // red
	case "critical":
		return 0x990000 // dark red
	default:
		return 0x888888 // gray
	}
}

// LogEntry records a sent notification for audit.
type LogEntry struct {
	ID      int64     `json:"id"`
	Channel string    `json:"channel"`
	EventID int64     `json:"event_id"`
	SentAt  time.Time `json:"sent_at"`
	Status  string    `json:"status"` // "ok" | "error"
	Error   string    `json:"error,omitempty"`
}

// NotificationLogger persists notification log entries. Implementations may
// write to a database, file, or no-op.
type NotificationLogger interface {
	LogNotification(entry LogEntry) error
}

// noopLogger discards log entries.
type noopLogger struct{}

func (noopLogger) LogNotification(LogEntry) error { return nil }

// rateKey uniquely identifies a (channel, event) pair for rate limiting.
type rateKey struct {
	channel string
	eventID int64
}

// Dispatcher routes alerts to one or more configured channels.
type Dispatcher struct {
	channels []Channel
	logger   NotificationLogger

	// Rate limiting: max 1 notification per event per channel per 5 minutes.
	mu       sync.Mutex
	lastSent map[rateKey]time.Time
	rateWindow time.Duration
}

// NewDispatcher creates a dispatcher with the given channels.
func NewDispatcher(channels ...Channel) *Dispatcher {
	return &Dispatcher{
		channels:   channels,
		logger:     noopLogger{},
		lastSent:   make(map[rateKey]time.Time),
		rateWindow: 5 * time.Minute,
	}
}

// SetLogger configures a notification logger.
func (d *Dispatcher) SetLogger(l NotificationLogger) {
	if l != nil {
		d.logger = l
	}
}

// rateLimited checks and records whether a send should be suppressed.
func (d *Dispatcher) rateLimited(channel string, eventID int64) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	key := rateKey{channel: channel, eventID: eventID}
	if last, ok := d.lastSent[key]; ok {
		if time.Since(last) < d.rateWindow {
			return true
		}
	}
	d.lastSent[key] = time.Now()
	return false
}

// cleanRateCache removes expired entries to prevent unbounded memory growth.
func (d *Dispatcher) cleanRateCache() {
	d.mu.Lock()
	defer d.mu.Unlock()
	now := time.Now()
	for k, t := range d.lastSent {
		if now.Sub(t) > d.rateWindow {
			delete(d.lastSent, k)
		}
	}
}

// Dispatch sends an alert to every enabled, non-rate-limited channel and
// returns a log entry per channel.
func (d *Dispatcher) Dispatch(ctx context.Context, alert Alert) []LogEntry {
	// Periodic cleanup — lightweight, runs inline.
	d.cleanRateCache()

	var entries []LogEntry
	for _, ch := range d.channels {
		if !ch.Enabled() {
			continue
		}
		entry := LogEntry{
			Channel: ch.Name(),
			EventID: alert.EventID,
			SentAt:  time.Now().UTC(),
		}
		if d.rateLimited(ch.Name(), alert.EventID) {
			log.Printf("[notify] %s rate-limited for event %d, skipping", ch.Name(), alert.EventID)
			continue
		}
		if err := ch.Send(ctx, alert); err != nil {
			entry.Status = "error"
			entry.Error = err.Error()
			log.Printf("[notify] %s send failed: %v", ch.Name(), err)
		} else {
			entry.Status = "ok"
		}
		entries = append(entries, entry)
		if err := d.logger.LogNotification(entry); err != nil {
			log.Printf("[notify] failed to log notification: %v", err)
		}
	}
	return entries
}

// TestAll sends a test message to every enabled channel.
func (d *Dispatcher) TestAll(ctx context.Context) map[string]error {
	results := make(map[string]error)
	for _, ch := range d.channels {
		if !ch.Enabled() {
			continue
		}
		results[ch.Name()] = ch.Test(ctx)
	}
	return results
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

// EnabledChannelNames returns only the enabled channels.
func (d *Dispatcher) EnabledChannelNames() []string {
	var names []string
	for _, ch := range d.channels {
		if ch.Enabled() {
			names = append(names, ch.Name())
		}
	}
	return names
}

// Validate checks that at least one channel is configured and enabled.
func (d *Dispatcher) Validate() error {
	for _, ch := range d.channels {
		if ch.Enabled() {
			return nil
		}
	}
	return fmt.Errorf("no notification channels enabled")
}
