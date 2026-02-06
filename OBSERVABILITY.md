# OpenTelemetry Observability Setup

## Overview

SleeperPy includes comprehensive observability with OpenTelemetry:
- **Traces**: End-to-end request flow visualization
- **Metrics**: Request counts, latency, cache efficiency, API calls
- **Logs**: Structured logging with trace correlation

## Local Development

### Start the Observability Stack

```bash
docker-compose -f docker-compose.otel.yml up -d
```

This starts:
- **OTEL Collector** (localhost:4317) - Receives telemetry
- **Jaeger UI** (localhost:16686) - Traces visualization
- **Prometheus** (localhost:9090) - Metrics storage
- **Grafana** (localhost:3000) - Unified dashboards

### Run Your App with OTEL

```bash
# Install OTEL dependencies first
go get go.opentelemetry.io/otel
go get go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc
go get go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc
go get go.opentelemetry.io/otel/sdk/trace
go get go.opentelemetry.io/otel/sdk/metric
go get go.opentelemetry.io/otel/sdk/resource
go get go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp

# Run with OTEL enabled
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 \
ENVIRONMENT=development \
go run .
```

### Access UIs

- **Jaeger**: http://localhost:16686 - View distributed traces
- **Prometheus**: http://localhost:9090 - Query metrics
- **Grafana**: http://localhost:3000 - Dashboards (auto-login enabled)

## Metrics Tracked

### HTTP Metrics
- `http.requests.total` - Total requests by route, method, status
- `http.request.duration` - Latency histogram (p50, p95, p99)
- `http.requests.active` - In-flight requests

### Application Metrics
- `cache.hits` / `cache.misses` - Cache efficiency
- `api.calls.total` - External API calls (Sleeper, Boris Chen, KTC)
- `api.call.duration` - API response times
- `leagues.analyzed` - Business metric
- `users.active` - Active users

## Structured Logging

Logs include trace IDs for correlation:

```go
import "sleeperpy/goapp/logger"

logger.Info(ctx, "fetching user leagues",
    slog.String("user_id", userID),
    slog.Int("league_count", len(leagues)),
)
```

## Adding Instrumentation

### Add Spans to Functions

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
)

var tracer = otel.Tracer("sleeperpy")

func myFunction(ctx context.Context) error {
    ctx, span := tracer.Start(ctx, "myFunction")
    defer span.End()

    span.SetAttributes(attribute.String("key", "value"))

    // Your code here

    return nil
}
```

### Record Metrics

```go
import "sleeperpy/goapp/otel"

otel.CacheHits.Add(ctx, 1,
    metric.WithAttributes(attribute.String("cache", "user")))
```

## Production Deployment

### Environment Variables

```bash
OTEL_EXPORTER_OTLP_ENDPOINT=your-collector:4317
ENVIRONMENT=production
```

### Sampling

- Development: 100% of traces
- Production: 10% of traces (configured in otel/otel.go)

### SaaS Options

Instead of self-hosting, you can use:
- **Honeycomb** - Free tier: 20GB/month
- **Grafana Cloud** - Free tier: 10k series, 50GB logs
- **New Relic** - Free tier: 100GB/month

Simply point `OTEL_EXPORTER_OTLP_ENDPOINT` to their collector endpoint.

## Troubleshooting

### Check OTEL Collector is running

```bash
docker ps | grep otel-collector
```

### View Collector Logs

```bash
docker logs <otel-collector-container-id>
```

### Test OTLP Endpoint

```bash
curl http://localhost:55679/debug/tracez
```

### Verify Metrics Export

```bash
curl http://localhost:8889/metrics
```

## Cost Considerations

### Self-Hosted (VPS)
- Small VPS (2 CPU, 4GB RAM): $10-15/month
- Medium VPS (4 CPU, 8GB RAM): $20-30/month
- Runs all components with 30-day retention

### SaaS
- Honeycomb: $0-20/month for small apps
- Grafana Cloud: $0-10/month
- New Relic: $0-15/month

## Next Steps

1. Instrument key functions in fetch.go with spans
2. Add HTTP middleware with otelhttp
3. Create custom Grafana dashboards
4. Set up alerts for error rates and latency
