package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Cache metrics
	CacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "url_cache_hits_total",
			Help: "Total number of cache hits",
		},
		[]string{"layer"}, // "l1" or "l2"
	)

	CacheMisses = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "url_cache_misses_total",
			Help: "Total number of cache misses",
		},
		[]string{"layer"},
	)

	CacheSize = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "url_cache_size",
			Help: "Current number of items in cache",
		},
		[]string{"layer"},
	)

	// Request metrics
	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "url_request_duration_seconds",
			Help:    "Request duration in seconds",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
		},
		[]string{"method", "status"},
	)

	RequestTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "url_requests_total",
			Help: "Total number of requests",
		},
		[]string{"method", "status"},
	)

	// Database metrics
	DatabaseQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "url_database_query_duration_seconds",
			Help:    "Database query duration in seconds",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
		},
		[]string{"operation"},
	)
)
