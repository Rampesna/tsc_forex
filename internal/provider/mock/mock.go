package mock

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/domain"
)

var errClosed = errors.New("provider closed")

const name = "mock"

// Provider emits synthetic XAUUSD / USDTRY ticks for local and CI use.
type Provider struct {
	cfg config.MockProviderConfig

	out    chan domain.Tick
	cancel context.CancelFunc

	mu         sync.RWMutex
	subscribed map[string]struct{}
	lastTick   time.Time
	lastErr    string
	reconnects int64
	closed     atomic.Bool
	running    atomic.Bool

	// FailAfter, when > 0, closes the stream after N ticks (failover tests).
	FailAfter int
	tickCount int64

	rng *rand.Rand
}

// New creates a mock market provider. Connect must be called before ticks flow.
func New(cfg config.MockProviderConfig) *Provider {
	if cfg.TickInterval <= 0 {
		cfg.TickInterval = 500 * time.Millisecond
	}
	if cfg.XAUUSDBase <= 0 {
		cfg.XAUUSDBase = 2350
	}
	if cfg.USDTRYBase <= 0 {
		cfg.USDTRYBase = 34.5
	}
	if cfg.JitterPercent <= 0 {
		cfg.JitterPercent = 0.05
	}
	return &Provider{
		cfg:        cfg,
		out:        make(chan domain.Tick, 256),
		subscribed: make(map[string]struct{}),
		rng:        rand.New(rand.NewSource(time.Now().UnixNano())),
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
	p.lastErr = ""
	p.mu.Unlock()

	p.running.Store(true)
	go p.loop(runCtx)
	return nil
}

func (p *Provider) Subscribe(symbol string) error {
	if p.closed.Load() {
		return errClosed
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.subscribed[symbol] = struct{}{}
	return nil
}

func (p *Provider) PriceChannel() <-chan domain.Tick { return p.out }

func (p *Provider) Health() domain.ProviderHealth {
	p.mu.RLock()
	defer p.mu.RUnlock()

	syms := make([]string, 0, len(p.subscribed))
	for s := range p.subscribed {
		syms = append(syms, s)
	}

	status := domain.ProviderUnknown
	switch {
	case p.closed.Load():
		status = domain.ProviderUnhealthy
	case p.running.Load() && p.lastErr == "":
		status = domain.ProviderHealthy
	case p.lastErr != "":
		status = domain.ProviderDegraded
	}

	return domain.ProviderHealth{
		Name:       name,
		Status:     status,
		LastTickAt: p.lastTick,
		LastError:  p.lastErr,
		Reconnects: atomic.LoadInt64(&p.reconnects),
		Subscribed: syms,
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

	symbols := []string{domain.SymbolXAUUSD, domain.SymbolUSDTRY}
	idx := 0

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if p.closed.Load() {
				return
			}

			p.mu.RLock()
			hasSubs := len(p.subscribed) > 0
			_, subXAU := p.subscribed[domain.SymbolXAUUSD]
			_, subTRY := p.subscribed[domain.SymbolUSDTRY]
			p.mu.RUnlock()

			if hasSubs {
				switch {
				case subXAU && subTRY:
					symbols = []string{domain.SymbolXAUUSD, domain.SymbolUSDTRY}
				case subXAU:
					symbols = []string{domain.SymbolXAUUSD}
				case subTRY:
					symbols = []string{domain.SymbolUSDTRY}
				default:
					continue
				}
			}

			sym := symbols[idx%len(symbols)]
			idx++

			tick := p.makeTick(sym)
			n := atomic.AddInt64(&p.tickCount, 1)

			p.mu.Lock()
			p.lastTick = tick.Timestamp
			p.mu.Unlock()

			select {
			case p.out <- tick:
			case <-ctx.Done():
				return
			}

			if p.FailAfter > 0 && int(n) >= p.FailAfter {
				p.mu.Lock()
				p.lastErr = "simulated disconnect"
				p.mu.Unlock()
				atomic.AddInt64(&p.reconnects, 1)
				return
			}
		}
	}
}

func (p *Provider) makeTick(symbol string) domain.Tick {
	p.mu.Lock()
	jitter := 1 + (p.rng.Float64()*2-1)*p.cfg.JitterPercent
	p.mu.Unlock()

	var mid float64
	switch symbol {
	case domain.SymbolXAUUSD:
		mid = p.cfg.XAUUSDBase * jitter
	case domain.SymbolUSDTRY:
		mid = p.cfg.USDTRYBase * jitter
	default:
		mid = 1
	}

	spread := mid * 0.0002
	bid := mid - spread/2
	ask := mid + spread/2

	return domain.Tick{
		Symbol:    symbol,
		Bid:       round6(bid),
		Ask:       round6(ask),
		Price:     round6(mid),
		Timestamp: time.Now().UTC(),
		Provider:  name,
	}
}

func round6(v float64) float64 {
	return math.Round(v*1e6) / 1e6
}
