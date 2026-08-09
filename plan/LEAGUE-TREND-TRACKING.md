# League Trend Tracking Across Seasons

Status: not started. Needs decisions below before implementation.

## User Story

"Show me how my team and league have changed over multiple seasons, not
just this week or this year" -- roster value trajectory, record history,
whether past trades and draft picks actually worked out, how the league's
competitive balance has shifted.

## Why this is a different kind of task than everything else built so far

Every existing feature in this app is stateless: each request live-fetches
from Sleeper/Boris Chen/DynastyProcess/FantasyCalc and computes the answer
on the spot, with short in-memory TTL caches (15 min to 24h) purely to
reduce upstream API load, not to persist history. Nothing survives a
process restart. Season-over-season trend tracking is fundamentally
different -- it requires remembering what things looked like in the past,
which means either real persistent storage or leaning on an external
source that already has that history.

## Open questions (need answers before design)

- Q1: What's actually worth tracking?
  - Roster/dynasty value over time (weekly or monthly snapshots)?
  - Win/loss record and standings history across seasons?
  - Draft pick outcomes (did a traded 1st actually hit, in hindsight)?
  - Some combination -- and if so, which is highest priority to ship first?
- Q2: Storage approach:
  - Add a real database (sqlite is the natural first choice given this is
    a single-instance Go binary today -- no infra changes needed, no new
    service to run)?
  - Keep it stateless and reconstruct history from Sleeper's own API where
    possible (Sleeper does expose past-season league/roster data by
    league_id chains -- worth checking how far back and how expensive)?
  - Some hybrid: snapshot dynasty/roster values periodically (can't get
    those retroactively from Sleeper), reconstruct record/standings
    on-demand from Sleeper's existing historical endpoints?
- Q3: Snapshot cadence and retention, if we do store snapshots -- daily?
  weekly? Keep forever or roll off after N seasons?
- Q4: Does this need to work retroactively for leagues that existed before
  this feature ships (no historical snapshots exist yet), or is it okay to
  start tracking from whenever it's built and only show trends going
  forward from there?

## Recommendation (not a final decision, just a starting point)

Start narrow: sqlite for periodic dynasty/roster-value snapshots only
(the one thing that can't be reconstructed after the fact from Sleeper's
API), reusing the existing `getValueTrend`/`rosterValueTrendCache` 24h
comparison logic as the seed -- that's already *almost* this feature, just
not durable across restarts and only comparing to "last time someone
checked" rather than a real history. Record/standings history can likely
be pulled live from Sleeper's existing league-chain API (previous_league_id
links one season to the next) without needing local storage at all, so
that part might not need Q2's tradeoff at all -- worth checking before
assuming a database is required for the whole feature.
