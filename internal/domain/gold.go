package domain

import "time"

// TroyOunceToGram is the standard conversion factor (XAU troy ounce → grams).
const TroyOunceToGram = 31.1034768

// Symbol identifiers used across providers and the calculator.
const (
	SymbolXAUUSD      = "XAUUSD"
	SymbolUSDTRY      = "USDTRY"
	SymbolGramGoldTRY = "XAU_GRAM_TRY"
)

// GramGold is the calculated gram gold price in TRY.
type GramGold struct {
	Symbol    string
	Price     float64
	MidPrice  float64
	Spread    float64
	XAUUSD    float64
	USDTRY    float64
	Provider  string
	Timestamp time.Time
}

// CalculateGramGoldTRY applies:
//
//	GramGoldTRY = (XAUUSD / 31.1034768) * USDTRY
func CalculateGramGoldTRY(xauusd, usdtry float64) float64 {
	if xauusd <= 0 || usdtry <= 0 {
		return 0
	}
	return (xauusd / TroyOunceToGram) * usdtry
}
