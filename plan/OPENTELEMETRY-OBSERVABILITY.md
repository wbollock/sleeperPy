# OpenTelemetry Observability

## Problem
No visibility into application behavior:
- Don't know which API calls are slow
- Can't track cache hit rates
- No distributed tracing
- Hard to debug performance issues
- No production metrics

## Solution
Full observability with OpenTelemetry (OTEL) for:
- 📊 **Metrics** - Request counts, latency, error rates, cache efficiency
- 🔍 **Traces** - End-to-end request flow visualization
- 📝 **Logs** - Structured logging with trace correlation

## Architecture

### Components

```
┌─────────────┐
│ SleeperPy   │
│ (Go App)    │
│             │
│ - OTEL SDK  │
│ - Instrumentation
└─────┬───────┘
      │ OTLP
      ▼
┌─────────────────┐
│ OTEL Collector  │
│ (Receive/Route) │
└────┬────────┬───┘
     │        │
     ▼        ▼
┌────────┐ ┌──────────┐
│ Jaeger │ │Prometheus│
│(Traces)│ │(Metrics) │
└────────┘ └──────────┘
     │        │
     └────┬───┘
          ▼
     ┌─────────┐
     │ Grafana │
     │(Unified)│
     └─────────┘
```

### Tech Stack

**Self-Hosted** (Recommended for start):
- **OpenTelemetry Collector** - Receive and route telemetry
- **Jaeger** - Distributed tracing UI
- **Prometheus** - Metrics storage and querying
- **Grafana** - Unified dashboards

**SaaS Options** (Easier but costs money):
- **Honeycomb** - Best DX, generous free tier (20GB/mo)
- **Grafana Cloud** - Free tier (10k series, 50GB logs)
- **New Relic** - Free tier (100GB/mo)

## What You'll Track

### HTTP Metrics
- `http.requests.total` - Total requests by route, method, status
- `http.request.duration` - Latency histogram (p50, p95, p99)
- `http.requests.active` - In-flight requests
- `http.request.size` - Request body size
- `http.response.size` - Response body size

### Application Metrics
- `cache.hits` / `cache.misses` - Cache efficiency by type (user, league, tiers, dynasty)
- `api.calls.total` - External API calls (sleeper, boris chen, ktc)
- `api.call.duration` - API response times
- `leagues.analyzed` - Business metric
- `users.active` - Active users (by cookie)
- `dynasty.leagues.percentage` - Feature usage

### Resource Metrics
- `process.cpu.usage` - CPU percentage
- `process.memory.usage` - Memory usage
- `go.goroutines` - Goroutine count
- `go.gc.duration` - GC pause times

### Traces
Every HTTP request becomes a trace with spans:
```
/lookup request (2.3s total)
├─ fetchSleeperUser (234ms)
│  └─ HTTP GET sleeper.app/user (189ms)
├─ fetchUserLeagues (892ms)
│  └─ HTTP GET sleeper.app/leagues (845ms)
├─ fetchBorisChenTiers (456ms) [cache HIT]
├─ fetchDynastyValues (1.2s) [cache MISS]
│  └─ HTTP GET ktc.com/values (1.1s)
└─ renderTemplate (123ms)
```

## Implementation

### Phase 1: Setup OTEL SDK (2 hours)

**File: `otel/otel.go`**
```go
package otel

import (
    "context"
    "log"
    "os"
    "time"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
    "go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
    "go.opentelemetry.io/otel/sdk/metric"
    "go.opentelemetry.io/otel/sdk/resource"
    "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

// Init initializes OpenTelemetry and returns cleanup function
func Init(ctx context.Context) func() {
    res, err := resource.New(ctx,
        resource.WithAttributes(
            semconv.ServiceName("sleeperpy"),
            semconv.ServiceVersion("1.0.0"),
            semconv.DeploymentEnvironment(getEnv()),
        ),
    )
    if err != nil {
        log.Fatalf("failed to create resource: %v", err)
    }

    // Setup trace provider
    tracerProvider := initTracer(ctx, res)
    otel.SetTracerProvider(tracerProvider)

    // Setup meter provider
    meterProvider := initMeter(ctx, res)
    otel.SetMeterProvider(meterProvider)

    // Return cleanup function
    return func() {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()

        if err := tracerProvider.Shutdown(ctx); err != nil {
            log.Printf("error shutting down tracer: %v", err)
        }
        if err := meterProvider.Shutdown(ctx); err != nil {
            log.Printf("error shutting down meter: %v", err)
        }
    }
}

func initTracer(ctx context.Context, res *resource.Resource) *trace.TracerProvider {
    exporter, err := otlptracegrpc.New(ctx,
        otlptracegrpc.WithEndpoint(getOTELEndpoint()),
        otlptracegrpc.WithInsecure(), // Use TLS in production
    )
    if err != nil {
        log.Fatalf("failed to create trace exporter: %v", err)
    }

    return trace.NewTracerProvider(
        trace.WithBatcher(exporter),
        trace.WithResource(res),
        trace.WithSampler(getSampler()),
    )
}

func initMeter(ctx context.Context, res *resource.Resource) *metric.MeterProvider {
    exporter, err := otlpmetricgrpc.New(ctx,
        otlpmetricgrpc.WithEndpoint(getOTELEndpoint()),
        otlpmetricgrpc.WithInsecure(),
    )
    if err != nil {
        log.Fatalf("failed to create metric exporter: %v", err)
    }

    return metric.NewMeterProvider(
        metric.WithReader(
            metric.NewPeriodicReader(exporter,
                metric.WithInterval(10*time.Second),
            ),
        ),
        metric.WithResource(res),
    )
}

func getOTELEndpoint() string {
    if endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); endpoint != "" {
        return endpoint
    }
    return "localhost:4317" // Default
}

func getEnv() string {
    if env := os.Getenv("ENVIRONMENT"); env != "" {
        return env
    }
    return "development"
}

func getSampler() trace.Sampler {
    // Sample 100% in dev, 10% in prod
    if getEnv() == "production" {
        return trace.TraceIDRatioBased(0.1)
    }
    return trace.AlwaysSample()
}
```

**File: `otel/metrics.go`**
```go
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
    APICallsTotal    metric.Int64Counter
    APICallDuration  metric.Float64Histogram

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
```

### Phase 2: Instrument HTTP (1 hour)

**File: `main.go`**
```go
import (
    "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
    "yourapp/otel"
)

func main() {
    ctx := context.Background()

    // Initialize OTEL
    cleanup := otel.Init(ctx)
    defer cleanup()

    otel.InitMetrics()

    // Wrap handlers with OTEL middleware
    http.Handle("/",
        otelhttp.NewHandler(
            http.HandlerFunc(indexHandler),
            "index",
        ),
    )
    http.Handle("/lookup",
        otelhttp.NewHandler(
            http.HandlerFunc(lookupHandler),
            "lookup",
        ),
    )

    log.Println("Starting server on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

### Phase 3: Add Spans to Functions (2 hours)

**File: `fetch.go`**
```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/codes"
    "yourapp/otel"
)

var tracer = otel.Tracer("sleeperpy")

func fetchSleeperUser(ctx context.Context, username string) (*SleeperUser, error) {
    ctx, span := tracer.Start(ctx, "fetchSleeperUser")
    defer span.End()

    span.SetAttributes(attribute.String("username", username))

    // Check cache
    if user, ok := userCache[username]; ok {
        otel.CacheHits.Add(ctx, 1,
            metric.WithAttributes(attribute.String("cache", "user")))
        span.SetAttributes(attribute.Bool("cache.hit", true))
        return user, nil
    }

    otel.CacheMisses.Add(ctx, 1,
        metric.WithAttributes(attribute.String("cache", "user")))
    span.SetAttributes(attribute.Bool("cache.hit", false))

    // API call
    start := time.Now()
    otel.APICallsTotal.Add(ctx, 1,
        metric.WithAttributes(
            attribute.String("service", "sleeper"),
            attribute.String("endpoint", "user"),
        ))

    resp, err := http.Get(fmt.Sprintf("https://api.sleeper.app/v1/user/%s", username))
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return nil, err
    }
    defer resp.Body.Close()

    duration := time.Since(start).Seconds()
    otel.APICallDuration.Record(ctx, duration,
        metric.WithAttributes(attribute.String("service", "sleeper")))

    // Parse response...
    span.SetStatus(codes.Ok, "success")
    return user, nil
}
```

### Phase 4: Structured Logging (1 hour)

**File: `logger/logger.go`**
```go
package logger

import (
    "context"
    "log/slog"
    "os"

    "go.opentelemetry.io/otel/trace"
)

var Logger *slog.Logger

func Init() {
    Logger = slog.New(
        slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
            Level: slog.LevelInfo,
        }),
    )
}

// WithTrace adds trace context to log attributes
func WithTrace(ctx context.Context, attrs ...any) []any {
    span := trace.SpanFromContext(ctx)
    if span.SpanContext().IsValid() {
        attrs = append(attrs,
            slog.String("trace_id", span.SpanContext().TraceID().String()),
            slog.String("span_id", span.SpanContext().SpanID().String()),
        )
    }
    return attrs
}

// Usage in code:
logger.Logger.InfoContext(ctx, "fetching user leagues",
    logger.WithTrace(ctx,
        slog.String("user_id", userID),
        slog.Int("league_count", len(leagues)),
    )...,
)
```

### Phase 5: Local Dev Stack (1 hour)

**File: `docker-compose.otel.yml`**
```yaml
version: '3.8'

services:
  # OpenTelemetry Collector
  otel-collector:
    image: otel/opentelemetry-collector-contrib:latest
    command: ["--config=/etc/otel-collector-config.yaml"]
    volumes:
      - ./otel-collector-config.yaml:/etc/otel-collector-config.yaml
    ports:
      - "4317:4317"  # OTLP gRPC
      - "4318:4318"  # OTLP HTTP
      - "55679:55679"  # zpages
      - "8889:8889"  # Prometheus exporter

  # Jaeger (Traces)
  jaeger:
    image: jaegertracing/all-in-one:latest
    environment:
      - COLLECTOR_OTLP_ENABLED=true
    ports:
      - "16686:16686"  # Jaeger UI
      - "14250:14250"  # Accept model.proto

  # Prometheus (Metrics)
  prometheus:
    image: prom/prometheus:latest
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
    ports:
      - "9090:9090"
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'

  # Grafana (Dashboards)
  grafana:
    image: grafana/grafana:latest
    environment:
      - GF_AUTH_ANONYMOUS_ENABLED=true
      - GF_AUTH_ANONYMOUS_ORG_ROLE=Admin
      - GF_FEATURE_TOGGLES_ENABLE=traceqlEditor
    volumes:
      - grafana-data:/var/lib/grafana
      - ./grafana/dashboards:/etc/grafana/provisioning/dashboards
      - ./grafana/datasources:/etc/grafana/provisioning/datasources
    ports:
      - "3000:3000"

volumes:
  grafana-data:
```

**File: `otel-collector-config.yaml`**
```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

processors:
  batch:
    timeout: 10s

  attributes:
    actions:
      - key: service.name
        value: sleeperpy
        action: upsert

exporters:
  # Jaeger for traces
  otlp/jaeger:
    endpoint: jaeger:4317
    tls:
      insecure: true

  # Prometheus for metrics
  prometheus:
    endpoint: "0.0.0.0:8889"
    namespace: sleeperpy

  # Debug logging
  logging:
    loglevel: info

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch, attributes]
      exporters: [otlp/jaeger, logging]

    metrics:
      receivers: [otlp]
      processors: [batch, attributes]
      exporters: [prometheus, logging]
```

**File: `prometheus.yml`**
```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'sleeperpy'
    static_configs:
      - targets: ['otel-collector:8889']
```

### Phase 6: Grafana Dashboards (2 hours)

**File: `grafana/datasources/datasources.yml`**
```yaml
apiVersion: 1

datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true

  - name: Jaeger
    type: jaeger
    access: proxy
    url: http://jaeger:16686
```

**Dashboard Panels**:
- Request rate (req/s)
- Request duration (p50, p95, p99)
- Error rate (%)
- Cache hit ratio
- API call counts by service
- Active goroutines
- Memory usage
- Top slowest endpoints

## Running Locally

```bash
# Start observability stack
docker-compose -f docker-compose.otel.yml up -d

# Run your app with OTEL enabled
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 \
ENVIRONMENT=development \
go run .

# Access UIs:
# - Jaeger: http://localhost:16686
# - Prometheus: http://localhost:9090
# - Grafana: http://localhost:3000
```

## Cost Analysis

### Self-Hosted (VPS)
- **Small VPS** (2 CPU, 4GB RAM): $10-15/month
- **Medium VPS** (4 CPU, 8GB RAM): $20-30/month
- Runs all components (Jaeger, Prometheus, Grafana)
- Data retention: 30 days default
- **Total: $10-30/month**

### SaaS

**Honeycomb** (Recommended if going SaaS):
- Free tier: 20GB/month events
- Pay-as-you-go: $1/GB after
- Best DX and query interface
- **Estimated: $0-20/month**

**Grafana Cloud**:
- Free tier: 10k series, 50GB logs
- Generous free tier for small apps
- **Estimated: $0-10/month**

**New Relic**:
- Free tier: 100GB/month
- More complex setup
- **Estimated: $0-15/month**

## Implementation Timeline

### Week 1: Core Setup (6 hours)
- Phase 1: Setup OTEL SDK (2h)
- Phase 2: Instrument HTTP (1h)
- Phase 3: Add spans to key functions (2h)
- Phase 4: Structured logging (1h)

### Week 2: Infrastructure (4 hours)
- Phase 5: Docker compose setup (1h)
- Phase 6: Grafana dashboards (2h)
- Testing and validation (1h)

**Total: 10 hours**

## Success Criteria

After implementation:
- ✅ Traces show up in Jaeger
- ✅ Metrics export to Prometheus
- ✅ Logs have trace correlation
- ✅ Can identify slow API calls
- ✅ Cache hit rates visible
- ✅ Error rates tracked
- ✅ Grafana dashboards working
- ✅ Works in local dev environment
- ✅ Documentation for deployment

## Future Enhancements

### Alerts
Set up alerts in Grafana for:
- Error rate > 5%
- p95 latency > 3s
- Cache hit ratio < 50%
- Goroutine leak (goroutines > 1000)

### Custom Metrics
- Dynasty league adoption rate
- Most popular features
- User retention (return visits)
- Peak usage times

### Log Aggregation
Add Loki for centralized logging:
- Search logs across deployments
- Correlate logs with traces
- Alert on error patterns

### Profiling
Integrate pprof data:
- CPU profiling
- Memory profiling
- Goroutine analysis
- Continuous profiling
