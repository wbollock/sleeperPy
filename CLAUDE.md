# Claude Agent Instructions for SleeperPy

## Auto-Resume Implementation Task

When instructed to "implement the plans" or similar, execute the following implementation sequence:

### Priority Order

**Phase 1-3: Critical Features (Complete First)**

1. **REMOVE-FLASHY-DESIGN.md** (~15 min)
   - Remove: elite-design.css, premium-design.css, data-viz.css, micro-interactions.css, transaction-viz.css
   - Keep: loading.css and loading.js (functional loading states)
   - Update templates/tiers.html and templates/index.html to remove deleted CSS links
   - Test that app still works with `go run .`
   - Commit: "chore: remove overly flashy elite design CSS"

2. **CLI-MODE.md** (~6 hours)
   - Create cli/ package with cli.go, commands.go, output.go, tests.go
   - Implement commands: user, league, tiers, dynasty-values, player, test
   - Add flags: --json, --debug, --cache-off
   - Update main.go to detect CLI mode and route to cli.Run()
   - Reuse existing fetch functions (fetchSleeperUser, fetchBorisChenTiers, etc.)
   - Test all CLI commands work correctly
   - Commit: "feat: add CLI mode for testing"

3. **OPENTELEMETRY-OBSERVABILITY.md** (~10 hours)
   - Install OTEL dependencies (go get go.opentelemetry.io/otel/...)
   - Create otel/ package with otel.go and metrics.go
   - Initialize OTEL SDK in main.go with cleanup
   - Wrap HTTP handlers with otelhttp middleware
   - Add spans to key functions (fetch.go, roster.go, dynasty.go)
   - Implement structured logging with trace correlation (logger/ package)
   - Create docker-compose.otel.yml with Jaeger, Prometheus, Grafana
   - Create otel-collector-config.yaml and prometheus.yml
   - Add documentation for running observability stack
   - Commit: "feat: add OpenTelemetry observability"

**Phase 4+: Remaining Features (After Phase 1-3)**

4. **Power User Improvements** (from power-user-improvements.md)
   - League selector dropdown for users with 5+ leagues
   - Searchable league list with grouping (Dynasty/Redraft)
   - Favorite/star leagues functionality
   - Better transaction display (two-column trade layout)
   - Transaction filters (All/Trades/Waivers/FA, date range)
   - User-centric trade views with dynasty value context
   - Remember last viewed league
   - Commit: "feat: add power user improvements for multi-league management"

5. **Admin Dashboard** (from feature-gaps-analysis.md)
   - Create /admin route with secret key authentication
   - Real-time metrics display (users online, lookups today, leagues viewed)
   - Growth metrics (daily/weekly/monthly active users)
   - Feature usage statistics (dynasty mode %, free agents clicks)
   - Error logs and debugging info
   - Browser/device breakdown
   - Most popular leagues/players
   - Use Prometheus metrics if available, otherwise in-memory counters
   - Commit: "feat: add admin dashboard with usage metrics"

6. **Enhanced League Features** (from feature-gaps-analysis.md)
   - League settings display (roster requirements, scoring format, league size)
   - Team standings table with current records
   - Playoff positioning indicator
   - Player search/filter within league view
   - Commit: "feat: add league settings, standings, and player search"

7. **Mobile/PWA Enhancements** (from feature-gaps-analysis.md)
   - Create manifest.json for PWA (add to home screen)
   - Service worker for offline support
   - Touch-friendly optimizations
   - Swipe gestures for league tabs (optional)
   - Bottom navigation for mobile
   - Commit: "feat: add PWA support and mobile optimizations"

8. **Additional Improvements** (from various plan docs)
   - Better error messages with retry buttons
   - Toast notifications for actions
   - Demo mode (show mock data without login)
   - Export league data to CSV (CLI command)
   - Commit per feature group

### Working Guidelines

**Code Quality:**
- Follow existing patterns in main.go, handlers.go, fetch.go, dynasty.go
- Use Go standard library (no frameworks)
- Keep it simple - don't over-engineer
- Only implement what's in the plan docs
- No backwards-compatibility hacks
- Delete unused code completely

**Git Workflow:**
- One commit per major feature (3 commits total)
- Use conventional commits: `feat:`, `fix:`, `chore:`, `docs:`
- ALWAYS add footer: `Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>`
- Test before committing when possible
- Use HEREDOC for multi-line commit messages:
  ```bash
  git commit -m "$(cat <<'EOF'
  feat: add CLI mode for testing

  - Create cli/ package with command router
  - Add user, league, tiers, dynasty-values, player, test commands
  - Support --json flag for machine-readable output
  - Integration test suite in cli test command

  Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
  EOF
  )"
  ```

**Testing:**
- Run `go run .` after phase 1 to verify web app still works
- Test CLI commands after phase 2: `./sleeperPy cli user testuser`
- Verify OTEL traces show up after phase 3: docker-compose -f docker-compose.otel.yml up
- If something breaks, fix it before moving on
- Use existing test mode: `go run . --test` (testuser mock data)

**Error Handling:**
- If a function doesn't exist, create it following existing patterns
- If you need architectural decisions, follow plan doc recommendations
- If you encounter ambiguity, choose the simpler approach
- Ask user only if truly blocked

### Project Context

**Architecture:**
- Backend: Go with standard library (no frameworks)
- Frontend: HTML templates (html/template) + vanilla JavaScript
- APIs: Sleeper, Boris Chen, DynastyProcess (KTC)
- Caching: In-memory (simple maps with mutexes)

**Key Files:**
- `main.go` - Entry point, server setup, template functions
- `handlers.go` - HTTP handlers (indexHandler, lookupHandler, etc.)
- `fetch.go` - API fetching (Sleeper, Boris Chen, KTC)
- `dynasty.go` - Dynasty features (news, breakouts, trade targets, etc.)
- `roster.go` - Roster processing and tier assignment
- `types.go` - All struct definitions
- `utils.go` - Utility functions
- `templates/tiers.html` - Main results page
- `templates/index.html` - Landing page

**Existing Functions to Reuse:**
- `fetchSleeperUser(username string)` - Get Sleeper user
- `fetchUserLeagues(userID string)` - Get user's leagues
- `fetchBorisChenTiers(format string)` - Get tiers
- `fetchDynastyValues()` - Get KTC values
- `fetchSleeperPlayers()` - Get all NFL players
- `analyzeLeague(leagueID, username string)` - Full league analysis

**Completed Features (see MEMORY.md):**
- Core tier analysis with Boris Chen
- Dynasty toolkit (news, breakouts, aging alerts, draft capital, trade targets, transactions, power rankings, rookies)
- Dynasty mode with collapseable sections
- Dark/light theme switcher
- Mobile responsive design
- Product pages (Privacy, ToS, About, FAQ, Pricing, SEO)

### Implementation Steps

**Phase 1: Remove Flashy Design (15 min)**
```bash
# 1. Remove CSS files
git rm static/elite-design.css static/premium-design.css static/data-viz.css static/micro-interactions.css static/transaction-viz.css

# 2. Edit templates/tiers.html - remove these lines:
# <link rel="stylesheet" href="/static/premium-design.css">
# <link rel="stylesheet" href="/static/data-viz.css">
# <link rel="stylesheet" href="/static/micro-interactions.css">
# <link rel="stylesheet" href="/static/elite-design.css">
# <link rel="stylesheet" href="/static/transaction-viz.css">

# 3. Check templates/index.html for same links

# 4. Test
go run .
# Visit localhost:8080, verify it works, loading states still present

# 5. Commit
git add -A
git commit -m "..."
```

**Phase 2: CLI Mode (6 hours)**

Create file structure:
```
cli/
├── cli.go       # Main router, Run(), printUsage()
├── commands.go  # cmdUser, cmdLeague, cmdTiers, etc.
├── output.go    # printLeagueAnalysis, JSON formatting
└── tests.go     # cmdTest, integration test suite
```

Key implementation details:
- CLI detection in main.go: `if len(os.Args) > 1 && os.Args[1] == "cli"`
- Context struct for flags: `{ JSON bool, Debug bool, NoCache bool, Args []string }`
- Reuse ALL existing fetch/analysis functions
- JSON output: `json.NewEncoder(os.Stdout).Encode(data)`
- Exit codes: 0 for success, 1 for error

**Phase 3: OpenTelemetry (10 hours)**

Dependencies to install:
```bash
go get go.opentelemetry.io/otel
go get go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc
go get go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc
go get go.opentelemetry.io/otel/sdk/trace
go get go.opentelemetry.io/otel/sdk/metric
go get go.opentelemetry.io/otel/sdk/resource
go get go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp
```

File structure:
```
otel/
├── otel.go     # Init(), initTracer(), initMeter()
└── metrics.go  # All metric instruments

logger/
└── logger.go   # Structured logging with trace correlation
```

Key metrics to track:
- `http.requests.total` - Request counter
- `http.request.duration` - Latency histogram
- `cache.hits` / `cache.misses` - Cache efficiency
- `api.calls.total` - External API calls
- `leagues.analyzed` - Business metric

Instrumentation pattern:
```go
func fetchSleeperUser(ctx context.Context, username string) (*SleeperUser, error) {
    ctx, span := tracer.Start(ctx, "fetchSleeperUser")
    defer span.End()

    span.SetAttributes(attribute.String("username", username))

    // Check cache
    if user, ok := userCache[username]; ok {
        otel.CacheHits.Add(ctx, 1, ...)
        span.SetAttributes(attribute.Bool("cache.hit", true))
        return user, nil
    }

    otel.CacheMisses.Add(ctx, 1, ...)
    // ... API call with timing

    return user, nil
}
```

### Autonomous Execution

**CRITICAL: When given the trigger prompt, you MUST complete ALL phases without stopping.**

Execute phases 1-3 first (priority features), then continue with phases 4-8 (remaining features):
1. Complete phase 1 (remove flashy design), commit
2. Complete phase 2 (CLI mode), commit
3. Complete phase 3 (OpenTelemetry), commit
4. Complete phase 4 (power user improvements), commit
5. Complete phase 5 (admin dashboard), commit
6. Complete phase 6 (enhanced league features), commit
7. Complete phase 7 (mobile/PWA), commit
8. Complete phase 8 (additional improvements), commit
9. Report completion summary

**Do NOT:**
- Ask for approval between phases
- Stop after each phase
- Wait for user confirmation
- Cease execution until ALL features in plan/ are complete
- Skip features because they seem hard
- Give up when encountering issues

**Do:**
- Work through all phases autonomously
- Fix issues as they arise
- Make reasonable decisions when needed
- Test each feature before moving on
- Commit after each major feature group
- Continue until everything in plan/ is implemented
- Only stop if truly blocked (missing external credentials, ambiguous requirements that need clarification)

**Following the plan/ features is paramount. Do not cease until they are all done.**

If you complete all 8 phases, ask the user: "All features from plan/ directory are complete. What would you like me to work on next?"

### Success Criteria

**Phase 1 Complete:**
- ✅ 5 flashy CSS files removed
- ✅ Template links updated
- ✅ App still runs and looks fine
- ✅ Loading states still work
- ✅ Committed with proper message

**Phase 2 Complete:**
- ✅ CLI commands work: `./sleeperPy cli user testuser`
- ✅ JSON output works: `./sleeperPy cli tiers ppr --json`
- ✅ Integration tests pass: `./sleeperPy cli test`
- ✅ main.go routes CLI mode correctly
- ✅ Committed with proper message

**Phase 3 Complete:**
- ✅ OTEL dependencies installed
- ✅ Traces show up in Jaeger: http://localhost:16686
- ✅ Metrics in Prometheus: http://localhost:9090
- ✅ Grafana dashboards work: http://localhost:3000
- ✅ HTTP handlers instrumented
- ✅ Key functions have spans
- ✅ Logs have trace correlation
- ✅ docker-compose.otel.yml works
- ✅ Committed with proper message

## General Project Guidelines

**Coding Style:**
- Go: Follow Go conventions, use `gofmt`
- No emojis in code or commits (unless explicitly requested)
- Comments only where logic isn't self-evident
- Simple error messages with actionable next steps

**Security:**
- No command injection vulnerabilities
- Validate user input
- No SQL injection (though we don't use SQL)
- Proper error handling without leaking internals

**Performance:**
- Cache API responses appropriately
- Don't block on slow operations
- Use goroutines only when needed
- Keep memory usage reasonable

**Documentation:**
- Update README.md if adding major features
- Add comments to complex algorithms
- Include examples in CLI help text
- Document environment variables

## Debugging Tips

**If app doesn't compile:**
- Check imports
- Run `go mod tidy`
- Verify function signatures match

**If tests fail:**
- Use `--test` flag for mock data
- Check API rate limits
- Verify cache isn't stale
- Enable debug logging: `--log=debug`

**If OTEL doesn't work:**
- Check OTEL_EXPORTER_OTLP_ENDPOINT env var
- Verify otel-collector is running: `docker ps`
- Check collector logs: `docker logs <container>`
- Visit zpages: http://localhost:55679/debug/tracez

## Reference

- Full implementation details in plan/ directory
- Project history in MEMORY.md
- Completed features listed in MEMORY.md
- Bug tracking in plan/current-bugs.md
