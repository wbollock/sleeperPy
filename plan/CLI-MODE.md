# CLI Mode for Testing

## Problem
Hard to test frontend logic because it's tightly coupled to the web server. Need a way to:
- Test API fetching without browser
- Debug roster calculations
- Validate dynasty value lookups
- Run integration tests in CI/CD
- Profile performance

## Solution
Add CLI mode that reuses existing backend logic but outputs to stdout instead of rendering HTML.

## Design

### Command Structure
```bash
# Main entry point
./sleeperPy cli <command> [args] [flags]

# Commands
./sleeperPy cli user <username>
./sleeperPy cli league <league_id> <username>
./sleeperPy cli tiers <format>
./sleeperPy cli dynasty-values
./sleeperPy cli player <name>
./sleeperPy cli test

# Flags
--json          Output JSON (machine readable)
--debug         Enable debug logging
--cache-off     Disable caching for testing
```

### Command Details

#### `cli user <username>`
**Purpose**: Fetch user's leagues

**Output**:
```
User: wbollock (ID: 123456789)
Found 3 leagues:

  - 5 Bags of Popcorn Dynasty (league_123)
    Type: dynasty, Scoring: ppr, Teams: 12

  - BDGE Dynasty League 60 (league_456)
    Type: dynasty, Scoring: ppr, Teams: 12

  - Summer Best Ball v44 (league_789)
    Type: bestball, Scoring: ppr, Teams: 12
```

**JSON Output** (`--json`):
```json
{
  "user_id": "123456789",
  "username": "wbollock",
  "leagues": [
    {
      "league_id": "league_123",
      "name": "5 Bags of Popcorn Dynasty",
      "type": "dynasty",
      "scoring": "ppr",
      "teams": 12
    }
  ]
}
```

#### `cli league <league_id> <username>`
**Purpose**: Analyze specific league

**Output**:
```
League: 5 Bags of Popcorn Dynasty
User: wbollock (Roster ID: 1)
Record: 8-5 (3rd place)

Starters:
  QB  Patrick Mahomes      Tier 1   Value: 3500
  RB  Bijan Robinson       Tier 2   Value: 4200
  RB  Jahmyr Gibbs         Tier 3   Value: 3800
  WR  CeeDee Lamb          Tier 1   Value: 5100
  WR  Amon-Ra St. Brown    Tier 2   Value: 4800
  TE  Sam LaPorta          Tier 2   Value: 2800
  FLEX James Cook           Tier 3   Value: 2900

Bench: 15 players
Total Roster Value: 48,500
Average Age: 24.3 years

Free Agent Upgrades: 2 available
```

#### `cli tiers <format>`
**Purpose**: Fetch Boris Chen tiers

**Formats**: `ppr`, `half-ppr`, `standard`, `superflex`

**Output**:
```
Boris Chen Tiers (ppr)
Fetched at: 2026-02-06 10:30:00

Tier 1 (Elite): 8 players
Tier 2 (Great): 12 players
Tier 3 (Good): 15 players
Tier 4 (Solid): 18 players
...

Cache: HIT (expires in 14m)
```

#### `cli dynasty-values`
**Purpose**: Fetch KTC dynasty values

**Output**:
```
Dynasty Values (KeepTradeCut)
Fetched at: 2026-02-06 10:30:00
Total players: 2,847

Top 10 Most Valuable:
  1. CeeDee Lamb (WR)          9,999
  2. Breece Hall (RB)          9,876
  3. Bijan Robinson (RB)       9,654
  4. Ja'Marr Chase (WR)        9,543
  5. Justin Jefferson (WR)     9,432
  ...

Cache: MISS (fetching fresh data)
```

#### `cli player <name>`
**Purpose**: Look up player info

**Output**:
```
Player: Patrick Mahomes
Position: QB
Age: 28
Team: KC

Boris Chen Tier (PPR): 1
Dynasty Value: 3,500

Status: Active
Injury: None
```

#### `cli test`
**Purpose**: Run integration tests

**Output**:
```
Running integration tests...

  Fetch test user (testuser)... ✅ PASS (234ms)
  Fetch Boris Chen tiers (ppr)... ✅ PASS (892ms)
  Fetch dynasty values (KTC)... ✅ PASS (1.2s)
  Analyze test league... ✅ PASS (456ms)
  Cache hit test... ✅ PASS (12ms)
  Cache miss test... ✅ PASS (234ms)

6 passed, 0 failed
Total time: 3.1s
```

## Architecture

### File Structure
```
.
├── main.go              # Entry point, route to CLI or web
├── cli/
│   ├── cli.go          # CLI router and commands
│   ├── commands.go     # Command implementations
│   ├── output.go       # Output formatting (text vs JSON)
│   └── tests.go        # Integration test suite
├── fetch.go            # Shared API fetching logic
├── roster.go           # Shared roster processing
└── types.go            # Shared data structures
```

### Main Entry Point

**File: `main.go`**
```go
func main() {
    // Parse flags
    testMode := flag.Bool("test", false, "Run in test mode")
    logLevel := flag.String("log", "info", "Log level")
    flag.Parse()

    // Check if CLI mode
    args := flag.Args()
    if len(args) > 0 && args[0] == "cli" {
        // Run CLI mode
        os.Exit(cli.Run(args[1:]))
    }

    // Otherwise run web server
    startWebServer(*testMode, *logLevel)
}
```

### CLI Package

**File: `cli/cli.go`**
```go
package cli

import (
    "flag"
    "fmt"
    "os"
)

// Run executes CLI commands
func Run(args []string) int {
    if len(args) == 0 {
        printUsage()
        return 1
    }

    // Parse CLI flags
    fs := flag.NewFlagSet("cli", flag.ExitOnError)
    jsonOutput := fs.Bool("json", false, "Output JSON")
    debug := fs.Bool("debug", false, "Enable debug logging")
    noCacheflag := fs.Bool("cache-off", false, "Disable caching")
    fs.Parse(args[1:])

    // Create CLI context
    ctx := &Context{
        JSON:     *jsonOutput,
        Debug:    *debug,
        NoCache:  *noCacheflag,
        Args:     fs.Args(),
    }

    // Route to command
    command := args[0]
    switch command {
    case "user":
        return cmdUser(ctx)
    case "league":
        return cmdLeague(ctx)
    case "tiers":
        return cmdTiers(ctx)
    case "dynasty-values":
        return cmdDynastyValues(ctx)
    case "player":
        return cmdPlayer(ctx)
    case "test":
        return cmdTest(ctx)
    default:
        fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
        printUsage()
        return 1
    }
}

type Context struct {
    JSON    bool
    Debug   bool
    NoCache bool
    Args    []string
}

func printUsage() {
    fmt.Println(`SleeperPy CLI

Usage:
  sleeperPy cli <command> [args] [flags]

Commands:
  user <username>              Fetch user's leagues
  league <league_id> <user>    Analyze specific league
  tiers <format>               Fetch Boris Chen tiers (ppr, half-ppr, standard, superflex)
  dynasty-values               Fetch KTC dynasty values
  player <name>                Look up player
  test                         Run integration tests

Flags:
  --json        Output JSON (machine readable)
  --debug       Enable debug logging
  --cache-off   Disable caching for testing

Examples:
  sleeperPy cli user wbollock
  sleeperPy cli league 123456789 wbollock --json
  sleeperPy cli tiers ppr
  sleeperPy cli test --debug`)
}
```

**File: `cli/commands.go`**
```go
package cli

import (
    "encoding/json"
    "fmt"
    "os"
)

func cmdUser(ctx *Context) int {
    if len(ctx.Args) < 1 {
        fmt.Fprintln(os.Stderr, "Usage: cli user <username>")
        return 1
    }

    username := ctx.Args[0]

    // Reuse existing fetchSleeperUser function
    user, err := fetchSleeperUser(username)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        return 1
    }

    // Fetch leagues
    leagues, err := fetchUserLeagues(user.UserID)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        return 1
    }

    // Output
    if ctx.JSON {
        json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
            "user_id":  user.UserID,
            "username": user.Username,
            "leagues":  leagues,
        })
    } else {
        fmt.Printf("User: %s (ID: %s)\n", user.Username, user.UserID)
        fmt.Printf("Found %d leagues:\n\n", len(leagues))
        for _, league := range leagues {
            fmt.Printf("  - %s (%s)\n", league.Name, league.LeagueID)
            fmt.Printf("    Type: %s, Scoring: %s, Teams: %d\n\n",
                league.Type, league.Scoring, league.TotalRosters)
        }
    }

    return 0
}

func cmdLeague(ctx *Context) int {
    if len(ctx.Args) < 2 {
        fmt.Fprintln(os.Stderr, "Usage: cli league <league_id> <username>")
        return 1
    }

    leagueID := ctx.Args[0]
    username := ctx.Args[1]

    // Reuse existing league analysis logic
    leagueData, err := analyzeLeague(leagueID, username)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        return 1
    }

    // Output
    if ctx.JSON {
        json.NewEncoder(os.Stdout).Encode(leagueData)
    } else {
        printLeagueAnalysis(leagueData)
    }

    return 0
}

// ... more command implementations
```

**File: `cli/output.go`**
```go
package cli

import "fmt"

func printLeagueAnalysis(data *LeagueData) {
    fmt.Printf("League: %s\n", data.Name)
    fmt.Printf("User: %s (Roster ID: %d)\n", data.Username, data.UserRosterID)
    if data.Record != "" {
        fmt.Printf("Record: %s\n", data.Record)
    }
    fmt.Println()

    fmt.Println("Starters:")
    for _, p := range data.Starters {
        fmt.Printf("  %-4s %-20s Tier %-2d  Value: %d\n",
            p.Pos, p.Name, p.Tier, p.DynastyValue)
    }
    fmt.Println()

    fmt.Printf("Bench: %d players\n", len(data.Bench))
    fmt.Printf("Total Roster Value: %d\n", data.TotalRosterValue)
    fmt.Printf("Average Age: %.1f years\n", data.AvgAge)
    fmt.Println()

    if len(data.FreeAgentUpgrades) > 0 {
        fmt.Printf("Free Agent Upgrades: %d available\n", len(data.FreeAgentUpgrades))
    }
}
```

**File: `cli/tests.go`**
```go
package cli

import (
    "fmt"
    "time"
)

func cmdTest(ctx *Context) int {
    fmt.Println("Running integration tests...\n")

    tests := []Test{
        {"Fetch test user (testuser)", testFetchUser},
        {"Fetch Boris Chen tiers (ppr)", testFetchTiers},
        {"Fetch dynasty values (KTC)", testFetchDynastyValues},
        {"Analyze test league", testAnalyzeLeague},
        {"Cache hit test", testCacheHit},
        {"Cache miss test", testCacheMiss},
    }

    passed := 0
    failed := 0
    var totalTime time.Duration

    for _, test := range tests {
        fmt.Printf("  %s... ", test.Name)
        start := time.Now()
        err := test.Fn()
        elapsed := time.Since(start)
        totalTime += elapsed

        if err != nil {
            fmt.Printf("❌ FAIL (%v)\n", err)
            if ctx.Debug {
                fmt.Printf("    Error: %v\n", err)
            }
            failed++
        } else {
            fmt.Printf("✅ PASS (%dms)\n", elapsed.Milliseconds())
            passed++
        }
    }

    fmt.Printf("\n%d passed, %d failed\n", passed, failed)
    fmt.Printf("Total time: %.1fs\n", totalTime.Seconds())

    if failed > 0 {
        return 1
    }
    return 0
}

type Test struct {
    Name string
    Fn   func() error
}

func testFetchUser() error {
    _, err := fetchSleeperUser("testuser")
    return err
}

func testFetchTiers() error {
    _, err := fetchBorisChenTiers("ppr")
    return err
}

// ... more test functions
```

## Benefits

### For Development
- ✅ Test backend logic without browser
- ✅ Easier debugging (no HTML rendering)
- ✅ Profile performance bottlenecks
- ✅ Validate API changes quickly

### For CI/CD
- ✅ Run integration tests in pipeline
- ✅ Check if external APIs are working
- ✅ Verify data parsing logic
- ✅ Catch regressions early

### For Users (Future)
- ✅ Scripting and automation
- ✅ Cron jobs to check values
- ✅ Export data to CSV
- ✅ Build custom tools on top

## Implementation Order

### Phase 1: Basic Commands (2 hours)
1. Create `cli/` package
2. Implement CLI router
3. Add `user` command
4. Add `tiers` command
5. Add `--json` flag support

### Phase 2: League Analysis (2 hours)
1. Add `league` command
2. Refactor league analysis to be reusable
3. Add pretty-print output
4. Test with real league data

### Phase 3: Testing (1 hour)
1. Add `test` command
2. Implement integration tests
3. Add `--debug` flag
4. Test in CI environment

### Phase 4: Polish (1 hour)
1. Add `dynasty-values` command
2. Add `player` command
3. Add `--cache-off` flag
4. Write documentation
5. Add examples to README

**Total: 6 hours**

## Testing Plan

### Manual Testing
```bash
# Test user lookup
./sleeperPy cli user wbollock
./sleeperPy cli user wbollock --json

# Test league analysis
./sleeperPy cli league 123456789 wbollock
./sleeperPy cli league 123456789 wbollock --json

# Test tiers
./sleeperPy cli tiers ppr
./sleeperPy cli tiers superflex --json

# Test dynasty values
./sleeperPy cli dynasty-values
./sleeperPy cli dynasty-values --json

# Run tests
./sleeperPy cli test
./sleeperPy cli test --debug
```

### CI Integration
```yaml
# .github/workflows/test.yml
name: Integration Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Build
        run: go build -o sleeperPy .

      - name: Run CLI tests
        run: ./sleeperPy cli test

      - name: Test user lookup
        run: ./sleeperPy cli user testuser --json
```

## Success Criteria
- ✅ Can fetch user leagues from CLI
- ✅ Can analyze league without web UI
- ✅ JSON output works for scripting
- ✅ Integration tests pass
- ✅ Works in CI/CD pipeline
- ✅ Documentation is clear
- ✅ Error messages are helpful

## Future Enhancements
- CSV export: `./sleeperPy cli export league_123 --format csv`
- Compare leagues: `./sleeperPy cli compare league1 league2`
- Watch mode: `./sleeperPy cli watch user wbollock`
- Shell completion: bash/zsh autocomplete
- Config file: `~/.sleeperpy/config.yaml`
