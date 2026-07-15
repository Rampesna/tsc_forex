package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/tsc-forex/tsc_forex/internal/calculator"
	"github.com/tsc-forex/tsc_forex/internal/domain"
)

const yahooMetaUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"

// marketMeta builds the per-kind "rich detail" payload shown on the asset
// detail screen (Binance-style). It layers external market data on top of our
// own live legs:
//   - crypto : Binance 24h ticker (high/low/open/volume/trades) + CoinGecko
//     fundamentals (market cap, supply, ATH/ATL, rank, dominance).
//   - stock  : Yahoo chart meta (day & 52-week ranges, volume, exchange).
//   - gold   : derived from XAU/USD + USD/TRY (ounce, gram-has, purity, weight).
//   - currency: parity, inverse and USD cross derived from the live legs.
//
// Responses are cached briefly per code so opening the detail page repeatedly
// does not hammer the upstreams.
type marketMeta struct {
	engine        *calculator.Engine
	client        *http.Client
	binance24hURL string
	coingeckoURL  string
	coingeckoGlob string
	yahooChartURL string
	log           *zap.Logger

	mu    sync.Mutex
	cache map[string]cachedDetail
	ttl   time.Duration

	globMu   sync.Mutex
	globAt   time.Time
	globMcap float64
}

type cachedDetail struct {
	at   time.Time
	data gin.H
}

// newMarketMeta wires the fetcher. cryptoBaseURL is the Binance price endpoint
// from config; the 24h endpoint is derived from it.
func newMarketMeta(engine *calculator.Engine, cryptoBaseURL string, log *zap.Logger) *marketMeta {
	b24 := "https://api.binance.com/api/v3/ticker/24hr"
	if strings.Contains(cryptoBaseURL, "/ticker/price") {
		b24 = strings.Replace(cryptoBaseURL, "/ticker/price", "/ticker/24hr", 1)
	}
	if log == nil {
		log = zap.NewNop()
	}
	return &marketMeta{
		engine:        engine,
		client:        &http.Client{Timeout: 7 * time.Second},
		binance24hURL: b24,
		coingeckoURL:  "https://api.coingecko.com/api/v3/coins/markets",
		coingeckoGlob: "https://api.coingecko.com/api/v3/global",
		yahooChartURL: "https://query1.finance.yahoo.com/v8/finance/chart/",
		log:           log,
		cache:         make(map[string]cachedDetail),
		ttl:           45 * time.Second,
	}
}

func (m *marketMeta) leg(code string) float64 {
	if m.engine == nil {
		return 0
	}
	if ip, ok := m.engine.Get(code); ok {
		return ip.Price
	}
	return 0
}

// Detail returns the cached-or-freshly-built rich payload for one instrument.
func (m *marketMeta) Detail(ctx context.Context, in domain.Instrument) gin.H {
	m.mu.Lock()
	if c, ok := m.cache[in.Code]; ok && time.Since(c.at) < m.ttl {
		d := c.data
		m.mu.Unlock()
		return d
	}
	m.mu.Unlock()

	data := gin.H{"code": in.Code, "kind": string(in.Kind)}
	switch in.Kind {
	case domain.KindCrypto:
		m.fillCrypto(ctx, in, data)
	case domain.KindStock:
		m.fillStock(ctx, in, data)
	case domain.KindGold:
		m.fillGold(in, data)
	case domain.KindCurrency:
		m.fillCurrency(in, data)
	}

	m.mu.Lock()
	m.cache[in.Code] = cachedDetail{at: time.Now(), data: data}
	m.mu.Unlock()
	return data
}

func (m *marketMeta) fillCrypto(ctx context.Context, in domain.Instrument, data gin.H) {
	usdtry := m.leg("USDTRY")
	data["price_try"] = m.leg(in.Code)

	// Binance 24h rolling stats (USD → TRY for the price-like fields).
	if usdtry > 0 {
		var t struct {
			PriceChangePercent string `json:"priceChangePercent"`
			WeightedAvgPrice   string `json:"weightedAvgPrice"`
			PrevClosePrice     string `json:"prevClosePrice"`
			LastPrice          string `json:"lastPrice"`
			OpenPrice          string `json:"openPrice"`
			HighPrice          string `json:"highPrice"`
			LowPrice           string `json:"lowPrice"`
			Volume             string `json:"volume"`
			QuoteVolume        string `json:"quoteVolume"`
			Count              int64  `json:"count"`
		}
		if err := m.getJSON(ctx, m.binance24hURL+"?symbol="+in.Code+"USDT", "", &t); err == nil {
			pf := func(s string) float64 { v, _ := strconv.ParseFloat(s, 64); return v }
			data["price_usd"] = pf(t.LastPrice)
			data["high24h"] = pf(t.HighPrice) * usdtry
			data["low24h"] = pf(t.LowPrice) * usdtry
			data["open24h"] = pf(t.OpenPrice) * usdtry
			data["prev_close"] = pf(t.PrevClosePrice) * usdtry
			data["weighted_avg"] = pf(t.WeightedAvgPrice) * usdtry
			data["volume_base"] = pf(t.Volume)
			data["quote_volume_usd"] = pf(t.QuoteVolume)
			data["trades"] = t.Count
			data["change_pct24h"] = pf(t.PriceChangePercent)
		} else {
			m.log.Debug("crypto 24h fetch failed", zap.String("code", in.Code), zap.Error(err))
		}
	}

	// CoinGecko fundamentals (USD): market cap, supply, ATH/ATL, rank.
	cg := domain.CryptoByCode[in.Code].CoinGeckoID
	if cg != "" {
		var arr []struct {
			MarketCap             *float64 `json:"market_cap"`
			MarketCapRank         *int     `json:"market_cap_rank"`
			FullyDilutedValuation *float64 `json:"fully_diluted_valuation"`
			TotalVolume           *float64 `json:"total_volume"`
			CirculatingSupply     *float64 `json:"circulating_supply"`
			TotalSupply           *float64 `json:"total_supply"`
			MaxSupply             *float64 `json:"max_supply"`
			ATH                   *float64 `json:"ath"`
			ATL                   *float64 `json:"atl"`
			ATHDate               string   `json:"ath_date"`
			ATLDate               string   `json:"atl_date"`
		}
		if err := m.getJSON(ctx, m.coingeckoURL+"?vs_currency=usd&ids="+cg, "", &arr); err == nil && len(arr) > 0 {
			c := arr[0]
			setF(data, "market_cap_usd", c.MarketCap)
			setI(data, "rank", c.MarketCapRank)
			setF(data, "fdv_usd", c.FullyDilutedValuation)
			setF(data, "total_volume_usd", c.TotalVolume)
			setF(data, "circulating_supply", c.CirculatingSupply)
			setF(data, "total_supply", c.TotalSupply)
			setF(data, "max_supply", c.MaxSupply)
			setF(data, "ath_usd", c.ATH)
			setF(data, "atl_usd", c.ATL)
			if c.ATHDate != "" {
				data["ath_date"] = c.ATHDate
			}
			if c.ATLDate != "" {
				data["atl_date"] = c.ATLDate
			}
			if c.MarketCap != nil {
				if g := m.globalMarketCap(ctx); g > 0 {
					data["dominance_pct"] = (*c.MarketCap / g) * 100
				}
			}
		} else if err != nil {
			m.log.Debug("coingecko fetch failed", zap.String("code", in.Code), zap.Error(err))
		}
	}
}

func (m *marketMeta) fillStock(ctx context.Context, in domain.Instrument, data gin.H) {
	// Native quote is USD for global equities, TRY for BIST. Expose both so the
	// client can show "$X" with a "≈ ₺Y" sub-line.
	native := m.leg(in.Code)
	ccy := in.Currency
	if ccy == "" {
		ccy = "TRY"
	}
	data["currency"] = ccy
	if ccy == "USD" {
		data["price_usd"] = native
		if usdtry := m.leg("USDTRY"); usdtry > 0 {
			data["price_try"] = native * usdtry
		}
	} else {
		data["price_try"] = native
	}
	var resp struct {
		Chart struct {
			Result []struct {
				Meta struct {
					RegularMarketDayHigh float64 `json:"regularMarketDayHigh"`
					RegularMarketDayLow  float64 `json:"regularMarketDayLow"`
					FiftyTwoWeekHigh     float64 `json:"fiftyTwoWeekHigh"`
					FiftyTwoWeekLow      float64 `json:"fiftyTwoWeekLow"`
					RegularMarketVolume  float64 `json:"regularMarketVolume"`
					PreviousClose        float64 `json:"previousClose"`
					ChartPreviousClose   float64 `json:"chartPreviousClose"`
					ExchangeName         string  `json:"exchangeName"`
					LongName             string  `json:"longName"`
					Currency             string  `json:"currency"`
				} `json:"meta"`
			} `json:"result"`
		} `json:"chart"`
	}
	// Resolve the Yahoo ticker from the catalog (BIST → "<code>.IS", global →
	// plain ticker) instead of hardcoding the ".IS" suffix.
	yahooSym := in.Code + ".IS"
	if def, ok := domain.StockByCode[in.Code]; ok && def.YahooSymbol != "" {
		yahooSym = def.YahooSymbol
	}
	reqURL := m.yahooChartURL + yahooSym + "?interval=1d&range=5d"
	if err := m.getJSON(ctx, reqURL, yahooMetaUA, &resp); err == nil && len(resp.Chart.Result) > 0 {
		meta := resp.Chart.Result[0].Meta
		data["day_high"] = meta.RegularMarketDayHigh
		data["day_low"] = meta.RegularMarketDayLow
		data["week52_high"] = meta.FiftyTwoWeekHigh
		data["week52_low"] = meta.FiftyTwoWeekLow
		data["volume"] = meta.RegularMarketVolume
		pc := meta.PreviousClose
		if pc == 0 {
			pc = meta.ChartPreviousClose
		}
		data["prev_close"] = pc
		if meta.ExchangeName != "" {
			data["exchange"] = meta.ExchangeName
		}
		if meta.LongName != "" {
			data["long_name"] = meta.LongName
		}
	} else if err != nil {
		m.log.Debug("stock meta fetch failed", zap.String("code", in.Code), zap.Error(err))
	}
}

func (m *marketMeta) fillGold(in domain.Instrument, data gin.H) {
	usdtry := m.leg("USDTRY")
	onsGold := m.leg("ONS_GOLD")
	var xau float64
	if usdtry > 0 {
		xau = onsGold / usdtry // ONS_GOLD = XAU/USD × USD/TRY ⇒ XAU/USD = ons/usdtry
	}
	data["price_try"] = m.leg(in.Code)
	data["ons_usd"] = xau
	data["ons_try"] = onsGold
	data["usd_try"] = usdtry
	data["gram_has_try"] = m.leg("GRAM_GOLD")
	data["gram_weight"] = in.GramWeight
	data["fineness"] = in.Fineness
	data["purity_label"] = purityLabel(in.Fineness)
}

func (m *marketMeta) fillCurrency(in domain.Instrument, data gin.H) {
	usdtry := m.leg("USDTRY")
	price := m.leg(in.Code)
	data["price_try"] = price
	if price > 0 {
		data["inverse"] = 1 / price
	}
	if usdtry > 0 && price > 0 {
		data["usd_cross"] = price / usdtry // 1 unit = X USD
	}
	data["usd_try"] = usdtry
	data["iso"] = strings.TrimSuffix(in.Code, "TRY")
}

// globalMarketCap returns the total crypto market cap (USD), cached 5 minutes,
// so per-coin dominance can be computed cheaply.
func (m *marketMeta) globalMarketCap(ctx context.Context) float64 {
	m.globMu.Lock()
	if time.Since(m.globAt) < 5*time.Minute && m.globMcap > 0 {
		v := m.globMcap
		m.globMu.Unlock()
		return v
	}
	m.globMu.Unlock()

	var g struct {
		Data struct {
			TotalMarketCap map[string]float64 `json:"total_market_cap"`
		} `json:"data"`
	}
	if err := m.getJSON(ctx, m.coingeckoGlob, "", &g); err != nil {
		return 0
	}
	v := g.Data.TotalMarketCap["usd"]
	m.globMu.Lock()
	m.globMcap = v
	m.globAt = time.Now()
	m.globMu.Unlock()
	return v
}

func (m *marketMeta) getJSON(ctx context.Context, reqURL, ua string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if ua != "" {
		req.Header.Set("User-Agent", ua)
	}
	resp, err := m.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func setF(d gin.H, k string, v *float64) {
	if v != nil {
		d[k] = *v
	}
}

func setI(d gin.H, k string, v *int) {
	if v != nil {
		d[k] = *v
	}
}

// purityLabel maps a millesimal fineness to a Turkish karat/millesimal label.
func purityLabel(fineness float64) string {
	switch {
	case fineness >= 0.999:
		return "Saf (995‰+)"
	case fineness >= 0.990:
		return "24 Ayar (995‰)"
	case fineness >= 0.910:
		return "22 Ayar (916‰)"
	case fineness >= 0.745:
		return "18 Ayar (750‰)"
	case fineness >= 0.580:
		return "14 Ayar (585‰)"
	default:
		return fmt.Sprintf("%.0f‰", fineness*1000)
	}
}
