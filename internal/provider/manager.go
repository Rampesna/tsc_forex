package provider

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/domain"
	"github.com/tsc-forex/tsc_forex/pkg/eventbus"
)

// Manager owns the active MarketProvider, reconnects with exponential
// backoff, and fails over across the configured provider chain.
// Downstream consumers only see Manager.PriceChannel() — never a raw adapter.
type Manager struct {
	cfg     config.ProviderConfig
	factory Factory
	log     *zap.Logger
	bus     *eventbus.Bus

	symbols []string

	out chan domain.Tick

	mu         sync.RWMutex
	active     MarketProvider
	activeName string
	chain      []string // primary + fallbacks
	chainIdx   int

	reconnects atomic.Int64
	switches   atomic.Int64
	lastTickAt atomic.Int64 // unix nano

	cancel context.CancelFunc
	wg     sync.WaitGroup
	closed atomic.Bool
}

// ManagerOption configures optional Manager dependencies.
type ManagerOption func(*Manager)

// WithBus publishes provider health / tick fan-in events on the in-process bus.
func WithBus(bus *eventbus.Bus) ManagerOption {
	return func(m *Manager) { m.bus = bus }
}

// WithLogger sets structured logging (provider, latency, error fields).
func WithLogger(log *zap.Logger) ManagerOption {
	return func(m *Manager) {
		if log != nil {
			m.log = log
		}
	}
}

// WithSymbols overrides the default XAUUSD + USDTRY subscription set.
func WithSymbols(symbols ...string) ManagerOption {
	return func(m *Manager) {
		if len(symbols) > 0 {
			m.symbols = append([]string(nil), symbols...)
		}
	}
}

// NewManager builds a manager. Call Run to start the supervision loop.
func NewManager(cfg config.ProviderConfig, factory Factory, opts ...ManagerOption) (*Manager, error) {
	if factory == nil {
		return nil, fmt.Errorf("provider factory is required")
	}
	chain := make([]string, 0, 1+len(cfg.Fallbacks))
	chain = append(chain, cfg.Primary)
	chain = append(chain, cfg.Fallbacks...)

	m := &Manager{
		cfg:     cfg,
		factory: factory,
		log:     zap.NewNop(),
		symbols: []string{domain.SymbolXAUUSD, domain.SymbolUSDTRY},
		out:     make(chan domain.Tick, 1024),
		chain:   chain,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m, nil
}

// PriceChannel is the unified tick stream for the calculator / event bus.
func (m *Manager) PriceChannel() <-chan domain.Tick { return m.out }

// ActiveName returns the currently selected provider name.
func (m *Manager) ActiveName() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.activeName
}

// ReconnectCount is the total reconnect attempts across all providers.
func (m *Manager) ReconnectCount() int64 { return m.reconnects.Load() }

// SwitchCount is how many times the active provider changed.
func (m *Manager) SwitchCount() int64 { return m.switches.Load() }

// Health returns health of the active provider plus manager counters.
func (m *Manager) Health() domain.ProviderHealth {
	m.mu.RLock()
	active := m.active
	name := m.activeName
	m.mu.RUnlock()

	h := domain.ProviderHealth{
		Name:       name,
		Status:     domain.ProviderUnknown,
		Reconnects: m.reconnects.Load(),
		Subscribed: append([]string(nil), m.symbols...),
	}
	if active != nil {
		h = active.Health()
		h.Reconnects = m.reconnects.Load()
	}
	if ns := m.lastTickAt.Load(); ns > 0 {
		h.LastTickAt = time.Unix(0, ns).UTC()
		if m.cfg.StaleThreshold > 0 && time.Since(h.LastTickAt) > m.cfg.StaleThreshold {
			h.Status = domain.ProviderDegraded
		}
	}
	return h
}

// Statuses returns health for every provider in the failover chain.
func (m *Manager) Statuses() []domain.ProviderHealth {
	m.mu.RLock()
	active := m.active
	activeName := m.activeName
	chain := append([]string(nil), m.chain...)
	m.mu.RUnlock()

	out := make([]domain.ProviderHealth, 0, len(chain))
	for _, name := range chain {
		if active != nil && name == activeName {
			out = append(out, m.Health())
			continue
		}
		out = append(out, domain.ProviderHealth{
			Name:   name,
			Status: domain.ProviderUnknown,
		})
	}
	return out
}

// Run supervises connect → subscribe → forward → reconnect/failover until ctx cancel.
func (m *Manager) Run(ctx context.Context) error {
	if m.closed.Load() {
		return ErrClosed
	}
	runCtx, cancel := context.WithCancel(ctx)
	m.cancel = cancel

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.supervise(runCtx)
	}()
	return nil
}

// Close stops supervision and the active provider.
func (m *Manager) Close() error {
	if !m.closed.CompareAndSwap(false, true) {
		return nil
	}
	if m.cancel != nil {
		m.cancel()
	}
	m.wg.Wait()

	m.mu.Lock()
	if m.active != nil {
		_ = m.active.Close()
		m.active = nil
	}
	m.mu.Unlock()

	close(m.out)
	return nil
}

func (m *Manager) supervise(ctx context.Context) {
	backoff := Backoff{Initial: m.cfg.ReconnectInitial, Max: m.cfg.ReconnectMax}
	const maxRetries = 3
	// A session longer than this is treated as healthy; the next disconnect
	// starts a fresh retry streak instead of immediately failing over.
	stableAfter := m.cfg.HealthInterval
	if stableAfter <= 0 {
		stableAfter = 5 * time.Second
	}

	retries := 0

	for {
		if ctx.Err() != nil || m.closed.Load() {
			return
		}

		name := m.chain[m.chainIdx%len(m.chain)]
		p, err := m.factory(name)
		if err != nil {
			m.log.Error("provider factory failed",
				zap.String("provider", name),
				zap.Error(err),
			)
			m.advanceFailover()
			retries = 0
			backoff.Reset()
			m.sleep(ctx, backoff.Next())
			continue
		}

		sessionStart := time.Now()
		if err := m.activate(ctx, p); err != nil {
			m.log.Warn("provider activate failed",
				zap.String("provider", name),
				zap.Error(err),
				zap.String("backoff", backoff.String()),
			)
			_ = p.Close()
			m.publishHealth(p.Health())
			m.reconnects.Add(1)
			retries++
			if retries >= maxRetries {
				m.advanceFailover()
				retries = 0
				backoff.Reset()
			}
			m.sleep(ctx, backoff.Next())
			continue
		}

		m.log.Info("provider active",
			zap.String("provider", name),
			zap.Strings("symbols", m.symbols),
		)

		// Block until the provider stream ends or context cancels.
		m.forward(ctx, p)

		sessionDur := time.Since(sessionStart)
		m.log.Warn("provider stream ended",
			zap.String("provider", name),
			zap.Duration("session", sessionDur),
			zap.Error(ctx.Err()),
		)
		_ = p.Close()
		m.mu.Lock()
		m.active = nil
		m.mu.Unlock()

		if ctx.Err() != nil || m.closed.Load() {
			return
		}

		m.reconnects.Add(1)
		if sessionDur >= stableAfter {
			retries = 0
		}
		retries++
		if retries >= maxRetries {
			m.log.Warn("retry budget exhausted, failing over",
				zap.String("provider", name),
				zap.Int("retries", retries),
			)
			m.advanceFailover()
			retries = 0
			backoff.Reset()
		}
		m.sleep(ctx, backoff.Next())
	}
}

func (m *Manager) activate(ctx context.Context, p MarketProvider) error {
	if err := p.Connect(ctx); err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	for _, sym := range m.symbols {
		if err := p.Subscribe(sym); err != nil {
			_ = p.Close()
			return fmt.Errorf("subscribe %s: %w", sym, err)
		}
	}

	m.mu.Lock()
	prev := m.activeName
	m.active = p
	m.activeName = p.Name()
	m.mu.Unlock()

	if prev != "" && prev != p.Name() {
		m.switches.Add(1)
	}
	m.publishHealth(p.Health())
	return nil
}

func (m *Manager) forward(ctx context.Context, p MarketProvider) {
	ch := p.PriceChannel()
	// Staleness is scoped to the current upstream session. Reusing the last
	// tick timestamp from a previous session would make a fresh connection
	// look stale at its first health check and cause a reconnect loop.
	lastReceivedAt := time.Now()

	var healthC <-chan time.Time
	if m.cfg.HealthInterval > 0 {
		t := time.NewTicker(m.cfg.HealthInterval)
		defer t.Stop()
		healthC = t.C
	}

	for {
		select {
		case <-ctx.Done():
			return
		case tick, ok := <-ch:
			if !ok {
				return
			}
			lastReceivedAt = time.Now()
			m.lastTickAt.Store(tick.Timestamp.UnixNano())
			if m.bus != nil {
				m.bus.Publish(eventbus.TopicTickReceived, tick)
			}
			select {
			case m.out <- tick:
			case <-ctx.Done():
				return
			default:
				// Drop rather than block the provider read loop under backpressure.
				m.log.Warn("tick dropped: manager out channel full",
					zap.String("provider", tick.Provider),
					zap.String("symbol", tick.Symbol),
				)
			}
		case <-healthC:
			m.publishHealth(p.Health())
			// Stale stream: force reconnect so failover can engage.
			if m.cfg.StaleThreshold > 0 {
				if time.Since(lastReceivedAt) > m.cfg.StaleThreshold {
					m.log.Warn("provider stale, forcing reconnect",
						zap.String("provider", p.Name()),
						zap.Duration("stale_threshold", m.cfg.StaleThreshold),
					)
					_ = p.Close()
					return
				}
			}
		}
	}
}

func (m *Manager) advanceFailover() {
	if len(m.chain) <= 1 {
		return
	}
	m.chainIdx = (m.chainIdx + 1) % len(m.chain)
	m.switches.Add(1)
	m.log.Warn("failing over to next provider",
		zap.String("next", m.chain[m.chainIdx]),
		zap.Int64("switches", m.switches.Load()),
	)
}

func (m *Manager) publishHealth(h domain.ProviderHealth) {
	if m.bus == nil {
		return
	}
	m.bus.Publish(eventbus.TopicProviderHealthChanged, h)
}

func (m *Manager) sleep(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}
