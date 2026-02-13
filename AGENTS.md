# Agent Instructions for SleeperPy

## CRITICAL GUIDELINES

### 1. TEST EVERYTHING YOURSELF FIRST

**BEFORE committing any change:**
- Build: `go build -o sleeperpy`
- Test with real user: `curl -s 'http://localhost:8888/dashboard?user=wboll'`
- Check for errors in output
- Test the specific feature you changed
- Verify on multiple screen sizes if UI-related

**Test users:**
- `wboll` - Real Sleeper user with dynasty leagues
- `testuser` - Mock data (use `--test` flag)

**Common test endpoints:**
- Dashboard: `http://localhost:8888/dashboard?user=wboll`
- Lookup: `http://localhost:8888/lookup?username=wboll`
- Check for errors: `grep -i error` in curl output

### 2. NO "AI SLOP" - PROFESSIONAL DESIGN ONLY

**AVOID:**
- ❌ Excessive emojis everywhere
- ❌ Over-the-top enthusiasm in UI text
- ❌ Flashy, gimmicky design elements
- ❌ Emoji in every single line of text
- ❌ "Gamification" that feels forced

**PREFER:**
- ✅ Clean, professional text
- ✅ Minimal, purposeful icons (1-2 per section max)
- ✅ Straightforward, helpful language
- ✅ Functional design over decorative
- ✅ User-focused, not marketing-speak

**Good example:**
```
Player News - Your Players (Top 3/12)
1. Breece Hall - OUT (knee injury, 2-week absence)
```

**Bad example:**
```
📰 What Changed This Week! 🔥 Your Players 🎯
1. 🔴 Breece Hall 💥 - OUT 😱 (knee injury 🤕)
```

**UI Text Guidelines:**
- Use clear, direct language
- Limit emojis to 1 per card/section header (if needed at all)
- Avoid exclamation marks unless truly critical
- Write like you're helping a friend, not selling a product
- Professional > Playful

### 3. RESPONSIVE DESIGN REQUIRED

**Every UI change must work on:**
- Mobile: 320px, 375px, 414px
- Tablet: 768px, 1024px
- Desktop: 1280px, 1920px

**Use flexible layouts:**
- CSS Grid with `auto-fill` / `auto-fit`
- Flexbox with `flex-wrap`
- `min-width` / `max-width`, not fixed widths
- Let content determine size

### 4. GIT WORKFLOW

**After EVERY change:**
1. Test locally (see above)
2. Build: `go build -o sleeperpy`
3. Commit with conventional commits: `feat:`, `fix:`, `chore:`
4. Always include Co-Authored-By footer
5. Push to remote
6. Note: You cannot restart systemd service (requires sudo)

**Commit format:**
```bash
git commit -m "$(cat <<'EOF'
fix: add year indicator to dashboard league cards

Duplicate league names (renewed leagues) now show year badge
to distinguish between seasons.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
EOF
)"
```

### 5. CODE QUALITY

**Follow existing patterns:**
- Go standard library only (no frameworks)
- Simple, readable code > clever code
- Delete unused code completely
- No backwards-compatibility hacks
- Fix bugs, don't add workarounds

**Error handling:**
- Log errors with context
- Show user-friendly messages
- Never show raw error strings in production UI

---

## Project Status

### Recently Completed (Phase 1)
✅ Feature #1: Cross-League Dashboard
✅ Feature #2: Weekly Action List
✅ Feature #3: Trade Fairness Detection
✅ Feature #4: News Signal Compression

### Known Issues
- Dashboard shows duplicate league names (needs year indicator) ← **FIX THIS NEXT**

---

## Project Architecture

**Backend:** Go with standard library
**Frontend:** HTML templates + vanilla JavaScript
**APIs:** Sleeper, Boris Chen, DynastyProcess (KTC)
**Caching:** In-memory with TTL

**Key Files:**
- `main.go` - Server setup, template functions
- `handlers.go` - HTTP handlers for all routes
- `fetch.go` - API fetching logic
- `dynasty.go` - Dynasty league features
- `roster.go` - Roster processing
- `types.go` - All struct definitions
- `templates/` - HTML templates

**Testing:**
- Test mode: `go run . --test` (use username "testuser")
- Debug mode: `go run . --log=debug`
- Real user: Use `wboll` for testing with actual data

---

## Current Focus

1. Fix dashboard duplicate league names (add year indicator)
2. Ensure all features are tested and working
3. Clean up any remaining emoji overuse
4. Verify responsive design on all breakpoints

---

## Debugging Real Data

**To inspect Sleeper API data:**
```bash
# Get user leagues
go run . cli user wboll

# Dynasty league ID: 1222367151910834176

# Fetch raw API data
curl -sS "https://api.sleeper.app/v1/league/1222367151910834176/traded_picks" -o /tmp/traded_picks.json
curl -sS "https://api.sleeper.app/v1/league/1222367151910834176/rosters" -o /tmp/rosters.json
curl -sS "https://api.sleeper.app/v1/league/1222367151910834176/users" -o /tmp/users.json
```

**Inspect with Python:**
```python
import json
with open('/tmp/rosters.json') as f:
    rosters = json.load(f)
# Map roster_id → owner_id → user name
```
