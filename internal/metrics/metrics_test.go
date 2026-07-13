package metrics_test

import (
	"testing"

	"github.com/tsc-forex/tsc_forex/internal/metrics"
)

func TestNewMetrics(t *testing.T) {
	t.Parallel()
	m := metrics.New("test")
	if m.Registry() == nil {
		t.Fatal("nil registry")
	}
	// Exercise collectors to ensure they are registered and usable.
	m.WSClients.Set(5)
	m.WSDroppedTotal.Inc()
	m.WSBroadcastTotal.Add(3)
	m.CalculationsTotal.Inc()
	m.CalcLatency.Observe(0.001)
	m.ProviderReconnects.WithLabelValues("mock").Inc()
	m.ProviderSwitches.Inc()
	m.ProviderLatency.WithLabelValues("mock", "XAUUSD").Observe(0.01)
	m.RedisLatency.Observe(0.001)
	m.RedisErrors.Inc()
	m.DBLatency.Observe(0.002)
	m.DBErrors.Inc()
	m.HTTPRequests.WithLabelValues("GET", "/x", "200").Inc()
	m.HTTPLatency.WithLabelValues("GET", "/x").Observe(0.05)

	mfs, err := m.Registry().Gather()
	if err != nil {
		t.Fatal(err)
	}
	if len(mfs) == 0 {
		t.Fatal("no metrics gathered")
	}
}

func TestNewMetricsDefaultNamespace(t *testing.T) {
	t.Parallel()
	m := metrics.New("")
	if m == nil || m.Registry() == nil {
		t.Fatal("nil metrics")
	}
}
