package engine

import (
	"testing"
	"time"
)

func TestCap5(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{0, 0},
		{1, 1},
		{5, 5},
		{6, 5},
		{100, 5},
		{-1, 0},
		{-100, 0},
	}
	for _, tc := range tests {
		got := cap5(tc.input)
		if got != tc.expected {
			t.Errorf("cap5(%d) = %d, want %d", tc.input, got, tc.expected)
		}
	}
}

func TestSignalBoard_Disabled(t *testing.T) {
	sb := NewSignalBoard(false)
	entry, err := sb.Calculate()
	if err != nil {
		t.Fatalf("Calculate failed: %v", err)
	}
	if entry != nil {
		t.Errorf("expected nil entry when disabled, got %+v", entry)
	}
}

func TestSignalBoard_NilStore(t *testing.T) {
	sb := NewSignalBoard(true)
	// No storage set — should return zeroed entry
	entry, err := sb.Calculate()
	if err != nil {
		t.Fatalf("Calculate failed: %v", err)
	}
	if entry == nil {
		t.Fatal("expected non-nil entry with nil store")
	}
	if entry.Military != 0 || entry.Cyber != 0 || entry.Financial != 0 || entry.Natural != 0 || entry.Health != 0 {
		t.Errorf("expected all zeros with nil store, got MIL=%d CYB=%d FIN=%d NAT=%d HLT=%d",
			entry.Military, entry.Cyber, entry.Financial, entry.Natural, entry.Health)
	}
}

func TestSignalBoard_EmptyDB(t *testing.T) {
	s := newTestStore(t)
	sb := NewSignalBoard(true)
	sb.SetStorage(s)

	entry, err := sb.Calculate()
	if err != nil {
		t.Fatalf("Calculate failed: %v", err)
	}
	if entry == nil {
		t.Fatal("expected non-nil entry")
	}
	// With no events, all domain scores should be 0
	if entry.Military != 0 {
		t.Errorf("expected military=0, got %d", entry.Military)
	}
	if entry.Cyber != 0 {
		t.Errorf("expected cyber=0, got %d", entry.Cyber)
	}
	if entry.Financial != 0 {
		t.Errorf("expected financial=0, got %d", entry.Financial)
	}
	if entry.Natural != 0 {
		t.Errorf("expected natural=0, got %d", entry.Natural)
	}
	if entry.Health != 0 {
		t.Errorf("expected health=0, got %d", entry.Health)
	}
}

func TestSignalBoard_MilitaryConflict(t *testing.T) {
	s := newTestStore(t)
	sb := NewSignalBoard(true)
	sb.SetStorage(s)

	now := time.Now().UTC()
	// Insert a critical conflict event
	e := newTestModelEvent("acled", "conflict", 40.0, -74.0, now.Add(-1*time.Hour))
	e.Severity = "critical"
	if err := s.StoreEvent(nil, e); err != nil {
		t.Fatalf("store event: %v", err)
	}

	entry, err := sb.Calculate()
	if err != nil {
		t.Fatalf("Calculate failed: %v", err)
	}
	if entry.Military < 1 {
		t.Errorf("expected military >= 1 with critical conflict event, got %d", entry.Military)
	}
}

func TestSignalBoard_NaturalEarthquake(t *testing.T) {
	s := newTestStore(t)
	sb := NewSignalBoard(true)
	sb.SetStorage(s)

	now := time.Now().UTC()
	// Insert a M7.5 earthquake
	e := newTestModelEvent("usgs", "earthquake", 35.0, 139.0, now.Add(-2*time.Hour))
	e.Magnitude = 7.5
	if err := s.StoreEvent(nil, e); err != nil {
		t.Fatalf("store event: %v", err)
	}

	entry, err := sb.Calculate()
	if err != nil {
		t.Fatalf("Calculate failed: %v", err)
	}
	if entry.Natural < 1 {
		t.Errorf("expected natural >= 1 with M7.5 earthquake, got %d", entry.Natural)
	}
}

func TestSignalBoard_HealthWHO(t *testing.T) {
	s := newTestStore(t)
	sb := NewSignalBoard(true)
	sb.SetStorage(s)

	now := time.Now().UTC()
	// Insert a WHO event
	e := newTestModelEvent("who", "health", 0.0, 0.0, now.Add(-1*24*time.Hour))
	e.Severity = "warning"
	e.Location.Coordinates = []float64{0.01, 0.01} // avoid 0,0 filter
	if err := s.StoreEvent(nil, e); err != nil {
		t.Fatalf("store event: %v", err)
	}

	entry, err := sb.Calculate()
	if err != nil {
		t.Fatalf("Calculate failed: %v", err)
	}
	if entry.Health < 1 {
		t.Errorf("expected health >= 1 with WHO warning event, got %d", entry.Health)
	}
}

func TestSignalBoard_FinancialVIX(t *testing.T) {
	s := newTestStore(t)
	sb := NewSignalBoard(true)
	sb.SetStorage(s)

	now := time.Now().UTC()
	// Insert a financial event with high VIX
	e := newTestModelEvent("financial_vix", "financial", 40.7, -74.0, now.Add(-2*time.Hour))
	e.Metadata = map[string]string{
		"vix": "40.5",
	}
	if err := s.StoreEvent(nil, e); err != nil {
		t.Fatalf("store event: %v", err)
	}

	entry, err := sb.Calculate()
	if err != nil {
		t.Fatalf("Calculate failed: %v", err)
	}
	if entry.Financial < 1 {
		t.Errorf("expected financial >= 1 with VIX > 35, got %d", entry.Financial)
	}
}

func TestSignalBoard_CyberCISA(t *testing.T) {
	s := newTestStore(t)
	sb := NewSignalBoard(true)
	sb.SetStorage(s)

	now := time.Now().UTC()
	// Insert a CISA KEV event
	e := newTestModelEvent("cisa_kev", "cyber", 38.9, -77.0, now.Add(-6*time.Hour))
	if err := s.StoreEvent(nil, e); err != nil {
		t.Fatalf("store event: %v", err)
	}

	entry, err := sb.Calculate()
	if err != nil {
		t.Fatalf("Calculate failed: %v", err)
	}
	if entry.Cyber < 1 {
		t.Errorf("expected cyber >= 1 with CISA KEV event, got %d", entry.Cyber)
	}
}

func TestSignalBoard_ScoreCapping(t *testing.T) {
	s := newTestStore(t)
	sb := NewSignalBoard(true)
	sb.SetStorage(s)

	now := time.Now().UTC()
	// Flood the database with many critical conflict events to try to exceed 5
	for i := 0; i < 20; i++ {
		e := newTestModelEvent("acled", "conflict", 40.0, -74.0, now.Add(-time.Duration(i)*time.Minute))
		e.Severity = "critical"
		if err := s.StoreEvent(nil, e); err != nil {
			t.Fatalf("store event %d: %v", i, err)
		}
	}

	entry, err := sb.Calculate()
	if err != nil {
		t.Fatalf("Calculate failed: %v", err)
	}
	if entry.Military > 5 {
		t.Errorf("military score should be capped at 5, got %d", entry.Military)
	}
}

func TestParseFloat(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
		wantErr  bool
	}{
		{"42.5", 42.5, false},
		{" 3.14 ", 3.14, false},
		{"0", 0, false},
		{"not-a-number", 0, true},
		{"", 0, true},
	}
	for _, tc := range tests {
		got, err := parseFloat(tc.input)
		if tc.wantErr && err == nil {
			t.Errorf("parseFloat(%q) expected error, got nil", tc.input)
		}
		if !tc.wantErr && err != nil {
			t.Errorf("parseFloat(%q) unexpected error: %v", tc.input, err)
		}
		if !tc.wantErr && got != tc.expected {
			t.Errorf("parseFloat(%q) = %f, want %f", tc.input, got, tc.expected)
		}
	}
}
