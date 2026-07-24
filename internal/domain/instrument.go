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
	{"MATIC", "Polygon (POL)", "POL28321-USD", "polygon-ecosystem-token"},
	{"LTC", "Litecoin", "LTC-USD", "litecoin"},
	{"ATOM", "Cosmos", "ATOM-USD", "cosmos"},
	{"XLM", "Stellar", "XLM-USD", "stellar"},
	{"SHIB", "Shiba Inu", "SHIB-USD", "shiba-inu"},
	{"BCH", "Bitcoin Cash", "BCH-USD", "bitcoin-cash"},
	{"UNI", "Uniswap", "UNI-USD", "uniswap"},
	{"NEAR", "NEAR Protocol", "NEAR-USD", "near"},
	{"APT", "Aptos", "APT-USD", "aptos"},
	{"ARB", "Arbitrum", "ARB-USD", "arbitrum"},
	{"OP", "Optimism", "OP-USD", "optimism"},
	{"INJ", "Injective", "INJ-USD", "injective-protocol"},
	{"FIL", "Filecoin", "FIL-USD", "filecoin"},
	{"ICP", "Internet Computer", "ICP-USD", "internet-computer"},
	{"AAVE", "Aave", "AAVE-USD", "aave"},
	{"FET", "Artificial Superintelligence", "FET-USD", "fetch-ai"},
	{"RENDER", "Render", "RENDER-USD", "render-token"},
	{"LDO", "Lido DAO", "LDO-USD", "lido-dao"},
	{"STX", "Stacks", "STX-USD", "blockstack"},
	{"TIA", "Celestia", "TIA-USD", "celestia"},
	{"SEI", "Sei", "SEI-USD", "sei-network"},
	{"SUI", "Sui", "SUI-USD", "sui"},
	{"PEPE", "Pepe", "PEPE-USD", "pepe"},
	{"WIF", "dogwifhat", "WIF-USD", "dogwifcoin"},
	{"BONK", "Bonk", "BONK-USD", "bonk"},
	{"FLOKI", "FLOKI", "FLOKI-USD", "floki"},
	{"ETC", "Ethereum Classic", "ETC-USD", "ethereum-classic"},
	{"ALGO", "Algorand", "ALGO-USD", "algorand"},
	{"VET", "VeChain", "VET-USD", "vechain"},
	{"SAND", "The Sandbox", "SAND-USD", "the-sandbox"},
	{"MANA", "Decentraland", "MANA-USD", "decentraland"},
	{"AXS", "Axie Infinity", "AXS-USD", "axie-infinity"},
	{"CRV", "Curve DAO", "CRV-USD", "curve-dao-token"},
	{"ENS", "Ethereum Name Service", "ENS-USD", "ethereum-name-service"},
	{"GRT", "The Graph", "GRT-USD", "the-graph"},
	{"FLOW", "Flow", "FLOW-USD", "flow"},
	{"IMX", "Immutable", "IMX-USD", "immutable-x"},
	{"RUNE", "THORChain", "RUNE-USD", "thorchain"},
	{"HBAR", "Hedera", "HBAR-USD", "hedera-hashgraph"},
	{"ONDO", "Ondo", "ONDO-USD", "ondo-finance"},
	{"WLD", "Worldcoin", "WLD-USD", "worldcoin-wld"},
	{"TAO", "Bittensor", "TAO-USD", "bittensor"},
	{"ENA", "Ethena", "ENA-USD", "ethena"},
	{"JUP", "Jupiter", "JUP-USD", "jupiter-exchange-solana"},
	{"PYTH", "Pyth Network", "PYTH-USD", "pyth-network"},
	{"STRK", "Starknet", "STRK-USD", "starknet"},
	{"BLUR", "Blur", "BLUR-USD", "blur"},
	{"APE", "ApeCoin", "APE-USD", "apecoin"},
	{"CHZ", "Chiliz", "CHZ-USD", "chiliz"},
	{"SNX", "Synthetix", "SNX-USD", "havven"},
	{"COMP", "Compound", "COMP-USD", "compound-governance-token"},
	{"ZEC", "Zcash", "ZEC-USD", "zcash"},
	{"QNT", "Quant", "QNT-USD", "quant-network"},
	{"EGLD", "MultiversX", "EGLD-USD", "elrond-erd-2"},
	{"KAVA", "Kava", "KAVA-USD", "kava"},
	{"ZRO", "LayerZero", "ZRO-USD", "layerzero"},
	{"PENDLE", "Pendle", "PENDLE-USD", "pendle"},
	{"CAKE", "PancakeSwap", "CAKE-USD", "pancakeswap-token"},
	{"GALA", "Gala", "GALA-USD", "gala"},
	{"ENJ", "Enjin Coin", "ENJ-USD", "enjincoin"},
	{"BAT", "Basic Attention Token", "BAT-USD", "basic-attention-token"},
	{"ANKR", "Ankr", "ANKR-USD", "ankr"},
	{"CELO", "Celo", "CELO-USD", "celo"},
	{"ROSE", "Oasis", "ROSE-USD", "oasis-network"},
	{"KSM", "Kusama", "KSM-USD", "kusama"},
	{"NEO", "Neo", "NEO-USD", "neo"},
	{"QTUM", "Qtum", "QTUM-USD", "qtum"},
	{"IOTA", "IOTA", "IOTA-USD", "iota"},
	{"XTZ", "Tezos", "XTZ-USD", "tezos"},
	{"MINA", "Mina", "MINA-USD", "mina-protocol"},
	{"CFX", "Conflux", "CFX-USD", "conflux-token"},
	{"ASTR", "Astar", "ASTR-USD", "astar"},
	{"THETA", "Theta Network", "THETA-USD", "theta-token"},
	{"NOT", "Notcoin", "NOT-USD", "notcoin"},
	{"ORDI", "ORDI", "ORDI-USD", "ordinals"},
	{"DYDX", "dYdX", "DYDX-USD", "dydx-chain"},
	{"SUPER", "SuperVerse", "SUPER-USD", "superfarm"},
	{"RSR", "Reserve Rights", "RSR-USD", "reserve-rights-token"},
	{"SKL", "SKALE", "SKL-USD", "skale"},
	{"JTO", "Jito", "JTO-USD", "jito-governance-token"},
	{"ETHFI", "ether.fi", "ETHFI-USD", "ether-fi"},
	{"ALT", "AltLayer", "ALT-USD", "altlayer"},
	{"PIXEL", "Pixels", "PIXEL-USD", "pixels"},
	{"CYBER", "Cyber", "CYBER-USD", "cyberconnect"},
	{"TRB", "Tellor", "TRB-USD", "tellor"},
	{"MASK", "Mask Network", "MASK-USD", "mask-network"},
	{"YGG", "Yield Guild Games", "YGG-USD", "yield-guild-games"},
	{"GMT", "GMT", "GMT-USD", "stepn"},
	{"GMX", "GMX", "GMX-USD", "gmx"},
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

// Stocks are BIST 100 equities tracked in TRY (Yahoo "<SYM>.IS").
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
	{"DSTKF", "Destek Faktoring", "DSTKF.IS", "TRY"},
	{"ENKAI", "Enka İnşaat", "ENKAI.IS", "TRY"},
	{"KTLEV", "Katılımevim", "KTLEV.IS", "TRY"},
	{"ASTOR", "Astor Enerji", "ASTOR.IS", "TRY"},
	{"VAKBN", "VakıfBank", "VAKBN.IS", "TRY"},
	{"HALKB", "Halkbank", "HALKB.IS", "TRY"},
	{"ODINE", "Odine Solutions", "ODINE.IS", "TRY"},
	{"CCOLA", "Coca-Cola İçecek", "CCOLA.IS", "TRY"},
	{"TRALT", "Türk Altın İşletmeleri", "TRALT.IS", "TRY"},
	{"GUBRF", "Gübre Fabrikaları", "GUBRF.IS", "TRY"},
	{"KLRHO", "Kiler Holding", "KLRHO.IS", "TRY"},
	{"TURSG", "Türkiye Sigorta", "TURSG.IS", "TRY"},
	{"AKSEN", "Aksa Enerji", "AKSEN.IS", "TRY"},
	{"ENJSA", "Enerjisa", "ENJSA.IS", "TRY"},
	{"AEFES", "Anadolu Efes", "AEFES.IS", "TRY"},
	{"MAGEN", "Margün Enerji", "MAGEN.IS", "TRY"},
	{"MGROS", "Migros", "MGROS.IS", "TRY"},
	{"OYAKC", "Oyak Çimento", "OYAKC.IS", "TRY"},
	{"TAVHL", "TAV Havalimanları", "TAVHL.IS", "TRY"},
	{"RALYH", "Ral Yatırım", "RALYH.IS", "TRY"},
	{"ENERY", "Enerya Enerji", "ENERY.IS", "TRY"},
	{"BRSAN", "Borusan Mannesmann", "BRSAN.IS", "TRY"},
	{"IEYHO", "Işıklar Enerji", "IEYHO.IS", "TRY"},
	{"MPARK", "MLP Sağlık", "MPARK.IS", "TRY"},
	{"PASEU", "Pasifik Eurasia", "PASEU.IS", "TRY"},
	{"EKGYO", "Emlak Konut GYO", "EKGYO.IS", "TRY"},
	{"ARCLK", "Arçelik", "ARCLK.IS", "TRY"},
	{"EUPWR", "Europower Enerji", "EUPWR.IS", "TRY"},
	{"PETKM", "Petkim", "PETKM.IS", "TRY"},
	{"KRDMD", "Kardemir (D)", "KRDMD.IS", "TRY"},
	{"BSOKE", "Batısöke Çimento", "BSOKE.IS", "TRY"},
	{"CVKMD", "CVK Maden", "CVKMD.IS", "TRY"},
	{"ANSGR", "Anadolu Sigorta", "ANSGR.IS", "TRY"},
	{"ISMEN", "İş Yatırım", "ISMEN.IS", "TRY"},
	{"GLRMK", "Gülermak", "GLRMK.IS", "TRY"},
	{"TKFEN", "Tekfen Holding", "TKFEN.IS", "TRY"},
	{"TRMET", "TR Anadolu Metal", "TRMET.IS", "TRY"},
	{"ECILC", "Eczacıbaşı İlaç", "ECILC.IS", "TRY"},
	{"BRYAT", "Borusan Yatırım", "BRYAT.IS", "TRY"},
	{"CIMSA", "Çimsa", "CIMSA.IS", "TRY"},
	{"AKSA", "Aksa Akrilik", "AKSA.IS", "TRY"},
	{"ALARK", "Alarko Holding", "ALARK.IS", "TRY"},
	{"CWENE", "CW Enerji", "CWENE.IS", "TRY"},
	{"GENIL", "Gen İlaç", "GENIL.IS", "TRY"},
	{"OTKAR", "Otokar", "OTKAR.IS", "TRY"},
	{"EFOR", "Efor Yatırım", "EFOR.IS", "TRY"},
	{"GESAN", "Girişim Elektrik", "GESAN.IS", "TRY"},
	{"DOHOL", "Doğan Holding", "DOHOL.IS", "TRY"},
	{"DOAS", "Doğuş Otomotiv", "DOAS.IS", "TRY"},
	{"ULKER", "Ülker", "ULKER.IS", "TRY"},
	{"SKBNK", "Şekerbank", "SKBNK.IS", "TRY"},
	{"SOKM", "Şok Marketler", "SOKM.IS", "TRY"},
	{"GRSEL", "Gür-Sel Turizm", "GRSEL.IS", "TRY"},
	{"TSKB", "TSKB", "TSKB.IS", "TRY"},
	{"MAVI", "Mavi Giyim", "MAVI.IS", "TRY"},
	{"GRTHO", "Grainturk Holding", "GRTHO.IS", "TRY"},
	{"PAHOL", "Pasifik Holding", "PAHOL.IS", "TRY"},
	{"BTCIM", "Batıçim", "BTCIM.IS", "TRY"},
	{"KUYAS", "Kuyaş Yatırım", "KUYAS.IS", "TRY"},
	{"PSGYO", "Pasifik GYO", "PSGYO.IS", "TRY"},
	{"TRENJ", "TR Doğal Enerji", "TRENJ.IS", "TRY"},
	{"HEKTS", "Hektaş", "HEKTS.IS", "TRY"},
	{"DAPGM", "DAP Gayrimenkul", "DAPGM.IS", "TRY"},
	{"SARKY", "Sarkuysan", "SARKY.IS", "TRY"},
	{"IZENR", "İzdemir Enerji", "IZENR.IS", "TRY"},
	{"FENER", "Fenerbahçe", "FENER.IS", "TRY"},
	{"BALSU", "Balsu Gıda", "BALSU.IS", "TRY"},
	{"ALTNY", "Altınay Savunma", "ALTNY.IS", "TRY"},
	{"MIATK", "MİA Teknoloji", "MIATK.IS", "TRY"},
	{"OBAMS", "Oba Makarna", "OBAMS.IS", "TRY"},
	{"PATEK", "Pasifik Teknoloji", "PATEK.IS", "TRY"},
	{"GSRAY", "Galatasaray", "GSRAY.IS", "TRY"},
	{"CANTE", "Çan2 Termik", "CANTE.IS", "TRY"},
	{"ZOREN", "Zorlu Enerji", "ZOREN.IS", "TRY"},
	{"ODAS", "Odaş Elektrik", "ODAS.IS", "TRY"},
	{"QUAGR", "Qua Granite", "QUAGR.IS", "TRY"},
	{"TUKAS", "Tukaş", "TUKAS.IS", "TRY"},
	{"BERA", "Bera Holding", "BERA.IS", "TRY"},
	{"EUREN", "Europen", "EUREN.IS", "TRY"},
	{"VESTL", "Vestel", "VESTL.IS", "TRY"},
	{"ESEN", "Esenboğa Elektrik", "ESEN.IS", "TRY"},
	{"REEDR", "Reeder Teknoloji", "REEDR.IS", "TRY"},
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
	{"AFN", "Afgan Afganisi"}, {"ALL", "Arnavut Leki"}, {"AMD", "Ermeni Dramı"}, {"AOA", "Angola Kvanzası"},
	{"BAM", "Bosna Markı"}, {"BDT", "Bangladeş Takası"}, {"BOB", "Bolivya Bolivyanosu"}, {"BYN", "Belarus Rublesi"},
	{"BZD", "Belize Doları"}, {"CRC", "Kosta Rika Kolonu"}, {"CUP", "Küba Pesosu"}, {"CVE", "Yeşil Burun Eskudosu"},
	{"DJF", "Cibuti Frangı"}, {"DOP", "Dominik Pesosu"}, {"ETB", "Etiyopya Birri"}, {"FJD", "Fiji Doları"},
	{"GHS", "Gana Sedisi"}, {"GMD", "Gambiya Dalasisi"}, {"GNF", "Gine Frangı"}, {"GTQ", "Guatemala Ketsali"},
	{"GYD", "Guyana Doları"}, {"HNL", "Honduras Lempirası"}, {"HTG", "Haiti Gurdu"}, {"IRR", "İran Riyali"},
	{"JMD", "Jamaika Doları"}, {"KGS", "Kırgız Somu"}, {"KHR", "Kamboçya Rieli"}, {"KMF", "Komor Frangı"},
	{"LAK", "Laos Kipi"}, {"LBP", "Lübnan Lirası"}, {"LKR", "Sri Lanka Rupisi"}, {"LRD", "Liberya Doları"},
	{"LYD", "Libya Dinarı"}, {"MDL", "Moldova Leyi"}, {"MGA", "Madagaskar Ariarisi"}, {"MKD", "Makedonya Dinarı"},
	{"MMK", "Myanmar Kyatı"}, {"MNT", "Moğol Tugriki"}, {"MOP", "Makao Patakası"}, {"MUR", "Mauritius Rupisi"},
	{"MVR", "Maldiv Rufiyası"}, {"MWK", "Malavi Kvaçası"}, {"MZN", "Mozambik Metikali"}, {"NAD", "Namibya Doları"},
	{"NIO", "Nikaragua Kordobası"}, {"NPR", "Nepal Rupisi"}, {"PEN", "Peru Solu"}, {"PGK", "Papua Yeni Gine Kinası"},
	{"PYG", "Paraguay Guaranisi"}, {"RSD", "Sırp Dinarı"}, {"RWF", "Ruanda Frangı"}, {"SCR", "Seyşeller Rupisi"},
	{"SDG", "Sudan Sterlini"}, {"SOS", "Somali Şilini"}, {"SRD", "Surinam Doları"}, {"SYP", "Suriye Lirası"},
	{"TJS", "Tacikistan Somonisi"}, {"TMT", "Türkmenistan Manatı"}, {"TTD", "Trinidad Tobago Doları"}, {"TZS", "Tanzanya Şilini"},
	{"UGX", "Uganda Şilini"}, {"UYU", "Uruguay Pesosu"}, {"VES", "Venezuela Bolivarı"}, {"XAF", "Orta Afrika CFA Frangı"},
	{"XCD", "Doğu Karayip Doları"}, {"XOF", "Batı Afrika CFA Frangı"}, {"XPF", "CFP Frangı"}, {"YER", "Yemen Riyali"},
	{"ZMW", "Zambiya Kvaçası"}, {"ANG", "Hollanda Antilleri Guldeni"}, {"AWG", "Aruba Florini"}, {"BBD", "Barbados Doları"},
	{"BMD", "Bermuda Doları"}, {"BND", "Brunei Doları"}, {"BSD", "Bahama Doları"}, {"BTN", "Bhutan Ngultrumu"},
	{"BWP", "Botsvana Pulası"}, {"CDF", "Kongo Frangı"}, {"ERN", "Eritre Nakfası"}, {"LSL", "Lesotho Lotisi"},
	{"MRU", "Moritanya Ugiyası"}, {"PAB", "Panama Balboası"}, {"SBD", "Solomon Adaları Doları"}, {"SZL", "Esvatini Lilangenisi"},
	{"TOP", "Tonga Paʻangası"}, {"VUV", "Vanuatu Vatusu"}, {"WST", "Samoa Talası"}, {"SLE", "Sierra Leone Leonesi"},
	{"STN", "São Tomé Dobrası"}, {"SSP", "Güney Sudan Sterlini"}, {"GIP", "Cebelitarık Sterlini"}, {"FKP", "Falkland Adaları Sterlini"},
	{"SHP", "Saint Helena Sterlini"}, {"KYD", "Cayman Adaları Doları"},
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
	out := make([]Instrument, 0, len(Currencies)+len(Cryptos)+len(Stocks)+len(GlobalStocks)+len(goldProducts()))
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
