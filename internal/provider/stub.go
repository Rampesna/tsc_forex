package provider

import (
	"context"

	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/domain"
)

// stubProvider is a placeholder adapter for future real feeds.
// It satisfies MarketProvider so the factory and failover chain can
// reference the name before the WS/REST client is implemented.
type stubProvider struct {
	name string
	cfg  config.APIProviderConfig
}

func newStub(name string, cfg config.APIProviderConfig) *stubProvider {
	return &stubProvider{name: name, cfg: cfg}
}

func (p *stubProvider) Name() string { return p.name }

func (p *stubProvider) Connect(context.Context) error { return ErrNotConfigured }

func (p *stubProvider) Subscribe(string) error { return ErrNotConfigured }

func (p *stubProvider) PriceChannel() <-chan domain.Tick {
	ch := make(chan domain.Tick)
	close(ch)
	return ch
}

func (p *stubProvider) Health() domain.ProviderHealth {
	return domain.ProviderHealth{
		Name:      p.name,
		Status:    domain.ProviderUnhealthy,
		LastError: ErrNotConfigured.Error(),
	}
}

func (p *stubProvider) Close() error { return nil }

var _ MarketProvider = (*stubProvider)(nil)
