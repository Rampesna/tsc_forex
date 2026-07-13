package provider

import (
	"fmt"
	"strings"

	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/provider/binance"
	"github.com/tsc-forex/tsc_forex/internal/provider/mock"
	"github.com/tsc-forex/tsc_forex/internal/provider/rest"
	"github.com/tsc-forex/tsc_forex/internal/provider/synthetic"
	"github.com/tsc-forex/tsc_forex/internal/provider/yahoo"
)

// Compile-time interface checks for concrete adapters.
var _ MarketProvider = (*mock.Provider)(nil)
var _ MarketProvider = (*synthetic.Provider)(nil)
var _ MarketProvider = (*rest.Provider)(nil)
var _ MarketProvider = (*binance.Provider)(nil)
var _ MarketProvider = (*yahoo.Provider)(nil)
var _ MarketProvider = (*stubProvider)(nil)

// DefaultFactory builds providers from application config.
// Unknown names return an error so misconfiguration fails at boot.
// Real adapters (TwelveData, Finnhub, …) replace stubs without changing callers.
func DefaultFactory(cfg config.ProviderConfig) Factory {
	return func(name string) (MarketProvider, error) {
		switch strings.ToLower(strings.TrimSpace(name)) {
		case "mock":
			return mock.New(cfg.Mock), nil
		case "synthetic":
			return synthetic.New(cfg.Synthetic), nil
		case "rest":
			return rest.New(cfg.REST), nil
		case "binance":
			return binance.New(cfg.REST), nil
		case "yahoo":
			return yahoo.New(cfg.REST), nil
		case "twelvedata":
			return newStub("twelvedata", cfg.TwelveData), nil
		case "finnhub":
			return newStub("finnhub", cfg.Finnhub), nil
		case "polygon":
			return newStub("polygon", cfg.Polygon), nil
		case "alphavantage":
			return newStub("alphavantage", cfg.AlphaVantage), nil
		default:
			return nil, fmt.Errorf("unknown provider %q", name)
		}
	}
}
