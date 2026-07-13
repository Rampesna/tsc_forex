package domain

import "time"

// ProviderHealthStatus describes connectivity of a market data provider.
type ProviderHealthStatus string

const (
	ProviderHealthy   ProviderHealthStatus = "healthy"
	ProviderDegraded  ProviderHealthStatus = "degraded"
	ProviderUnhealthy ProviderHealthStatus = "unhealthy"
	ProviderUnknown   ProviderHealthStatus = "unknown"
)

// ProviderHealth is a snapshot of provider operational state.
type ProviderHealth struct {
	Name          string
	Status        ProviderHealthStatus
	LastTickAt    time.Time
	LastError     string
	Reconnects    int64
	Subscribed    []string
	LatencyMillis int64
}
