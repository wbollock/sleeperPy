# Dynasty Mode Layout: Better Use of Box Space

Status: resolved. See "Resolution" below -- implemented without a
screenshot by reasoning about the table CSS directly rather than
waiting on the open questions below (kept for context).

## User report

"The boxes leave a ton of empty negative space in dynasty mode on the left
hand side." Reported after the free-agent/dynasty-value boxes on the right
side of the page were already tightened up (see below) -- the left-hand
side (roster/starters area) is the part still flagged as wasteful.

## What's already been fixed (for context, don't redo)

The free-agent/dynasty-value section (below the roster) used to stack
full-width single-column tables with no card styling. That's now:
- Wrapped in `.fa-grid`, a responsive 2-column grid (`static/tiers.css`,
  `.fa-grid` / `.fa-section`), 1 column under 900px.
- `.fa-section` given real card treatment (border/background/padding)
  instead of bare tables floating in whitespace.

This covers "Recommended Free Agents", "Your Lowest-Value Players vs. Best
Available", "🚕 Taxi Squad Suggestions", and "Suggested Add/Drop" in
`templates/tiers.html`.

## What's still reported as empty

The roster/starters area on the left is `.roster-card` inside
`.league-grid-container` (`static/tiers.css`), which uses a
`minmax(0, 1.6fr) minmax(360px, 1fr)` split (roster left, dynasty toolkit
right) on screens >= 1100px. The roster table itself is `width: 100%`
inside that card, so the *table* stretches to fill the column -- the
reported empty space is likely one of:

- Short/centered cell content (Tier, Dynasty Value, Bye) in a wide column
  reading as "empty" even though the table technically fills the box.
- The roster-card being much shorter than the dynasty-toolkit column next
  to it (draft picks + league age + trade targets stacked), leaving a
  visual gap below the roster table while the right column keeps going.
- Something specific to a particular dynasty sub-view (e.g. the Bench
  table, or the summary/footer rows) not covered by the general
  `.pretty-table { width: 100% }` rule.

## Open questions (need answers before implementation)

- Q1: Screenshot or exact page section -- which box specifically? "Left
  hand side" narrows it to `.roster-card`, but not which table/row inside
  it.
- Q2: Is this about horizontal empty space (a box that doesn't stretch)
  or vertical (a short box next to a much taller one)? The fix is
  different for each -- CSS width vs. a masonry/equal-height layout vs.
  moving content between the two columns.
- Q3: Does this reproduce on desktop-width windows only, or also at the
  900-1100px range where `.league-grid-container` is still single-column?

## Suggested next step

Get a screenshot of the dynasty tiers page (real league, not mock -- mock
test-mode leagues are all redraft and won't show this layout at all) and
diagnose from there rather than guessing further.

## Resolution

Went with hypothesis 1 (short/centered cell content in a too-wide column)
without a screenshot: the roster table has only 4 mostly-short columns
(Position, Player, Age, Dynasty Value) under the default
`table-layout: auto`, which spreads leftover width from `width: 100%`
roughly evenly across every column rather than proportionally to content
-- so Position/Age/Dynasty Value ballooned out with a lot of empty padding
around short centered text, most visible on the roster-card side of the
grid since it gets the wider `1.6fr` track.

Fix: `static/tiers.css` adds `.roster-table { table-layout: fixed; }`
(scoped to this one table so it doesn't affect the page's other tables),
and `templates/tiers.html` adds an explicit `<colgroup>` giving Player the
majority of the row width (~55-66%) and capping the narrow columns.

That wasn't the actual complaint -- user feedback after this landed
confirmed it's Q2 (vertical mismatch): `.dynasty-toolkit` stacks roughly
20 `toolkit-card` boxes (draft picks, trade targets, league age, etc.) in
a single flex column confined to the grid's narrow right track, while
`roster-card` is just one table. That height mismatch left a large empty
gap under the roster on the left once it ran out of content, while the
right column kept going.

Second fix: `.league-grid-container` now uses
`grid-template-columns: repeat(auto-fit, minmax(380px, 1fr))` instead of
a fixed `1.6fr 1fr` split, and `.dynasty-toolkit` is set to
`display: contents` so its `toolkit-card` children become direct grid
items -- they flow into whatever columns/rows are actually free
(including next to and below the roster card) instead of being trapped
in one tall single-file stack. `roster-card` gets `grid-column: span 2`
so it stays the widest/most prominent box.

Q3 (900-1100px behavior, where `.league-grid-container` is still a
single column) wasn't specifically re-verified, but the single-column
path is unchanged by this fix.
