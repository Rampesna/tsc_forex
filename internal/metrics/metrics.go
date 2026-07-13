package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus collectors for the service.
// Injected via DI; no global state.
type Metrics struct {
	reg *prometheus.Registry

	WSClients        prometheus.Gauge
	WSDroppedTotal   prometheus.Counter
	WSBroadcastTotal prometheus.Counter

	CalculationsTotal prometheus.Counter
	CalcLatency       prometheus.Histogram

	ProviderReconnects *prometheus.CounterVec
	ProviderSwitches   prometheus.Counter
	ProviderLatency    *prometheus.HistogramVec

	RedisLatency prometheus.Histogram
	RedisErrors  prometheus.Counter

	DBLatency prometheus.Histogram
	DBErrors  prometheus.Counter

	HTTPRequests *prometheus.CounterVec
	HTTPLatency  *prometheus.HistogramVec
}

// New builds all collectors registered on a dedicated registry.
func New(namespace string) *Metrics {
	if namespace == "" {
		namespace = "tsc_forex"
	}
	reg := prometheus.NewRegistry()
	reg.MustRegister(prometheus.NewGoCollector())
	reg.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	f := promauto.With(reg)

	return &Metrics{
		reg: reg,
		WSClients: f.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace, Subsystem: "ws", Name: "connected_clients",
			Help: "Currently connected WebSocket clients.",
		}),
		WSDroppedTotal: f.NewCounter(prometheus.CounterOpts{
			Namespace: namespace, Subsystem: "ws", Name: "dropped_messages_total",
			Help: "Messages dropped due to slow clients or full queues.",
		}),
		WSBroadcastTotal: f.NewCounter(prometheus.CounterOpts{
			Namespace: namespace, Subsystem: "ws", Name: "broadcasts_total",
			Help: "Total broadcast fan-outs performed.",
		}),
		CalculationsTotal: f.NewCounter(prometheus.CounterOpts{
			Namespace: namespace, Subsystem: "calc", Name: "calculations_total",
			Help: "Total gram-gold recalculations.",
		}),
		CalcLatency: f.NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace, Subsystem: "calc", Name: "latency_seconds",
			Help:    "Tick-to-calculation latency.",
			Buckets: prometheus.ExponentialBuckets(0.0001, 2, 12),
		}),
		ProviderReconnects: f.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace, Subsystem: "provider", Name: "reconnects_total",
			Help: "Provider reconnect attempts.",
		}, []string{"provider"}),
		ProviderSwitches: f.NewCounter(prometheus.CounterOpts{
			Namespace: namespace, Subsystem: "provider", Name: "switches_total",
			Help: "Provider failover switches.",
		}),
		ProviderLatency: f.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace, Subsystem: "provider", Name: "tick_latency_seconds",
			Help:    "Provider tick delivery latency (now - tick timestamp).",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 12),
		}, []string{"provider", "symbol"}),
		RedisLatency: f.NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace, Subsystem: "redis", Name: "op_latency_seconds",
			Help:    "Redis operation latency.",
			Buckets: prometheus.ExponentialBuckets(0.0001, 2, 12),
		}),
		RedisErrors: f.NewCounter(prometheus.CounterOpts{
			Namespace: namespace, Subsystem: "redis", Name: "errors_total",
			Help: "Redis operation errors.",
		}),
		DBLatency: f.NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace, Subsystem: "db", Name: "op_latency_seconds",
			Help:    "Database operation latency.",
			Buckets: prometheus.ExponentialBuckets(0.0005, 2, 12),
		}),
		DBErrors: f.NewCounter(prometheus.CounterOpts{
			Namespace: namespace, Subsystem: "db", Name: "errors_total",
			Help: "Database operation errors.",
		}),
		HTTPRequests: f.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace, Subsystem: "http", Name: "requests_total",
			Help: "HTTP requests by route and status.",
		}, []string{"method", "route", "status"}),
		HTTPLatency: f.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace, Subsystem: "http", Name: "request_latency_seconds",
			Help:    "HTTP request latency by route.",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "route"}),
	}
}

// Registry exposes the underlying registry for the /metrics handler.
func (m *Metrics) Registry() *prometheus.Registry { return m.reg }
