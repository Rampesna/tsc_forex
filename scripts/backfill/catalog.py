"""Instrument catalog mirrored from internal/domain/instrument.go + gold.go.

Keep this in sync with the Go source. Values (gram weights, fineness, provider
symbols) must match ComputeInstrumentPrice so backfilled prices line up with
live data written by the running engine.
"""

# Standard troy ounce -> gram (internal/domain/gold.go).
TROY_OUNCE_TO_GRAM = 31.1034768

# Millesimal fineness (internal/domain/instrument.go).
FINENESS_24K = 0.995  # "has" / gram gold
FINENESS_22K = 0.916  # coins & bilezik
FINENESS_18K = 0.750
FINENESS_14K = 0.585

# --- Currencies (56) : code == "<ISO>TRY", kind "currency", priced in TRY ---
CURRENCIES = [
    "USD", "EUR", "GBP", "JPY",
    "CHF", "CAD", "AUD", "NZD",
    "CNY", "RUB", "SAR", "AED",
    "KWD", "QAR", "BHD", "OMR",
    "JOD", "ILS", "INR", "PKR",
    "IQD", "EGP", "ZAR", "SEK",
    "NOK", "DKK", "PLN", "CZK",
    "HUF", "RON", "BGN", "UAH",
    "KRW", "SGD", "HKD", "TWD",
    "THB", "MYR", "IDR", "PHP",
    "VND", "MXN", "BRL", "ARS",
    "CLP", "COP", "AZN", "KZT",
    "GEL", "MAD", "DZD", "TND",
    "NGN", "KES", "UZS", "ISK",
]

# --- Gold products (10) : derived from XAUUSD + USDTRY ---
# (code, gram_weight, fineness)
GOLD_PRODUCTS = [
    ("ONS_GOLD", TROY_OUNCE_TO_GRAM, 1.0),
    ("GRAM_GOLD", 1.0, FINENESS_24K),
    ("CEYREK_GOLD", 1.754, FINENESS_22K),
    ("YARIM_GOLD", 3.508, FINENESS_22K),
    ("TAM_GOLD", 7.016, FINENESS_22K),
    ("CUMHURIYET_GOLD", 7.216, FINENESS_22K),
    ("ATA_GOLD", 7.216, FINENESS_22K),
    ("GRAM_22K", 1.0, FINENESS_22K),
    ("GRAM_18K", 1.0, FINENESS_18K),
    ("GRAM_14K", 1.0, FINENESS_14K),
]

# --- Crypto (17) : code == symbol, priced from <SYM>-USD (Yahoo) x USDTRY ---
# (code, yahoo_symbol, binance_symbol)
CRYPTOS = [
    ("BTC", "BTC-USD", "BTCUSDT"),
    ("ETH", "ETH-USD", "ETHUSDT"),
    ("BNB", "BNB-USD", "BNBUSDT"),
    ("SOL", "SOL-USD", "SOLUSDT"),
    ("XRP", "XRP-USD", "XRPUSDT"),
    ("ADA", "ADA-USD", "ADAUSDT"),
    ("DOGE", "DOGE-USD", "DOGEUSDT"),
    ("TRX", "TRX-USD", "TRXUSDT"),
    ("AVAX", "AVAX-USD", "AVAXUSDT"),
    ("DOT", "DOT-USD", "DOTUSDT"),
    ("LINK", "LINK-USD", "LINKUSDT"),
    ("MATIC", "MATIC-USD", "MATICUSDT"),
    ("LTC", "LTC-USD", "LTCUSDT"),
    ("ATOM", "ATOM-USD", "ATOMUSDT"),
    ("XLM", "XLM-USD", "XLMUSDT"),
    ("SHIB", "SHIB-USD", "SHIBUSDT"),
    ("BCH", "BCH-USD", "BCHUSDT"),
]

# --- BIST stocks (18) : code == symbol, Yahoo "<SYM>.IS", native TRY ---
STOCKS = [
    "THYAO", "GARAN", "AKBNK", "ISCTR",
    "YKBNK", "SISE", "EREGL", "KCHOL",
    "SAHOL", "ASELS", "BIMAS", "TUPRS",
    "FROTO", "TCELL", "PGSUS", "TTKOM",
    "TOASO", "SASA",
]

# --- Global stocks (50) : code == symbol, native USD, plain Yahoo ticker ---
# Mirror of domain.GlobalStocks. Stored/charted in USD (matches the live engine),
# with a TRY equivalent derived at display time.  (code, yahoo_symbol)
GLOBAL_STOCKS = [
    ("NVDA", "NVDA"), ("AAPL", "AAPL"), ("MSFT", "MSFT"), ("GOOGL", "GOOGL"),
    ("AMZN", "AMZN"), ("META", "META"), ("AVGO", "AVGO"), ("TSLA", "TSLA"),
    ("BRKB", "BRK-B"), ("TSM", "TSM"), ("LLY", "LLY"), ("WMT", "WMT"),
    ("JPM", "JPM"), ("V", "V"), ("MA", "MA"), ("ORCL", "ORCL"),
    ("NVO", "NVO"), ("XOM", "XOM"), ("UNH", "UNH"), ("COST", "COST"),
    ("HD", "HD"), ("PG", "PG"), ("JNJ", "JNJ"), ("NFLX", "NFLX"),
    ("BAC", "BAC"), ("ABBV", "ABBV"), ("KO", "KO"), ("SAP", "SAP"),
    ("CRM", "CRM"), ("CVX", "CVX"), ("TMUS", "TMUS"), ("WFC", "WFC"),
    ("ASML", "ASML"), ("BABA", "BABA"), ("MRK", "MRK"), ("AMD", "AMD"),
    ("PEP", "PEP"), ("LIN", "LIN"), ("ADBE", "ADBE"), ("TM", "TM"),
    ("AZN", "AZN"), ("NVS", "NVS"), ("MCD", "MCD"), ("SHEL", "SHEL"),
    ("ABT", "ABT"), ("TMO", "TMO"), ("GE", "GE"), ("CSCO", "CSCO"),
    ("INTC", "INTC"), ("DIS", "DIS"),
]

# Bid/ask spread fractions used by the engine (ComputeInstrumentPrice).
SPREAD_GOLD = 0.001    # ~10 bps
SPREAD_OTHER = 0.0005  # ~5 bps


def gold_price(xauusd: float, usdtry: float, gram_weight: float, fineness: float) -> float:
    """Replicates ComputeInstrumentPrice for gold: (XAUUSD/troy)*USDTRY*gram*fineness."""
    gram_has = (xauusd / TROY_OUNCE_TO_GRAM) * usdtry
    return gram_has * gram_weight * fineness


def bid_ask(price: float, spread_frac: float):
    spread = price * spread_frac
    return price - spread / 2.0, price + spread / 2.0
