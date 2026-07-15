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
	dayOpen  map[string]float64 // instrument code → first price of the day (baseline)
	dayKey   string             // current day (Istanbul) the baselines belong to
	loc      *time.Location     // market day timezone (Europe/Istanbul)
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
	// Market day boundary in Istanbul time (change % is measured against the
	// first price observed after local midnight). Fall back to UTC if the tzdata
	// is unavailable in the runtime image.
	loc, err := time.LoadLocation("Europe/Istanbul")
	if err != nil {
		loc = time.UTC
	}
	e := &Engine{
		bus:      bus,
		log:      log,
		epsilon:  1e-6,
		currency: make(map[string]float64),
		prev:     make(map[string]float64),
		dayOpen:  make(map[string]float64),
		loc:      loc,
	}
	e.latest.Store(map[string]domain.InstrumentPrice{})
	return e
}

// SeedDayOpen sets the per-instrument day-open baselines (loaded from the DB at
// startup) for the given Istanbul day key (YYYY-MM-DD). Safe to call before the
// engine starts receiving ticks so change % survives restarts.
func (e *Engine) SeedDayOpen(opens map[string]float64, dayKey string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	for code, v := range opens {
		if v > 0 {
			e.dayOpen[code] = v
		}
	}
	if dayKey != "" {
		e.dayKey = dayKey
	}
}

// DayKey returns today's market-day key in the engine timezone.
func (e *Engine) DayKey() string { return time.Now().In(e.loc).Format("2006-01-02") }

// DayStart returns local (Istanbul) midnight for today, in UTC — the lower
// bound for loading today's opening prices from the DB.
func (e *Engine) DayStart() time.Time {
	now := time.Now().In(e.loc)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, e.loc).UTC()
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
	switch {
	case t.Symbol == domain.SymbolXAUUSD:
		e.xau = t.Mid()
	case t.Symbol == domain.SymbolUSDTRY:
		e.usd = t.Mid()
		e.currency[domain.SymbolUSDTRY] = t.Mid()
	case domain.IsDirectTRYSymbol(t.Symbol):
		// Currencies, crypto and BIST equities all carry a direct TRY value.
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

	// Roll the day-open baselines at local (Istanbul) midnight so change % is
	// measured from the day's first price.
	if dk := now.In(e.loc).Format("2006-01-02"); dk != e.dayKey {
		e.dayKey = dk
		e.dayOpen = make(map[string]float64, len(domain.Catalog))
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
		// Skip immaterial moves using a RELATIVE threshold so both high-value
		// (BTC ~2.5M) and micro-priced (SHIB ~0.0002) instruments update: an
		// absolute epsilon would freeze SHIB whose per-tick move is < 1e-6.
		if had && last != 0 {
			threshold := e.epsilon * math.Abs(last)
			if threshold < 1e-9 {
				threshold = 1e-9
			}
			if math.Abs(price-last) < threshold {
				continue
			}
		}
		e.prev[in.Code] = price
		// Establish the day-open baseline on the first price seen today, then
		// report change relative to it (day open → latest), not tick-over-tick.
		open, hasOpen := e.dayOpen[in.Code]
		if !hasOpen || open == 0 {
			open = price
			e.dayOpen[in.Code] = price
		}
		var change, changePct float64
		if open != 0 {
			change = price - open
			changePct = (change / open) * 100
		}
		// Native currency + TRY equivalent. Global equities are quoted in USD;
		// everything else is already TRY. The TRY figure powers the "≈ ₺X" sub-line.
		ccy := in.Currency
		if ccy == "" {
			ccy = "TRY"
		}
		priceTRY := price
		if ccy == "USD" && usd > 0 {
			priceTRY = price * usd
		}
		ip := domain.InstrumentPrice{
			Code:        in.Code,
			DisplayName: in.DisplayName,
			Kind:        string(in.Kind),
			Price:       price,
			Bid:         bid,
			Ask:         ask,
			Change:      change,
			ChangePct:   changePct,
			Currency:    ccy,
			PriceTRY:    priceTRY,
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
