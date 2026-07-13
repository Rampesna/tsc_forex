package yahoo

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

const name = "yahoo"

// Yahoo Finance public chart endpoint (no API key required, reachable where
// Binance is blocked). Each symbol is polled and the latest regularMarketPrice
// is emitted as a live tick.
//
//   - GC=F      : COMEX gold futures (USD / troy ounce) → XAUUSD
//   - USDTRY=X  : USD/TRY spot
//   - EURTRY=X  : EUR/TRY spot
//   - GBPTRY=X  : GBP/TRY spot
//   - JPYTRY=X  : JPY/TRY spot
//
// Yahoo quotes the TRY pairs directly, so no cross-rate derivation is needed.
var yahooChartBase = "https://query1.finance.yahoo.com/v8/finance/chart/"

// symbolMap maps our canonical base symbols to Yahoo tickers.
var symbolMap = []struct {
	canonical string
	yahoo     string
}{
	{domain.SymbolXAUUSD, "GC=F"},
	{domain.SymbolUSDTRY, "USDTRY=X"},
	{domain.SymbolEURTRY, "EURTRY=X"},
	{domain.SymbolGBPTRY, "GBPTRY=X"},
	{domain.SymbolJPYTRY, "JPYTRY=X"},
}

// Provider polls Yahoo Finance for the full base-leg set. Implements
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
}

// yahooChart is the minimal shape we parse from the chart endpoint.
type yahooChart struct {
	Chart struct {
		Result []struct {
			Meta struct {
				RegularMarketPrice float64 `json:"regularMarketPrice"`
				Currency           string  `json:"currency"`
			} `json:"meta"`
		} `json:"result"`
		Error *struct {
			Code        string `json:"code"`
			Description string `json:"description"`
		} `json:"error"`
	} `json:"chart"`
}

// New builds a Yahoo Finance polling provider.
func New(cfg config.RESTProviderConfig) *Provider {
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 2 * time.Second
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 6 * time.Second
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

// Fetch polls all symbols concurrently and returns the derived base-leg ticks.
// Exported for testing against httptest servers.
func (p *Provider) Fetch(ctx context.Context) ([]domain.Tick, error) {
	type res struct {
		canonical string
		price     float64
		err       error
	}
	results := make(chan res, len(symbolMap))
	var wg sync.WaitGroup
	for _, m := range symbolMap {
		wg.Add(1)
		go func(canonical, yahooSym string) {
			defer wg.Done()
			price, err := p.fetchSymbol(ctx, yahooSym)
			results <- res{canonical: canonical, price: price, err: err}
		}(m.canonical, m.yahoo)
	}
	go func() { wg.Wait(); close(results) }()

	now := time.Now().UTC()
	out := make([]domain.Tick, 0, len(symbolMap))
	var firstErr error
	for r := range results {
		if r.err != nil {
			if firstErr == nil {
				firstErr = r.err
			}
			continue
		}
		if r.price <= 0 {
			continue
		}
		spread := r.price * 0.001
		out = append(out, domain.Tick{
			Symbol:    r.canonical,
			Bid:       r.price - spread/2,
			Ask:       r.price + spread/2,
			Price:     r.price,
			Timestamp: now,
			Provider:  name,
		})
	}
	if len(out) == 0 {
		if firstErr != nil {
			return nil, firstErr
		}
		return nil, fmt.Errorf("no prices parsed from yahoo response")
	}
	return out, nil
}

func (p *Provider) fetchSymbol(ctx context.Context, yahooSym string) (float64, error) {
	reqURL := yahooChartBase + yahooSym + "?interval=1m&range=1d"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return 0, err
	}
	// Yahoo rejects requests without a browser-like User-Agent.
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("yahoo http %d for %s", resp.StatusCode, yahooSym)
	}
	var chart yahooChart
	if err := json.NewDecoder(resp.Body).Decode(&chart); err != nil {
		return 0, err
	}
	if chart.Chart.Error != nil {
		return 0, fmt.Errorf("yahoo error for %s: %s", yahooSym, chart.Chart.Error.Code)
	}
	if len(chart.Chart.Result) == 0 {
		return 0, fmt.Errorf("yahoo empty result for %s", yahooSym)
	}
	price := chart.Chart.Result[0].Meta.RegularMarketPrice
	if price <= 0 {
		return 0, fmt.Errorf("yahoo zero price for %s", yahooSym)
	}
	return price, nil
}
