package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tsc-forex/tsc_forex/internal/calculator"
	"github.com/tsc-forex/tsc_forex/internal/provider"
	"github.com/tsc-forex/tsc_forex/internal/utils"
)

// Pinger is satisfied by the Redis cache and DB health probes.
type Pinger interface {
	Ping(ctx context.Context) error
}

// SystemHandler serves health, version, and provider status.
type SystemHandler struct {
	provider  *provider.Manager
	calc      *calculator.Calculator
	redis     Pinger
	dbPing    func(ctx context.Context) error
	clientsFn func() int64
}

// SystemDeps groups optional system collaborators.
type SystemDeps struct {
	Provider  *provider.Manager
	Calc      *calculator.Calculator
	Redis     Pinger
	DBPing    func(ctx context.Context) error
	ClientsFn func() int64
}

// NewSystemHandler builds the system handler.
func NewSystemHandler(d SystemDeps) *SystemHandler {
	return &SystemHandler{
		provider:  d.Provider,
		calc:      d.Calc,
		redis:     d.Redis,
		dbPing:    d.DBPing,
		clientsFn: d.ClientsFn,
	}
}

// Health godoc
// @Summary  Service health
// @Tags     system
// @Produce  json
// @Success  200 {object} map[string]interface{}
// @Router   /health [get]
func (h *SystemHandler) Health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	deps := gin.H{}
	healthy := true

	if h.redis != nil {
		if err := h.redis.Ping(ctx); err != nil {
			deps["redis"] = "down"
			healthy = false
		} else {
			deps["redis"] = "up"
		}
	}
	if h.dbPing != nil {
		if err := h.dbPing(ctx); err != nil {
			deps["database"] = "down"
			healthy = false
		} else {
			deps["database"] = "up"
		}
	}

	body := gin.H{
		"status": "ok",
		"time":   time.Now().UTC(),
		"deps":   deps,
	}
	if h.clientsFn != nil {
		body["ws_clients"] = h.clientsFn()
	}
	if h.provider != nil {
		ph := h.provider.Health()
		body["provider"] = gin.H{"name": ph.Name, "status": ph.Status, "active": h.provider.ActiveName()}
	}
	if h.calc != nil {
		body["calculations"] = h.calc.CalculationCount()
		if g, ok := h.calc.Latest(); ok {
			body["gold"] = gin.H{"symbol": g.Symbol, "price": g.Price, "ts": g.Timestamp}
		}
	}

	code := http.StatusOK
	if !healthy {
		body["status"] = "degraded"
		code = http.StatusServiceUnavailable
	}
	c.JSON(code, body)
}

// Version godoc
// @Summary  Build version
// @Tags     system
// @Produce  json
// @Success  200 {object} map[string]string
// @Router   /version [get]
func (h *SystemHandler) Version(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version":    utils.Version,
		"commit":     utils.Commit,
		"build_time": utils.BuildTime,
	})
}

// Providers godoc
// @Summary  Provider status
// @Tags     system
// @Produce  json
// @Success  200 {object} map[string]interface{}
// @Router   /providers [get]
func (h *SystemHandler) Providers(c *gin.Context) {
	if h.provider == nil {
		c.JSON(http.StatusOK, gin.H{"providers": []ProviderStatusResponse{}})
		return
	}
	active := h.provider.ActiveName()
	statuses := h.provider.Statuses()
	out := make([]ProviderStatusResponse, 0, len(statuses))
	for _, s := range statuses {
		out = append(out, ProviderStatusResponse{
			Name:          s.Name,
			Status:        string(s.Status),
			LastTickAt:    s.LastTickAt,
			LastError:     s.LastError,
			Reconnects:    s.Reconnects,
			Subscribed:    s.Subscribed,
			LatencyMillis: s.LatencyMillis,
			Active:        s.Name == active,
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"active":    active,
		"switches":  h.provider.SwitchCount(),
		"providers": out,
	})
}
