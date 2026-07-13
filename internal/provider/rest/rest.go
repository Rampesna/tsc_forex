package rest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/domain"
)

var errClosed = errors.New("provider closed")

const name = "rest"

// Provider polls external REST APIs for FX and gold, deriving all base legs
// (USDTRY, EURTRY, GBPTRY, JPYTRY, XAUUSD) every PollInterval. It is the
// production path for real market data; the synthetic provider is the offline
// fallback. Failures are surfaced via Health so the Manager can fail over.
type Provider struct {
	cfg    config.RESTProviderConfig
	client *http.Client

	out    chan domain.Tick
	cancel context.CancelFunc

	mu         sync.RWMutex
	lastTick   time.Time
	lastErr    string
	reconnects int64
	closed     atomic.Bool
	running    atomic.Bool
}

// erAPIResponse matches the open.er-api.com / exchangerate.host "latest" shape.
type erAPIResponse struct {
	Result string             `json:"result"`
	Rates  map[string]float64 `json:"rates"`
}

// goldResponse is a permissive shape for a gold source; the first non-zero of
// these fields is used as XAU price in USD (per troy ounce).
type goldResponse struct {
	XAUUSD float64 `json:"XAUUSD"`
	Price  float64 `json:"price"`
	Gold   float64 `json:"gold"`
}

// New builds a REST polling provider.
func New(cfg config.RESTProviderConfig) *Provider {
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 5 * time.Second
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 4 * time.Second
	}
	if cfg.FXBaseURL == "" {
		cfg.FXBaseURL = "https://open.er-api.com/v6/latest/USD"
	}
	return &Provider{
		cfg:    cfg,
		client: &http.Client{Timeout: cfg.Timeout},
		out:    make(chan domain.Tick, 512),
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
	case p.lastErr != "":
		status = domain.ProviderDegraded
	case p.running.Load():
		status = domain.ProviderHealthy
	}
	return domain.ProviderHealth{
		Name:       name,
		Status:     status,
		LastTickAt: p.lastTick,
		LastError:  p.lastErr,
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

	ticker := time.NewTicker(p.cfg.PollInterval)
	defer ticker.Stop()

	p.pollOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if p.closed.Load() {
				return
			}
			p.pollOnce(ctx)
		}
	}
}

func (p *Provider) pollOnce(ctx context.Context) {
	ticks, err := p.Fetch(ctx)
	if err != nil {
		p.mu.Lock()
		p.lastErr = err.Error()
		p.mu.Unlock()
		atomic.AddInt64(&p.reconnects, 1)
		return
	}
	now := time.Now().UTC()
	p.mu.Lock()
	p.lastErr = ""
	p.lastTick = now
	p.mu.Unlock()

	for _, t := range ticks {
		select {
		case p.out <- t:
		case <-ctx.Done():
			return
		}
	}
}

// Fetch performs a single poll and returns the derived base-leg ticks.
// Exported for testing against httptest servers.
func (p *Provider) Fetch(ctx context.Context) ([]domain.Tick, error) {
	fx, err := p.fetchFX(ctx)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	out := make([]domain.Tick, 0, 5)
	add := func(sym string, price float64) {
		if price <= 0 {
			return
		}
		spread := price * 0.001
		out = append(out, domain.Tick{
			Symbol:    sym,
			Bid:       price - spread/2,
			Ask:       price + spread/2,
			Price:     price,
			Timestamp: now,
			Provider:  name,
		})
	}

	// open.er-api base=USD → rates are "target per 1 USD". Derive every world
	// currency vs TRY: XXX/TRY = (TRY per USD) / (XXX per USD).
	tryPerUSD := fx["TRY"]
	add(domain.SymbolUSDTRY, tryPerUSD)
	for _, c := range domain.Currencies {
		if c.ISO == "USD" {
			continue
		}
		if rate := fx[c.ISO]; rate > 0 {
			add(c.ISO+"TRY", tryPerUSD/rate)
		}
	}

	if p.cfg.GoldURL != "" {
		if xau, err := p.fetchGold(ctx); err == nil {
			add(domain.SymbolXAUUSD, xau)
		}
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("no rates parsed from FX response")
	}
	return out, nil
}

func (p *Provider) fetchFX(ctx context.Context) (map[string]float64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.cfg.FXBaseURL, nil)
	if err != nil {
		return nil, err
	}
	if p.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fx http %d", resp.StatusCode)
	}
	var r erAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, err
	}
	if len(r.Rates) == 0 {
		return nil, fmt.Errorf("empty rates")
	}
	return r.Rates, nil
}

func (p *Provider) fetchGold(ctx context.Context) (float64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.cfg.GoldURL, nil)
	if err != nil {
		return 0, err
	}
	if p.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("gold http %d", resp.StatusCode)
	}
	var g goldResponse
	if err := json.NewDecoder(resp.Body).Decode(&g); err != nil {
		return 0, err
	}
	for _, v := range []float64{g.XAUUSD, g.Price, g.Gold} {
		if v > 0 {
			return v, nil
		}
	}
	return 0, fmt.Errorf("no gold price field")
}
