# Simplify Design + CLI Mode + OpenTelemetry Integration

## Overview
Three improvements to make SleeperPy more maintainable and testable:
1. **Remove flashy design** - Simplify the "elite" CSS that's too much
2. **Add CLI mode** - Test backend logic without web interface
3. **Add OpenTelemetry** - Comprehensive logs, metrics, and traces

---

## Part 1: Simplify the Flashy Design

### Problem
Recent commits added excessive visual effects:
- `elite-design.css` - 686 lines of animations, glow effects, glassmorphism
- `transaction-viz.css` - 439 lines of animated trade visualizations
- `premium-design.css`, `data-viz.css`, `micro-interactions.css`
- Custom fonts (Orbitron, Rajdhani, Space Mono)
- Shimmer, pulse, glow, shine effects everywhere

### Goal
Keep it **clean, fast, and readable**. Remove:
- ❌ Glow effects and shimmer animations
- ❌ Glassmorphism and complex gradients
- ❌ Custom athletic fonts (stick to system fonts)
- ❌ Pulse and shine micro-interactions
- ❌ Tactical grid overlays
- ❌ Championship badge animations

Keep:
- ✅ Simple, readable typography (system fonts)
- ✅ Clear color coding for tiers
- ✅ Responsive layout
- ✅ Dark/light theme toggle
- ✅ Basic hover states
- ✅ Fast loading

### Implementation

**Option 1: Remove CSS files entirely** (quickest)
```bash
git rm static/elite-design.css
git rm static/transaction-viz.css
git rm static/premium-design.css
git rm static/data-viz.css
git rm static/micro-interactions.css
```

Remove from templates:
```html
<!-- Delete these lines from templates/tiers.html -->
<link rel="stylesheet" href="/static/elite-design.css">
<link rel="stylesheet" href="/static/transaction-viz.css">
<link rel="stylesheet" href="/static/premium-design.css">
<link rel="stylesheet" href="/static/data-viz.css">
<link rel="stylesheet" href="/static/micro-interactions.css">
```

**Option 2: Simplify the CSS** (more work but keeps some improvements)
- Remove all animations/transitions
- Remove custom fonts
- Remove glow/shimmer effects
- Keep basic layout improvements
- Keep transaction display clarity (without animations)

### Recommendation
**Option 1** - Just remove it all. The app worked fine before these files. You can always add subtle improvements later.

---

## Part 2: CLI Mode for Testing

### Goal
Run backend logic from command line without starting the web server. Useful for:
- Testing API fetching logic
- Debugging roster calculations
- Validating dynasty value lookups
- Running integration tests
- CI/CD pipeline testing

### Design

**CLI Commands**:
```bash
# Fetch user leagues
./sleeperPy cli user <username>

# Analyze specific league
./sleeperPy cli league <league_id> <username>

# Test Boris Chen tier fetching
./sleeperPy cli tiers <format>  # ppr, half-ppr, standard, superflex

# Test dynasty values fetch
./sleeperPy cli dynasty-values

# Test player data
./sleeperPy cli player <player_name>

# Run all integration tests
./sleeperPy cli test
```

**Output Format**:
- JSON for machine parsing
- Pretty-printed for human reading
- `--json` flag for strict JSON output

### Implementation

**1. Create CLI package** (`cli/cli.go`):
```go
package cli

import (
    "encoding/json"
    "fmt"
    "os"
)

func RunCLI(args []string) {
    if len(args) < 2 {
        printUsage()
        os.Exit(1)
    }

    command := args[1]

    switch command {
    case "user":
        if len(args) < 3 {
            fmt.Println("Usage: sleeperPy cli user <username>")
            os.Exit(1)
        }
        handleUser(args[2])
    case "league":
        if len(args) < 4 {
            fmt.Println("Usage: sleeperPy cli league <league_id> <username>")
            os.Exit(1)
        }
        handleLeague(args[2], args[3])
    case "tiers":
        format := "ppr"
        if len(args) >= 3 {
            format = args[2]
        }
        handleTiers(format)
    case "dynasty-values":
        handleDynastyValues()
    case "player":
        if len(args) < 3 {
            fmt.Println("Usage: sleeperPy cli player <name>")
            os.Exit(1)
        }
        handlePlayer(args[2])
    case "test":
        runIntegrationTests()
    default:
        fmt.Printf("Unknown command: %s\n", command)
        printUsage()
        os.Exit(1)
    }
}

func printUsage() {
    fmt.Println(`SleeperPy CLI

Usage:
  sleeperPy cli user <username>              Fetch user's leagues
  sleeperPy cli league <league_id> <user>    Analyze specific league
  sleeperPy cli tiers <format>               Fetch Boris Chen tiers
  sleeperPy cli dynasty-values               Fetch dynasty values
  sleeperPy cli player <name>                Look up player
  sleeperPy cli test                         Run integration tests

Flags:
  --json    Output in JSON format (machine readable)
  --debug   Enable debug logging
`)
}

func handleUser(username string) {
    // Reuse existing fetchSleeperUser function
    user, err := fetchSleeperUser(username)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error fetching user: %v\n", err)
        os.Exit(1)
    }

    // Fetch leagues
    leagues, err := fetchUserLeagues(user.UserID)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error fetching leagues: %v\n", err)
        os.Exit(1)
    }

    // Print results
    if hasJSONFlag() {
        json.NewEncoder(os.Stdout).Encode(leagues)
    } else {
        fmt.Printf("User: %s (ID: %s)\n", user.Username, user.UserID)
        fmt.Printf("Found %d leagues:\n\n", len(leagues))
        for _, league := range leagues {
            fmt.Printf("  - %s (%s)\n", league.Name, league.LeagueID)
            fmt.Printf("    Type: %s, Scoring: %s\n", league.Type, league.ScoringSettings)
        }
    }
}

func handleLeague(leagueID, username string) {
    // Reuse existing league analysis logic
    // This is the same code path as the web handler
    // but outputs to stdout instead of rendering HTML

    leagueData, err := analyzeLeague(leagueID, username)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error analyzing league: %v\n", err)
        os.Exit(1)
    }

    if hasJSONFlag() {
        json.NewEncoder(os.Stdout).Encode(leagueData)
    } else {
        // Pretty print
        fmt.Printf("League: %s\n", leagueData.Name)
        fmt.Printf("Roster:\n")
        for _, player := range leagueData.Starters {
            fmt.Printf("  %s (%s) - Tier %d, Value: %d\n",
                player.Name, player.Pos, player.Tier, player.DynastyValue)
        }
    }
}

func handleTiers(format string) {
    tiers, err := fetchBorisChenTiers(format)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error fetching tiers: %v\n", err)
        os.Exit(1)
    }

    if hasJSONFlag() {
        json.NewEncoder(os.Stdout).Encode(tiers)
    } else {
        fmt.Printf("Boris Chen Tiers (%s)\n\n", format)
        for _, tier := range tiers {
            fmt.Printf("Tier %d: %d players\n", tier.TierNum, len(tier.Players))
        }
    }
}

func handleDynastyValues() {
    values, err := fetchDynastyValues()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error fetching dynasty values: %v\n", err)
        os.Exit(1)
    }

    if hasJSONFlag() {
        json.NewEncoder(os.Stdout).Encode(values)
    } else {
        fmt.Printf("Dynasty Values (KeepTradeCut)\n")
        fmt.Printf("Fetched %d player values\n", len(values))
        fmt.Printf("Top 10 most valuable:\n\n")

        // Sort and show top 10
        // ... implementation
    }
}

func runIntegrationTests() {
    fmt.Println("Running integration tests...\n")

    tests := []struct {
        name string
        fn   func() error
    }{
        {"Fetch test user", testFetchUser},
        {"Fetch Boris Chen tiers", testFetchTiers},
        {"Fetch dynasty values", testFetchDynastyValues},
        {"Analyze test league", testAnalyzeLeague},
    }

    passed := 0
    failed := 0

    for _, test := range tests {
        fmt.Printf("  %s... ", test.name)
        err := test.fn()
        if err != nil {
            fmt.Printf("❌ FAIL: %v\n", err)
            failed++
        } else {
            fmt.Printf("✅ PASS\n")
            passed++
        }
    }

    fmt.Printf("\n%d passed, %d failed\n", passed, failed)
    if failed > 0 {
        os.Exit(1)
    }
}

func hasJSONFlag() bool {
    for _, arg := range os.Args {
        if arg == "--json" {
            return true
        }
    }
    return false
}
```

**2. Update main.go**:
```go
func main() {
    // Check if CLI mode
    if len(os.Args) > 1 && os.Args[1] == "cli" {
        cli.RunCLI(os.Args)
        return
    }

    // Otherwise, run web server
    startWebServer()
}
```

**3. Add tests** (`cli/cli_test.go`):
```go
func TestCLIUser(t *testing.T) {
    // Test user command
}

func TestCLILeague(t *testing.T) {
    // Test league command
}

func TestCLITiers(t *testing.T) {
    // Test tiers command
}
```

### Benefits
- ✅ Test backend without browser
- ✅ Easier to debug
- ✅ CI/CD integration
- ✅ Scripting and automation
- ✅ Performance testing
- ✅ Validate API changes

---

## Part 3: OpenTelemetry Integration

### Goal
Full observability: logs, metrics, and distributed traces.

**What you get**:
- 📊 **Metrics**: Request counts, latency, error rates, cache hits
- 📝 **Logs**: Structured logging with trace correlation
- 🔍 **Traces**: End-to-end request flow visualization
- 🎯 **Monitoring**: Real-time dashboard of app health

### Architecture

**Components**:
1. **OpenTelemetry SDK** - Instrument Go app
2. **OTLP Exporter** - Export to collector
3. **OpenTelemetry Collector** - Receive and route telemetry
4. **Backends**: Choose one or more:
   - **Jaeger** - Distributed tracing (open source)
   - **Prometheus** - Metrics (you already use this?)
   - **Loki** - Log aggregation
   - **Grafana** - Unified dashboard

**OR use a SaaS** (easier but costs money):
- Honeycomb (generous free tier, best DX)
- Grafana Cloud (free tier available)
- New Relic (has free tier)
- DataDog (expensive but powerful)

### Implementation

**1. Install dependencies**:
```bash
go get go.opentelemetry.io/otel
go get go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc
go get go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc
go get go.opentelemetry.io/otel/sdk/trace
go get go.opentelemetry.io/otel/sdk/metric
go get go.opentelemetry.io/otel/sdk/resource
go get go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp
```

**2. Initialize OTEL** (`otel.go`):
```go
package main

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

func initOTEL(ctx context.Context) func() {
    res, err := resource.New(ctx,
        resource.WithAttributes(
            semconv.ServiceName("sleeperpy"),
            semconv.ServiceVersion("1.0.0"),
        ),
    )
    if err != nil {
        log.Fatalf("failed to create resource: %v", err)
    }

    // Trace exporter
    traceExporter, err := otlptracegrpc.New(ctx,
        otlptracegrpc.WithEndpoint(getOTELEndpoint()),
        otlptracegrpc.WithInsecure(), // Use TLS in production
    )
    if err != nil {
        log.Fatalf("failed to create trace exporter: %v", err)
    }

    tracerProvider := trace.NewTracerProvider(
        trace.WithBatcher(traceExporter),
        trace.WithResource(res),
        trace.WithSampler(trace.AlwaysSample()), // Sample 100% for now
    )
    otel.SetTracerProvider(tracerProvider)

    // Metric exporter
    metricExporter, err := otlpmetricgrpc.New(ctx,
        otlpmetricgrpc.WithEndpoint(getOTELEndpoint()),
        otlpmetricgrpc.WithInsecure(),
    )
    if err != nil {
        log.Fatalf("failed to create metric exporter: %v", err)
    }

    meterProvider := metric.NewMeterProvider(
        metric.WithReader(metric.NewPeriodicReader(metricExporter, metric.WithInterval(10*time.Second))),
        metric.WithResource(res),
    )
    otel.SetMeterProvider(meterProvider)

    // Return cleanup function
    return func() {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        if err := tracerProvider.Shutdown(ctx); err != nil {
            log.Printf("error shutting down tracer provider: %v", err)
        }
        if err := meterProvider.Shutdown(ctx); err != nil {
            log.Printf("error shutting down meter provider: %v", err)
        }
    }
}

func getOTELEndpoint() string {
    endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
    if endpoint == "" {
        endpoint = "localhost:4317" // Default OTLP gRPC port
    }
    return endpoint
}
```

**3. Instrument HTTP handlers** (`main.go`):
```go
import (
    "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/metric"
)

var (
    tracer = otel.Tracer("sleeperpy")
    meter  = otel.Meter("sleeperpy")

    // Metrics
    httpRequestsTotal metric.Int64Counter
    httpRequestDuration metric.Float64Histogram
    cacheHits metric.Int64Counter
    cacheMisses metric.Int64Counter
    apiCallsTotal metric.Int64Counter
)

func initMetrics() {
    var err error

    httpRequestsTotal, err = meter.Int64Counter(
        "http.requests.total",
        metric.WithDescription("Total number of HTTP requests"),
    )
    if err != nil {
        log.Fatal(err)
    }

    httpRequestDuration, err = meter.Float64Histogram(
        "http.request.duration",
        metric.WithDescription("HTTP request duration in seconds"),
        metric.WithUnit("s"),
    )
    if err != nil {
        log.Fatal(err)
    }

    cacheHits, err = meter.Int64Counter(
        "cache.hits",
        metric.WithDescription("Number of cache hits"),
    )
    if err != nil {
        log.Fatal(err)
    }

    cacheMisses, err = meter.Int64Counter(
        "cache.misses",
        metric.WithDescription("Number of cache misses"),
    )
    if err != nil {
        log.Fatal(err)
    }

    apiCallsTotal, err = meter.Int64Counter(
        "api.calls.total",
        metric.WithDescription("Total API calls to external services"),
    )
    if err != nil {
        log.Fatal(err)
    }
}

func main() {
    ctx := context.Background()

    // Initialize OTEL
    cleanup := initOTEL(ctx)
    defer cleanup()

    initMetrics()

    // Wrap handlers with OTEL middleware
    http.Handle("/", otelhttp.NewHandler(http.HandlerFunc(indexHandler), "index"))
    http.Handle("/lookup", otelhttp.NewHandler(http.HandlerFunc(lookupHandler), "lookup"))
    http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

    log.Println("Starting server on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

**4. Add spans to functions** (`fetch.go`):
```go
func fetchSleeperUser(username string) (*SleeperUser, error) {
    ctx, span := tracer.Start(context.Background(), "fetchSleeperUser")
    defer span.End()

    span.SetAttributes(attribute.String("username", username))

    // Check cache
    if user, ok := userCache[username]; ok {
        cacheHits.Add(ctx, 1, metric.WithAttributes(attribute.String("cache", "user")))
        span.SetAttributes(attribute.Bool("cache.hit", true))
        return user, nil
    }

    cacheMisses.Add(ctx, 1, metric.WithAttributes(attribute.String("cache", "user")))
    span.SetAttributes(attribute.Bool("cache.hit", false))

    // API call
    apiCallsTotal.Add(ctx, 1, metric.WithAttributes(
        attribute.String("service", "sleeper"),
        attribute.String("endpoint", "user"),
    ))

    resp, err := http.Get(fmt.Sprintf("https://api.sleeper.app/v1/user/%s", username))
    if err != nil {
        span.RecordError(err)
        return nil, err
    }
    defer resp.Body.Close()

    // Parse response
    // ...

    return user, nil
}
```

**5. Structured logging** (`log.go`):
```go
import (
    "log/slog"
    "os"

    "go.opentelemetry.io/otel/trace"
)

var logger *slog.Logger

func initLogger() {
    logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelInfo,
    }))
}

func logWithTrace(ctx context.Context, level slog.Level, msg string, attrs ...any) {
    span := trace.SpanFromContext(ctx)
    if span.SpanContext().IsValid() {
        attrs = append(attrs,
            slog.String("trace_id", span.SpanContext().TraceID().String()),
            slog.String("span_id", span.SpanContext().SpanID().String()),
        )
    }

    logger.Log(ctx, level, msg, attrs...)
}

// Usage:
logWithTrace(ctx, slog.LevelInfo, "fetching user leagues",
    slog.String("user_id", userID),
    slog.Int("league_count", len(leagues)),
)
```

**6. Docker Compose for local dev** (`docker-compose.yml`):
```yaml
version: '3.8'

services:
  # OpenTelemetry Collector
  otel-collector:
    image: otel/opentelemetry-collector:latest
    command: ["--config=/etc/otel-collector-config.yaml"]
    volumes:
      - ./otel-collector-config.yaml:/etc/otel-collector-config.yaml
    ports:
      - "4317:4317"  # OTLP gRPC
      - "4318:4318"  # OTLP HTTP
      - "55679:55679"  # zpages

  # Jaeger for traces
  jaeger:
    image: jaegertracing/all-in-one:latest
    ports:
      - "16686:16686"  # Jaeger UI
      - "14250:14250"  # Accept model.proto

  # Prometheus for metrics
  prometheus:
    image: prom/prometheus:latest
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
    ports:
      - "9090:9090"

  # Grafana for visualization
  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      - GF_AUTH_ANONYMOUS_ENABLED=true
      - GF_AUTH_ANONYMOUS_ORG_ROLE=Admin
    volumes:
      - grafana-storage:/var/lib/grafana

volumes:
  grafana-storage:
```

**7. OTEL Collector config** (`otel-collector-config.yaml`):
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

exporters:
  jaeger:
    endpoint: jaeger:14250
    tls:
      insecure: true

  prometheus:
    endpoint: "0.0.0.0:8889"

  logging:
    loglevel: debug

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [jaeger, logging]
    metrics:
      receivers: [otlp]
      processors: [batch]
      exporters: [prometheus, logging]
```

### Running Locally

```bash
# Start observability stack
docker-compose up -d

# Run your app
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 go run .

# Access dashboards:
# - Jaeger UI: http://localhost:16686
# - Prometheus: http://localhost:9090
# - Grafana: http://localhost:3000
```

### What You'll See

**Jaeger (Traces)**:
- End-to-end request flow
- How long each function takes
- Which API calls are slow
- Error stack traces
- Dependency graph

**Prometheus (Metrics)**:
- Request rate
- Error rate
- Cache hit ratio
- API call counts
- Response time percentiles

**Grafana (Dashboards)**:
- Unified view of everything
- Custom dashboards
- Alerts

### Metrics to Track

**HTTP Metrics**:
- `http.requests.total` - Total requests
- `http.request.duration` - Latency
- `http.requests.active` - In-flight requests
- `http.request.size` - Request body size
- `http.response.size` - Response body size

**Application Metrics**:
- `cache.hits` / `cache.misses` - Cache efficiency
- `api.calls.total` - External API usage
- `leagues.analyzed` - Business metric
- `users.active` - Active users
- `dynasty.leagues.percentage` - Feature usage

**Resource Metrics**:
- `process.cpu.usage` - CPU %
- `process.memory.usage` - Memory usage
- `go.goroutines` - Goroutine count

### Cost Considerations

**Self-hosted (Free)**:
- Run Jaeger + Prometheus + Grafana on VPS
- ~$5-10/month for small VPS
- Full control, unlimited retention

**SaaS (Easier but $$$)**:
- Honeycomb: Free tier (20GB/month)
- Grafana Cloud: Free tier (10k series, 50GB logs)
- New Relic: Free tier (100GB/month)

**Recommendation**: Start self-hosted, migrate to SaaS if needed.

---

## Implementation Order

### Phase 1: Simplify Design (30 minutes)
1. Remove flashy CSS files
2. Remove CSS links from templates
3. Test that app still works
4. Commit: "chore: remove overly flashy elite design"

### Phase 2: CLI Mode (4-6 hours)
1. Create `cli/` package
2. Implement basic commands (user, league, tiers)
3. Add `--json` flag support
4. Update main.go to detect CLI mode
5. Test all commands
6. Add documentation
7. Commit: "feat: add CLI mode for testing"

### Phase 3: OpenTelemetry (6-8 hours)
1. Install OTEL dependencies
2. Initialize OTEL in main.go
3. Add HTTP middleware
4. Instrument key functions with spans
5. Add custom metrics
6. Set up structured logging
7. Create docker-compose for local dev
8. Document how to use
9. Commit: "feat: add OpenTelemetry observability"

### Phase 4: Testing & Docs (2-3 hours)
1. Test CLI mode with real data
2. Verify OTEL traces in Jaeger
3. Create Grafana dashboards
4. Write README documentation
5. Add examples

---

## Success Criteria

After implementation:

### Design
- ✅ No glow/shimmer/pulse effects
- ✅ Clean, readable UI
- ✅ Fast page loads
- ✅ System fonts only

### CLI Mode
- ✅ Can fetch user data from command line
- ✅ Can analyze league without web UI
- ✅ JSON output works for scripting
- ✅ Integration tests pass

### OpenTelemetry
- ✅ Traces show up in Jaeger
- ✅ Metrics export to Prometheus
- ✅ Logs have trace correlation
- ✅ Can debug slow requests
- ✅ Cache hit rates visible
- ✅ API call counts tracked

---

## Questions for You

1. **Design**: Remove all flashy CSS or keep some of it?
   - Recommend: Remove all, start clean

2. **CLI**: What's the primary use case?
   - Testing? Integration tests? Scripting?
   - This affects output format and features

3. **OTEL**: Self-hosted or SaaS?
   - Recommend: Start self-hosted (Jaeger + Prometheus + Grafana)
   - Can migrate to Honeycomb/Grafana Cloud later

4. **OTEL**: What do you want to monitor most?
   - Slow API calls?
   - Cache efficiency?
   - Error rates?
   - User behavior?

5. **Priority**: Which part first?
   - Design simplification (quickest)
   - CLI mode (most useful for testing)
   - OTEL (best for production monitoring)

---

## Next Steps

Let me know:
1. Should I remove the flashy CSS files?
2. Should I implement the CLI mode?
3. Should I set up OpenTelemetry?

I can do all three or start with one. What's your priority?
