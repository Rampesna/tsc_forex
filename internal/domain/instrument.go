package domain

import "time"

// InstrumentKind classifies a tradable instrument.
type InstrumentKind string

const (
	KindCurrency InstrumentKind = "currency"
	KindGold     InstrumentKind = "gold"
)

// Base market symbols emitted by providers (raw feed legs).
const (
	SymbolEURTRY = "EURTRY"
	SymbolGBPTRY = "GBPTRY"
	SymbolJPYTRY = "JPYTRY"
	// SymbolXAUUSD and SymbolUSDTRY are declared in gold.go.
)

// Gold coin/product fineness (millesimal) used across the Turkish market.
const (
	Fineness24K = 0.995 // "has" / gram gold
	Fineness22K = 0.916 // coins & bilezik
	Fineness18K = 0.750
	Fineness14K = 0.585
)

// Instrument describes a computed, user-facing tradable value in TRY.
type Instrument struct {
	Code        string         // stable id, e.g. "GRAM_GOLD", "USDTRY", "CEYREK_GOLD"
	DisplayName string         // human name (TR)
	Kind        InstrumentKind // currency | gold
	Base        string         // provider base symbol for currency (e.g. USDTRY); "" for derived gold
	GramWeight  float64        // gold: pure gram weight of the product
	Fineness    float64        // gold: millesimal fineness
	Unit        string         // display unit
}

// Catalog is the full set of instruments the platform tracks.
// Weights follow common Turkish market conventions.
var Catalog = []Instrument{
	// --- Currencies (vs TRY) ---
	{Code: "USDTRY", DisplayName: "Amerikan Doları", Kind: KindCurrency, Base: SymbolUSDTRY, Unit: "₺"},
	{Code: "EURTRY", DisplayName: "Euro", Kind: KindCurrency, Base: SymbolEURTRY, Unit: "₺"},
	{Code: "GBPTRY", DisplayName: "İngiliz Sterlini", Kind: KindCurrency, Base: SymbolGBPTRY, Unit: "₺"},
	{Code: "JPYTRY", DisplayName: "Japon Yeni", Kind: KindCurrency, Base: SymbolJPYTRY, Unit: "₺"},

	// --- Gold products (derived from XAUUSD + USDTRY) ---
	// ONS: 1 troy ounce in USD → shown in TRY here for consistency.
	{Code: "ONS_GOLD", DisplayName: "Ons Altın", Kind: KindGold, GramWeight: TroyOunceToGram, Fineness: 1.0, Unit: "₺"},
	{Code: "GRAM_GOLD", DisplayName: "Gram Altın (Has)", Kind: KindGold, GramWeight: 1.0, Fineness: Fineness24K, Unit: "₺"},
	{Code: "CEYREK_GOLD", DisplayName: "Çeyrek Altın", Kind: KindGold, GramWeight: 1.754, Fineness: Fineness22K, Unit: "₺"},
	{Code: "YARIM_GOLD", DisplayName: "Yarım Altın", Kind: KindGold, GramWeight: 3.508, Fineness: Fineness22K, Unit: "₺"},
	{Code: "TAM_GOLD", DisplayName: "Tam Altın", Kind: KindGold, GramWeight: 7.016, Fineness: Fineness22K, Unit: "₺"},
	{Code: "CUMHURIYET_GOLD", DisplayName: "Cumhuriyet Altını", Kind: KindGold, GramWeight: 7.216, Fineness: Fineness22K, Unit: "₺"},
	{Code: "ATA_GOLD", DisplayName: "Ata Altın", Kind: KindGold, GramWeight: 7.216, Fineness: Fineness22K, Unit: "₺"},
	{Code: "GRAM_22K", DisplayName: "22 Ayar Bilezik", Kind: KindGold, GramWeight: 1.0, Fineness: Fineness22K, Unit: "₺"},
	{Code: "GRAM_18K", DisplayName: "18 Ayar Altın", Kind: KindGold, GramWeight: 1.0, Fineness: Fineness18K, Unit: "₺"},
	{Code: "GRAM_14K", DisplayName: "14 Ayar Altın", Kind: KindGold, GramWeight: 1.0, Fineness: Fineness14K, Unit: "₺"},
}

// CatalogByCode indexes the catalog for O(1) lookup.
var CatalogByCode = func() map[string]Instrument {
	m := make(map[string]Instrument, len(Catalog))
	for _, in := range Catalog {
		m[in.Code] = in
	}
	return m
}()

// BaseSymbols returns the provider feed symbols required to compute the catalog.
func BaseSymbols() []string {
	return []string{SymbolXAUUSD, SymbolUSDTRY, SymbolEURTRY, SymbolGBPTRY, SymbolJPYTRY}
}

// InstrumentPrice is a computed, user-facing price for a single instrument.
type InstrumentPrice struct {
	Code        string    `json:"code"`
	DisplayName string    `json:"display_name"`
	Kind        string    `json:"kind"`
	Price       float64   `json:"price"`
	Bid         float64   `json:"bid"`
	Ask         float64   `json:"ask"`
	Change      float64   `json:"change"`         // absolute change vs previous
	ChangePct   float64   `json:"change_pct"`     // percent change vs previous
	Provider    string    `json:"provider"`
	Timestamp   time.Time `json:"timestamp"`
}

// ComputeInstrumentPrice derives a TRY price for the instrument given base legs.
//   - Currency: uses its TRY rate directly.
//   - Gold: (XAUUSD / troy) * USDTRY * gramWeight * fineness.
func ComputeInstrumentPrice(in Instrument, xauusd, usdtry float64, currencyTRY map[string]float64) (price, bid, ask float64, ok bool) {
	switch in.Kind {
	case KindCurrency:
		p := currencyTRY[in.Base]
		if p <= 0 {
			return 0, 0, 0, false
		}
		spread := p * 0.0005
		return p, p - spread/2, p + spread/2, true
	case KindGold:
		if xauusd <= 0 || usdtry <= 0 {
			return 0, 0, 0, false
		}
		gramHas := (xauusd / TroyOunceToGram) * usdtry
		p := gramHas * in.GramWeight * in.Fineness
		if p <= 0 {
			return 0, 0, 0, false
		}
		spread := p * 0.001 // ~10 bps synthetic retail spread
		return p, p - spread/2, p + spread/2, true
	default:
		return 0, 0, 0, false
	}
}
