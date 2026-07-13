package provider

import (
	"context"
	"errors"

	"github.com/tsc-forex/tsc_forex/internal/domain"
)

// ErrNotConfigured is returned when a provider adapter is registered but not ready.
var ErrNotConfigured = errors.New("provider not configured")

// ErrClosed is returned when operating on a closed provider.
var ErrClosed = errors.New("provider closed")

// MarketProvider is the swappable market-data source contract.
// Concrete adapters (Mock, TwelveData, Finnhub, Polygon, AlphaVantage)
// must implement this interface — never hardcode provider logic in services.
type MarketProvider interface {
	// Name returns a stable identifier (e.g. "mock", "twelvedata").
	Name() string

	// Connect establishes the upstream session (WS preferred, REST fallback).
	Connect(ctx context.Context) error

	// Subscribe requests updates for a canonical symbol (XAUUSD, USDTRY, …).
	Subscribe(symbol string) error

	// PriceChannel returns the inbound tick stream. Closed when the provider stops.
	PriceChannel() <-chan domain.Tick

	// Health returns a point-in-time operational snapshot.
	Health() domain.ProviderHealth

	// Close releases resources. Idempotent.
	Close() error
}

// Factory constructs a named provider from application config.
type Factory func(name string) (MarketProvider, error)
