# Dynasty League Offseason Features - Detailed Plan

## Overview
This plan outlines features to enhance the SleeperPy application for dynasty leagues during the offseason. Dynasty leagues require different tooling than redraft leagues, especially when there are no active matchups.

## Current Problem
- Leagues without matchups (offseason) are currently skipped in `main.go:319-324`
- Dynasty leagues are not distinguished from redraft leagues
- No dynasty-specific features exist

## Phase 1: Make Leagues Visible in Offseason

### 1.1 Remove Matchup Requirement for Dynasty Leagues
**Goal**: Display dynasty leagues even when `matchups` endpoint returns empty/error

**Implementation**:
- Modify `lookupHandler()` in `main.go` around line 319-324
- Instead of `continue` when no matchups found, check if league is dynasty
- If dynasty AND offseason, skip matchup-dependent features but still display league
- Store a flag `HasMatchups` in `LeagueData` struct

**Code Changes**:
```go
// In LeagueData struct (line ~198)
type LeagueData struct {
    LeagueName      string
    Scoring         string
    IsDynasty       bool  // NEW
    HasMatchups     bool  // NEW
    // ... existing fields
}

// In lookupHandler (line ~319)
matchups, err := fetchJSONArray(...)
hasMatchups := (err == nil && len(matchups) > 0)

// Don't skip league if it's dynasty, just mark it
if !hasMatchups {
    log.Printf("[INFO] No matchups for league %s (possibly offseason)", leagueName)
    // Continue processing but with HasMatchups = false
}
```

### 1.2 Dynasty League Detection
**Goal**: Identify which leagues are dynasty leagues

**Detection Strategy**:
Based on research, Sleeper API doesn't have an explicit `type: "dynasty"` field. We need to infer it from:

1. **League settings** - Check for:
   - `settings.keeper_deadline` > 0 (indicates keeper/dynasty)
   - Multiple draft years in league history
   - Presence of future draft picks in transactions

2. **Heuristic approach** (simpler, start here):
   - Check if league has `settings.keeper_deadline` field
   - OR: Prompt user to manually mark dynasty leagues (least technical debt)
   - OR: Check league name for "dynasty" keyword as fallback

**Implementation**:
```go
func isDynastyLeague(league map[string]interface{}) bool {
    // Check settings for keeper_deadline
    if settings, ok := league["settings"].(map[string]interface{}); ok {
        if keeperDeadline, ok := settings["keeper_deadline"].(float64); ok && keeperDeadline > 0 {
            return true
        }
    }

    // Fallback: check league name for "dynasty" keyword
    if name, ok := league["name"].(string); ok {
        nameLower := strings.ToLower(name)
        return strings.Contains(nameLower, "dynasty")
    }

    return false
}
```

### 1.3 UI Changes for Dynasty Indication
**Goal**: Mark dynasty leagues with a star/badge in the UI

**Implementation**:
- Add star ⭐ or badge to league tab name for dynasty leagues
- Update `templates/tiers.html` line 33:
  ```html
  <button class="league-tab" {{if eq $i 0}}id="defaultTab"{{end}} onclick="showLeagueTab(event, 'league{{$i}}')">
      {{if $l.IsDynasty}}⭐ {{end}}{{$l.LeagueName}} ({{$l.Scoring}})
  </button>
  ```

**Testing**:
- Run with Jesse's dynasty league "Dynasty League 2025 10-Team Dynasty SF PPR TEP"
- Verify star appears in UI
- Verify league loads in offseason without matchups

## Phase 2: Dynasty-Specific Features

### 2.1 Keep Trade Cut (KTC) Player Values
**Goal**: Show KTC values for all players on roster

**Data Source**:
- KTC provides a public CSV/API at https://keeptradecut.com/dynasty-rankings
- Alternative: Scrape their rankings page (less ideal)
- Cache with 24-hour TTL (rankings don't change that often)

**Implementation**:
```go
// New struct for KTC data
type KTCValue struct {
    PlayerName string
    Value      int     // 0-10000 scale
    Trend      string  // "up", "down", "stable"
    AgeAdjusted int    // Value adjusted for age
}

// Cache similar to Boris tiers
var ktcCache = &tiersCache{
    data:      make(map[string]map[string]KTCValue),
    timestamp: make(map[string]time.Time),
    ttl:       24 * time.Hour,
}

func fetchKTCValues() map[string]KTCValue {
    // Fetch from KTC API/CSV
    // Parse and return map[playerID]KTCValue
}

// Add to PlayerRow struct
type PlayerRow struct {
    // ... existing fields
    KTCValue    int    // NEW - only for dynasty leagues
    KTCTrend    string // NEW - "↑", "↓", "→"
}
```

**UI Changes**:
- Add KTC column to roster table for dynasty leagues only
- Color code values: green (>7000), yellow (4000-7000), red (<4000)
- Show trend arrows next to value

**Template Changes** (`templates/tiers.html`):
```html
{{if $l.IsDynasty}}
<th>KTC Value</th>
{{end}}

{{if $l.IsDynasty}}
<td>{{.KTCValue}} {{.KTCTrend}}</td>
{{end}}
```

### 2.2 Top 5 Rookies To-Be-Drafted
**Goal**: Show top rookie prospects for upcoming draft

**Data Source**:
- KeepTradeCut rookie rankings
- DynastyNerds rookie rankings
- Or manual curated list for 2025 draft

**Implementation**:
```go
type RookieProspect struct {
    Name     string
    Position string
    College  string
    KTCValue int
    Rank     int // Overall rank among rookies
}

func getTopRookies(year int) []RookieProspect {
    // Fetch from external source or static list
    // Return top 5
}

// Add to LeagueData for dynasty leagues
type LeagueData struct {
    // ... existing fields
    TopRookies []RookieProspect // NEW - only for dynasty
}
```

**UI**:
- New section below free agents: "Top Rookie Prospects for 2025 Draft"
- Show name, position, college, projected value
- Only display for dynasty leagues

### 2.3 Available Draft Picks
**Goal**: Show what draft picks the user has

⚠️ **KNOWN ISSUE**: Draft picks ownership logic may not be 100% accurate. The current implementation may show picks that were traded away or not show acquired picks correctly. This needs debugging with real API data to fix the ownership tracking logic.

**Data Source**:
- Sleeper API: `/v1/league/{league_id}/traded_picks` endpoint
- Shows all traded picks (both acquired and traded away)
- Combine with default picks to show complete picture

**Implementation**:
```go
type DraftPick struct {
    Round  int
    Year   int
    Owner  string // "You" or other team name
    Original string // Original owner if traded
}

func getAvailablePicks(leagueID, userID string) []DraftPick {
    // 1. Get traded picks from API
    tradedPicks := fetchJSONArray(fmt.Sprintf("https://api.sleeper.app/v1/league/%s/traded_picks", leagueID))

    // 2. Determine default picks (1 per round per year for next 3 years)
    // 3. Adjust for traded picks
    // 4. Return sorted list
}

// Add to LeagueData
type LeagueData struct {
    // ... existing fields
    DraftPicks []DraftPick // NEW - dynasty only
}
```

**UI**:
- Section: "Your Draft Capital"
- Group by year, then by round
- Highlight acquired picks in green, traded away in red
- Show pick origin if traded (e.g., "2025 Round 1 (from Team X)")

### 2.4 Team Needs Analysis
**Goal**: Identify positional weaknesses on roster

**Implementation**:
```go
type TeamNeed struct {
    Position     string
    Severity     string // "Critical", "Moderate", "Depth"
    Reason       string // Human-readable explanation
    SuggestedFAs []PlayerRow // Top 3 FAs for this position
}

func analyzeTeamNeeds(starters, bench []PlayerRow, isDynasty bool) []TeamNeed {
    needs := []TeamNeed{}

    // For each position, analyze:
    // 1. Starter tier quality (if tier > 5, it's a need)
    // 2. Bench depth (if no bench player for position, it's a need)
    // 3. For dynasty: Age concerns (if player is old, project future need)

    // Example logic:
    // - No starter or tier > 7: "Critical"
    // - Starter tier 5-7: "Moderate"
    // - Weak bench depth: "Depth"

    return needs
}

// Add to LeagueData
type LeagueData struct {
    // ... existing fields
    TeamNeeds []TeamNeed // NEW
}
```

**UI**:
- Section: "Team Needs Analysis"
- List needs by severity (Critical → Moderate → Depth)
- For each need, show suggested free agents
- Color code: Red (Critical), Yellow (Moderate), Blue (Depth)

### 2.5 Additional Dynasty Features (Nice-to-Have)

#### 2.5.1 Age Profile
- Show average age of roster
- Highlight aging players (>30 for RB, >32 for WR/TE/QB)
- "Rebuild" vs "Win Now" classification based on age

#### 2.5.2 Total Roster Value (KTC)
- Sum all KTC values
- Show percentile rank vs other teams in league
- Track value changes week-over-week

#### 2.5.3 Dynasty Trade Calculator
- Input: Players you're trading vs receiving
- Output: KTC value difference
- Show if trade is fair/unfair

#### 2.5.4 Startup Draft Simulator
- For dynasty leagues preparing for startup drafts
- Mock draft interface
- Show ADP (average draft position) data

## Phase 3: Offseason-Specific UI

### 3.1 Modify Win Probability Section
**Problem**: Win probability doesn't make sense in offseason

**Solution**:
- If `HasMatchups == false`, hide win probability section
- Replace with dynasty-focused metrics:
  - Roster strength score
  - Championship window indicator
  - Rebuild progress meter

### 3.2 Adjust Bench Recommendations
**Problem**: "Swap" recommendations assume current week optimization

**Solution**:
- For dynasty offseason, change focus to long-term value
- Instead of "swap in", show "buy low candidates" (players with bad tiers but high KTC)
- Show "sell high candidates" (players with good tiers but declining KTC)

## Implementation Order (Recommended)

### Sprint 1: Foundation (2-3 hours)
1. Remove matchup requirement for dynasty leagues
2. Add dynasty detection logic
3. Mark dynasty leagues with star in UI
4. Test with Jesse's league

### Sprint 2: KTC Integration (3-4 hours)
1. Fetch and cache KTC values
2. Add KTC column to roster tables
3. Show total roster value
4. Test with real data

### Sprint 3: Draft Picks & Rookies (2-3 hours)
1. Fetch traded picks from API
2. Calculate available draft capital
3. Add top rookies section
4. UI for both features

### Sprint 4: Team Needs (2-3 hours)
1. Implement needs analysis algorithm
2. Connect to existing FA recommendations
3. Add UI section
4. Test and refine

### Sprint 5: Polish (1-2 hours)
1. Offseason-specific UI adjustments
2. Age profile and additional metrics
3. Mobile responsiveness
4. Documentation

## Technical Considerations

### API Rate Limits
- Sleeper API: No documented rate limits, but be respectful
- KTC: May have rate limits, implement proper caching
- Use exponential backoff for retries

### Caching Strategy
- KTC values: 24-hour TTL
- Rookie rankings: 7-day TTL (updated weekly)
- Draft picks: 1-hour TTL (can change frequently during season)
- Store in memory (current approach) or consider Redis for production

### Testing
- Create mock dynasty league in `mock_server.go`
- Add `settings.keeper_deadline` to mock data
- Create test cases for:
  - Dynasty detection
  - Offseason display (no matchups)
  - KTC value display
  - Draft pick calculation

### Error Handling
- If KTC API is down, gracefully degrade (show without values)
- If traded picks endpoint fails, show default picks only
- Log all errors but don't break page rendering

## Open Questions for Jesse

1. **Dynasty Detection**: Do you want to rely on heuristics (league name, settings) or manually configure which leagues are dynasty?

2. **KTC Data Source**: KeepTradeCut doesn't have an official API. Options:
   - Web scraping (fragile)
   - Manual CSV updates (low maintenance)
   - Find alternative API (DynastyNerds, FantasyPros)

3. **Rookie Rankings**: Should we hard-code 2025 rookies or integrate with a live API?

4. **Scope**: Which features are MVP (must-have) vs nice-to-have? I recommend:
   - **MVP**: Offseason visibility, dynasty star, KTC values, draft picks
   - **V2**: Rookies, team needs, age profile
   - **V3**: Trade calculator, startup draft sim

5. **UI Real Estate**: The page is already dense. Should dynasty features be:
   - Separate tab/section?
   - Collapsed by default?
   - Separate page entirely?

## Success Metrics

After implementation, we should validate:
- ✅ Dynasty leagues are visible in offseason
- ✅ Dynasty leagues are clearly marked with ⭐
- ✅ KTC values display correctly for all players
- ✅ Draft picks are accurately calculated
- ✅ Team needs make sense based on roster
- ✅ Page loads in < 2 seconds even with extra data
- ✅ No errors in console or logs
- ✅ Mobile-friendly display

## Next Steps

1. Get Jesse's feedback on this plan
2. Clarify open questions
3. Start with Sprint 1 (foundation)
4. Iterate based on real usage
