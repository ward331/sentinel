package model

// FinancialOverview holds market snapshot data.
type FinancialOverview struct {
	VIX           float64 `json:"vix"`
	VIXChange     float64 `json:"vix_change"`
	BTCUSD        float64 `json:"btc_usd"`
	BTCChange24h  float64 `json:"btc_change_24h"`
	ETHUSD        float64 `json:"eth_usd"`
	ETHChange24h  float64 `json:"eth_change_24h"`
	OilWTI        float64 `json:"oil_wti"`
	OilChange     float64 `json:"oil_change"`
	Gold          float64 `json:"gold"`
	GoldChange    float64 `json:"gold_change"`
	Yield10Y      float64 `json:"yield_10y"`
	Yield2Y       float64 `json:"yield_2y"`
	CurveInverted bool    `json:"curve_inverted"`
	FearGreed     int     `json:"fear_greed"`
	UpdatedAt     string  `json:"updated_at"`
}
