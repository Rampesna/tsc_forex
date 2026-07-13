package domain

import "testing"

func TestCalculateGramGoldTRY(t *testing.T) {
	t.Parallel()

	got := CalculateGramGoldTRY(3100.0, 34.0)
	want := (3100.0 / TroyOunceToGram) * 34.0
	if got != want {
		t.Fatalf("CalculateGramGoldTRY = %v, want %v", got, want)
	}

	if CalculateGramGoldTRY(0, 34) != 0 {
		t.Fatal("expected 0 for non-positive XAUUSD")
	}
	if CalculateGramGoldTRY(3100, -1) != 0 {
		t.Fatal("expected 0 for non-positive USDTRY")
	}
}

func TestTickMidAndSpread(t *testing.T) {
	t.Parallel()

	tick := Tick{Bid: 100, Ask: 102, Price: 101}
	if mid := tick.Mid(); mid != 101 {
		t.Fatalf("Mid = %v, want 101", mid)
	}
	if spread := tick.Spread(); spread != 2 {
		t.Fatalf("Spread = %v, want 2", spread)
	}

	priceOnly := Tick{Price: 99}
	if priceOnly.Mid() != 99 {
		t.Fatalf("Mid with price only = %v, want 99", priceOnly.Mid())
	}
	if priceOnly.Spread() != 0 {
		t.Fatalf("Spread with price only = %v, want 0", priceOnly.Spread())
	}
}
