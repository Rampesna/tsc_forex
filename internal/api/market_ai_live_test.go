package api

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/domain"
)

// This test is skipped in normal CI. Run it manually with OPENAI_API_KEY to
// validate the production Responses API schema, web search and citations.
func TestMarketResearchLive(t *testing.T) {
	if os.Getenv("TSC_LIVE_AI_TEST") != "1" {
		t.Skip("set TSC_LIVE_AI_TEST=1 to run the paid live research test")
	}
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		t.Skip("OPENAI_API_KEY is not set")
	}
	handler := NewMarketAIHandler(config.AIConfig{
		Enabled: true, APIKey: key, Model: "gpt-5.6-sol",
		BaseURL: "https://api.openai.com/v1", Timeout: 30 * time.Second,
		MaxOutputTokens: 8000, ReasoningEffort: "high", WebSearch: true,
		SearchContext: "high", MaxToolCalls: 4,
	}, nil)
	price := domain.InstrumentPrice{
		Code: "GRAM_GOLD", DisplayName: "Gram Altın (Has)", Kind: "gold",
		Price: 6014.85, PriceTRY: 6014.85, Bid: 6011.84, Ask: 6017.86,
		ChangePct: -1.73, Currency: "TRY", Provider: "test", Timestamp: time.Now().UTC(),
	}
	day, week, month, quarter, year := -1.73, -0.8, 3.4, 11.2, 48.6
	contextData := []MarketDataPoint{{
		Code: price.Code, DisplayName: price.DisplayName, Price: price.Price, Currency: price.Currency,
		Change1D: &day, Change1W: &week, Change1M: &month, Change3M: &quarter, Change1Y: &year,
		Timestamp: price.Timestamp.Format(time.RFC3339),
	}}
	envelope, err := handler.startBackground(context.Background(), price, contextData, "tsc_live_schema_test")
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(6 * time.Minute)
	for envelope.Status == "queued" || envelope.Status == "in_progress" {
		if time.Now().After(deadline) {
			t.Fatalf("background response %s did not complete; status=%s", envelope.ID, envelope.Status)
		}
		time.Sleep(3 * time.Second)
		envelope, err = handler.retrieveBackground(context.Background(), envelope.ID)
		if err != nil {
			t.Fatal(err)
		}
	}
	result, err := handler.finalizeResponse(envelope, price, contextData)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.TimeHorizons) != 3 || len(result.Scenarios) != 3 || len(result.Sources) < 3 {
		t.Fatalf("incomplete research: horizons=%d scenarios=%d sources=%d", len(result.TimeHorizons), len(result.Scenarios), len(result.Sources))
	}
}
