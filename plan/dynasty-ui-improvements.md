# Dynasty UI Improvements Plan

## Overview
This plan addresses multiple UI/UX improvements for dynasty mode based on user feedback:
1. ✅ Default to dynasty mode for dynasty leagues
2. ✅ Fix age sorting bug
3. ⚠️ Fix draft picks ownership logic (KNOWN ISSUE - may not work correctly, needs debugging with real API data)
4. Improve horizontal layout utilization
5. Add collapseable sections
6. Implement KTC-based trade target suggestions

## Current Problems

### 1. Age Sorting Bug
**Issue**: League Age Analysis table is not sorted by average age
**Current behavior**: Teams appear in random order
**Expected behavior**: Teams sorted oldest to youngest (descending by AvgAge)
**Location**: `main.go` - line where `teamAges` slice is sorted

### 2. Draft Picks Ownership Bug
**Issue**: Draft picks show incorrect ownership (e.g., "from gdyche" when user actually traded that pick away)
**Root cause**: Backend logic is incorrectly determining which picks belong to the user
**Location**: `main.go` - draft picks fetching/processing logic around line 1200-1250

### 3. Default Mode
**Issue**: Dynasty leagues default to "In-Season" mode but should default to "Dynasty" mode
**Location**: `templates/tiers.html` - JavaScript initialization and button active states

### 4. Vertical Layout
**Issue**: All content is stacked vertically in one long column
**Desired**: Use horizontal space better, organize into separate "cards"

### 5. No Collapseable Sections
**Issue**: Page is very long, no way to collapse sections
**Desired**: Make sections collapseable to reduce scrolling

## Proposed Architecture Changes

### Layout Structure (Dynasty Mode)

Current structure:
```
┌─────────────────────────────┐
│ Roster Table (with footer)  │
│  - Starters                 │
│  - Summary rows             │
│  - Draft picks              │
│  - League age               │
│  - Bench                    │
└─────────────────────────────┘
│ Free Agents Table           │
└─────────────────────────────┘
```

Proposed structure:
```
┌────────────────────┬────────────────────┐
│  Roster & Bench    │  Dynasty Toolkit   │
│  (Card 1)          │  (Card 2)          │
│  - Starters        │  ┌──────────────┐  │
│  - Summary         │  │ Draft Capital│  │
│  - Bench           │  │ [Collapseable]│  │
│                    │  └──────────────┘  │
│                    │  ┌──────────────┐  │
│                    │  │ League Age   │  │
│                    │  │ [Collapseable]│  │
│                    │  └──────────────┘  │
│                    │  ┌──────────────┐  │
│                    │  │ Trade Targets│  │
│                    │  │ [Collapseable]│  │
│                    │  └──────────────┘  │
└────────────────────┴────────────────────┘
│  Free Agents (Full Width)               │
│  [Collapseable]                         │
└─────────────────────────────────────────┘
```

### Card System

**Card 1: Roster & Bench**
- Single table with starters and bench
- Summary rows for avg tier, opponent tier, win probability (in-season mode)
- Summary rows for total roster value and avg age (dynasty mode)
- Always visible (not collapseable since it's primary content)

**Card 2: Dynasty Toolkit (Sidebar)**
- Draft Capital section (collapseable)
- League Age Analysis section (collapseable)
- Trade Targets section (collapseable, NEW)
- Fixed width sidebar on desktop, stacks below on mobile

### Component Details

#### 1. Draft Capital Section
**Status**: Exists but buggy
**Changes needed**:
- Fix ownership logic
- Make collapseable
- Move to sidebar card

**Backend fix** (`main.go`):
```go
// Current logic around line 1200-1250
// Problem: We're showing picks that the user traded away

// Need to check:
// - If pick.owner_id == userID, it's the user's pick
// - If pick.previous_owner_id == userID, user traded it away (DON'T show)
// - If pick.owner_id == userID AND pick.previous_owner_id != userID, show "from <team>"
```

**Frontend**:
- Add collapse/expand button
- Keep existing grid layout
- Keep "from <team>" annotation for acquired picks

#### 2. League Age Analysis Section
**Status**: Exists but sorting broken
**Changes needed**:
- Fix sorting
- Make collapseable
- Move to sidebar card

**Backend fix** (`main.go`):
```go
// Find the sort function for teamAges
// Current (WRONG):
sort.Slice(teamAges, func(i, j int) bool {
    return teamAges[i].RosterValue > teamAges[j].RosterValue
})

// Should be (CORRECT):
sort.Slice(teamAges, func(i, j int) bool {
    return teamAges[i].AvgAge > teamAges[j].AvgAge
})
```

**Frontend**:
- Add collapse/expand button
- Keep existing table layout
- Move to sidebar

#### 3. Trade Targets Section (NEW)
**Status**: Doesn't exist
**Purpose**: Suggest 2-3 potential trade partners based on KTC value distribution

**Algorithm**:
1. Calculate positional KTC distribution for user's team
   - Total QB KTC value
   - Total RB KTC value
   - Total WR KTC value
   - Total TE KTC value

2. Calculate same for all other teams in league

3. Find complementary teams:
   - If user has high WR% and low RB%, find teams with high RB% and low WR%
   - If user has high RB% and low WR%, find teams with high WR% and low RB%
   - Same for QB/TE combinations

4. Rank potential trade partners by "complementarity score"

5. Show top 2-3 partners with explanation

**Backend** (`main.go`):
```go
type TradeTarget struct {
    TeamName        string
    Reason          string // "Has RBs, needs WRs"
    YourSurplus     string // "WR"
    TheirSurplus    string // "RB"
    YourSurplusKTC  int    // Total KTC value you have in surplus position
    TheirSurplusKTC int    // Total KTC value they have in surplus position
}

type PositionalKTC struct {
    QB int
    RB int
    WR int
    TE int
}

func calculatePositionalKTC(players []PlayerRow) PositionalKTC {
    // Sum KTC values by position
}

func findTradeTargets(userRoster []PlayerRow, allTeams map[string][]PlayerRow, userID string) []TradeTarget {
    userKTC := calculatePositionalKTC(userRoster)

    // Calculate percentages
    userTotal := userKTC.QB + userKTC.RB + userKTC.WR + userKTC.TE
    userQBPct := float64(userKTC.QB) / float64(userTotal)
    userRBPct := float64(userKTC.RB) / float64(userTotal)
    // etc...

    // Determine user's surplus and deficit positions
    // (position with >35% is surplus, <15% is deficit)

    targets := []TradeTarget{}

    for teamID, roster := range allTeams {
        if teamID == userID {
            continue
        }

        teamKTC := calculatePositionalKTC(roster)
        // Calculate their percentages

        // Check for complementarity
        // If user has WR surplus and RB deficit,
        // and team has RB surplus and WR deficit,
        // they're a good trade target

        // Calculate complementarity score
        // Add to targets if score is high
    }

    // Sort by score, return top 3
    return targets[:min(3, len(targets))]
}

// Add to LeagueData struct:
type LeagueData struct {
    // ... existing fields
    TradeTargets []TradeTarget
}
```

**Frontend** (`templates/tiers.html`):
```html
<div class="toolkit-card collapseable">
    <div class="card-header" onclick="toggleSection('trade-targets-{{$i}}')">
        <span class="card-title">Trade Targets</span>
        <span class="collapse-icon" id="trade-targets-{{$i}}-icon">▼</span>
    </div>
    <div class="card-content" id="trade-targets-{{$i}}-content">
        {{range $l.TradeTargets}}
        <div class="trade-target-item">
            <div class="target-team">{{.TeamName}}</div>
            <div class="target-reason">{{.Reason}}</div>
            <div class="target-details">
                <span class="your-surplus">You have: {{.YourSurplus}} ({{.YourSurplusKTC}} value)</span>
                <span class="their-surplus">They have: {{.TheirSurplus}} ({{.TheirSurplusKTC}} value)</span>
            </div>
        </div>
        {{end}}
    </div>
</div>
```

#### 4. Default Mode
**Changes needed** (`templates/tiers.html`):

Current:
```html
<button class="mode-btn active" onclick="switchMode({{$i}}, 'inseason')">In-Season</button>
<button class="mode-btn" onclick="switchMode({{$i}}, 'dynasty')">Dynasty</button>

<!-- Content starts with in-season visible -->
<span class="inseason-desc">Weekly matchups & tier-based recommendations</span>
<span class="dynasty-desc" style="display:none;">Long-term value & dynasty toolkit</span>
```

Proposed:
```html
<button class="mode-btn" onclick="switchMode({{$i}}, 'inseason')">In-Season</button>
<button class="mode-btn active" onclick="switchMode({{$i}}, 'dynasty')">Dynasty</button>

<!-- Content starts with dynasty visible -->
<span class="inseason-desc" style="display:none;">Weekly matchups & tier-based recommendations</span>
<span class="dynasty-desc">Long-term value & dynasty toolkit</span>

<!-- Table content -->
<th class="inseason-only" style="display:none;">Tier</th>
<th class="dynasty-only">Age</th>

<!-- And in the initialization script at the bottom -->
<script>
// After tab click, trigger dynasty mode for each dynasty league
document.addEventListener('DOMContentLoaded', function() {
    {{range $i, $l := .Leagues}}
    {{if $l.IsDynasty}}
    // Initialize league {{$i}} in dynasty mode
    const league{{$i}} = document.getElementById('league{{$i}}');
    if (league{{$i}}) {
        // Set initial visibility
        league{{$i}}.querySelectorAll('.inseason-only').forEach(el => el.style.display = 'none');
        league{{$i}}.querySelectorAll('.dynasty-only').forEach(el => el.style.display = '');
    }
    {{end}}
    {{end}}
});
</script>
```

#### 5. Collapseable Sections

**JavaScript** (add to `templates/tiers.html`):
```javascript
function toggleSection(sectionId) {
    const content = document.getElementById(sectionId + '-content');
    const icon = document.getElementById(sectionId + '-icon');

    if (content.style.display === 'none') {
        content.style.display = '';
        icon.textContent = '▼';
    } else {
        content.style.display = 'none';
        icon.textContent = '▶';
    }
}
```

**CSS**:
```css
.toolkit-card {
    background: rgba(30, 38, 54, 0.5);
    border-radius: 12px;
    margin-bottom: 16px;
    border: 1px solid rgba(123, 176, 255, 0.15);
}

.card-header {
    padding: 16px;
    cursor: pointer;
    display: flex;
    justify-content: space-between;
    align-items: center;
    transition: background 0.2s ease;
}

.card-header:hover {
    background: rgba(123, 176, 255, 0.08);
}

.card-title {
    font-size: 1.15em;
    font-weight: 700;
    color: #7bb0ff;
    text-transform: uppercase;
    letter-spacing: 0.5px;
}

.collapse-icon {
    color: #9fb3d4;
    font-size: 0.9em;
    transition: transform 0.2s ease;
}

.card-content {
    padding: 0 16px 16px 16px;
}
```

#### 6. Horizontal Layout

**CSS changes** (`templates/tiers.html`):
```css
@media (min-width: 1024px) {
    .league-content {
        display: grid;
        grid-template-columns: 1fr 400px;
        gap: 24px;
    }

    .roster-card {
        grid-column: 1;
    }

    .dynasty-toolkit {
        grid-column: 2;
    }

    .fa-section {
        grid-column: 1 / -1; /* Full width */
    }
}

@media (max-width: 1023px) {
    .league-content {
        display: block;
    }
}
```

**HTML structure**:
```html
<div class="league-content" id="league{{$i}}">
    {{if $l.IsDynasty}}
    <!-- Mode toggle -->
    {{end}}

    <div class="roster-card">
        <!-- Existing roster table -->
        <!-- Move all dynasty-specific footer content OUT of table -->
    </div>

    {{if $l.IsDynasty}}
    <div class="dynasty-toolkit dynasty-only">
        <div class="toolkit-card collapseable">
            <!-- Draft Capital -->
        </div>

        <div class="toolkit-card collapseable">
            <!-- League Age Analysis -->
        </div>

        <div class="toolkit-card collapseable">
            <!-- Trade Targets -->
        </div>
    </div>
    {{end}}

    <!-- Free Agents (full width) -->
</div>
```

## Implementation Order

### Step 1: Quick Fixes (Bugs)
**Time estimate**: 30 minutes
1. Fix age sorting bug in main.go
2. Fix draft picks ownership logic in main.go
3. Default to dynasty mode in templates/tiers.html

### Step 2: Layout Restructure
**Time estimate**: 1 hour
1. Extract dynasty content from table footer
2. Create card-based layout with grid
3. Move draft capital to sidebar
4. Move league age to sidebar
5. Update CSS for horizontal layout
6. Test responsive behavior

### Step 3: Collapseable Sections
**Time estimate**: 30 minutes
1. Add toggleSection JavaScript function
2. Add collapse icons to card headers
3. Make draft capital collapseable
4. Make league age collapseable
5. Make free agents collapseable
6. Store collapse state in localStorage (optional enhancement)

### Step 4: Trade Targets Feature
**Time estimate**: 2-3 hours
1. Backend: Add PositionalKTC struct
2. Backend: Add calculatePositionalKTC function
3. Backend: Add findTradeTargets function
4. Backend: Add TradeTarget struct and field to LeagueData
5. Backend: Call findTradeTargets in lookupHandler
6. Frontend: Add trade targets card HTML
7. Frontend: Add trade targets styling
8. Test with real league data
9. Refine algorithm based on results

## Testing Plan

### Manual Testing Checklist
- [ ] Dynasty league defaults to Dynasty mode on page load
- [ ] League Age Analysis table is sorted oldest to youngest
- [ ] Draft picks only show picks I currently own
- [ ] Acquired picks show "from <team>" annotation
- [ ] Layout uses horizontal space on desktop (side-by-side)
- [ ] Layout stacks vertically on mobile
- [ ] Draft capital section collapses/expands
- [ ] League age section collapses/expands
- [ ] Free agents section collapses/expands
- [ ] Trade targets show 2-3 relevant suggestions
- [ ] Trade target suggestions make sense based on roster composition
- [ ] Collapse state persists when switching between leagues (optional)
- [ ] All existing functionality still works (in-season mode, tier highlighting, etc.)

### Edge Cases
- League with no traded picks
- League with only acquired picks
- League with only traded-away picks
- Team with balanced roster (no clear trade targets)
- Team with extreme imbalance (all WRs, no RBs)
- Mobile viewport
- Tablet viewport

## Open Questions for Jesse

1. **Trade Target Thresholds**: What percentage should constitute a "surplus" or "deficit"?
   - Proposed: >35% is surplus, <15% is deficit
   - Alternative: Could use standard deviation from league average

2. **Trade Target Scoring**: How to weight complementarity?
   - Simple approach: Binary (they have what you need AND you have what they need)
   - Complex approach: Weighted score based on degree of mismatch

3. **Collapse State**: Should collapsed/expanded state persist when switching between league tabs?
   - Pros: Better UX, user preference remembered
   - Cons: Requires localStorage, slightly more complex

4. **Draft Picks API**: The Sleeper API `/v1/league/{league_id}/traded_picks` returns picks that have been traded. Need to confirm the structure:
   - Does `owner_id` represent current owner?
   - Does `previous_owner_id` represent who traded it away?
   - Does it include future default picks or only traded ones?

5. **Mobile Layout**: For sidebar content on mobile, should it:
   - Stack below roster (current plan)
   - Be in a horizontal scrollable carousel
   - Be hidden behind a "Show Dynasty Tools" button

## File Changes Summary

### Files to Modify
1. `main.go` - Bug fixes and trade targets backend logic
2. `templates/tiers.html` - Layout restructure, collapseable sections, trade targets UI

### Files to Create
- None (this plan document already exists)

### Files to Test
- All existing functionality
- New features

## Success Criteria

After implementation, the following should be true:

1. ✅ Dynasty leagues open in Dynasty mode by default
2. ✅ League Age Analysis sorts correctly (oldest to youngest)
3. ✅ Draft picks show correct ownership
4. ✅ Acquired picks annotated with "from <team>"
5. ✅ Desktop layout uses horizontal space efficiently
6. ✅ Mobile layout stacks vertically
7. ✅ Draft capital section is collapseable
8. ✅ League age section is collapseable
9. ✅ Trade targets section shows 2-3 suggestions
10. ✅ Trade suggestions are relevant and actionable
11. ✅ All existing features still work
12. ✅ No visual regressions
13. ✅ Page loads in <2 seconds

## Notes

- Keep existing "from <team>" annotation (Jesse likes it)
- Don't change anything that's already working well
- Follow existing design patterns and color scheme
- Make minimal changes to achieve goals (YAGNI principle)
- Test thoroughly before considering done
