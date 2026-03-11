package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFinancialMarketsProvider_Name(t *testing.T) {
	p := NewFinancialMarketsProvider(&Config{Enabled: true})
	if p.Name() != "financial_markets" {
		t.Errorf("expected 'financial_markets', got %q", p.Name())
	}
}

func TestFinancialMarketsProvider_Enabled(t *testing.T) {
	p := NewFinancialMarketsProvider(&Config{Enabled: true})
	if !p.Enabled() {
		t.Error("expected enabled=true")
	}

	p2 := NewFinancialMarketsProvider(&Config{Enabled: false})
	if p2.Enabled() {
		t.Error("expected enabled=false")
	}

	p3 := NewFinancialMarketsProvider(nil)
	if !p3.Enabled() {
		t.Error("expected enabled=true with nil config (default)")
	}
}

func TestFinancialMarketsProvider_VIXParsing(t *testing.T) {
	// Simulate Alpha Vantage VIX response
	vixData := map[string]interface{}{
		"Meta Data": map[string]string{
			"1. Information": "Daily Time Series",
			"2. Symbol":      "VIX",
		},
		"Time Series (Daily)": map[string]interface{}{
			"2026-03-10": map[string]interface{}{
				"1. open":   "32.50",
				"2. high":   "35.20",
				"3. low":    "31.80",
				"4. close":  "34.75",
				"5. volume": "0",
			},
			"2026-03-09": map[string]interface{}{
				"1. open":   "28.00",
				"2. high":   "33.10",
				"3. low":    "27.50",
				"4. close":  "32.50",
				"5. volume": "0",
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(vixData)
	}))
	defer server.Close()

	p := NewFinancialMarketsProvider(&Config{Enabled: true})

	// We can't easily override the URL in fetchVIXData, so test parseVIXData directly
	events, err := p.parseVIXData(vixData)
	if err != nil {
		t.Fatalf("parseVIXData failed: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 VIX event, got %d", len(events))
	}

	ev := events[0]
	if ev.Source != "financial_vix" {
		t.Errorf("expected source 'financial_vix', got %q", ev.Source)
	}
	if ev.Category != "financial" {
		t.Errorf("expected category 'financial', got %q", ev.Category)
	}
	// VIX 34.75 -> high severity (>= 30)
	if ev.Severity != "high" {
		t.Errorf("expected severity 'high' for VIX 34.75, got %q", ev.Severity)
	}
}

func TestFinancialMarketsProvider_VIXSeverity(t *testing.T) {
	p := NewFinancialMarketsProvider(&Config{Enabled: true})

	tests := []struct {
		vix      float64
		severity string
	}{
		{45.0, "critical"},
		{35.0, "high"},
		{22.0, "medium"},
		{15.0, "low"},
		{8.0, "low"},
	}

	for _, tc := range tests {
		got := p.determineVIXSeverity(tc.vix)
		if string(got) != tc.severity {
			t.Errorf("VIX %.1f: expected severity %q, got %q", tc.vix, tc.severity, got)
		}
	}
}

func TestFinancialMarketsProvider_VIXMagnitude(t *testing.T) {
	p := NewFinancialMarketsProvider(&Config{Enabled: true})

	tests := []struct {
		vix           float64
		expectedMag   float64
	}{
		{45.0, 8.0},
		{35.0, 6.5},
		{22.0, 5.0},
		{15.0, 3.5},
		{12.0, 2.5},
		{8.0, 1.5},
	}

	for _, tc := range tests {
		got := p.calculateVIXMagnitude(tc.vix)
		if got != tc.expectedMag {
			t.Errorf("VIX %.1f: expected magnitude %.1f, got %.1f", tc.vix, tc.expectedMag, got)
		}
	}
}

func TestFinancialMarketsProvider_Fetch_NoAPIServer(t *testing.T) {
	// When the API is unreachable, Fetch should fall back to sample data
	p := NewFinancialMarketsProvider(&Config{Enabled: true})

	events, err := p.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	// Should return sample events even if API is down
	if len(events) == 0 {
		t.Error("expected fallback sample events, got 0")
	}
}

func TestFinancialMarketsProvider_SampleEvents(t *testing.T) {
	p := NewFinancialMarketsProvider(&Config{Enabled: true})

	// Test sample generators directly
	vixEvents := p.generateSampleVIXEvents()
	if len(vixEvents) == 0 {
		t.Error("expected sample VIX events")
	}

	oilEvents := p.generateSampleOilEvents()
	if len(oilEvents) == 0 {
		t.Error("expected sample oil events")
	}

	cryptoEvents := p.generateSampleCryptoEvents()
	if len(cryptoEvents) == 0 {
		t.Error("expected sample crypto events")
	}

	treasuryEvents := p.generateSampleTreasuryEvents()
	if len(treasuryEvents) == 0 {
		t.Error("expected sample treasury events")
	}
}

func TestFinancialMarketsProvider_VIXTitle(t *testing.T) {
	p := NewFinancialMarketsProvider(&Config{Enabled: true})

	title := p.generateVIXTitle(45.0)
	if title == "" {
		t.Error("expected non-empty title")
	}

	title = p.generateVIXTitle(12.0)
	if title == "" {
		t.Error("expected non-empty title for low VIX")
	}
}
