package domain_test

import (
	"testing"

	"github.com/tsc-forex/tsc_forex/internal/domain"
)

func TestCatalogIntegrity(t *testing.T) {
	t.Parallel()
	if len(domain.Catalog) != len(domain.CatalogByCode) {
		t.Fatalf("catalog index mismatch: %d vs %d", len(domain.Catalog), len(domain.CatalogByCode))
	}
	for _, in := range domain.Catalog {
		got, ok := domain.CatalogByCode[in.Code]
		if !ok || got.Code != in.Code {
			t.Fatalf("missing index for %s", in.Code)
		}
		if in.Kind == domain.KindGold && (in.GramWeight <= 0 || in.Fineness <= 0) {
			t.Fatalf("gold %s has invalid weight/fineness", in.Code)
		}
		if in.Kind == domain.KindCurrency && in.Base == "" {
			t.Fatalf("currency %s missing base symbol", in.Code)
		}
	}
	// Base symbols = XAUUSD + every world currency vs TRY.
	if want := 1 + len(domain.Currencies); len(domain.BaseSymbols()) != want {
		t.Fatalf("expected %d base symbols, got %d", want, len(domain.BaseSymbols()))
	}
}

func TestComputeCurrency(t *testing.T) {
	t.Parallel()
	in := domain.CatalogByCode["USDTRY"]
	price, bid, ask, ok := domain.ComputeInstrumentPrice(in, 0, 0, map[string]float64{"USDTRY": 34.5})
	if !ok || price != 34.5 {
		t.Fatalf("price = %v ok=%v", price, ok)
	}
	if !(bid < price && ask > price) {
		t.Fatalf("bid/ask invalid: %v/%v", bid, ask)
	}
}

func TestComputeGoldProducts(t *testing.T) {
	t.Parallel()
	xau, usd := 2350.0, 34.5
	gramHas := (xau / domain.TroyOunceToGram) * usd

	gram := domain.CatalogByCode["GRAM_GOLD"]
	p, _, _, ok := domain.ComputeInstrumentPrice(gram, xau, usd, nil)
	if !ok {
		t.Fatal("gram compute failed")
	}
	want := gramHas * 1.0 * domain.Fineness24K
	if diff := p - want; diff > 1e-6 || diff < -1e-6 {
		t.Fatalf("gram price = %v want %v", p, want)
	}

	// Çeyrek should be larger than gram (heavier).
	ceyrek := domain.CatalogByCode["CEYREK_GOLD"]
	pc, _, _, _ := domain.ComputeInstrumentPrice(ceyrek, xau, usd, nil)
	if pc <= p {
		t.Fatalf("ceyrek %v should exceed gram %v", pc, p)
	}

	// Invalid legs → not ok.
	if _, _, _, ok := domain.ComputeInstrumentPrice(gram, 0, usd, nil); ok {
		t.Fatal("expected failure with zero xau")
	}
}
