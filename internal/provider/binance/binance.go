package binance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/domain"
)

var errClosed = errors.New("provider closed")

const name = "binance"

// Binance public endpoints (no API key required).
//   - PAXGUSDT: PAX Gold, 1 token ≈ 1 fine troy ounce of gold → real-time XAUUSD.
//   - USDTTRY:  Tether/TRY ≈ USD/TRY, real-time.
// These update on every trade, giving a genuinely LIVE ticker. The slower
// EUR/GBP/JPY-per-USD cross ratios come from a free FX source (er-api) and are
// refreshed periodically; because the TRY leg (USDTTRY) is live, every derived
// TRY pair moves in real time too.
const (
	symbolGold     = "PAXGUSDT"
	symbolUSDTRY   = "USDTTRY"
	fxRefreshEvery = 10 * time.Minute
)

// binanceBatchURL is a var (not const) so tests can point it at an httptest server.
var binanceBatchURL = "https://api.binance.com/api/v3/ticker/price"

// Provider polls Binance for the live gold + USDTRY legs and derives the full
// base-leg set (XAUUSD, USDTRY, EURTRY, GBPTRY, JPYTRY). Implements
// provider.MarketProvider.
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

	// cached FX cross ratios (target currency per 1 USD).
	fxMu      sync.RWMutex
	eurPerUSD float64
	gbpPerUSD float64
	jpyPerUSD float64
	fxAt      time.Time
}

type binanceTicker struct {
	Symbol string `json:"symbol"`
	Price  string `json:"price"`
}

type erAPIResponse struct {
	Result string             `json:"result"`
	Rates  map[string]float64 `json:"rates"`
}

// New builds a Binance polling provider.
func New(cfg config.RESTProviderConfig) *Provider {
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 2 * time.Second
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

	// Prime FX cross ratios once up-front (best effort).
	_ = p.refreshFX(ctx)

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
	// Refresh slow FX ratios occasionally.
	p.fxMu.RLock()
	stale := time.Since(p.fxAt) > fxRefreshEvery || p.eurPerUSD == 0
	p.fxMu.RUnlock()
	if stale {
		_ = p.refreshFX(ctx)
	}

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

// Fetch performs a single poll of Binance and returns the derived base-leg
// ticks. Exported for testing against httptest servers.
func (p *Provider) Fetch(ctx context.Context) ([]domain.Tick, error) {
	xau, usdtry, err := p.fetchBinance(ctx)
	if err != nil {
		return nil, err
	}

	p.fxMu.RLock()
	eur, gbp, jpy := p.eurPerUSD, p.gbpPerUSD, p.jpyPerUSD
	p.fxMu.RUnlock()

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

	add(domain.SymbolXAUUSD, xau)
	add(domain.SymbolUSDTRY, usdtry)
	if eur > 0 {
		add(domain.SymbolEURTRY, usdtry/eur)
	}
	if gbp > 0 {
		add(domain.SymbolGBPTRY, usdtry/gbp)
	}
	if jpy > 0 {
		add(domain.SymbolJPYTRY, usdtry/jpy)
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("no prices parsed from binance response")
	}
	return out, nil
}

func (p *Provider) fetchBinance(ctx context.Context) (xau, usdtry float64, err error) {
	q := url.Values{}
	q.Set("symbols", `["`+symbolGold+`","`+symbolUSDTRY+`"]`)
	reqURL := binanceBatchURL + "?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return 0, 0, err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("binance http %d", resp.StatusCode)
	}
	var tickers []binanceTicker
	if err := json.NewDecoder(resp.Body).Decode(&tickers); err != nil {
		return 0, 0, err
	}
	for _, t := range tickers {
		v, perr := strconv.ParseFloat(t.Price, 64)
		if perr != nil {
			continue
		}
		switch t.Symbol {
		case symbolGold:
			xau = v
		case symbolUSDTRY:
			usdtry = v
		}
	}
	if xau <= 0 || usdtry <= 0 {
		return 0, 0, fmt.Errorf("binance missing PAXGUSDT/USDTTRY prices")
	}
	return xau, usdtry, nil
}

func (p *Provider) refreshFX(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.cfg.FXBaseURL, nil)
	if err != nil {
		return err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fx http %d", resp.StatusCode)
	}
	var r erAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return err
	}
	if len(r.Rates) == 0 {
		return fmt.Errorf("empty fx rates")
	}
	p.fxMu.Lock()
	if v := r.Rates["EUR"]; v > 0 {
		p.eurPerUSD = v
	}
	if v := r.Rates["GBP"]; v > 0 {
		p.gbpPerUSD = v
	}
	if v := r.Rates["JPY"]; v > 0 {
		p.jpyPerUSD = v
	}
	p.fxAt = time.Now()
	p.fxMu.Unlock()
	return nil
}
