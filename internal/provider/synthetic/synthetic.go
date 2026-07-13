package synthetic

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/domain"
)

var errClosed = errors.New("provider closed")

const name = "synthetic"

// leg couples a base symbol with its current price and a random-walk state.
type leg struct {
	symbol string
	price  float64
}

// Provider emits synthetic ticks for ALL base legs the catalog needs
// (XAUUSD, USDTRY, EURTRY, GBPTRY, JPYTRY). Unlike the two-symbol mock,
// this populates the full multi-instrument pipeline offline. It performs a
// small random walk per leg so charts and alerts behave realistically.
type Provider struct {
	cfg config.SyntheticConfig

	out    chan domain.Tick
	cancel context.CancelFunc

	mu         sync.RWMutex
	legs       []leg
	lastTick   time.Time
	reconnects int64
	closed     atomic.Bool
	running    atomic.Bool

	rng *rand.Rand
}

// New builds a synthetic provider with sane defaults.
func New(cfg config.SyntheticConfig) *Provider {
	if cfg.TickInterval <= 0 {
		cfg.TickInterval = 5 * time.Second
	}
	if cfg.XAUUSDBase <= 0 {
		cfg.XAUUSDBase = 2350
	}
	if cfg.USDTRYBase <= 0 {
		cfg.USDTRYBase = 34.5
	}
	if cfg.EURTRYBase <= 0 {
		cfg.EURTRYBase = 37.5
	}
	if cfg.GBPTRYBase <= 0 {
		cfg.GBPTRYBase = 44.0
	}
	if cfg.JPYTRYBase <= 0 {
		cfg.JPYTRYBase = 0.23
	}
	if cfg.JitterPercent <= 0 {
		cfg.JitterPercent = 0.03
	}
	return &Provider{
		cfg: cfg,
		out: make(chan domain.Tick, 512),
		legs: []leg{
			{symbol: domain.SymbolXAUUSD, price: cfg.XAUUSDBase},
			{symbol: domain.SymbolUSDTRY, price: cfg.USDTRYBase},
			{symbol: domain.SymbolEURTRY, price: cfg.EURTRYBase},
			{symbol: domain.SymbolGBPTRY, price: cfg.GBPTRYBase},
			{symbol: domain.SymbolJPYTRY, price: cfg.JPYTRYBase},
		},
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (p *Provider) Name() string { return name }

func (p *Provider) Connect(ctx context.Context) error {
	if p.closed.Load() {
		return errClosed
	}
	if p.running.Load() {
		return nil
	}
	runCtx, cancel := context.WithCancel(ctx)
	p.mu.Lock()
	p.cancel = cancel
	p.mu.Unlock()
	p.running.Store(true)
	go p.loop(runCtx)
	return nil
}

// Subscribe is a no-op: the synthetic provider always streams the full leg set.
func (p *Provider) Subscribe(string) error {
	if p.closed.Load() {
		return errClosed
	}
	return nil
}

func (p *Provider) PriceChannel() <-chan domain.Tick { return p.out }

func (p *Provider) Health() domain.ProviderHealth {
	p.mu.RLock()
	defer p.mu.RUnlock()
	status := domain.ProviderUnknown
	switch {
	case p.closed.Load():
		status = domain.ProviderUnhealthy
	case p.running.Load():
		status = domain.ProviderHealthy
	}
	return domain.ProviderHealth{
		Name:       name,
		Status:     status,
		LastTickAt: p.lastTick,
		Reconnects: atomic.LoadInt64(&p.reconnects),
		Subscribed: domain.BaseSymbols(),
	}
}

func (p *Provider) Close() error {
	if !p.closed.CompareAndSwap(false, true) {
		return nil
	}
	p.mu.Lock()
	if p.cancel != nil {
		p.cancel()
	}
	p.mu.Unlock()
	p.running.Store(false)
	return nil
}

func (p *Provider) loop(ctx context.Context) {
	defer func() {
		p.running.Store(false)
		close(p.out)
	}()

	ticker := time.NewTicker(p.cfg.TickInterval)
	defer ticker.Stop()

	// Emit one immediate full snapshot so consumers get data without waiting.
	p.emitAll(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if p.closed.Load() {
				return
			}
			p.emitAll(ctx)
		}
	}
}

func (p *Provider) emitAll(ctx context.Context) {
	now := time.Now().UTC()
	p.mu.Lock()
	for i := range p.legs {
		// Random walk: nudge by up to ±jitter/2 each tick, clamp to ±jitter of base.
		step := (p.rng.Float64() - 0.5) * p.cfg.JitterPercent * p.legs[i].price
		p.legs[i].price += step
		if p.legs[i].price <= 0 {
			p.legs[i].price = -p.legs[i].price + 0.01
		}
	}
	snapshot := make([]leg, len(p.legs))
	copy(snapshot, p.legs)
	p.lastTick = now
	p.mu.Unlock()

	for _, l := range snapshot {
		spread := l.price * 0.0008
		tick := domain.Tick{
			Symbol:    l.symbol,
			Bid:       l.price - spread/2,
			Ask:       l.price + spread/2,
			Price:     l.price,
			Timestamp: now,
			Provider:  name,
		}
		select {
		case p.out <- tick:
		case <-ctx.Done():
			return
		}
	}
}
