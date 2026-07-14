package yahoo

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/domain"
)

// TestLiveFetchBreadth hits the real Yahoo + er-api endpoints. It only runs when
// TSC_LIVE=1 so CI (no network / rate limits) never flakes on it.
func TestLiveFetchBreadth(t *testing.T) {
	if os.Getenv("TSC_LIVE") != "1" {
		t.Skip("live network test; set TSC_LIVE=1 to run")
	}
	p := New(config.RESTProviderConfig{Timeout: 8 * time.Second})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	ticks, err := p.Fetch(ctx)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	seen := map[string]float64{}
	for _, tk := range ticks {
		seen[tk.Symbol] = tk.Price
	}
	t.Logf("got %d ticks; USDTRY=%v XAUUSD=%v EURTRY=%v CHFTRY=%v INRTRY=%v",
		len(ticks), seen[domain.SymbolUSDTRY], seen[domain.SymbolXAUUSD],
		seen["EURTRY"], seen["CHFTRY"], seen["INRTRY"])
	t.Logf("crypto TRY: BTC=%v ETH=%v SOL=%v | stocks TRY: THYAO=%v GARAN=%v ASELS=%v",
		seen["BTC"], seen["ETH"], seen["SOL"], seen["THYAO"], seen["GARAN"], seen["ASELS"])
	if len(ticks) < 40 {
		t.Fatalf("expected broad currency coverage, got %d ticks", len(ticks))
	}
}
