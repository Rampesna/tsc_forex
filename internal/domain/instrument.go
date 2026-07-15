package domain

import "time"

// InstrumentKind classifies a tradable instrument.
type InstrumentKind string

const (
	KindCurrency InstrumentKind = "currency"
	KindGold     InstrumentKind = "gold"
	KindCrypto   InstrumentKind = "crypto"
	KindStock    InstrumentKind = "stock"
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

// Instrument describes a computed, user-facing tradable value.
type Instrument struct {
	Code        string         // stable id, e.g. "GRAM_GOLD", "USDTRY", "CEYREK_GOLD"
	DisplayName string         // human name (TR)
	Kind        InstrumentKind // currency | gold | crypto | stock
	Base        string         // provider base symbol for currency (e.g. USDTRY); "" for derived gold
	GramWeight  float64        // gold: pure gram weight of the product
	Fineness    float64        // gold: millesimal fineness
	Unit        string         // display unit symbol (₺ or $)
	Currency    string         // native quote currency: "TRY" (default) or "USD" (global equities)
}

// CurrencyDef defines a world fiat currency quoted against TRY. The provider
// base symbol and instrument code are both "<ISO>TRY" (e.g. "USDTRY").
type CurrencyDef struct {
	ISO         string // ISO-4217 code, e.g. "USD"
	DisplayName string // Turkish name
}

// CryptoDef defines a cryptocurrency tracked against TRY. Priced from a USD
// quote (Binance "<SYM>USDT", Yahoo "<SYM>-USD" fallback) × live USD/TRY.
// Code == Base == Symbol (e.g. "BTC"). CoinGeckoID feeds the detail page
// (market cap, supply, ATH/ATL, rank).
type CryptoDef struct {
	Symbol      string // canonical short symbol, e.g. "BTC" (instrument code + base)
	DisplayName string
	YahooSymbol string // Yahoo Finance ticker, e.g. "BTC-USD"
	CoinGeckoID string // CoinGecko coin id, e.g. "bitcoin"
}

// Cryptos are the major coins the platform tracks (priced in TRY).
var Cryptos = []CryptoDef{
	{"BTC", "Bitcoin", "BTC-USD", "bitcoin"},
	{"ETH", "Ethereum", "ETH-USD", "ethereum"},
	{"BNB", "BNB", "BNB-USD", "binancecoin"},
	{"SOL", "Solana", "SOL-USD", "solana"},
	{"XRP", "XRP", "XRP-USD", "ripple"},
	{"ADA", "Cardano", "ADA-USD", "cardano"},
	{"DOGE", "Dogecoin", "DOGE-USD", "dogecoin"},
	{"TRX", "TRON", "TRX-USD", "tron"},
	{"AVAX", "Avalanche", "AVAX-USD", "avalanche-2"},
	{"DOT", "Polkadot", "DOT-USD", "polkadot"},
	{"LINK", "Chainlink", "LINK-USD", "chainlink"},
	{"MATIC", "Polygon", "MATIC-USD", "matic-network"},
	{"LTC", "Litecoin", "LTC-USD", "litecoin"},
	{"ATOM", "Cosmos", "ATOM-USD", "cosmos"},
	{"XLM", "Stellar", "XLM-USD", "stellar"},
	{"SHIB", "Shiba Inu", "SHIB-USD", "shiba-inu"},
	{"BCH", "Bitcoin Cash", "BCH-USD", "bitcoin-cash"},
}

// CryptoByCode indexes crypto defs by canonical symbol for O(1) lookup.
var CryptoByCode = func() map[string]CryptoDef {
	m := make(map[string]CryptoDef, len(Cryptos))
	for _, c := range Cryptos {
		m[c.Symbol] = c
	}
	return m
}()

// StockDef defines an equity. BIST names are quoted natively in TRY (YahooSymbol
// ends in ".IS"); global names are quoted in USD (plain Yahoo ticker). Code ==
// Base == Symbol. Currency is "TRY" or "USD".
type StockDef struct {
	Symbol      string // instrument code + base (e.g. "THYAO", "AAPL")
	DisplayName string
	YahooSymbol string // Yahoo Finance ticker (e.g. "THYAO.IS", "AAPL")
	Currency    string // "TRY" (BIST) or "USD" (global)
}

// Stocks are liquid BIST-30 names tracked in TRY.
var Stocks = []StockDef{
	{"THYAO", "Türk Hava Yolları", "THYAO.IS", "TRY"},
	{"GARAN", "Garanti BBVA", "GARAN.IS", "TRY"},
	{"AKBNK", "Akbank", "AKBNK.IS", "TRY"},
	{"ISCTR", "İş Bankası (C)", "ISCTR.IS", "TRY"},
	{"YKBNK", "Yapı Kredi", "YKBNK.IS", "TRY"},
	{"SISE", "Şişecam", "SISE.IS", "TRY"},
	{"EREGL", "Ereğli Demir Çelik", "EREGL.IS", "TRY"},
	{"KCHOL", "Koç Holding", "KCHOL.IS", "TRY"},
	{"SAHOL", "Sabancı Holding", "SAHOL.IS", "TRY"},
	{"ASELS", "Aselsan", "ASELS.IS", "TRY"},
	{"BIMAS", "BİM Mağazalar", "BIMAS.IS", "TRY"},
	{"TUPRS", "Tüpraş", "TUPRS.IS", "TRY"},
	{"FROTO", "Ford Otosan", "FROTO.IS", "TRY"},
	{"TCELL", "Turkcell", "TCELL.IS", "TRY"},
	{"PGSUS", "Pegasus", "PGSUS.IS", "TRY"},
	{"TTKOM", "Türk Telekom", "TTKOM.IS", "TRY"},
	{"TOASO", "Tofaş", "TOASO.IS", "TRY"},
	{"SASA", "Sasa Polyester", "SASA.IS", "TRY"},
}

// GlobalStocks are the world's largest USD-listed companies (US primaries and
// ADRs) by market cap. Priced natively in USD; a TRY equivalent is derived from
// live USD/TRY for display. Codes avoid punctuation (e.g. BRKB → "BRK-B").
var GlobalStocks = []StockDef{
	{"NVDA", "Nvidia", "NVDA", "USD"},
	{"AAPL", "Apple", "AAPL", "USD"},
	{"MSFT", "Microsoft", "MSFT", "USD"},
	{"GOOGL", "Alphabet (Google)", "GOOGL", "USD"},
	{"AMZN", "Amazon", "AMZN", "USD"},
	{"META", "Meta Platforms", "META", "USD"},
	{"AVGO", "Broadcom", "AVGO", "USD"},
	{"TSLA", "Tesla", "TSLA", "USD"},
	{"BRKB", "Berkshire Hathaway", "BRK-B", "USD"},
	{"TSM", "TSMC", "TSM", "USD"},
	{"LLY", "Eli Lilly", "LLY", "USD"},
	{"WMT", "Walmart", "WMT", "USD"},
	{"JPM", "JPMorgan Chase", "JPM", "USD"},
	{"V", "Visa", "V", "USD"},
	{"MA", "Mastercard", "MA", "USD"},
	{"ORCL", "Oracle", "ORCL", "USD"},
	{"NVO", "Novo Nordisk", "NVO", "USD"},
	{"XOM", "Exxon Mobil", "XOM", "USD"},
	{"UNH", "UnitedHealth", "UNH", "USD"},
	{"COST", "Costco", "COST", "USD"},
	{"HD", "Home Depot", "HD", "USD"},
	{"PG", "Procter & Gamble", "PG", "USD"},
	{"JNJ", "Johnson & Johnson", "JNJ", "USD"},
	{"NFLX", "Netflix", "NFLX", "USD"},
	{"BAC", "Bank of America", "BAC", "USD"},
	{"ABBV", "AbbVie", "ABBV", "USD"},
	{"KO", "Coca-Cola", "KO", "USD"},
	{"SAP", "SAP", "SAP", "USD"},
	{"CRM", "Salesforce", "CRM", "USD"},
	{"CVX", "Chevron", "CVX", "USD"},
	{"TMUS", "T-Mobile US", "TMUS", "USD"},
	{"WFC", "Wells Fargo", "WFC", "USD"},
	{"ASML", "ASML", "ASML", "USD"},
	{"BABA", "Alibaba", "BABA", "USD"},
	{"MRK", "Merck", "MRK", "USD"},
	{"AMD", "AMD", "AMD", "USD"},
	{"PEP", "PepsiCo", "PEP", "USD"},
	{"LIN", "Linde", "LIN", "USD"},
	{"ADBE", "Adobe", "ADBE", "USD"},
	{"TM", "Toyota", "TM", "USD"},
	{"AZN", "AstraZeneca", "AZN", "USD"},
	{"NVS", "Novartis", "NVS", "USD"},
	{"MCD", "McDonald's", "MCD", "USD"},
	{"SHEL", "Shell", "SHEL", "USD"},
	{"ABT", "Abbott", "ABT", "USD"},
	{"TMO", "Thermo Fisher", "TMO", "USD"},
	{"GE", "GE Aerospace", "GE", "USD"},
	{"CSCO", "Cisco", "CSCO", "USD"},
	{"INTC", "Intel", "INTC", "USD"},
	{"DIS", "Walt Disney", "DIS", "USD"},
}

// AllStocks returns BIST + global equities in one slice.
func AllStocks() []StockDef {
	out := make([]StockDef, 0, len(Stocks)+len(GlobalStocks))
	out = append(out, Stocks...)
	out = append(out, GlobalStocks...)
	return out
}

// StockByCode indexes every equity (BIST + global) by its instrument code, so
// callers can resolve the Yahoo ticker and quote currency.
var StockByCode = func() map[string]StockDef {
	all := AllStocks()
	m := make(map[string]StockDef, len(all))
	for _, s := range all {
		m[s.Symbol] = s
	}
	return m
}()

// Currencies is the full set of world currencies the platform tracks against
// TRY. USD/EUR/GBP/JPY lead (most traded in TR); the rest give global coverage.
// All are quoted in TRY and derived from live USD/TRY × USD-based cross rates.
var Currencies = []CurrencyDef{
	{"USD", "Amerikan Doları"}, {"EUR", "Euro"}, {"GBP", "İngiliz Sterlini"}, {"JPY", "Japon Yeni"},
	{"CHF", "İsviçre Frangı"}, {"CAD", "Kanada Doları"}, {"AUD", "Avustralya Doları"}, {"NZD", "Yeni Zelanda Doları"},
	{"CNY", "Çin Yuanı"}, {"RUB", "Rus Rublesi"}, {"SAR", "Suudi Arabistan Riyali"}, {"AED", "BAE Dirhemi"},
	{"KWD", "Kuveyt Dinarı"}, {"QAR", "Katar Riyali"}, {"BHD", "Bahreyn Dinarı"}, {"OMR", "Umman Riyali"},
	{"JOD", "Ürdün Dinarı"}, {"ILS", "İsrail Şekeli"}, {"INR", "Hindistan Rupisi"}, {"PKR", "Pakistan Rupisi"},
	{"IQD", "Irak Dinarı"}, {"EGP", "Mısır Lirası"}, {"ZAR", "Güney Afrika Randı"}, {"SEK", "İsveç Kronu"},
	{"NOK", "Norveç Kronu"}, {"DKK", "Danimarka Kronu"}, {"PLN", "Polonya Zlotisi"}, {"CZK", "Çek Korunası"},
	{"HUF", "Macar Forinti"}, {"RON", "Rumen Leyi"}, {"BGN", "Bulgar Levası"}, {"UAH", "Ukrayna Grivnası"},
	{"KRW", "Güney Kore Wonu"}, {"SGD", "Singapur Doları"}, {"HKD", "Hong Kong Doları"}, {"TWD", "Tayvan Doları"},
	{"THB", "Tayland Bahtı"}, {"MYR", "Malezya Ringiti"}, {"IDR", "Endonezya Rupiahı"}, {"PHP", "Filipin Pesosu"},
	{"VND", "Vietnam Dongu"}, {"MXN", "Meksika Pesosu"}, {"BRL", "Brezilya Reali"}, {"ARS", "Arjantin Pesosu"},
	{"CLP", "Şili Pesosu"}, {"COP", "Kolombiya Pesosu"}, {"AZN", "Azerbaycan Manatı"}, {"KZT", "Kazakistan Tengesi"},
	{"GEL", "Gürcü Larisi"}, {"MAD", "Fas Dirhemi"}, {"DZD", "Cezayir Dinarı"}, {"TND", "Tunus Dinarı"},
	{"NGN", "Nijerya Nairası"}, {"KES", "Kenya Şilini"}, {"UZS", "Özbekistan Somu"}, {"ISK", "İzlanda Kronu"},
}

// goldProducts are derived from XAUUSD + USDTRY using Turkish market weights.
func goldProducts() []Instrument {
	return []Instrument{
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
}

// Catalog is the full set of instruments the platform tracks: every world
// currency (vs TRY), Turkish gold products, crypto and BIST equities.
var Catalog = func() []Instrument {
	out := make([]Instrument, 0, len(Currencies)+len(Cryptos)+len(Stocks)+10)
	for _, c := range Currencies {
		out = append(out, Instrument{
			Code:        c.ISO + "TRY",
			DisplayName: c.DisplayName,
			Kind:        KindCurrency,
			Base:        c.ISO + "TRY",
			Unit:        "₺",
		})
	}
	out = append(out, goldProducts()...)
	for _, c := range Cryptos {
		out = append(out, Instrument{
			Code:        c.Symbol,
			DisplayName: c.DisplayName,
			Kind:        KindCrypto,
			Base:        c.Symbol,
			Unit:        "₺",
		})
	}
	for _, s := range AllStocks() {
		unit := "₺"
		ccy := s.Currency
		if ccy == "" {
			ccy = "TRY"
		}
		if ccy == "USD" {
			unit = "$"
		}
		out = append(out, Instrument{
			Code:        s.Symbol,
			DisplayName: s.DisplayName,
			Kind:        KindStock,
			Base:        s.Symbol,
			Unit:        unit,
			Currency:    ccy,
		})
	}
	return out
}()

// currencySymbolSet indexes every currency base symbol for O(1) membership.
var currencySymbolSet = func() map[string]bool {
	m := make(map[string]bool, len(Currencies))
	for _, c := range Currencies {
		m[c.ISO+"TRY"] = true
	}
	return m
}()

// IsCurrencySymbol reports whether a provider symbol is a tracked TRY currency.
func IsCurrencySymbol(sym string) bool { return currencySymbolSet[sym] }

// directTRYSymbolSet indexes every symbol whose provider tick is already a TRY
// value read straight from the map: currencies, crypto and BIST equities.
var directTRYSymbolSet = func() map[string]bool {
	all := AllStocks()
	m := make(map[string]bool, len(Currencies)+len(Cryptos)+len(all))
	for _, c := range Currencies {
		m[c.ISO+"TRY"] = true
	}
	for _, c := range Cryptos {
		m[c.Symbol] = true
	}
	// Equities (BIST in TRY, global in USD) are stored verbatim by the engine;
	// the native quote currency is carried on the instrument for display.
	for _, s := range all {
		m[s.Symbol] = true
	}
	return m
}()

// IsDirectTRYSymbol reports whether a provider symbol carries a direct TRY value
// (currency, crypto or stock) that the engine stores verbatim.
func IsDirectTRYSymbol(sym string) bool { return directTRYSymbolSet[sym] }

// CryptoSymbols returns every tracked crypto symbol (== instrument code).
func CryptoSymbols() []string {
	out := make([]string, 0, len(Cryptos))
	for _, c := range Cryptos {
		out = append(out, c.Symbol)
	}
	return out
}

// StockSymbols returns every tracked equity symbol (BIST + global).
func StockSymbols() []string {
	all := AllStocks()
	out := make([]string, 0, len(all))
	for _, s := range all {
		out = append(out, s.Symbol)
	}
	return out
}

// CurrencyBaseSymbols returns every "<ISO>TRY" feed symbol.
func CurrencyBaseSymbols() []string {
	out := make([]string, 0, len(Currencies))
	for _, c := range Currencies {
		out = append(out, c.ISO+"TRY")
	}
	return out
}

// CatalogByCode indexes the catalog for O(1) lookup.
var CatalogByCode = func() map[string]Instrument {
	m := make(map[string]Instrument, len(Catalog))
	for _, in := range Catalog {
		m[in.Code] = in
	}
	return m
}()

// BaseSymbols returns the provider feed symbols required to compute the catalog:
// gold (XAUUSD) plus every world currency vs TRY.
func BaseSymbols() []string {
	return append([]string{SymbolXAUUSD}, CurrencyBaseSymbols()...)
}

// InstrumentPrice is a computed, user-facing price for a single instrument.
type InstrumentPrice struct {
	Code        string    `json:"code"`
	DisplayName string    `json:"display_name"`
	Kind        string    `json:"kind"`
	Price       float64   `json:"price"` // native quote (TRY, or USD for global equities)
	Bid         float64   `json:"bid"`
	Ask         float64   `json:"ask"`
	Change      float64   `json:"change"`     // absolute change vs day open (native currency)
	ChangePct   float64   `json:"change_pct"` // percent change vs day open
	Currency    string    `json:"currency"`   // native currency of Price: "TRY" or "USD"
	PriceTRY    float64   `json:"price_try"`  // TRY equivalent (== Price for TRY; Price×USDTRY for USD)
	Provider    string    `json:"provider"`
	Timestamp   time.Time `json:"timestamp"`
}

// ComputeInstrumentPrice derives a TRY price for the instrument given base legs.
//   - Currency: uses its TRY rate directly.
//   - Gold: (XAUUSD / troy) * USDTRY * gramWeight * fineness.
func ComputeInstrumentPrice(in Instrument, xauusd, usdtry float64, currencyTRY map[string]float64) (price, bid, ask float64, ok bool) {
	switch in.Kind {
	case KindCurrency, KindCrypto, KindStock:
		// All three carry a direct TRY value keyed by their base symbol.
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
