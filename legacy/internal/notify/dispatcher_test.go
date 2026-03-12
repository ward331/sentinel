package notify

import (
	"context"
	"sync"
	"testing"
	"time"
)

// mockChannel is a test double for Channel.
type mockChannel struct {
	name    string
	enabled bool
	sent    []Alert
	mu      sync.Mutex
	sendErr error
}

func (m *mockChannel) Name() string   { return m.name }
func (m *mockChannel) Enabled() bool  { return m.enabled }
func (m *mockChannel) Test(ctx context.Context) error { return nil }

func (m *mockChannel) Send(ctx context.Context, alert Alert) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sendErr != nil {
		return m.sendErr
	}
	m.sent = append(m.sent, alert)
	return nil
}

func (m *mockChannel) sentCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.sent)
}

func TestDispatcher_RoutesToEnabledChannels(t *testing.T) {
	ch1 := &mockChannel{name: "telegram", enabled: true}
	ch2 := &mockChannel{name: "email", enabled: true}

	d := NewDispatcher(ch1, ch2)
	alert := Alert{
		EventID:  1,
		Title:    "Test Alert",
		Body:     "Something happened",
		Severity: "warning",
	}

	entries := d.Dispatch(context.Background(), alert)
	if len(entries) != 2 {
		t.Errorf("expected 2 log entries, got %d", len(entries))
	}
	if ch1.sentCount() != 1 {
		t.Errorf("expected telegram to receive 1 alert, got %d", ch1.sentCount())
	}
	if ch2.sentCount() != 1 {
		t.Errorf("expected email to receive 1 alert, got %d", ch2.sentCount())
	}
}

func TestDispatcher_SkipsDisabledChannels(t *testing.T) {
	enabled := &mockChannel{name: "telegram", enabled: true}
	disabled := &mockChannel{name: "email", enabled: false}

	d := NewDispatcher(enabled, disabled)
	alert := Alert{EventID: 1, Title: "Test"}

	entries := d.Dispatch(context.Background(), alert)
	if len(entries) != 1 {
		t.Errorf("expected 1 log entry (disabled skipped), got %d", len(entries))
	}
	if enabled.sentCount() != 1 {
		t.Errorf("expected telegram to receive alert")
	}
	if disabled.sentCount() != 0 {
		t.Errorf("expected disabled email to receive 0 alerts, got %d", disabled.sentCount())
	}
}

func TestDispatcher_RateLimiting(t *testing.T) {
	ch := &mockChannel{name: "telegram", enabled: true}
	d := NewDispatcher(ch)

	alert := Alert{EventID: 42, Title: "Earthquake"}

	// First dispatch should succeed
	entries1 := d.Dispatch(context.Background(), alert)
	if len(entries1) != 1 {
		t.Errorf("expected 1 log entry on first dispatch, got %d", len(entries1))
	}

	// Second dispatch with same event ID within 5 minutes should be rate-limited
	entries2 := d.Dispatch(context.Background(), alert)
	if len(entries2) != 0 {
		t.Errorf("expected 0 log entries (rate-limited), got %d", len(entries2))
	}
	if ch.sentCount() != 1 {
		t.Errorf("expected only 1 send (rate-limited second), got %d", ch.sentCount())
	}
}

func TestDispatcher_DifferentEventsNotRateLimited(t *testing.T) {
	ch := &mockChannel{name: "telegram", enabled: true}
	d := NewDispatcher(ch)

	alert1 := Alert{EventID: 1, Title: "Event 1"}
	alert2 := Alert{EventID: 2, Title: "Event 2"}

	d.Dispatch(context.Background(), alert1)
	d.Dispatch(context.Background(), alert2)

	if ch.sentCount() != 2 {
		t.Errorf("different events should not be rate-limited, expected 2 sends, got %d", ch.sentCount())
	}
}

func TestDispatcher_ChannelNames(t *testing.T) {
	ch1 := &mockChannel{name: "telegram", enabled: true}
	ch2 := &mockChannel{name: "email", enabled: false}
	ch3 := &mockChannel{name: "discord", enabled: true}

	d := NewDispatcher(ch1, ch2, ch3)

	names := d.ChannelNames()
	if len(names) != 3 {
		t.Errorf("expected 3 channel names, got %d", len(names))
	}

	enabledNames := d.EnabledChannelNames()
	if len(enabledNames) != 2 {
		t.Errorf("expected 2 enabled channel names, got %d", len(enabledNames))
	}
}

func TestDispatcher_Validate(t *testing.T) {
	// No enabled channels
	d1 := NewDispatcher(&mockChannel{name: "x", enabled: false})
	if err := d1.Validate(); err == nil {
		t.Error("expected validation error with no enabled channels")
	}

	// At least one enabled
	d2 := NewDispatcher(&mockChannel{name: "x", enabled: true})
	if err := d2.Validate(); err != nil {
		t.Errorf("unexpected validation error: %v", err)
	}
}

func TestDispatcher_AddChannel(t *testing.T) {
	d := NewDispatcher()
	if len(d.ChannelNames()) != 0 {
		t.Error("expected 0 channels initially")
	}

	d.AddChannel(&mockChannel{name: "telegram", enabled: true})
	if len(d.ChannelNames()) != 1 {
		t.Error("expected 1 channel after AddChannel")
	}
}

func TestSeverityEmoji(t *testing.T) {
	tests := []struct {
		severity string
		wantNon  string // just ensure non-empty
	}{
		{"info", ""},
		{"watch", ""},
		{"warning", ""},
		{"alert", ""},
		{"critical", ""},
		{"unknown", ""},
	}
	for _, tc := range tests {
		got := SeverityEmoji(tc.severity)
		if got == "" {
			t.Errorf("SeverityEmoji(%q) returned empty string", tc.severity)
		}
	}
}

func TestSeverityColor(t *testing.T) {
	tests := []struct {
		severity string
		want     int
	}{
		{"info", 0x00cc00},
		{"watch", 0xffcc00},
		{"warning", 0xff8800},
		{"alert", 0xff0000},
		{"critical", 0x990000},
		{"unknown", 0x888888},
	}
	for _, tc := range tests {
		got := SeverityColor(tc.severity)
		if got != tc.want {
			t.Errorf("SeverityColor(%q) = 0x%06x, want 0x%06x", tc.severity, got, tc.want)
		}
	}
}

func TestDispatcher_CleanRateCache(t *testing.T) {
	ch := &mockChannel{name: "test", enabled: true}
	d := NewDispatcher(ch)
	d.rateWindow = 1 * time.Millisecond // very short window for testing

	alert := Alert{EventID: 1, Title: "Test"}
	d.Dispatch(context.Background(), alert)

	// Wait for rate window to expire
	time.Sleep(5 * time.Millisecond)

	// Clean + dispatch again should work
	entries := d.Dispatch(context.Background(), alert)
	if len(entries) != 1 {
		t.Errorf("expected send to succeed after rate window expired, got %d entries", len(entries))
	}
}
