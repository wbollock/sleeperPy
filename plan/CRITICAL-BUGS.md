# Critical Bugs - SleeperPy

## Priority: BLOCKING - Fix Before Any New Features

---

## Bug #1: Draft Pick Ownership is Inverted

### Severity: CRITICAL
**Impact**: Users see draft picks they traded away, and don't see picks they acquired.

### Description
The draft picks feature is showing incorrect ownership. When a user trades a pick TO another team, the app incorrectly displays that pick as owned BY the user with a "from <team>" annotation.

### User Report
**Actual Trade**: User traded 2026 Round 1 pick TO gdyche
**App Display**: Shows "2026 Round 1 (from wboll)" ❌
**Expected Display**: Should NOT show this pick at all (user doesn't own it) ✅

### Root Cause Analysis
The Sleeper API `/v1/league/{league_id}/traded_picks` endpoint fields are being misinterpreted:

**Current Assumptions (LIKELY WRONG)**:
- `roster_id` = current owner of the pick
- `owner_id` = original owner (default owner)
- `previous_owner_id` = previous owner before current trade

**Actual API Behavior (SUSPECTED)**:
The field meanings might be reversed or different. Need to test with real API data.

**Evidence**:
- User traded pick TO gdyche → pick should have `roster_id` = gdyche's roster
- But app shows pick as owned by user → suggests `roster_id` might mean something else
- The "from wboll" annotation means `owner_id` = wboll's roster
- This creates impossible state: "You own a pick from yourself"

### Code Location
**File**: `handlers.go`
**Lines**: 1079-1275 (draft picks logic)

Key problematic sections:
```go
// Line 1161-1162: Field interpretation
rosterID, _ := trade["roster_id"].(float64)        // Current owner after trade?
originalRosterID, _ := trade["owner_id"].(float64) // Original owner (default owner)?

// Line 1188: Updates ownership
pickOwnership[key] = int(rosterID)

// Line 1211-1216: Filters for user's picks
if ownerRosterID != int(userRosterID) {
    continue  // Skip picks not owned by user
}

// Line 1233-1240: Annotates acquired picks
if originalRosterID != int(userRosterID) {
    originalName = origOwner  // Shows "from <team>"
}
```

### Fix Strategy

**Step 1: Verify API Field Meanings**
Run the app with debug logging for a league with known traded picks:
```bash
go run . --log=debug
```

Check the "TRADED PICKS RAW DATA" section and compare against actual trades in Sleeper app.

**Step 2: Test All Scenarios**
For each traded pick, verify:
1. User's original pick traded away → should NOT appear in list
2. User acquired someone else's pick → should show "from <team>"
3. User's original pick not traded → should show without "from" annotation
4. Multi-hop trades (A→B→C) → verify current owner is correct

**Step 3: Correct Field Interpretation**
Once API fields are understood, update the logic:
- Ensure `pickOwnership[key]` contains the TRUE current owner
- Ensure the extraction filter (line 1211-1216) only includes user's picks
- Ensure "from" annotation shows the correct original owner

**Step 4: Add Validation**
- Log every pick's journey: "Pick X: was owned by Y, now owned by Z"
- Compare against Sleeper app's trade history
- Add unit tests with mock API data

### Alternative Hypothesis
**Possibility**: The API's `roster_id` might refer to the roster the pick BELONGS TO (original team), and we need to track trades separately through `previous_owner_id` chain.

If this is true, the logic needs complete rewrite:
- Start with each team's default picks (roster owns their own picks)
- For each trade, follow the chain: owner_id → previous_owner_id → roster_id
- Build ownership map by following trade history, not just reading current state

### Testing Data Needed
To fix this, we need debug output from a real league showing:
1. Raw API response for traded_picks
2. User's actual roster_id
3. Trade history from Sleeper app (who traded what to whom)
4. Current ownership state in Sleeper app

Then compare against what our app displays.

---

## Bug #2: UI Layout is Broken - Text Cutting Off

### Severity: HIGH
**Impact**: App is unusable on many screen sizes, unprofessional appearance

### Description
The current box-based layout is failing:
- Text cutting off across the app
- Boxes not scaling properly
- Layout breaks on different screen sizes
- Too rigid, doesn't handle varying content lengths

### User Report
"the UI of the app needs help the boxes are not holding up and text is cutting all over the plan. we need a flexible, clean, and satisifying UI that can scale for all these basica nd premium features not relying on a series of boxes anymore."

### Root Cause
**Design Approach**: Current CSS relies heavily on fixed-height boxes, rigid grid layouts, and multiple overlapping stylesheets
**Problem**: Dynasty toolkit has variable content (news feeds, transactions, age analysis, etc.) that doesn't fit predictable box sizes

### Affected Files
- `templates/tiers.html` - Main results page with dynasty toolkit
- `templates/index.html` - Landing page
- `static/styles.css` - Base styles
- `static/loading.css` - Loading states (KEEP THIS)
- Multiple deleted elite CSS files (already removed in cleanup)

### Fix Strategy

**Complete UI Redesign Required**

**Goals**:
1. **Flexible, Not Fixed**: Use flexbox/grid with auto-sizing, not fixed heights
2. **Content-First**: Let content determine layout, not boxes constraining content
3. **Scalable**: Handle both basic tier view AND full dynasty toolkit gracefully
4. **Clean & Satisfying**: Modern, professional look without flashiness
5. **No More Boxes**: Replace rigid box metaphor with flowing sections

**Design Principles**:
- **Card-based BUT flexible**: Cards expand to fit content, not truncate
- **Responsive by default**: Mobile-first design that scales up
- **Whitespace**: Use padding/margins effectively, don't cram content
- **Typography**: Proper text sizing, line-height, word-wrap
- **Progressive disclosure**: Collapsible sections work properly
- **Visual hierarchy**: Clear information architecture

**Technical Approach**:

1. **Audit Current CSS** (~2 hours)
   - Identify all fixed widths/heights
   - Find overflow:hidden and text truncation
   - Map breakpoint issues
   - List conflicting styles

2. **Create New Foundation** (~4 hours)
   - New `flexible-layout.css` with modern CSS Grid
   - Proper responsive utilities
   - Consistent spacing system (--spacing-sm, --spacing-md, etc.)
   - Typography scale (--text-sm, --text-base, --text-lg, etc.)
   - Remove all fixed dimensions

3. **Rebuild Key Sections** (~6 hours)
   - Tier display (main content area)
   - Dynasty toolkit sidebar/sections
   - Transaction history (two-column trade layout)
   - News feed (variable-length content)
   - Draft picks display
   - Mobile layout (stack vertically, not side-by-side)

4. **Polish & Test** (~2 hours)
   - Test on mobile (320px, 375px, 414px)
   - Test on tablet (768px, 1024px)
   - Test on desktop (1280px, 1920px)
   - Test with real league data (varying content lengths)
   - Test dark mode compatibility

**Inspiration** (DO NOT COPY, just principles):
- Tailwind's utility-first approach (spacing, typography)
- Linear.app's clean information density
- GitHub's flexible responsive grid
- Notion's content-first layout

**What to Keep**:
- Dark/light theme switcher (works fine)
- Loading states (loading.css is good)
- Color scheme (just make it more flexible)
- Dynasty mode toggle (functionality is fine)

**What to Remove**:
- All fixed pixel heights on content containers
- Overflow:hidden on text content
- Rigid box metaphor CSS
- Complex multi-stylesheet architecture

### Expected Outcome
After redesign:
- Text never cuts off (wraps properly)
- Layout adapts to content length
- Works on all screen sizes
- Professional, modern appearance
- Maintains existing functionality (dark mode, collapsible sections, etc.)
- Single cohesive stylesheet (or modular system)

### Time Estimate
**Total**: ~14 hours for complete UI overhaul
- Worth it - current UI is blocking user satisfaction
- Will prevent future layout issues
- Makes adding new features easier

---

## Implementation Order

### MUST FIX BOTH BEFORE ANY NEW FEATURES

**Priority 1: Draft Picks Bug** (~4-8 hours)
- Blocking dynasty league users
- Data accuracy issue (trust problem)
- Need to understand API first (debugging session)
- Then implement fix
- Then test thoroughly

**Priority 2: UI Redesign** (~14 hours)
- Blocking user experience
- Affects all users, all screen sizes
- Makes app look unprofessional
- Required before adding any new features (would inherit broken layout)

**Total**: ~18-22 hours to fix critical issues

---

## Post-Fix Validation

**Draft Picks Validation**:
- [ ] Test with multiple dynasty leagues
- [ ] Verify all trade scenarios (acquire, trade away, multi-hop)
- [ ] Compare app display vs Sleeper app for 100% accuracy
- [ ] Add automated tests with mock API data
- [ ] Document API field meanings in code comments

**UI Validation**:
- [ ] Test on 6+ screen sizes (mobile, tablet, desktop)
- [ ] Test with short content (1 player news item)
- [ ] Test with long content (50 transactions)
- [ ] Test all dynasty toolkit sections
- [ ] Test in both light and dark modes
- [ ] Get user feedback on new design

---

## After Critical Fixes: What's Next?

Only after BOTH bugs are fixed, we can consider new features:

**Potential Future Work** (from archived plans):
- CLI mode for testing/debugging
- OpenTelemetry observability
- Power user improvements (multi-league management)
- Admin dashboard
- Enhanced league features
- Mobile PWA enhancements

**But NONE of these until critical bugs are resolved.**

---

## Notes

- All previous feature plans moved to `plan/archive/`
- Focus is 100% on fixing these two critical bugs
- No new features until these are stable
- User satisfaction depends on core functionality working correctly
- A broken UI + wrong data = unusable app, regardless of features
