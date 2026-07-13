package calculator

import (
	"context"
	"math"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"github.com/tsc-forex/tsc_forex/internal/domain"
	"github.com/tsc-forex/tsc_forex/pkg/eventbus"
)

const defaultEpsilon = 1e-8

// Calculator maintains the latest XAUUSD and USDTRY legs and emits GramGold
// updates. All mutable state is owned by a single Run goroutine (no mutex).
type Calculator struct {
	troy    float64
	epsilon float64
	bus     *eventbus.Bus
	log     *zap.Logger

	// latest is an atomic snapshot for REST / health readers.
	latest atomic.Value // domain.GramGold

	calcs     atomic.Int64
	publishes atomic.Int64
}

// Option configures the calculator.
type Option func(*Calculator)

// WithBus publishes TopicGramGoldUpdated after each material change.
func WithBus(bus *eventbus.Bus) Option {
	return func(c *Calculator) { c.bus = bus }
}

// WithLogger sets structured logging.
func WithLogger(log *zap.Logger) Option {
	return func(c *Calculator) {
		if log != nil {
			c.log = log
		}
	}
}

// WithTroyOunceToGram overrides the conversion factor (tests / config).
func WithTroyOunceToGram(v float64) Option {
	return func(c *Calculator) {
		if v > 0 {
			c.troy = v
		}
	}
}

// WithEpsilon sets the minimum absolute price delta required to publish.
func WithEpsilon(eps float64) Option {
	return func(c *Calculator) {
		if eps >= 0 {
			c.epsilon = eps
		}
	}
}

// New creates a calculator. Call Run with a tick stream.
func New(opts ...Option) *Calculator {
	c := &Calculator{
		troy:    domain.TroyOunceToGram,
		epsilon: defaultEpsilon,
		log:     zap.NewNop(),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Latest returns the last published GramGold, or false if not yet ready.
func (c *Calculator) Latest() (domain.GramGold, bool) {
	v := c.latest.Load()
	if v == nil {
		return domain.GramGold{}, false
	}
	g, ok := v.(domain.GramGold)
	return g, ok && g.Price > 0
}

// CalculationCount is the number of successful recalculations (both legs present).
func (c *Calculator) CalculationCount() int64 { return c.calcs.Load() }

// PublishCount is how many updates were emitted (after change detection).
func (c *Calculator) PublishCount() int64 { return c.publishes.Load() }

// Run consumes ticks until ctx cancel or ticks closed.
// Only this goroutine mutates leg state — lock-free hot path.
func (c *Calculator) Run(ctx context.Context, ticks <-chan domain.Tick) {
	var (
		xau       domain.Tick
		usd       domain.Tick
		hasXAU    bool
		hasUSD    bool
		lastPrice float64
		ready     bool
	)

	publish := func(g domain.GramGold) {
		c.latest.Store(g)
		c.publishes.Add(1)
		if c.bus != nil {
			c.bus.Publish(eventbus.TopicGramGoldUpdated, g)
		}
		c.log.Debug("gram gold updated",
			zap.Float64("price", g.Price),
			zap.Float64("xauusd", g.XAUUSD),
			zap.Float64("usdtry", g.USDTRY),
			zap.String("provider", g.Provider),
		)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case tick, ok := <-ticks:
			if !ok {
				return
			}
			switch tick.Symbol {
			case domain.SymbolXAUUSD:
				xau = tick
				hasXAU = true
			case domain.SymbolUSDTRY:
				usd = tick
				hasUSD = true
			default:
				continue
			}

			if !hasXAU || !hasUSD {
				continue
			}

			g, ok := c.compute(xau, usd)
			if !ok {
				continue
			}
			c.calcs.Add(1)

			if ready && math.Abs(g.Price-lastPrice) < c.epsilon {
				continue
			}
			ready = true
			lastPrice = g.Price
			publish(g)
		}
	}
}

func (c *Calculator) compute(xau, usd domain.Tick) (domain.GramGold, bool) {
	xauMid := xau.Mid()
	usdMid := usd.Mid()
	price := (xauMid / c.troy) * usdMid
	if price <= 0 || math.IsNaN(price) || math.IsInf(price, 0) {
		return domain.GramGold{}, false
	}

	// Synthetic gram bid/ask from each leg's bid/ask when available.
	var bid, ask float64
	if xau.Bid > 0 && xau.Ask > 0 && usd.Bid > 0 && usd.Ask > 0 {
		bid = (xau.Bid / c.troy) * usd.Bid
		ask = (xau.Ask / c.troy) * usd.Ask
	} else {
		bid, ask = price, price
	}
	mid := (bid + ask) / 2
	spread := ask - bid

	provider := xau.Provider
	if usd.Provider != "" && usd.Provider != provider {
		provider = xau.Provider + "+" + usd.Provider
	}

	ts := xau.Timestamp
	if usd.Timestamp.After(ts) {
		ts = usd.Timestamp
	}
	if ts.IsZero() {
		ts = time.Now().UTC()
	}

	return domain.GramGold{
		Symbol:    domain.SymbolGramGoldTRY,
		Price:     price,
		MidPrice:  mid,
		Spread:    spread,
		XAUUSD:    xauMid,
		USDTRY:    usdMid,
		Provider:  provider,
		Timestamp: ts.UTC(),
	}, true
}
