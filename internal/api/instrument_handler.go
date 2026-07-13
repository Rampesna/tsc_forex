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
)

// InstrumentHandler serves the full multi-instrument catalog (all gold products
// + currencies): live prices, catalog metadata, and per-instrument history.
type InstrumentHandler struct {
	engine *calculator.Engine
	repo   repository.PriceRepository // may be nil in minimal mode
	log    *zap.Logger
}

// NewInstrumentHandler builds the handler.
func NewInstrumentHandler(engine *calculator.Engine, repo repository.PriceRepository, log *zap.Logger) *InstrumentHandler {
	if log == nil {
		log = zap.NewNop()
	}
	return &InstrumentHandler{engine: engine, repo: repo, log: log}
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
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "history storage not configured"})
		return
	}
	code := c.Param("code")
	if _, ok := domain.CatalogByCode[code]; !ok {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "unknown instrument"})
		return
	}
	from := parseTime(c.Query("from"))
	to := parseTime(c.Query("to"))

	if interval := c.Query("interval"); interval != "" {
		if !repository.SupportedInterval(interval) {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "unsupported interval"})
			return
		}
		candles, err := h.repo.Aggregate(c.Request.Context(), code, interval, from, to)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "aggregation failed"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": code, "interval": interval, "candles": candles})
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
	if h.repo == nil {
		c.JSON(http.StatusOK, gin.H{"code": code, "last": last})
		return
	}
	s, err := h.repo.Stats(c.Request.Context(), code, time.Now().Add(-24*time.Hour))
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

// Sparklines returns the last N prices per instrument for the market-list mini
// charts. Defaults to 10 points.
// @Summary Sparkline series for all instruments
// @Tags    instruments
// @Router  /instruments/sparklines [get]
func (h *InstrumentHandler) Sparklines(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusOK, gin.H{"points": gin.H{}})
		return
	}
	points, _ := strconv.Atoi(c.DefaultQuery("points", "10"))
	series, err := h.repo.Sparklines(c.Request.Context(), points)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "sparkline query failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"points": series})
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
		Provider:    ip.Provider,
		Timestamp:   ip.Timestamp,
	}
}
