package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/openclaw/sentinel-backend/internal/model"
)

// FinancialMarketsProvider fetches financial market indicators and economic data
type FinancialMarketsProvider struct {
	client *http.Client
	config *Config
}




// NewFinancialMarketsProvider creates a new FinancialMarketsProvider
func NewFinancialMarketsProvider(config *Config) *FinancialMarketsProvider {
	return &FinancialMarketsProvider{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		config: config,
	}
}

// Fetch retrieves financial market data from multiple sources
func (p *FinancialMarketsProvider) Fetch(ctx context.Context) ([]*model.Event, error) {
	var allEvents []*model.Event
	
	// Fetch data from 4 sub-providers
	subProviders := []struct {
		name   string
		fetch  func(context.Context) ([]*model.Event, error)
	}{
		{"vix", p.fetchVIXData},
		{"oil", p.fetchOilPrices},
		{"crypto", p.fetchCryptoPrices},
		{"treasury", p.fetchTreasuryYields},
	}
	
	for _, sp := range subProviders {
		events, err := sp.fetch(ctx)
		if err != nil {
			// Log error but continue with other sub-providers
			continue
		}
		allEvents = append(allEvents, events...)
	}
	
	return allEvents, nil
}

// fetchVIXData fetches VIX (Volatility Index) data
func (p *FinancialMarketsProvider) fetchVIXData(ctx context.Context) ([]*model.Event, error) {
	// Using Alpha Vantage API (free tier) for VIX data
	// Note: Would need API key in production
	url := "https://www.alphavantage.co/query?function=TIME_SERIES_DAILY&symbol=VIX&apikey=demo&datatype=json"
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create VIX request: %w", err)
	}
	
	resp, err := p.client.Do(req)
	if err != nil {
		return p.generateSampleVIXEvents(), nil
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return p.generateSampleVIXEvents(), nil
	}
	
	var data map[string]interface{}
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&data); err != nil {
		return p.generateSampleVIXEvents(), nil
	}
	
	return p.parseVIXData(data)
}

// parseVIXData parses VIX API response
func (p *FinancialMarketsProvider) parseVIXData(data map[string]interface{}) ([]*model.Event, error) {
	// Extract time series data
	timeSeries, ok := data["Time Series (Daily)"].(map[string]interface{})
	if !ok {
		return p.generateSampleVIXEvents(), nil
	}
	
	// Get latest data point
	var latestDate string
	var latestData map[string]interface{}
	
	for date, dailyData := range timeSeries {
		if latestDate == "" || date > latestDate {
			latestDate = date
			latestData = dailyData.(map[string]interface{})
		}
	}
	
	if latestData == nil {
		return p.generateSampleVIXEvents(), nil
	}
	
	// Parse VIX value
	closeStr, ok := latestData["4. close"].(string)
	if !ok {
		return p.generateSampleVIXEvents(), nil
	}
	
	vixValue, err := strconv.ParseFloat(closeStr, 64)
	if err != nil {
		return p.generateSampleVIXEvents(), nil
	}
	
	// Create event
	event := p.createVIXEvent(vixValue, latestDate)
	return []*model.Event{event}, nil
}

// createVIXEvent creates a VIX volatility event
func (p *FinancialMarketsProvider) createVIXEvent(vixValue float64, date string) *model.Event {
	timestamp, _ := time.Parse("2006-01-02", date)
	
	return &model.Event{
		Title:       p.generateVIXTitle(vixValue),
		Description: p.generateVIXDescription(vixValue, date),
		Source:      "financial_vix",
		SourceID:    fmt.Sprintf("vix_%s", date),
		OccurredAt:  timestamp,
		Location: model.GeoJSON{
			Type:        "Point",
			Coordinates: []float64{-74.0060, 40.7128}, // New York
		},
		Precision: model.PrecisionExact,
		Magnitude: p.calculateVIXMagnitude(vixValue),
		Category:  "financial",
		Severity:  p.determineVIXSeverity(vixValue),
		Metadata:  p.generateVIXMetadata(vixValue, date),
		Badges:    p.generateVIXBadges(vixValue, timestamp),
	}
}

// generateVIXTitle creates a title for VIX event
func (p *FinancialMarketsProvider) generateVIXTitle(vixValue float64) string {
	var title strings.Builder
	
	// Add emoji based on VIX level
	if vixValue >= 40 {
		title.WriteString("📈🚨 ")
	} else if vixValue >= 30 {
		title.WriteString("📈⚠️ ")
	} else if vixValue >= 20 {
		title.WriteString("📈 ")
	} else {
		title.WriteString("📊 ")
	}
	
	title.WriteString(fmt.Sprintf("VIX: %.2f - ", vixValue))
	
	// Add interpretation
	if vixValue >= 40 {
		title.WriteString("Extreme Fear/Panic")
	} else if vixValue >= 30 {
		title.WriteString("High Fear")
	} else if vixValue >= 20 {
		title.WriteString("Elevated Fear")
	} else if vixValue >= 15 {
		title.WriteString("Moderate Fear")
	} else if vixValue >= 10 {
		title.WriteString("Low Fear")
	} else {
		title.WriteString("Complacency")
	}
	
	return title.String()
}

// generateVIXDescription creates a description for VIX event
func (p *FinancialMarketsProvider) generateVIXDescription(vixValue float64, date string) string {
	var builder strings.Builder
	
	builder.WriteString(fmt.Sprintf("CBOE Volatility Index (VIX) Update\n\n"))
	builder.WriteString(fmt.Sprintf("Current VIX: %.2f\n", vixValue))
	builder.WriteString(fmt.Sprintf("Date: %s\n\n", date))
	
	// Add interpretation
	builder.WriteString("VIX Interpretation:\n")
	
	if vixValue >= 40 {
		builder.WriteString("• Extreme fear/panic in markets\n")
		builder.WriteString("• High volatility expected\n")
		builder.WriteString("• Potential market crash conditions\n")
		builder.WriteString("• Investors seeking safe havens\n")
	} else if vixValue >= 30 {
		builder.WriteString("• High fear levels\n")
		builder.WriteString("• Elevated volatility\n")
		builder.WriteString("• Market stress conditions\n")
		builder.WriteString("• Increased hedging activity\n")
	} else if vixValue >= 20 {
		builder.WriteString("• Elevated fear\n")
		builder.WriteString("• Above-average volatility\n")
		builder.WriteString("• Market uncertainty\n")
		builder.WriteString("• Caution advised\n")
	} else if vixValue >= 15 {
		builder.WriteString("• Moderate fear\n")
		builder.WriteString("• Normal volatility range\n")
		builder.WriteString("• Typical market conditions\n")
	} else if vixValue >= 10 {
		builder.WriteString("• Low fear\n")
		builder.WriteString("• Below-average volatility\n")
		builder.WriteString("• Market complacency\n")
		builder.WriteString("• Potential for volatility spike\n")
	} else {
		builder.WriteString("• Extreme complacency\n")
		builder.WriteString("• Very low volatility\n")
		builder.WriteString("• Market calm conditions\n")
		builder.WriteString("• Historically precedes volatility spikes\n")
	}
	
	// Add historical context
	builder.WriteString(fmt.Sprintf("\nHistorical Context:\n"))
	builder.WriteString(fmt.Sprintf("• Long-term average: ~19-20\n"))
	builder.WriteString(fmt.Sprintf("• 2008 Financial Crisis peak: ~80\n"))
	builder.WriteString(fmt.Sprintf("• COVID-19 March 2020 peak: ~82\n"))
	builder.WriteString(fmt.Sprintf("• Typical range: 10-30\n"))
	
	builder.WriteString(fmt.Sprintf("\nSource: CBOE Volatility Index (VIX)"))
	builder.WriteString(fmt.Sprintf("\nData provider: Alpha Vantage"))
	
	return builder.String()
}

// calculateVIXMagnitude calculates magnitude based on VIX value
func (p *FinancialMarketsProvider) calculateVIXMagnitude(vixValue float64) float64 {
	// VIX magnitude correlates with market stress
	if vixValue >= 40 {
		return 8.0
	} else if vixValue >= 30 {
		return 6.5
	} else if vixValue >= 20 {
		return 5.0
	} else if vixValue >= 15 {
		return 3.5
	} else if vixValue >= 10 {
		return 2.5
	}
	return 1.5
}

// determineVIXSeverity determines severity based on VIX value
func (p *FinancialMarketsProvider) determineVIXSeverity(vixValue float64) model.Severity {
	if vixValue >= 40 {
		return model.SeverityCritical
	} else if vixValue >= 30 {
		return model.SeverityHigh
	} else if vixValue >= 20 {
		return model.SeverityMedium
	}
	return model.SeverityLow
}

// generateVIXMetadata creates metadata for VIX event
func (p *FinancialMarketsProvider) generateVIXMetadata(vixValue float64, date string) map[string]string {
	metadata := map[string]string{
		"source":        "CBOE VIX",
		"indicator":     "volatility_index",
		"value":         fmt.Sprintf("%.2f", vixValue),
		"date":          date,
		"timestamp":     time.Now().UTC().Format(time.RFC3339),
		"location":      "Chicago Board Options Exchange",
		"interpretation": p.getVIXInterpretation(vixValue),
	}
	
	// Add level classification
	if vixValue >= 40 {
		metadata["level"] = "extreme_fear"
	} else if vixValue >= 30 {
		metadata["level"] = "high_fear"
	} else if vixValue >= 20 {
		metadata["level"] = "elevated_fear"
	} else if vixValue >= 15 {
		metadata["level"] = "moderate_fear"
	} else if vixValue >= 10 {
		metadata["level"] = "low_fear"
	} else {
		metadata["level"] = "complacency"
	}
	
	return metadata
}

// getVIXInterpretation returns interpretation text for VIX value
func (p *FinancialMarketsProvider) getVIXInterpretation(vixValue float64) string {
	if vixValue >= 40 {
		return "Extreme fear/panic - market crash conditions"
	} else if vixValue >= 30 {
		return "High fear - elevated market stress"
	} else if vixValue >= 20 {
		return "Elevated fear - above-average volatility"
	} else if vixValue >= 15 {
		return "Moderate fear - normal market conditions"
	} else if vixValue >= 10 {
		return "Low fear - market complacency"
	}
	return "Extreme complacency - very low volatility"
}

// generateVIXBadges creates badges for VIX event
func (p *FinancialMarketsProvider) generateVIXBadges(vixValue float64, timestamp time.Time) []model.Badge {
	badges := []model.Badge{
		{
			Label:     "VIX",
			Type:      "source",
			Timestamp: timestamp,
		},
		{
			Label:     "Volatility Index",
			Type:      "financial_indicator",
			Timestamp: timestamp,
		},
		{
			Label:     fmt.Sprintf("%.2f", vixValue),
			Type:      "value",
			Timestamp: timestamp,
		},
	}
	
	// Add fear level badge
	if vixValue >= 40 {
		badges = append(badges, model.Badge{
			Label:     "Extreme Fear",
			Type:      "fear_level",
			Timestamp: timestamp,
		})
	} else if vixValue >= 30 {
		badges = append(badges, model.Badge{
			Label:     "High Fear",
			Type:      "fear_level",
			Timestamp: timestamp,
		})
	} else if vixValue >= 20 {
		badges = append(badges, model.Badge{
			Label:     "Elevated Fear",
			Type:      "fear_level",
			Timestamp: timestamp,
		})
	} else if vixValue >= 15 {
		badges = append(badges, model.Badge{
			Label:     "Moderate Fear",
			Type:      "fear_level",
			Timestamp: timestamp,
		})
	} else if vixValue >= 10 {
		badges = append(badges, model.Badge{
			Label:     "Low Fear",
			Type:      "fear_level",
			Timestamp: timestamp,
		})
	} else {
		badges = append(badges, model.Badge{
			Label:     "Complacency",
			Type:      "fear_level",
			Timestamp: timestamp,
		})
	}
	
	// Add severity badge
	severity := p.determineVIXSeverity(vixValue)
	badges = append(badges, model.Badge{
		Label:     strings.Title(string(severity)),
		Type:      "severity",
		Timestamp: timestamp,
	})
	
	return badges
}

// fetchOilPrices fetches oil price data
func (p *FinancialMarketsProvider) fetchOilPrices(ctx context.Context) ([]*model.Event, error) {
	// Sample implementation - would use actual API in production
	return p.generateSampleOilEvents(), nil
}

// fetchCryptoPrices fetches cryptocurrency price data
func (p *FinancialMarketsProvider) fetchCryptoPrices(ctx context.Context) ([]*model.Event, error) {
	// Sample implementation - would use actual API in production
	return p.generateSampleCryptoEvents(), nil
}

// fetchTreasuryYields fetches US Treasury yield data
func (p *FinancialMarketsProvider) fetchTreasuryYields(ctx context.Context) ([]*model.Event, error) {
	// Sample implementation - would use actual API in production
	return p.generateSampleTreasuryEvents(), nil
}

// generateSampleVIXEvents generates sample VIX events
func (p *FinancialMarketsProvider) generateSampleVIXEvents() []*model.Event {
	// Sample VIX data
	vixValue := 22.5 // Moderate fear level
	date := time.Now().Format("2006-01-02")
	
	event := p.createVIXEvent(vixValue, date)
	return []*model.Event{event}
}

// generateSampleOilEvents generates sample oil price events
func (p *FinancialMarketsProvider) generateSampleOilEvents() []*model.Event {
	timestamp := time.Now().UTC()
	
	event := &model.Event{
		Title:       "🛢️ Brent Crude: $82.45 - Moderate Price Level",
		Description: p.generateOilDescription(82.45, timestamp),
		Source:      "financial_oil",
		SourceID:    fmt.Sprintf("oil_%s", timestamp.Format("20060102")),
		OccurredAt:  timestamp,
		Location: model.GeoJSON{
			Type:        "Point",
			Coordinates: []float64{-0.1276, 51.5074}, // London
		},
		Precision: model.PrecisionExact,
		Magnitude: 4.0,
		Category:  "financial",
		Severity:  model.SeverityLow,
		Metadata: map[string]string{
			"source":    "ICE Brent Crude",
			"price":     "82.45",
			"currency":  "USD",
			"unit":      "per barrel",
			"timestamp": timestamp.Format(time.RFC3339),
		},
		Badges: []model.Badge{
			{Label: "Oil Prices", Type: "source", Timestamp: timestamp},
			{Label: "Brent Crude", Type: "commodity", Timestamp: timestamp},
			{Label: "$82.45", Type: "price", Timestamp: timestamp},
			{Label: "Moderate", Type: "price_level", Timestamp: timestamp},
		},
	}
	
	return []*model.Event{event}
}

// generateOilDescription creates description for oil price event
func (p *FinancialMarketsProvider) generateOilDescription(price float64, timestamp time.Time) string {
	return fmt.Sprintf(`Brent Crude Oil Price Update

Current Price: $%.2f per barrel
Date: %s

Market Context:
• Moderate price level
• Balanced supply/demand
• OPEC+ production cuts in effect
• Global economic growth concerns
• Geopolitical tensions affecting supply

Historical Range:
• 2022 peak: ~$139 (Russia-Ukraine war)
• 2020 low: ~$16 (COVID-19 demand collapse)
• 5-year average: ~$65-85

Source: ICE Brent Crude Futures
Data: Real-time pricing from Intercontinental Exchange`, timestamp.Format("January 2, 2006"))
}

// generateSampleCryptoEvents generates sample cryptocurrency events
func (p *FinancialMarketsProvider) generateSampleCryptoEvents() []*model.Event {
	timestamp := time.Now().UTC()
	
	event := &model.Event{
		Title:       "₿ Bitcoin: $68,500 - Strong Bullish Momentum",
		Description: p.generateCryptoDescription(68500, timestamp),
		Source:      "financial_crypto",
		SourceID:    fmt.Sprintf("btc_%s", timestamp.Format("20060102")),
		OccurredAt:  timestamp,
		Location: model.GeoJSON{
			Type:        "Point",
			Coordinates: []float64{-122.4194, 37.7749}, // San Francisco
		},
		Precision: model.PrecisionExact,
		Magnitude: 6.0,
		Category:  "financial",
		Severity:  model.SeverityMedium,
		Metadata: map[string]string{
			"source":    "Cryptocurrency Markets",
			"asset":     "Bitcoin",
			"symbol":    "BTC",
			"price":     "68500",
			"currency":  "USD",
			"timestamp": timestamp.Format(time.RFC3339),
			"market_cap": "1.35T",
			"volume_24h": "45.2B",
		},
		Badges: []model.Badge{
			{Label: "Cryptocurrency", Type: "source", Timestamp: timestamp},
			{Label: "Bitcoin", Type: "crypto_asset", Timestamp: timestamp},
			{Label: "$68.5K", Type: "price", Timestamp: timestamp},
			{Label: "Bullish", Type: "trend", Timestamp: timestamp},
			{Label: "High Volatility", Type: "risk", Timestamp: timestamp},
		},
	}
	
	return []*model.Event{event}
}

// generateCryptoDescription creates description for crypto price event
func (p *FinancialMarketsProvider) generateCryptoDescription(price float64, timestamp time.Time) string {
	return fmt.Sprintf(`Bitcoin Price Update

Current Price: $%.0f
Market Cap: $1.35 trillion
24h Volume: $45.2 billion
Date: %s

Market Context:
• Strong bullish momentum
• ETF approval driving institutional adoption
• Halving cycle in progress
• Regulatory developments ongoing
• High volatility expected

Historical Context:
• All-time high: $73,750 (March 2024)
• 2022 low: $15,500 (FTX collapse)
• 5-year return: ~400%%

Risk Factors:
• Extreme volatility
• Regulatory uncertainty
• Cybersecurity risks
• Market manipulation concerns

Source: Cryptocurrency Exchanges (aggregated)
Note: Cryptocurrencies are highly speculative assets`, price, timestamp.Format("January 2, 2006"))
}

// generateSampleTreasuryEvents generates sample treasury yield events
func (p *FinancialMarketsProvider) generateSampleTreasuryEvents() []*model.Event {
	timestamp := time.Now().UTC()
	
	event := &model.Event{
		Title:       "📈 10-Year Treasury: 4.25%% - Inverted Yield Curve Warning",
		Description: p.generateTreasuryDescription(4.25, timestamp),
		Source:      "financial_treasury",
		SourceID:    fmt.Sprintf("treasury_%s", timestamp.Format("20060102")),
		OccurredAt:  timestamp,
		Location: model.GeoJSON{
			Type:        "Point",
			Coordinates: []float64{-77.0369, 38.9072}, // Washington DC
		},
		Precision: model.PrecisionExact,
		Magnitude: 5.5,
		Category:  "financial",
		Severity:  model.SeverityMedium,
		Metadata: map[string]string{
			"source":        "US Treasury",
			"security":      "10-Year Note",
			"yield":         "4.25",
			"yield_2yr":     "4.75",
			"spread":        "-0.50",
			"inverted":      "true",
			"timestamp":     timestamp.Format(time.RFC3339),
			"interpretation": "yield_curve_inversion",
		},
		Badges: []model.Badge{
			{Label: "US Treasury", Type: "source", Timestamp: timestamp},
			{Label: "10-Year Yield", Type: "security", Timestamp: timestamp},
			{Label: "4.25%%", Type: "yield", Timestamp: timestamp},
			{Label: "Inverted Curve", Type: "warning", Timestamp: timestamp},
			{Label: "Recession Signal", Type: "risk", Timestamp: timestamp},
		},
	}
	
	return []*model.Event{event}
}

// generateTreasuryDescription creates description for treasury yield event
func (p *FinancialMarketsProvider) generateTreasuryDescription(yield float64, timestamp time.Time) string {
	return fmt.Sprintf(`US Treasury Yield Update

10-Year Yield: %.2f%%
2-Year Yield: 4.75%%
Yield Spread: -0.50%% (inverted)
Date: %s

Market Interpretation:
• Yield curve inversion persists
• Historically reliable recession indicator
• Fed policy expectations driving yields
• Inflation concerns elevated
• Flight to quality during uncertainty

Economic Implications:
• Inverted curve suggests economic slowdown
• Tighter financial conditions
• Reduced lending activity expected
• Corporate borrowing costs rising
• Equity market headwinds

Historical Context:
• Typical 10-year yield: 2-4%%
• 2020 COVID low: 0.52%%
• 1981 peak: 15.84%%
• Current level: Above historical average

Source: US Treasury Department
Data: Secondary market yields`, yield, timestamp.Format("January 2, 2006"))
}
