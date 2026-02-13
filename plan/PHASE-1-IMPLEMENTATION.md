# Phase 1: Free Core Utility - Implementation Specs

## User Decisions (Feb 13, 2026)

**Feature #1: Cross-League Dashboard**
- Q1: Season overview focus - "how are my teams trending overall"
- Q2: Medium density (6-8 metrics per league)
- Q3: New tab (keep current flow, add dashboard option)

**Feature #2: Weekly Action List**
- Q4: Balanced personality (clear opportunities, some suggestions)
- Q5: 1 tier threshold (show more swaps, might be noisy - can tune later)
- Q6: Dismissible with checkboxes (if architecture allows)

**Feature #3: Trade Fairness + "Fleeced" Flag**
- Q7: Hybrid approach (flag extreme value gaps, note context separately)
- Q8: ~15% value delta (aggressive detection, iterate as needed)
- Q9: Subtle display (small badges, muted colors)

**Feature #4: News Signal Compression**
- Q10: Dynamic time window (this week during season, total offseason during offseason)
- Q11: Your players only (my roster across all leagues)
- Q12: Top 3 headlines (most critical only)

---

## Feature #1: Cross-League Dashboard

### **User Story**
"Show me how all my teams are trending this season - which leagues I'm dominating, which need attention, and where I should focus my time."

### **UX Design**

**New Route**: `/dashboard` (accessible via nav tab)

**Layout**: Grid of league cards (2-3 columns on desktop, 1 on mobile)

**Each card shows (6-8 metrics)**:
```
┌─────────────────────────────────────┐
│ 🏈 League Name (Dynasty • SF • 12T) │
│                                     │
│ 📊 Roster Value: 8,450 (#3/12)     │
│ 📈 Trend: ↗ +5% this month          │
│ 👴 Avg Age: 26.2 (#7/12)            │
│ 📅 Record: 8-5 (Playoffs ✓)        │
│ 🎯 Draft Picks: 2026 1st, 2027 1st │
│ ⚡ Actions: 2 pending               │
│                                     │
│ [View League →]                     │
└─────────────────────────────────────┘
```

**Metrics Priority**:
1. **Roster Value + Rank** (dynasty leagues only)
2. **Trend indicator** (+/- % from last check, via cached values)
3. **Avg Age + Rank** (dynasty leagues only)
4. **Record + Playoff status** (if in-season)
5. **Draft capital summary** (dynasty leagues only)
6. **Action count** (from Weekly Action List)
7. **League type badges** (Dynasty, SF, PPR, size)
8. **Last updated** timestamp

### **Data Requirements**

**New Types** (`types.go`):
```go
type LeagueSummary struct {
    LeagueID          string
    LeagueName        string
    Scoring           string
    IsDynasty         bool
    IsSuperFlex       bool
    LeagueSize        int

    // Dynasty metrics
    TotalRosterValue  int
    ValueRank         int  // 1-12
    ValueTrend        string // "↗ +5%", "↘ -3%", "→ stable"
    AvgAge            float64
    AgeRank           int
    DraftPicksSummary string // "2026 1st, 2027 1st, 2nd"

    // Season metrics
    Record            string // "8-5" or empty if offseason
    PlayoffStatus     string // "Clinched", "In Hunt", "Eliminated", ""

    // Action items
    ActionCount       int

    LastUpdated       time.Time
}

type DashboardPage struct {
    Username        string
    LeagueSummaries []LeagueSummary
    TotalLeagues    int
    DynastyCount    int
    RedraftCount    int
}
```

**Value Trend Calculation**:
- Cache dynasty values per league in memory with timestamp
- On dashboard load, compare current values to cached (24h ago)
- Calculate: `(current - cached) / cached * 100`
- Display: "↗ +5%" (green), "↘ -3%" (red), "→ stable" (gray)

### **Implementation Steps**

1. **Add route** (`main.go`):
   ```go
   http.HandleFunc("/dashboard", tiersHandler) // reuse handler, add dashboard mode
   ```

2. **New handler mode** (`handlers.go`):
   ```go
   func buildDashboardPage(username string) (*DashboardPage, error) {
       // Fetch all user leagues (already have this)
       // For each league:
       //   - Fetch basic league data
       //   - Calculate roster value + rank (dynasty only)
       //   - Calculate avg age + rank (dynasty only)
       //   - Get draft picks summary
       //   - Get record (if in-season)
       //   - Count pending actions (defer to Feature #2)
       //   - Calculate trend (compare to cache)
       // Return DashboardPage
   }
   ```

3. **Value trend cache** (`cache.go` or new `trend_cache.go`):
   ```go
   var valueTrendCache = make(map[string]CachedValue) // key: "username:leagueID"

   type CachedValue struct {
       RosterValue int
       Timestamp   time.Time
   }

   func getValueTrend(username, leagueID string, currentValue int) string {
       // Check cache, compare, return trend string
   }
   ```

4. **New template** (`templates/dashboard.html`):
   - Hero section: "Your Fantasy Football Empire"
   - Stats bar: "X Total Leagues • Y Dynasty • Z Redraft"
   - Grid of league cards
   - Click card → navigate to `/tiers?user=X&league=Y`

5. **Navigation update** (`templates/*.html`):
   - Add "Dashboard" tab to nav (between Home and current league view)

### **Edge Cases**
- No dynasty values → hide value/age/trend metrics
- Offseason → hide record/playoff status
- No trades/cached data → show "→ stable" trend
- New league (first visit) → show "New" badge, no trend

### **Testing Checklist**
- [ ] Dashboard loads for user with 1 league
- [ ] Dashboard loads for user with 5+ leagues
- [ ] Dynasty leagues show value/age metrics
- [ ] Redraft leagues hide dynasty metrics
- [ ] Trend calculation works (mock cache data)
- [ ] Mobile responsive (cards stack vertically)
- [ ] Click card navigates to correct league
- [ ] Dashboard tab in nav highlights when active

---

## Feature #2: Weekly Action List

### **User Story**
"Tell me what to do first this week - give me a balanced checklist of clear opportunities without overwhelming me."

### **UX Design**

**Location**: Top of league view page (above roster display) + dynasty toolkit sidebar

**Compact version (top of page)**:
```
⚡ 2 Actions This Week
1. ✓ Swap Starter: Start Jahmyr Gibbs over James Conner (+1.2 tiers)
2. Check Waiver Wire: Tank Bigsby available (would start over Conner)

[View All Actions →]
```

**Detailed version (dynasty toolkit sidebar)**:
```
📋 Weekly Action List

✓ 1. Swap Starter
   Start: Jahmyr Gibbs (Tier 2.1)
   Bench: James Conner (Tier 3.3)
   Impact: +1.2 tier upgrade
   [Quick Swap →]

☐ 2. Check Waiver Wire
   Target: Tank Bigsby (Tier 2.8)
   Would replace: James Conner (Tier 3.3)
   Impact: +0.5 tier upgrade
   [View Player →]

☐ 3. Consider Trade
   Surplus: WR depth (4 top-24 WRs)
   Need: RB2 upgrade
   Target teams: TeamX, TeamY
   [View Targets →]
```

### **Action Priority Algorithm**

```go
type Action struct {
    Priority    int    // 1-5 (1 = highest)
    Category    string // "swap", "waiver", "trade", "injury", "lineup"
    Title       string // "Swap Starter"
    Description string // "Start Jahmyr Gibbs over James Conner"
    Impact      string // "+1.2 tier upgrade"
    Link        string // "/tiers?user=X&league=Y#player-123"
    Completed   bool   // User checked it off
    WeekID      string // "2026-W14" for persistence
}

func buildWeeklyActions(league LeagueData, userPrefs UserPreferences) []Action {
    actions := []Action{}

    // 1. Lineup optimization (in-season only)
    if isInSeason() {
        // Check if starters can be improved by bench
        for starter in league.Starters {
            for bench in league.Bench {
                if bench.Tier - starter.Tier >= 1.0 { // Q5: 1 tier threshold
                    actions = append(actions, Action{
                        Priority: 1,
                        Category: "swap",
                        Title: "Swap Starter",
                        Description: fmt.Sprintf("Start %s over %s", bench.Name, starter.Name),
                        Impact: fmt.Sprintf("+%.1f tier upgrade", bench.Tier - starter.Tier),
                    })
                }
            }
        }
    }

    // 2. Waiver wire opportunities
    for fa in league.TopFreeAgentsByValue {
        for starter in league.Starters {
            if fa.Tier - starter.Tier >= 1.0 { // Q5: 1 tier threshold
                actions = append(actions, Action{
                    Priority: 2,
                    Category: "waiver",
                    Title: "Check Waiver Wire",
                    Description: fmt.Sprintf("%s available (would start over %s)", fa.Name, starter.Name),
                    Impact: fmt.Sprintf("+%.1f tier upgrade", fa.Tier - starter.Tier),
                })
            }
        }
    }

    // 3. Injury alerts (starters only)
    for starter in league.Starters {
        if starter.InjuryStatus in ["Out", "Doubtful", "IR"] {
            actions = append(actions, Action{
                Priority: 1,
                Category: "injury",
                Title: "Injury Alert",
                Description: fmt.Sprintf("%s is %s - adjust lineup", starter.Name, starter.InjuryStatus),
                Impact: "Avoid 0 points",
            })
        }
    }

    // 4. Trade opportunities (dynasty only)
    if league.IsDynasty && len(league.TradeTargets) > 0 {
        // Find positional imbalances
        posBreakdown := calculatePositionalKTC(league.FullRoster)
        if hasImbalance(posBreakdown) {
            actions = append(actions, Action{
                Priority: 3,
                Category: "trade",
                Title: "Consider Trade",
                Description: fmt.Sprintf("Surplus: %s depth, Need: %s upgrade", surplusPos, deficitPos),
                Impact: fmt.Sprintf("%d targets available", len(league.TradeTargets)),
            })
        }
    }

    // Sort by priority, limit to top 5
    sort.Slice(actions, func(i, j int) bool {
        return actions[i].Priority < actions[j].Priority
    })

    if len(actions) > 5 {
        actions = actions[:5]
    }

    return actions
}
```

### **Data Requirements**

**New Types** (`types.go`):
```go
type Action struct {
    Priority    int
    Category    string
    Title       string
    Description string
    Impact      string
    Link        string
    Completed   bool
    WeekID      string
}

// Add to LeagueData
type LeagueData struct {
    // ... existing fields
    WeeklyActions []Action
}
```

**Persistence** (Q6: if architecture allows):
- Store completed actions in browser localStorage (client-side)
- Key: `actions_{username}_{leagueID}_{weekID}`
- Value: JSON array of completed action titles
- Clear on new week (server sends current week ID)

**Alternative (simpler)**: No persistence, always show current state

### **Implementation Steps**

1. **New file** (`actions.go`):
   ```go
   func buildWeeklyActions(league LeagueData) []Action
   func hasImbalance(pos PositionalKTC) bool
   func getCurrentWeekID() string // "2026-W14"
   ```

2. **Update handler** (`handlers.go`):
   ```go
   // In tiersHandler, after building LeagueData:
   leagueData.WeeklyActions = buildWeeklyActions(leagueData)
   ```

3. **Update template** (`templates/tiers.html`):
   - Add compact actions section at top (above roster)
   - Add detailed actions card to dynasty toolkit sidebar
   - Add JavaScript for checkbox persistence (localStorage)

4. **Tuning knobs** (for future iteration):
   ```go
   const (
       TIER_THRESHOLD_SWAP   = 1.0 // Q5: currently 1 tier
       TIER_THRESHOLD_WAIVER = 1.0
       MAX_ACTIONS           = 5
   )
   ```

### **Edge Cases**
- No actions → show "✅ Looking good! No urgent actions this week"
- Offseason → focus on trade/draft actions, hide lineup swaps
- Redraft league → hide dynasty-specific actions
- Multiple actions for same player → deduplicate (highest priority wins)

### **Testing Checklist**
- [ ] Actions generated for league with obvious swaps
- [ ] Tier threshold works (1.0 tier gap triggers action)
- [ ] Priority sorting works (injuries > swaps > waivers > trades)
- [ ] Max 5 actions enforced
- [ ] Checkboxes toggle (if implemented)
- [ ] localStorage persistence works (if implemented)
- [ ] No actions case shows positive message
- [ ] Mobile responsive display

---

## Feature #3: Trade Fairness + "Fleeced" Flag

### **User Story**
"Show me if recent trades were lopsided - flag extreme value gaps while recognizing that contending vs rebuilding strategies are valid."

### **UX Design**

**Updated trade display** (in Recent Transactions section):

**Before**:
```
Trade: TeamA ⇄ TeamB
TeamA gave: Player X (2500 KTC)
TeamB gave: Player Y (2000 KTC), 2026 1st (800 KTC)
```

**After**:
```
Trade: TeamA ⇄ TeamB  [TeamB +12%] 🟢
TeamA gave: Player X (2500 KTC)
TeamB gave: Player Y (2000 KTC), 2026 1st (800 KTC)
Context: TeamB rebuilding, acquired future value
```

**Fleeced example**:
```
Trade: TeamA ⇄ TeamB  [TeamA +28%] 🔴 FLEECED
TeamA gave: Bench RB (500 KTC), 2028 3rd (50 KTC)
TeamB gave: Elite WR (3000 KTC)
Context: Extreme value gap - verify trade validity
```

### **Fleeced Detection Logic**

```go
type TradeFairness struct {
    Winner           string      // "TeamA" or "TeamB" or "Fair"
    ValueDelta       int         // Absolute KTC difference
    ValueDeltaPct    float64     // % of smaller team's roster value
    Fleeced          bool        // Extreme gap flag
    Context          string      // "Competing strategy", "Rebuilding", "Extreme value gap"
    WinnerTeam       string      // Team that got better value
    DisplayBadge     string      // "🟢 +12%", "🔴 FLEECED", "→ Fair"
}

func calculateTradeFairness(trade Transaction, rosters map[int][]PlayerRow) TradeFairness {
    // Calculate total value each team gave
    teamAGave := sumKTCValues(trade.TeamAAssets)
    teamBGave := sumKTCValues(trade.TeamBAssets)

    delta := abs(teamAGave - teamBGave)

    // Get each team's total roster value
    teamARosterValue := sumKTCValues(rosters[trade.TeamARosterID])
    teamBRosterValue := sumKTCValues(rosters[trade.TeamBRosterID])

    // Calculate delta as % of SMALLER team's roster (more generous)
    smallerRoster := min(teamARosterValue, teamBRosterValue)
    deltaPct := float64(delta) / float64(smallerRoster) * 100

    // Determine winner
    winner := "Fair"
    winnerTeam := ""
    if teamAGave > teamBGave {
        winner = "TeamB"
        winnerTeam = trade.TeamBName
    } else if teamBGave > teamAGave {
        winner = "TeamA"
        winnerTeam = trade.TeamAName
    }

    // Fleeced threshold: ~15% (Q8)
    fleeced := deltaPct >= 15.0

    // Context detection (Q7: hybrid approach)
    context := ""
    if deltaPct < 5.0 {
        context = "Fair trade"
    } else if deltaPct < 15.0 {
        // Check if it's a strategic trade
        if isRebuildingStrategy(trade, rosters) {
            context = fmt.Sprintf("%s rebuilding - acquiring future value", winnerTeam)
        } else if isCompetingStrategy(trade, rosters) {
            context = fmt.Sprintf("%s competing - acquired win-now pieces", winnerTeam)
        } else {
            context = "Moderate value gap"
        }
    } else {
        context = "Extreme value gap - verify trade validity"
    }

    // Display badge (Q9: subtle)
    badge := ""
    if fleeced {
        badge = fmt.Sprintf("🔴 %s +%.0f%% FLEECED", winnerTeam, deltaPct)
    } else if deltaPct >= 5.0 {
        badge = fmt.Sprintf("🟢 %s +%.0f%%", winnerTeam, deltaPct)
    } else {
        badge = "→ Fair trade"
    }

    return TradeFairness{
        Winner:        winner,
        ValueDelta:    delta,
        ValueDeltaPct: deltaPct,
        Fleeced:       fleeced,
        Context:       context,
        WinnerTeam:    winnerTeam,
        DisplayBadge:  badge,
    }
}

// Helper: detect rebuilding strategy
func isRebuildingStrategy(trade Transaction, rosters map[int][]PlayerRow) bool {
    // Winner received mostly picks + young players (age < 24)
    // Winner gave away older players (age > 26)
    // OR winner's roster avg age decreased significantly
    // TODO: implement heuristics
    return false
}

// Helper: detect competing strategy
func isCompetingStrategy(trade Transaction, rosters map[int][]PlayerRow) bool {
    // Winner received proven starters (top-24 positional players)
    // Winner gave away picks + young players
    // OR winner's roster avg age increased
    // TODO: implement heuristics
    return false
}
```

### **Data Requirements**

**Update Types** (`types.go`):
```go
type Transaction struct {
    // ... existing fields
    Fairness TradeFairness  // NEW
}

type TradeFairness struct {
    Winner        string
    ValueDelta    int
    ValueDeltaPct float64
    Fleeced       bool
    Context       string
    WinnerTeam    string
    DisplayBadge  string
}
```

### **Implementation Steps**

1. **Update** (`dynasty.go` or new `trade_fairness.go`):
   ```go
   func calculateTradeFairness(trade Transaction, rosters map[int][]PlayerRow) TradeFairness
   func isRebuildingStrategy(trade Transaction, rosters map[int][]PlayerRow) bool
   func isCompetingStrategy(trade Transaction, rosters map[int][]PlayerRow) bool
   ```

2. **Update** (`handlers.go`):
   ```go
   // In tiersHandler, when building transactions:
   for i := range recentTransactions {
       recentTransactions[i].Fairness = calculateTradeFairness(recentTransactions[i], allRosters)
   }
   ```

3. **Update template** (`templates/tiers.html`):
   - Add fairness badge to each trade
   - Add context text below trade details
   - Style: subtle green/red badges (Q9)

### **Iteration Plan** (Q8: "1-ish but maybe need to iterate")

Start with **15% threshold**, then tune based on feedback:
- If too many false positives → increase to 18-20%
- If missing obvious fleeces → decrease to 12-13%
- Consider league-specific thresholds (competitive vs casual)

### **Edge Cases**
- No dynasty values → skip fairness calculation
- Pick-only trades → use pick values from KTC
- 3-team trades → calculate pairwise fairness (A vs B, B vs C, A vs C)
- Devy/taxi players → include if dynasty values available

### **Testing Checklist**
- [ ] Fair trade (5% delta) shows "Fair trade"
- [ ] Moderate gap (10% delta) shows winner badge
- [ ] Fleeced trade (20% delta) shows FLEECED flag
- [ ] Context detection works (rebuild vs compete)
- [ ] Badge styling is subtle (not distracting)
- [ ] Mobile responsive display
- [ ] No dynasty values → skip fairness calc

---

## Feature #4: News Signal Compression

### **User Story**
"Show me the top 3 most critical news items for MY players - during the season focus on this week, during offseason show the whole offseason summary."

### **UX Design**

**Location**: Top of News Feed card (replaces current full feed)

**In-Season View** (Week 1 - Week 18):
```
📰 What Changed This Week (Your Players)

1. 🔴 Breece Hall (OUT) - Knee injury, 2-week absence expected
   Impact: Start Isaiah Davis as RB2

2. 📈 Jayden Reed - WR1 upside this week (Doubs out)
   Impact: Flex over DJ Moore

3. 💬 Travis Etienne - Split backfield concerns vs Jaguars
   Impact: Monitor snap count
```

**Offseason View** (Post-playoffs to Pre-draft):
```
📰 Offseason Summary (Last 3 Months) - Your Players

1. 🏈 Justin Jefferson - Contract extension, locked in through 2028
   Impact: Dynasty value stable

2. 📈 Bijan Robinson - Added 15 lbs muscle, RB1 buzz
   Impact: Top-5 dynasty RB

3. 📉 Kyle Pitts - New OC rumors, target share concerns
   Impact: Monitor training camp reports
```

### **News Filtering Algorithm**

```go
type CompressedNews struct {
    TimeWindow    string        // "This Week" or "Last 3 Months"
    TopHeadlines  []PlayerNews  // Max 3 items
    TotalItems    int           // Total news items for user's players
}

func compressPlayerNews(allNews []PlayerNews, userPlayers []string, isDynasty bool) CompressedNews {
    // 1. Filter: Only news for user's players (Q11)
    userNews := []PlayerNews{}
    for _, news := range allNews {
        if contains(userPlayers, news.PlayerName) {
            userNews = append(userNews, news)
        }
    }

    // 2. Time window: Dynamic based on season state (Q10)
    timeWindow := "This Week"
    daysBack := 7

    if isOffseason() {
        timeWindow = "Last 3 Months"
        daysBack = 90
    }

    // Filter by time window
    recentNews := []PlayerNews{}
    cutoff := time.Now().AddDate(0, 0, -daysBack)
    for _, news := range userNews {
        if news.Timestamp.After(cutoff) {
            recentNews = append(recentNews, news)
        }
    }

    // 3. Score each news item by importance
    for i := range recentNews {
        recentNews[i].ImportanceScore = calculateImportanceScore(recentNews[i])
    }

    // 4. Sort by importance, take top 3 (Q12)
    sort.Slice(recentNews, func(i, j int) bool {
        return recentNews[i].ImportanceScore > recentNews[j].ImportanceScore
    })

    topHeadlines := recentNews
    if len(topHeadlines) > 3 {
        topHeadlines = topHeadlines[:3]
    }

    return CompressedNews{
        TimeWindow:   timeWindow,
        TopHeadlines: topHeadlines,
        TotalItems:   len(recentNews),
    }
}

// Importance scoring heuristic
func calculateImportanceScore(news PlayerNews) int {
    score := 0

    // Injury status = highest priority
    if news.InjuryStatus == "Out" || news.InjuryStatus == "IR" {
        score += 100
    } else if news.InjuryStatus == "Doubtful" {
        score += 80
    } else if news.InjuryStatus == "Questionable" {
        score += 50
    }

    // Dynasty value change (if available)
    if news.ValueChange > 500 {
        score += 70  // Big riser
    } else if news.ValueChange < -500 {
        score += 60  // Big faller
    }

    // Starter vs bench (user's roster context)
    if news.IsStarter {
        score += 40
    }

    // Recency (newer = higher score)
    hoursSince := time.Since(news.Timestamp).Hours()
    if hoursSince < 24 {
        score += 30
    } else if hoursSince < 72 {
        score += 15
    }

    // Keywords in news text
    keywords := []string{
        "injury", "out", "IR", "doubtful",
        "trade", "traded", "acquired",
        "suspension", "suspended",
        "promoted", "starter", "RB1", "WR1",
    }
    for _, kw := range keywords {
        if strings.Contains(strings.ToLower(news.Description), kw) {
            score += 10
        }
    }

    return score
}

// Helper: check if offseason
func isOffseason() bool {
    now := time.Now()
    month := now.Month()

    // Offseason: February - August
    // In-season: September - January
    return month >= time.February && month <= time.August
}
```

### **Data Requirements**

**Update Types** (`types.go`):
```go
type PlayerNews struct {
    // ... existing fields
    ImportanceScore int        // NEW: 0-200 range
    ValueChange     int        // NEW: KTC delta (if tracked)
    IsStarter       bool       // NEW: user's starter vs bench
}

type CompressedNews struct {
    TimeWindow   string
    TopHeadlines []PlayerNews
    TotalItems   int
}

// Add to LeagueData
type LeagueData struct {
    // ... existing fields
    CompressedNews CompressedNews  // Replace PlayerNewsFeed in UI
}
```

### **Implementation Steps**

1. **Update** (`dynasty.go`):
   ```go
   func compressPlayerNews(allNews []PlayerNews, userPlayers []string, isDynasty bool) CompressedNews
   func calculateImportanceScore(news PlayerNews) int
   func isOffseason() bool
   ```

2. **Update** (`handlers.go`):
   ```go
   // In tiersHandler, after aggregating player news:
   userPlayerIDs := extractPlayerIDs(startersRows, benchRows)
   leagueData.CompressedNews = compressPlayerNews(playerNewsFeed, userPlayerIDs, isDynasty)
   ```

3. **Update template** (`templates/tiers.html`):
   - Replace full news feed with compressed view
   - Show "Top 3 Headlines" section
   - Add "View All News (X items) →" link to expand full feed
   - Highlight importance with icons (🔴 injury, 📈 riser, 💬 news)

### **Edge Cases**
- No news for user's players → show "No news for your players this week"
- Less than 3 items → show all available items
- Offseason + no news → show "Quiet offseason - check back during training camp"
- Non-dynasty league → focus on injury/availability news only

### **Testing Checklist**
- [ ] In-season mode shows "This Week" (7 days)
- [ ] Offseason mode shows "Last 3 Months" (90 days)
- [ ] Only user's players included (not league-wide)
- [ ] Top 3 most important items shown
- [ ] Injury status prioritized (Out > Doubtful > Questionable)
- [ ] Importance scoring works (manual verification)
- [ ] "View All News" link works
- [ ] No news case shows helpful message

---

## Implementation Order

✅ **Execute in sequence: #1 → #2 → #3 → #4**

**Estimated Timeline**:
- Feature #1 (Cross-League Dashboard): 8-12 hours
- Feature #2 (Weekly Action List): 6-8 hours
- Feature #3 (Trade Fairness): 4-6 hours
- Feature #4 (News Compression): 3-4 hours

**Total Phase 1**: ~21-30 hours

---

## Next Steps

1. Review this spec doc
2. Approve approach for Feature #1
3. Begin implementation of Feature #1
4. Test, iterate, commit
5. Move to Feature #2
6. Repeat until Phase 1 complete

After Phase 1 complete → Plan Phase 2 features with user input.
