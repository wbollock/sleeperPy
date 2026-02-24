# Roadmap: Free + Premium + Cross-Platform

## Goals
- Make the free tier extremely useful week-to-week.
- Build clear premium value without blocking core workflows.
- De-risk platform dependency (cross-platform last).

## Phase 1 — Free Core Utility (MVP)
**Focus:** Daily usefulness + trade clarity

**Deliverables (clear definitions):**
- **Cross-League Dashboard:** Aggregated cards for every league showing roster value, avg age, injuries, picks, and top action items.
- **Weekly Action List:** Per-league checklist that is deterministic (non-LLM) so users get consistent advice.
- **Trade Fairness + Fleeced flag:** Clear value delta + “fleeced” label when thresholds are exceeded.
- **News Digest:** A compressed list of high-impact updates for quick scanning.

**Success Criteria:**
- No UI truncation or overflow.
- Results appear in <3s on typical leagues.
- Users can identify 3+ actions in under 60 seconds.

## Ordered Feature List (All Phases)
1. Cross‑League Dashboard
2. Weekly Action List
3. Trade Fairness Snapshot + “Fleeced” Flag
4. News Signal Compression
5. Trade Retrospective Analyzer
6. League Context Cards
7. Value Change Tracker
8. LLM Strategy Studio
9. Trade Negotiation Coach
10. Advanced Waiver Model
11. Season Planner
12. Rookie Draft Needs (Premium Dynasty)
13. Feature gating + usage limits
14. Account linking (multiple Sleeper usernames)
15. Provider Interface
16. ESPN/Yahoo read‑only imports
17. Weekly email summary (low priority)
18. Public status page (sanitized operational health)

1) **Cross-League Dashboard (Free)**
- **User Story:** “Show me how my teams are doing without clicking each league.”
- **UX:** New top section in `templates/tiers.html` with cards per league.
- **Data Inputs:** `LeagueData` values already computed:
  - `TotalRosterValue`, `UserAvgAge`, `DraftPicks` count, `TradeTargets` count, `PlayerNewsFeed` count.
- **New Types:** `type LeagueSummary struct { LeagueName, Scoring string; LeagueSize int; TotalRosterValue int; AvgAge float64; DraftPickCount int; TradeTargetCount int; Alerts []string }`
- **Implementation Steps:**
  1. In `handlers.go`, after building `leagueResults`, map each to `LeagueSummary`.
  2. Compute `Alerts` with small heuristics (e.g., injury spikes, no depth at RB).
  3. Add `TiersPage.LeagueSummaries []LeagueSummary`.
  4. Render summary cards above league selector.
- **Edge Cases:** No dynasty values → skip value fields. No matchups → omit win prob.

2) **Weekly Action List (Free)**
- **User Story:** “Tell me what to do first.”
- **UX:** Checklist card in dynasty toolkit; also for redraft below roster table.
- **Data Inputs:** `Starters`, `Bench`, `TopFreeAgentsByValue`, injury flags (if available).
- **Algorithm (deterministic):**
  - If `IsTierWorseThanBench`, add “Swap in X for Y”.
  - If top FA improves a starter by >=2 tiers, add waiver action.
  - If bench has elite tier but is not starting, add “Consider starting”.
  - Cap list at 5 items.
- **Implementation Steps:**
  1. Create `actions.go` with `buildWeeklyActions(league LeagueData) []string`.
  2. Populate `LeagueData.WeeklyActions`.
  3. Render in `templates/tiers.html`.

3) **Trade Fairness Snapshot + “Fleeced” Flag (Free)**
- **User Story:** “Was this trade lopsided?”
- **UX:** On each trade, show `NetValue` and “Fleeced” badge if applicable.
- **Logic:** 
  - Compute dynasty value delta: `Team1GaveValue - Team2GaveValue`.
  - Compute starter impact: number of top-24 positional players swapped.
  - Fleeced if value delta > 20% of receiving team’s roster value OR tier delta >= 3 on 2+ starters.
- **Implementation Steps:**
  1. Extend `Transaction` with `NetValue`, `Fleeced`, `FleecedSide`, `ValueDeltaPct`.
  2. Update `fetchRecentTransactions` to compute per-team totals.
  3. Add display in `templates/tiers.html`.
  4. Add tests with mock trades.

4) **News Signal Compression (Free)**
- **User Story:** “Tell me what matters without reading every blurb.”
- **UX:** A “What changed today” list at the top of the news card.
- **Logic:** Group by player, pick most recent high-impact items.
- **Implementation Steps:**
  1. Add `LeagueData.NewsDigest []PlayerNews`.
  2. In `aggregatePlayerNews`, create digest from `PlayerNewsFeed`.
  3. Add small tags for injury status.

## Phase 2 — Free Depth + Retrospective
**Focus:** Long-term evaluation + context

**Deliverables:**
- Retrospective trade analysis with winners/losers over time.
- League context cards for strategy alignment.
- Value change tracking for buy/sell signals.

5) **Trade Retrospective Analyzer (Free)**
- **User Story:** “Who actually won that trade?”
- **UX:** Card listing prior trades and winner/loser over time.
- **Logic:** Compare value at trade time vs current value.
- **Implementation Steps:**
  1. Create `TradeSnapshot` struct with `Timestamp`, `Assets`, `ValuesAtTrade`.
  2. Cache snapshots in a local file or simple JSON cache keyed by league ID.
  3. On each lookup, update with current values and compute winner.
  4. Render summary in `templates/tiers.html`.

6) **League Context Cards (Free)**
- **User Story:** “What’s my team direction?”
- **UX:** Small cards (Age rank, Value rank, Pick count, Scarcity).
- **Implementation Steps:**
  1. Add `ContextCard` struct (title, value, trend).
  2. Build list in `handlers.go`.
  3. Render in the dynasty toolkit.

7) **Value Change Tracker (Free-lite)**
- **User Story:** “Who’s rising or falling?”
- **UX:** List of top risers/fallers.
- **Implementation Steps:**
  1. Cache DynastyProcess values snapshot daily.
  2. Compute deltas and store in memory.
  3. Render top 5 risers/fallers.

## Phase 3 — Premium SaaS Layer
**Focus:** Insights + negotiation help

**Deliverables:**
- Premium LLM strategy outputs
- Trade negotiation automation
- Waiver + season planning models

8) **LLM Strategy Studio (Premium)**
- **Status:** Implemented.
- **Next Enhancements:**
  - Add “priority matrix” (needs/surpluses).
  - Add risk watchlist for injury/age.

9) **Trade Negotiation Coach (Premium)**
- **User Story:** “Give me a trade offer and message.”
- **UX:** Button on trade targets card.
- **Implementation Steps:**
  1. Build LLM prompt using surpluses/deficits.
  2. Add template fallback if LLM disabled.
  3. Render draft trade proposal + message.

10) **Advanced Waiver Model (Premium)**
- **User Story:** “Who should I bid on and how much?”
- **Logic:** Score by tier delta + scarcity + playoff schedule.
- **Implementation Steps:**
  1. Extend FA data with usage/role.
  2. Add FAAB estimator.
  3. Render recommendations.

11) **Season Planner (Premium)**
- **User Story:** “When should I trade vs hold?”
- **UX:** Roadmap card with calendar view.
- **Implementation Steps:**
  1. Build schedule difficulty profile.
  2. Map to action recommendations.
  3. Render timeline.

12) **Rookie Draft Needs (Premium - Dynasty)**
- **User Story:** “What should I target in the rookie draft?”
- **UX:** Card listing priority positions, suggested archetypes.
- **Logic:** Age curve + depth + positional scarcity + pick inventory.
- **Output:** Ranked positions + pick strategy guidance.

## Phase 4 — Monetization & Ops
**Deliverables:**
- Feature gating with usage limits.
- Account linking for multiple Sleeper usernames.
- Public status page with sanitized operational metrics.

18) **Public Status Page (Ops/Public)**
- **User Story:** “Let me quickly see if SleeperPy is healthy without exposing admin internals.”
- **UX:** Public `/status` page with uptime, aggregate usage counters, and coarse health status.
- **Security/Safety:** No user agents, no path-level data, no raw errors, no admin auth details.
- **Implementation Steps:**
  1. Add a dedicated public status handler and template.
  2. Reuse aggregate counters (uptime, lookups, leagues, errors) only.
  3. Compute a coarse health state from aggregate error rate.
  4. Keep `/admin` and `/admin/api` private and unchanged.

## Phase 5 — Cross-Platform (Last)
**Priority:** mitigate API risk

15) **Provider Interface**
- **Goal:** Abstract Sleeper so ESPN/Yahoo can be added without rewriting logic.
- **Implementation Steps:**
  1. Define interface `Provider` in new file `providers/provider.go`.
  2. Implement `SleeperProvider`.
  3. Refactor `handlers.go` to use provider instance.

16) **ESPN/Yahoo Read-Only Imports**
- **Goal:** read-only support for ESPN/Yahoo.
- **Implementation Steps:**
  1. Build provider adapters with mapping.
  2. Normalize data into internal structs.

## Low Priority (Nice-to-Have)
- **Weekly Email Summary**
  - Digest of all leagues with action items + top risks.
  - Triggered weekly cron; requires email provider.
