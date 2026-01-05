# Dynasty Offseason Features - Prioritized Plan

## Overview
Dynasty leagues need different tools during the offseason when there are no active matchups. This plan focuses on features that help managers make roster decisions, track player value changes, and stay informed about their players.

## Priority 1: Player News Feed ⭐ HIGHEST PRIORITY

### What It Does
Shows aggregated news for all players on your roster in a chronological feed. Helps managers stay informed during the offseason when they're not checking matchups weekly.

### Data Source Options

**Option 1: Sleeper Player News API** (RECOMMENDED - easiest)
- Endpoint: `https://api.sleeper.app/v1/players/nfl`
- Each player object has `news` field with recent news
- Also has `injury_status`, `injury_notes`
- Free, no API key required
- Already fetching this data for player lookups

**Option 2: FantasyPros API**
- Has player news endpoint
- Requires API key (may have free tier limits)
- More comprehensive news coverage

**Option 3: ESPN/Yahoo APIs**
- More complex authentication
- Not worth the effort for news alone

### Implementation

**Backend** (`main.go`):
```go
type PlayerNews struct {
    PlayerName   string
    Position     string
    NewsText     string
    Source       string
    Timestamp    time.Time
    InjuryStatus string // "Out", "Questionable", "Doubtful", etc.
    IsStarter    bool   // True if in starting lineup
    DynastyValue int    // To show value context
}

type LeagueData struct {
    // ... existing fields
    PlayerNewsFeed []PlayerNews // All news for user's roster players
}

func aggregatePlayerNews(rosterPlayerIDs []string, players map[string]interface{}, startersIDs []string) []PlayerNews {
    newsFeed := []PlayerNews{}

    for _, pid := range rosterPlayerIDs {
        if p, ok := players[pid].(map[string]interface{}); ok {
            name := getPlayerName(p)
            pos, _ := p["position"].(string)

            // Check if player has news
            if newsObj, ok := p["news"].(map[string]interface{}); ok {
                newsText, _ := newsObj["text"].(string)
                source, _ := newsObj["source"].(string)

                // Parse timestamp if available
                var timestamp time.Time
                if ts, ok := newsObj["timestamp"].(float64); ok {
                    timestamp = time.Unix(int64(ts), 0)
                }

                // Get injury status
                injuryStatus, _ := p["injury_status"].(string)

                // Check if starter
                isStarter := false
                for _, sid := range startersIDs {
                    if sid == pid {
                        isStarter = true
                        break
                    }
                }

                if newsText != "" {
                    newsFeed = append(newsFeed, PlayerNews{
                        PlayerName:   name,
                        Position:     pos,
                        NewsText:     newsText,
                        Source:       source,
                        Timestamp:    timestamp,
                        InjuryStatus: injuryStatus,
                        IsStarter:    isStarter,
                    })
                }
            }
        }
    }

    // Sort by timestamp (newest first)
    sort.Slice(newsFeed, func(i, j int) bool {
        return newsFeed[i].Timestamp.After(newsFeed[j].Timestamp)
    })

    return newsFeed
}
```

**Frontend** (`templates/tiers.html`):
```html
{{if $l.IsDynasty}}
<div class="dynasty-toolkit dynasty-only">
    <!-- Player News Feed - NEW, TOP PRIORITY -->
    <div class="toolkit-card collapseable">
        <div class="card-header" onclick="toggleSection('player-news-{{$i}}')">
            <span class="card-title">
                Player News
                <span class="news-count">{{len $l.PlayerNewsFeed}}</span>
            </span>
            <span class="collapse-icon" id="player-news-{{$i}}-icon">▼</span>
        </div>
        <div class="card-content" id="player-news-{{$i}}-content">
            {{if $l.PlayerNewsFeed}}
                <div class="news-feed">
                    {{range $l.PlayerNewsFeed}}
                    <div class="news-item {{if .InjuryStatus}}news-injury{{end}}">
                        <div class="news-header">
                            <span class="news-player">
                                {{.PlayerName}} ({{.Position}})
                                {{if .IsStarter}}<span class="starter-badge">⭐</span>{{end}}
                            </span>
                            <span class="news-time">{{formatTime .Timestamp}}</span>
                        </div>
                        {{if .InjuryStatus}}
                        <div class="news-injury-status">{{.InjuryStatus}}</div>
                        {{end}}
                        <div class="news-text">{{.NewsText}}</div>
                        {{if .Source}}
                        <div class="news-source">Source: {{.Source}}</div>
                        {{end}}
                    </div>
                    {{end}}
                </div>
            {{else}}
                <div class="empty-state">
                    <div class="empty-state-icon">📰</div>
                    <div class="empty-state-title">No Recent News</div>
                    <div class="empty-state-text">
                        All your players are flying under the radar. Check back later for updates.
                    </div>
                </div>
            {{end}}
        </div>
    </div>

    <!-- Existing cards: Draft Capital, League Age, Trade Targets -->
</div>
{{end}}
```

**CSS**:
```css
.news-feed {
    max-height: 500px;
    overflow-y: auto;
}

.news-item {
    padding: 12px;
    margin-bottom: 8px;
    background: rgba(30, 38, 54, 0.5);
    border-left: 3px solid rgba(123, 176, 255, 0.3);
    border-radius: 4px;
}

.news-item.news-injury {
    border-left-color: rgba(255, 123, 123, 0.6);
    background: rgba(255, 123, 123, 0.05);
}

.news-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 6px;
}

.news-player {
    font-weight: 600;
    color: #7bb0ff;
}

.starter-badge {
    font-size: 0.8em;
    margin-left: 4px;
}

.news-time {
    font-size: 0.85em;
    color: #9fb3d4;
}

.news-injury-status {
    display: inline-block;
    padding: 2px 8px;
    margin-bottom: 6px;
    font-size: 0.85em;
    font-weight: 600;
    background: rgba(255, 123, 123, 0.2);
    color: #ff7b7b;
    border-radius: 4px;
}

.news-text {
    color: #d4dde8;
    line-height: 1.5;
    margin-bottom: 4px;
}

.news-source {
    font-size: 0.8em;
    color: #9fb3d4;
    font-style: italic;
}

.news-count {
    display: inline-block;
    padding: 2px 8px;
    margin-left: 8px;
    background: rgba(123, 176, 255, 0.2);
    border-radius: 10px;
    font-size: 0.8em;
    font-weight: 600;
}
```

### Enhancements
- Filter by news type (injury, trade, depth chart)
- Filter by position
- Filter by starters only
- Mark news as "read" and persist state
- Push notifications for critical news (future)

### Time Estimate
**3-4 hours** (including styling and testing)

---

## Priority 2: Recent League Transactions

### What It Does
Shows recent trades, adds/drops, and waiver claims in the league. Helps managers track what other teams are doing during the offseason.

### Data Source
- Sleeper API: `/v1/league/{league_id}/transactions/{round}`
- Returns all transactions for a given week/round
- Need to aggregate last N weeks for offseason view

### Implementation

**Backend**:
```go
type Transaction struct {
    Type        string    // "trade", "waiver", "free_agent"
    Timestamp   time.Time
    Description string    // "Team A traded Player X to Team B for Player Y"
    TeamNames   []string
    PlayerNames []string
}

func fetchRecentTransactions(leagueID string, numWeeks int) []Transaction {
    // Fetch transactions for last N weeks
    // Parse and format into readable descriptions
    // Sort by timestamp (newest first)
}
```

**Frontend**:
- Similar card layout to Player News
- Show trades, adds, drops with team names
- Color code by transaction type

### Time Estimate
**2-3 hours**

---

## Priority 3: Breakout Candidates

### What It Does
Highlights young players on the user's bench who have high upside potential. Uses age + dynasty value to identify buy-low opportunities.

### Algorithm
```go
func findBreakoutCandidates(benchRows []PlayerRow) []PlayerRow {
    candidates := []PlayerRow{}

    for _, row := range benchRows {
        // Criteria:
        // 1. Age < 25 (young)
        // 2. Dynasty value > 500 (has some value)
        // 3. Currently on bench (not starting)
        // 4. Position is RB/WR/TE (skill positions)

        if row.Age > 0 && row.Age < 25 &&
           row.DynastyValue > 500 &&
           (row.Pos == "RB" || row.Pos == "WR" || row.Pos == "TE") {
            candidates = append(candidates, row)
        }
    }

    // Sort by dynasty value (highest upside first)
    sort.Slice(candidates, func(i, j int) bool {
        return candidates[i].DynastyValue > candidates[j].DynastyValue
    })

    return candidates
}
```

### Display
- Section in dynasty toolkit
- Show player name, age, dynasty value, position
- Highlight why they're a breakout candidate (age, value trend)

### Time Estimate
**1-2 hours** (simple algorithm, mostly frontend)

---

## Priority 4: Aging Players Alert

### What It Does
Flags players on the roster who are approaching the end of their fantasy relevance. Suggests selling before value crashes.

### Algorithm
```go
func findAgingPlayers(startersRows, benchRows []PlayerRow) []PlayerRow {
    aging := []PlayerRow{}

    allPlayers := append(startersRows, benchRows...)

    for _, row := range allPlayers {
        isAging := false
        reason := ""

        // Position-specific age thresholds
        switch row.Pos {
        case "RB":
            if row.Age >= 28 {
                isAging = true
                reason = "RBs decline sharply after age 28"
            }
        case "WR", "TE":
            if row.Age >= 30 {
                isAging = true
                reason = "Pass catchers typically decline after 30"
            }
        case "QB":
            if row.Age >= 35 {
                isAging = true
                reason = "QBs can play longer but decline risk increases"
            }
        }

        if isAging && row.DynastyValue > 1000 {
            // Only flag if they still have trade value
            aging = append(aging, row)
        }
    }

    return aging
}
```

### Display
- Warning section in dynasty toolkit
- Show player name, age, current value, suggested action
- Sort by urgency (oldest first)

### Time Estimate
**1-2 hours**

---

## Priority 5: Dynasty Trade Analyzer

### What It Does
Let users input potential trades and see the dynasty value comparison. Shows if the trade is fair or if one side is winning.

### Implementation

**Backend**:
```go
type TradeAnalysis struct {
    YourPlayers    []PlayerRow
    TheirPlayers   []PlayerRow
    YourTotalValue int
    TheirTotalValue int
    Difference     int // Positive = you win, negative = they win
    Verdict        string // "Fair", "You Win", "They Win"
}

func analyzeProposedTrade(yourPlayerIDs, theirPlayerIDs []string, dynastyValues map[string]DynastyValue, isSuperFlex bool) TradeAnalysis {
    // Calculate total value for each side
    // Determine winner based on value difference
    // Consider picks if included
}
```

**Frontend**:
- Interactive UI with player selection
- Real-time value calculation
- Visual comparison (bar chart showing value difference)
- "Add Pick" button to include draft picks in trade

### Complexity
This is more complex - requires interactive JavaScript, player search/autocomplete, etc.

### Time Estimate
**4-5 hours** (interactive UI is more work)

---

## Priority 6: Rookie Draft Rankings

### What It Does
Shows top rookie prospects for the upcoming draft with projected dynasty values.

### Data Source
- DynastyProcess CSV has rookie data
- Or manually curate top 20 rookies for 2025
- Update once per year (low maintenance)

### Display
- Simple table: Rank, Name, Position, College, Projected Value
- Highlight which rounds they'll likely go in startup drafts
- Compare to current roster values

### Time Estimate
**1-2 hours** (mostly data entry if manual)

---

## Priority 7: Startup Draft Simulator (Future)

### What It Does
Mock draft interface for dynasty startup drafts. Shows ADP, allows picking players, gives draft grade.

### Complexity
Very high - requires:
- ADP data
- Draft state management
- AI opponents or manual picks
- Real-time draft board updates

### Time Estimate
**10-15 hours** (major feature)

### Status
**Defer to V2** - too complex for initial offseason launch

---

## Implementation Order (Recommended)

### Phase 1: Core Offseason Features (Week 1)
1. **Player News Feed** - 3-4 hours
2. **Breakout Candidates** - 1-2 hours
3. **Aging Players Alert** - 1-2 hours

**Total: 5-8 hours**

### Phase 2: League Intelligence (Week 2)
4. **Recent League Transactions** - 2-3 hours
5. **Rookie Draft Rankings** - 1-2 hours (if manual data)

**Total: 3-5 hours**

### Phase 3: Advanced Tools (Week 3+)
6. **Dynasty Trade Analyzer** - 4-5 hours
7. **Startup Draft Simulator** - Future / V2

---

## Technical Considerations

### Caching Strategy
- **Player news**: Refresh every 15 minutes (news changes frequently)
- **Transactions**: Refresh every hour (low frequency in offseason)
- **Rookie rankings**: Refresh weekly (static data)

### API Rate Limits
- Sleeper has no documented rate limits, but be respectful
- Cache aggressively to minimize requests
- Consider adding retry logic with exponential backoff

### Performance
- Player news feed could get large (100+ players × multiple news items)
- Limit to most recent 50 news items
- Lazy load older news if needed
- Add "Load More" button for pagination

### Mobile Responsiveness
- News feed should scroll well on mobile
- Keep cards stacked vertically on small screens
- Ensure text is readable (no tiny fonts)

---

## Success Metrics

After implementation:
- ✅ Player news shows for all roster players
- ✅ News updates within 15 minutes of API changes
- ✅ Injury statuses are clearly highlighted
- ✅ Starters are distinguished from bench players
- ✅ News feed is collapseable to save space
- ✅ Breakout candidates make sense (young + value)
- ✅ Aging alerts catch sell-high candidates
- ✅ Page loads in < 3 seconds with all features

---

## Open Questions

1. **News Freshness**: How often should we poll for new player news?
   - Proposal: 15 minutes during season, 1 hour in offseason

2. **News Filtering**: Should we filter out old news (e.g., > 7 days)?
   - Proposal: Show last 7 days by default, "Show All" button for older

3. **Mobile Layout**: Should player news be in sidebar or main column on mobile?
   - Proposal: Stack below roster on mobile (same as current toolkit)

4. **Notification System**: Should we add browser notifications for breaking news?
   - Proposal: Phase 2 feature, not MVP

5. **Trade Analyzer Scope**: Include draft picks in trade value calculations?
   - Proposal: Yes, but pick values are estimates (e.g., early 1st = ~3000 value)

---

## Next Steps

1. Get Jesse's feedback on priority order
2. Start with Player News Feed (highest value, reasonable effort)
3. Test with real dynasty league to validate data sources
4. Iterate based on user feedback
5. Add Phase 2 features once Phase 1 is stable
