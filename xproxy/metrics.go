package xproxy

import (
	"github.com/prometheus/client_golang/prometheus"
)

var xproxy_roundtrips_total = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "x",
		Subsystem: "proxy",
		Name:      "roundtrips_total",
		Help:      "The total number of xproxy round trips.",
	},
	[]string{"service", "status"},
)

var xproxy_roundtrips_latency = prometheus.NewSummaryVec(
	prometheus.SummaryOpts{
		Namespace: "x",
		Subsystem: "proxy",
		Name:      "roundtrips_latency",
		Help:      "The latency of xproxy round trips.",
	},
	[]string{"service"},
)

func RegisterMetrics() {
	prometheus.MustRegister(xproxy_roundtrips_total)
	prometheus.MustRegister(xproxy_roundtrips_latency)
}
