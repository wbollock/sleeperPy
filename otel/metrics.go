package otel

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

var (
	// HTTP Metrics
	HTTPRequestsTotal   metric.Int64Counter
	HTTPRequestDuration metric.Float64Histogram
	HTTPRequestsActive  metric.Int64UpDownCounter

	// Cache Metrics
	CacheHits   metric.Int64Counter
	CacheMisses metric.Int64Counter

	// API Call Metrics
	APICallsTotal   metric.Int64Counter
	APICallDuration metric.Float64Histogram

	// Business Metrics
	LeaguesAnalyzed metric.Int64Counter
	UsersActive     metric.Int64UpDownCounter
)

func InitMetrics() {
	meter := otel.Meter("sleeperpy")

	// HTTP
	HTTPRequestsTotal, _ = meter.Int64Counter(
		"http.requests.total",
		metric.WithDescription("Total HTTP requests"),
	)
	HTTPRequestDuration, _ = meter.Float64Histogram(
		"http.request.duration",
		metric.WithDescription("HTTP request duration"),
		metric.WithUnit("s"),
	)
	HTTPRequestsActive, _ = meter.Int64UpDownCounter(
		"http.requests.active",
		metric.WithDescription("Active HTTP requests"),
	)

	// Cache
	CacheHits, _ = meter.Int64Counter(
		"cache.hits",
		metric.WithDescription("Cache hits"),
	)
	CacheMisses, _ = meter.Int64Counter(
		"cache.misses",
		metric.WithDescription("Cache misses"),
	)

	// API Calls
	APICallsTotal, _ = meter.Int64Counter(
		"api.calls.total",
		metric.WithDescription("External API calls"),
	)
	APICallDuration, _ = meter.Float64Histogram(
		"api.call.duration",
		metric.WithDescription("API call duration"),
		metric.WithUnit("s"),
	)

	// Business
	LeaguesAnalyzed, _ = meter.Int64Counter(
		"leagues.analyzed",
		metric.WithDescription("Leagues analyzed"),
	)
	UsersActive, _ = meter.Int64UpDownCounter(
		"users.active",
		metric.WithDescription("Active users"),
	)
}
