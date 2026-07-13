package domain

import "time"

// Tick represents a single market price update from any provider.
type Tick struct {
	Symbol    string
	Bid       float64
	Ask       float64
	Price     float64
	Timestamp time.Time
	Provider  string
}

// Mid returns the mid price. Prefers Bid/Ask when both are positive.
func (t Tick) Mid() float64 {
	if t.Bid > 0 && t.Ask > 0 {
		return (t.Bid + t.Ask) / 2
	}
	return t.Price
}

// Spread returns Ask - Bid when both sides are present; otherwise 0.
func (t Tick) Spread() float64 {
	if t.Bid > 0 && t.Ask > 0 {
		return t.Ask - t.Bid
	}
	return 0
}
