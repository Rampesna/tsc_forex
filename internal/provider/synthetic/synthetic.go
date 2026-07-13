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
		cfg:  cfg,
		out:  make(chan domain.Tick, 1024),
		legs: buildLegs(cfg),
		rng:  rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// approxUSDRate is roughly "units of currency per 1 USD" — used only to seed
// realistic offline synthetic prices. Real prices come from live providers.
var approxUSDRate = map[string]float64{
	"USD": 1, "EUR": 0.92, "GBP": 0.79, "JPY": 157, "CHF": 0.88, "CAD": 1.37,
	"AUD": 1.52, "NZD": 1.64, "CNY": 7.25, "RUB": 90, "SAR": 3.75, "AED": 3.67,
	"KWD": 0.31, "QAR": 3.64, "BHD": 0.38, "OMR": 0.38, "JOD": 0.71, "ILS": 3.7,
	"INR": 83.5, "PKR": 278, "IQD": 1310, "EGP": 48, "ZAR": 18.5, "SEK": 10.6,
	"NOK": 10.7, "DKK": 6.9, "PLN": 3.95, "CZK": 23, "HUF": 360, "RON": 4.6,
	"BGN": 1.8, "UAH": 41, "KRW": 1360, "SGD": 1.35, "HKD": 7.8, "TWD": 32.5,
	"THB": 36, "MYR": 4.7, "IDR": 16000, "PHP": 58, "VND": 25400, "MXN": 18.5,
	"BRL": 5.4, "ARS": 900, "CLP": 950, "COP": 4000, "AZN": 1.7, "KZT": 470,
	"GEL": 2.7, "MAD": 9.9, "DZD": 134, "TND": 3.1, "NGN": 1500, "KES": 129,
	"UZS": 12600, "ISK": 138,
}

// buildLegs seeds XAUUSD plus every world currency vs TRY. Majors use the
// configured bases; the rest derive from USDTRYBase × approximate cross rates.
func buildLegs(cfg config.SyntheticConfig) []leg {
	usdtry := cfg.USDTRYBase
	legs := []leg{{symbol: domain.SymbolXAUUSD, price: cfg.XAUUSDBase}}
	for _, c := range domain.Currencies {
		var price float64
		switch c.ISO {
		case "USD":
			price = cfg.USDTRYBase
		case "EUR":
			price = cfg.EURTRYBase
		case "GBP":
			price = cfg.GBPTRYBase
		case "JPY":
			price = cfg.JPYTRYBase
		default:
			if r := approxUSDRate[c.ISO]; r > 0 {
				price = usdtry / r
			} else {
				price = usdtry // safe fallback so the leg is always positive
			}
		}
		legs = append(legs, leg{symbol: c.ISO + "TRY", price: price})
	}
	return legs
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
