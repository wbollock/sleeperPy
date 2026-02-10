# Claude Agent Instructions for SleeperPy

## CRITICAL PRIORITY: FIX BUGS FIRST

**ALL FEATURE WORK IS SUSPENDED** until these two critical bugs are fixed:

### Bug #1: Draft Pick Ownership Inverted (BLOCKING) 🔴

**Issue**: Draft picks show wrong ownership
- User trades pick away → app incorrectly shows they still own it
- User receives pick → might not show correctly

**Example**:
- User traded 2026 Round 1 pick TO gdyche
- App displays: "2026 Round 1 (from wboll)" ❌
- Should display: Nothing (user doesn't own it) ✅

**Root Cause**: Sleeper API field interpretation wrong
- Code location: `handlers.go` lines 1079-1275
- Current assumptions about API fields are likely incorrect:
  - Assumes `roster_id` = current owner
  - Assumes `owner_id` = original owner
- User data proves this is wrong

**Fix Steps**:
1. Run app with `--log=debug` on a real league with traded picks
2. Examine "TRADED PICKS RAW DATA" output
3. Compare against actual Sleeper app trade history
4. Determine correct field meanings
5. Rewrite draft picks logic with correct interpretation
6. Test all scenarios: acquired picks, traded away picks, original picks
7. Add validation to prevent regression

**Test Cases Required**:
- User's original pick (not traded) → show without annotation
- User acquired pick from Team A → show "from Team A"
- User traded pick to Team B → DON'T show at all
- Multi-hop trades (A→B→C) → show correct current owner

**Estimated Time**: 4-8 hours (includes debugging + fix + testing)

---

### Bug #2: UI Layout Broken (BLOCKING) 🔴

**Issue**: Layout is broken across the app
- Text cutting off everywhere
- Boxes not holding content properly
- Fails on different screen sizes
- Rigid box-based design doesn't scale

**User Report**: "the UI of the app needs help the boxes are not holding up and text is cutting all over the plan. we need a flexible, clean, and satisifying UI that can scale for all these basica nd premium features not relying on a series of boxes anymore."

**Root Cause**: Poor CSS architecture
- Fixed-height boxes constraining variable content
- Rigid grid layout not responsive
- Multiple conflicting stylesheets
- `overflow:hidden` cutting off text
- Box metaphor doesn't fit dynasty toolkit's variable content

**Fix Required**: Complete UI Redesign

**Design Goals**:
1. **Flexible, not fixed**: CSS Grid/Flexbox with auto-sizing
2. **Content-first**: Let content determine size, not boxes constraining content
3. **Scalable**: Works for basic tier view AND full dynasty toolkit
4. **Clean & professional**: Modern look without excessive flashiness
5. **No more rigid boxes**: Flowing sections that adapt to content

**Implementation Steps**:

1. **Audit Current CSS** (~2 hours)
   - Find all fixed widths/heights → convert to flexible
   - Find overflow:hidden on text → remove
   - Map responsive breakpoint issues
   - Identify conflicting styles

2. **Create New Foundation** (~4 hours)
   - New modular CSS architecture
   - CSS Grid for main layout (not fixed boxes)
   - Consistent spacing system (CSS custom properties)
   - Typography scale (proper sizing, line-height, word-wrap)
   - Mobile-first responsive design
   - Remove all fixed dimensions on content

3. **Rebuild Key Sections** (~6 hours)
   - Tier display (main content area)
   - Dynasty toolkit (sidebar/cards with flexible height)
   - Transaction history (proper two-column layout)
   - News feed (variable-length content)
   - Draft picks display
   - Mobile stacking (vertical, not cramped horizontal)

4. **Test & Polish** (~2 hours)
   - Test on mobile sizes (320px, 375px, 414px)
   - Test on tablets (768px, 1024px)
   - Test on desktop (1280px, 1920px)
   - Test with varying content (1 item vs 50 items)
   - Test dark mode compatibility
   - Verify collapsible sections work

**Files to Modify**:
- `templates/tiers.html` - Main results page
- `templates/index.html` - Landing page
- `static/styles.css` - Complete rewrite or new modular approach
- KEEP `static/loading.css` - Loading states work fine

**What to Keep**:
- Dark/light theme toggle (functional)
- Loading states (good)
- Color scheme (just make flexible)
- Collapsible sections (functionality is fine)

**What to Remove**:
- Fixed pixel heights on content containers
- Overflow:hidden on text
- Rigid box CSS paradigm
- Complex multi-stylesheet conflicts

**Estimated Time**: 14 hours total

---

## Implementation Priority

**MUST COMPLETE IN ORDER - NO EXCEPTIONS**

### Phase 1: Fix Draft Picks Bug
- Debug API fields with real data
- Rewrite draft picks logic with correct interpretation
- Test all trade scenarios
- Commit: "fix: correct draft pick ownership logic"
- **BLOCK**: Do not proceed until this is 100% accurate

### Phase 2: Fix UI Layout
- Audit and document current CSS issues
- Design new flexible layout system
- Implement responsive CSS Grid foundation
- Rebuild all major sections with flexible design
- Test on all screen sizes
- Commit: "fix: redesign UI for flexible, scalable layout"
- **BLOCK**: Do not proceed until UI works on all screens

### Phase 3: Only After Both Bugs Fixed
- User approval required
- All previous feature plans are in `plan/archive/`
- Do NOT implement archived features without explicit user request
- Focus remains on stability and usability

---

## Suspended Features

All feature plans have been moved to `plan/archive/`:
- CLI mode
- OpenTelemetry observability
- Power user improvements
- Admin dashboard
- Enhanced league features
- Mobile PWA enhancements

**DO NOT implement these until critical bugs are fixed and user approves.**

---

## Working Guidelines (For Bug Fixes)

**Code Quality:**
- Follow existing patterns in handlers.go, main.go, etc.
- Use Go standard library (no frameworks)
- Keep it simple - don't over-engineer
- Fix the bug, don't add features
- No backwards-compatibility hacks
- Delete unused code completely

**Git Workflow:**
- Commit after each bug fix
- Use conventional commits: `fix:`, not `feat:`
- ALWAYS add footer: `Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>`
- Test thoroughly before committing
- Use HEREDOC for multi-line commit messages:
  ```bash
  git commit -m "$(cat <<'EOF'
  fix: correct draft pick ownership logic

  - Debug API showed roster_id is actually [correct meaning]
  - Rewrite logic to properly track pick ownership
  - Add validation to prevent showing traded-away picks

  Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
  EOF
  )"
  ```

**Testing:**
- Run `go run . --log=debug` to see detailed API data
- Test with real leagues (not just testuser)
- Verify against Sleeper app's actual data
- Test all edge cases before committing

**Debugging:**
- Use existing debug logging infrastructure
- Add more debug logs if needed to understand API
- Compare raw API data vs expected behavior
- Document findings in code comments

---

## Project Context

**Architecture:**
- Backend: Go with standard library (no frameworks)
- Frontend: HTML templates (html/template) + vanilla JavaScript
- APIs: Sleeper, Boris Chen, DynastyProcess (KTC)
- Caching: In-memory (simple maps with mutexes)

**Key Files:**
- `main.go` - Entry point, server setup, template functions
- `handlers.go` - HTTP handlers, **DRAFT PICKS LOGIC HERE (lines 1079-1275)**
- `fetch.go` - API fetching (Sleeper, Boris Chen, KTC)
- `dynasty.go` - Dynasty features (news, breakouts, trade targets)
- `roster.go` - Roster processing and tier assignment
- `types.go` - All struct definitions (includes DraftPick)
- `utils.go` - Utility functions
- `templates/tiers.html` - Main results page, **UI LAYOUT ISSUES HERE**
- `templates/index.html` - Landing page
- `static/styles.css` - Main stylesheet, **NEEDS REDESIGN**

**Completed Features:**
- Core tier analysis with Boris Chen
- Dynasty toolkit (news, breakouts, aging alerts, trade targets, transactions, power rankings, rookies)
- Dynasty mode with collapseable sections
- Dark/light theme switcher
- Mobile responsive design (BROKEN - needs fix)
- Product pages (Privacy, ToS, About, FAQ, Pricing, SEO)

---

## Debugging the Draft Picks Bug

**Run with debug logging:**
```bash
go run . --log=debug
```

**Visit a dynasty league in the browser, then check terminal output for:**
1. "TRADED PICKS RAW DATA" - Shows API response
2. "APPLYING TRADED PICKS" - Shows how logic interprets trades
3. "EXTRACTING USER PICKS" - Shows which picks are included in final list

**Compare terminal output against Sleeper app's trade history.**

**Questions to answer from debug output:**
- What does `roster_id` actually represent?
- What does `owner_id` actually represent?
- What does `previous_owner_id` actually represent?
- For a pick the user traded away, who does the API say owns it now?
- For a pick the user acquired, who does the API say owns it now?

---

## Reference

- **Critical bugs documented in**: `plan/CRITICAL-BUGS.md`
- **All feature plans archived in**: `plan/archive/`
- **Project history in**: `MEMORY.md`
- **Current focus**: Fix bugs, not add features
