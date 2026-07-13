package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/tsc-forex/tsc_forex/internal/cache"
	"github.com/tsc-forex/tsc_forex/internal/calculator"
	"github.com/tsc-forex/tsc_forex/internal/domain"
	"github.com/tsc-forex/tsc_forex/internal/repository"
)

// GoldHandler serves current and historical gram-gold prices.
type GoldHandler struct {
	calc  *calculator.Calculator
	cache *cache.RedisCache
	repo  repository.GoldRepository
	log   *zap.Logger
}

// NewGoldHandler builds the gold handler. cache and repo may be nil in minimal mode.
func NewGoldHandler(calc *calculator.Calculator, c *cache.RedisCache, repo repository.GoldRepository, log *zap.Logger) *GoldHandler {
	if log == nil {
		log = zap.NewNop()
	}
	return &GoldHandler{calc: calc, cache: c, repo: repo, log: log}
}

// Current godoc
// @Summary      Current gram gold price
// @Description  Returns the latest calculated gram gold (XAU gram TRY) price.
// @Tags         gold
// @Produce      json
// @Success      200 {object} GoldResponse
// @Failure      503 {object} ErrorResponse
// @Router       /gold/current [get]
func (h *GoldHandler) Current(c *gin.Context) {
	// Prefer in-process calculator snapshot (fastest, always fresh).
	if h.calc != nil {
		if g, ok := h.calc.Latest(); ok {
			c.JSON(http.StatusOK, toGoldResponse(g, "calculator"))
			return
		}
	}
	// Fall back to Redis cache (shared across instances).
	if h.cache != nil {
		if g, err := h.cache.GetLatest(c.Request.Context()); err == nil {
			c.JSON(http.StatusOK, toGoldResponse(g, "cache"))
			return
		}
	}
	c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "price not available yet"})
}

// History godoc
// @Summary      Historical gram gold prices
// @Description  Returns historical prices, optionally aggregated into OHLC candles.
// @Tags         gold
// @Produce      json
// @Param        symbol   query string false "Symbol (default XAU_GRAM_TRY)"
// @Param        from     query string false "RFC3339 start time"
// @Param        to       query string false "RFC3339 end time"
// @Param        limit    query int    false "Max rows (raw mode)"
// @Param        interval query string false "Aggregation: 1m,5m,15m,30m,1h,4h,1d"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} ErrorResponse
// @Failure      503 {object} ErrorResponse
// @Router       /gold/history [get]
func (h *GoldHandler) History(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "history storage not configured"})
		return
	}
	symbol := c.DefaultQuery("symbol", domain.SymbolGramGoldTRY)
	from := parseTime(c.Query("from"))
	to := parseTime(c.Query("to"))

	if interval := c.Query("interval"); interval != "" {
		if !repository.SupportedInterval(interval) {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "unsupported interval"})
			return
		}
		candles, err := h.repo.Aggregate(c.Request.Context(), symbol, interval, from, to)
		if err != nil {
			h.log.Warn("aggregate failed", zap.Error(err))
			c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "aggregation failed"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"symbol": symbol, "interval": interval, "candles": candles})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "1000"))
	rows, err := h.repo.History(c.Request.Context(), symbol, from, to, limit)
	if err != nil {
		h.log.Warn("history query failed", zap.Error(err))
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "history query failed"})
		return
	}
	items := make([]HistoryItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, HistoryItem{
			Price:     r.Price,
			MidPrice:  r.MidPrice,
			Spread:    r.Spread,
			XAUUSD:    r.XAUUSD,
			USDTRY:    r.USDTRY,
			Provider:  r.Provider,
			CreatedAt: r.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"symbol": symbol, "count": len(items), "items": items})
}

func toGoldResponse(g domain.GramGold, source string) GoldResponse {
	return GoldResponse{
		Symbol:    g.Symbol,
		Price:     g.Price,
		MidPrice:  g.MidPrice,
		Spread:    g.Spread,
		XAUUSD:    g.XAUUSD,
		USDTRY:    g.USDTRY,
		Provider:  g.Provider,
		Timestamp: g.Timestamp,
		Source:    source,
	}
}

func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	if unix, err := strconv.ParseInt(s, 10, 64); err == nil {
		return time.Unix(unix, 0).UTC()
	}
	return time.Time{}
}
