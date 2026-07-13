package calculator

import (
	"math"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"github.com/tsc-forex/tsc_forex/internal/domain"
	"github.com/tsc-forex/tsc_forex/pkg/eventbus"
)

// Engine maintains the latest base legs (XAUUSD, USDTRY, EURTRY, GBPTRY,
// JPYTRY) and recomputes the FULL instrument catalog (all gold products +
// currencies) whenever any leg changes. Materially changed instruments are
// published on TopicInstrumentUpdated for broadcast / persistence / alerts.
//
// It subscribes to the in-process bus (TopicTickReceived) rather than owning
// a channel, so it coexists with the flagship gram-gold Calculator.
type Engine struct {
	bus     *eventbus.Bus
	log     *zap.Logger
	epsilon float64

	mu       sync.Mutex
	xau      float64
	usd      float64
	currency map[string]float64 // base symbol (e.g. USDTRY) → TRY rate
	prev     map[string]float64 // instrument code → last published price
	provider string

	// latest is an atomic snapshot for REST readers.
	latest atomic.Value // map[string]domain.InstrumentPrice

	updates atomic.Int64
}

// NewEngine builds a multi-instrument engine. Call Attach to wire the bus.
func NewEngine(bus *eventbus.Bus, log *zap.Logger) *Engine {
	if log == nil {
		log = zap.NewNop()
	}
	e := &Engine{
		bus:      bus,
		log:      log,
		epsilon:  1e-6,
		currency: make(map[string]float64),
		prev:     make(map[string]float64),
	}
	e.latest.Store(map[string]domain.InstrumentPrice{})
	return e
}

// Attach subscribes the engine to tick events on the bus.
func (e *Engine) Attach() {
	if e.bus == nil {
		return
	}
	e.bus.Subscribe(eventbus.TopicTickReceived, func(ev eventbus.Event) {
		tick, ok := ev.Payload.(domain.Tick)
		if !ok {
			return
		}
		e.onTick(tick)
	})
}

// UpdateCount is the number of instrument updates published.
func (e *Engine) UpdateCount() int64 { return e.updates.Load() }

// Snapshot returns all currently-known instrument prices sorted by code.
func (e *Engine) Snapshot() []domain.InstrumentPrice {
	m, _ := e.latest.Load().(map[string]domain.InstrumentPrice)
	out := make([]domain.InstrumentPrice, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	return out
}

// Get returns the latest price for one instrument code.
func (e *Engine) Get(code string) (domain.InstrumentPrice, bool) {
	m, _ := e.latest.Load().(map[string]domain.InstrumentPrice)
	p, ok := m[code]
	return p, ok
}

func (e *Engine) onTick(t domain.Tick) {
	e.mu.Lock()
	switch t.Symbol {
	case domain.SymbolXAUUSD:
		e.xau = t.Mid()
	case domain.SymbolUSDTRY:
		e.usd = t.Mid()
		e.currency[domain.SymbolUSDTRY] = t.Mid()
	case domain.SymbolEURTRY, domain.SymbolGBPTRY, domain.SymbolJPYTRY:
		e.currency[t.Symbol] = t.Mid()
	default:
		e.mu.Unlock()
		return
	}
	if t.Provider != "" {
		e.provider = t.Provider
	}

	xau, usd := e.xau, e.usd
	provider := e.provider
	// Copy currency map for lock-free compute.
	cur := make(map[string]float64, len(e.currency))
	for k, v := range e.currency {
		cur[k] = v
	}

	now := t.Timestamp
	if now.IsZero() {
		now = time.Now().UTC()
	}

	changed := make([]domain.InstrumentPrice, 0, 4)
	// Rebuild the latest map (copy-on-write for atomic snapshot).
	prevMap, _ := e.latest.Load().(map[string]domain.InstrumentPrice)
	next := make(map[string]domain.InstrumentPrice, len(prevMap)+len(domain.Catalog))
	for k, v := range prevMap {
		next[k] = v
	}

	for _, in := range domain.Catalog {
		price, bid, ask, ok := domain.ComputeInstrumentPrice(in, xau, usd, cur)
		if !ok {
			continue
		}
		last, had := e.prev[in.Code]
		if had && math.Abs(price-last) < e.epsilon {
			continue
		}
		var change, changePct float64
		if had && last != 0 {
			change = price - last
			changePct = (change / last) * 100
		}
		e.prev[in.Code] = price
		ip := domain.InstrumentPrice{
			Code:        in.Code,
			DisplayName: in.DisplayName,
			Kind:        string(in.Kind),
			Price:       price,
			Bid:         bid,
			Ask:         ask,
			Change:      change,
			ChangePct:   changePct,
			Provider:    provider,
			Timestamp:   now.UTC(),
		}
		next[in.Code] = ip
		changed = append(changed, ip)
	}
	e.latest.Store(next)
	e.mu.Unlock()

	for _, ip := range changed {
		e.updates.Add(1)
		if e.bus != nil {
			e.bus.Publish(eventbus.TopicInstrumentUpdated, ip)
		}
	}
}
