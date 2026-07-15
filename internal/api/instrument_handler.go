package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/tsc-forex/tsc_forex/internal/calculator"
	"github.com/tsc-forex/tsc_forex/internal/domain"
	"github.com/tsc-forex/tsc_forex/internal/repository"
	"github.com/tsc-forex/tsc_forex/internal/service"
)

// InstrumentHandler serves the full multi-instrument catalog (all gold products
// + currencies): live prices, catalog metadata, and per-instrument history.
type InstrumentHandler struct {
	engine  *calculator.Engine
	repo    repository.PriceRepository // may be nil in minimal mode
	candles *service.CandleStore       // hot path for charts/sparklines/stats
	meta    *marketMeta                // per-kind rich detail (external data)
	log     *zap.Logger
}

// NewInstrumentHandler builds the handler. cryptoBaseURL is the Binance price
// endpoint (from config) used to derive the 24h ticker URL for crypto detail.
func NewInstrumentHandler(engine *calculator.Engine, repo repository.PriceRepository, candles *service.CandleStore, cryptoBaseURL string, log *zap.Logger) *InstrumentHandler {
	if log == nil {
		log = zap.NewNop()
	}
	return &InstrumentHandler{
		engine:  engine,
		repo:    repo,
		candles: candles,
		meta:    newMarketMeta(engine, cryptoBaseURL, log),
		log:     log,
	}
}

// Catalog returns the static instrument catalog metadata.
// @Summary List all supported instruments
// @Tags    instruments
// @Router  /instruments [get]
func (h *InstrumentHandler) Catalog(c *gin.Context) {
	out := make([]gin.H, 0, len(domain.Catalog))
	for _, in := range domain.Catalog {
		out = append(out, gin.H{
			"code":         in.Code,
			"display_name": in.DisplayName,
			"kind":         string(in.Kind),
			"unit":         in.Unit,
		})
	}
	c.JSON(http.StatusOK, gin.H{"count": len(out), "instruments": out})
}

// Prices returns the latest computed price for every instrument.
// @Summary Current prices for all instruments
// @Tags    instruments
// @Router  /instruments/prices [get]
func (h *InstrumentHandler) Prices(c *gin.Context) {
	if h.engine == nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "engine not ready"})
		return
	}
	snap := h.engine.Snapshot()
	out := make([]InstrumentResponse, 0, len(snap))
	for _, ip := range snap {
		out = append(out, toInstrumentResponse(ip))
	}
	if len(out) == 0 {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "prices not available yet"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"count": len(out), "prices": out})
}

// Get returns the latest price for a single instrument code.
// @Summary Current price for one instrument
// @Tags    instruments
// @Router  /instruments/{code} [get]
func (h *InstrumentHandler) Get(c *gin.Context) {
	code := c.Param("code")
	if _, ok := domain.CatalogByCode[code]; !ok {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "unknown instrument"})
		return
	}
	if h.engine != nil {
		if ip, ok := h.engine.Get(code); ok {
			c.JSON(http.StatusOK, toInstrumentResponse(ip))
			return
		}
	}
	if h.repo != nil {
		if rec, err := h.repo.Latest(c.Request.Context(), code); err == nil {
			c.JSON(http.StatusOK, InstrumentResponse{
				Code:      rec.Code,
				Kind:      rec.Kind,
				Price:     rec.Price,
				Bid:       rec.Bid,
				Ask:       rec.Ask,
				Provider:  rec.Provider,
				Timestamp: rec.CreatedAt,
			})
			return
		}
	}
	c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "price not available yet"})
}

// History returns raw ticks or OHLC candles for one instrument.
// @Summary Historical prices for one instrument
// @Tags    instruments
// @Router  /instruments/{code}/history [get]
func (h *InstrumentHandler) History(c *gin.Context) {
	code := c.Param("code")
	if _, ok := domain.CatalogByCode[code]; !ok {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "unknown instrument"})
		return
	}
	from := parseTime(c.Query("from"))
	to := parseTime(c.Query("to"))

	// Candle charts read from the pre-aggregated candle store (memory + indexed
	// candle table), never by aggregating raw ticks at request time.
	if interval := c.Query("interval"); interval != "" {
		if h.candles == nil {
			c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "candles not configured"})
			return
		}
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "1000"))
		candles, err := h.candles.Candles(c.Request.Context(), code, interval, from, to, limit)
		if err != nil {
			if err == service.ErrUnsupportedInterval {
				c.JSON(http.StatusBadRequest, ErrorResponse{Error: "unsupported interval"})
				return
			}
			c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "aggregation failed"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": code, "interval": interval, "candles": candles})
		return
	}

	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "history storage not configured"})
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "1000"))
	rows, err := h.repo.History(c.Request.Context(), code, from, to, limit)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "history query failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": code, "count": len(rows), "items": rows})
}

// Stats returns a 24h rolling summary (open/high/low/last/change) for one
// instrument, powering the Binance-style header on the detail screen.
// @Summary 24h stats for one instrument
// @Tags    instruments
// @Router  /instruments/{code}/stats [get]
func (h *InstrumentHandler) Stats(c *gin.Context) {
	code := c.Param("code")
	if _, ok := domain.CatalogByCode[code]; !ok {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "unknown instrument"})
		return
	}
	// Live "last" price from the engine (freshest), falling back to stored last.
	var last float64
	if h.engine != nil {
		if ip, ok := h.engine.Get(code); ok {
			last = ip.Price
		}
	}
	// Fast path: today's open/high/low/close from the in-memory daily candle —
	// no DB. "Open" here is UTC-day open of the 1d candle.
	if h.candles != nil {
		if st := h.candles.Stats(code); st.OK {
			if last == 0 {
				last = st.Close
			}
			var change, changePct float64
			if st.Open > 0 {
				change = last - st.Open
				changePct = (change / st.Open) * 100
			}
			c.JSON(http.StatusOK, gin.H{
				"code": code, "open": st.Open, "high": st.High, "low": st.Low,
				"last": last, "change": change, "change_pct": changePct, "count": st.Count,
			})
			return
		}
	}
	if h.repo == nil {
		c.JSON(http.StatusOK, gin.H{"code": code, "last": last})
		return
	}
	// Fallback: rolling-window stats from the DB (Istanbul market day).
	s, err := h.repo.Stats(c.Request.Context(), code, startOfMarketDay())
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "stats query failed"})
		return
	}
	if last == 0 {
		last = s.Last
	}
	var change, changePct float64
	if s.Open > 0 {
		change = last - s.Open
		changePct = (change / s.Open) * 100
	}
	c.JSON(http.StatusOK, gin.H{
		"code":       code,
		"open":       s.Open,
		"high":       s.High,
		"low":        s.Low,
		"last":       last,
		"change":     change,
		"change_pct": changePct,
		"count":      s.Count,
	})
}

// Detail returns kind-specific rich market data for one instrument, powering
// the Binance-style detail sections (crypto: market cap/supply/ATH; stock:
// 52-week range/volume; gold: ounce/purity/weight; currency: parity/cross).
// @Summary Rich per-kind detail for one instrument
// @Tags    instruments
// @Router  /instruments/{code}/detail [get]
func (h *InstrumentHandler) Detail(c *gin.Context) {
	code := c.Param("code")
	in, ok := domain.CatalogByCode[code]
	if !ok {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "unknown instrument"})
		return
	}
	if h.meta == nil {
		c.JSON(http.StatusOK, gin.H{"code": code, "kind": string(in.Kind)})
		return
	}
	c.JSON(http.StatusOK, h.meta.Detail(c.Request.Context(), in))
}

// Sparklines returns the last N prices per instrument for the market-list mini
// charts. Defaults to 10 points.
// @Summary Sparkline series for all instruments
// @Tags    instruments
// @Router  /instruments/sparklines [get]
func (h *InstrumentHandler) Sparklines(c *gin.Context) {
	points, _ := strconv.Atoi(c.DefaultQuery("points", "10"))
	// Served entirely from the in-memory sparkline ring — zero DB.
	if h.candles != nil {
		series := h.candles.SparklinesAll(points)
		if len(series) > 0 {
			c.JSON(http.StatusOK, gin.H{"points": series})
			return
		}
	}
	// Fallback (cold start, before the ring fills): last N from DB.
	if h.repo == nil {
		c.JSON(http.StatusOK, gin.H{"points": gin.H{}})
		return
	}
	series, err := h.repo.Sparklines(c.Request.Context(), points)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "sparkline query failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"points": series})
}

// marketLoc is the timezone whose local midnight defines the trading day used
// for change % (Istanbul). Falls back to UTC if tzdata is unavailable.
var marketLoc = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Istanbul")
	if err != nil {
		return time.UTC
	}
	return loc
}()

// startOfMarketDay returns local (Istanbul) midnight for today, in UTC.
func startOfMarketDay() time.Time {
	now := time.Now().In(marketLoc)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, marketLoc).UTC()
}

func toInstrumentResponse(ip domain.InstrumentPrice) InstrumentResponse {
	return InstrumentResponse{
		Code:        ip.Code,
		DisplayName: ip.DisplayName,
		Kind:        ip.Kind,
		Price:       ip.Price,
		Bid:         ip.Bid,
		Ask:         ip.Ask,
		Change:      ip.Change,
		ChangePct:   ip.ChangePct,
		Currency:    ip.Currency,
		PriceTRY:    ip.PriceTRY,
		Provider:    ip.Provider,
		Timestamp:   ip.Timestamp,
	}
}
