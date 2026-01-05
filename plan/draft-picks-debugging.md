# Draft Picks Ownership Debugging Guide

## Problem Statement

Draft picks are showing incorrect ownership information. For example, showing "from gdyche" when the user actually traded that pick away to gdyche (not acquired from gdyche).

## Expected Behavior

1. **Only show picks currently owned by the user**
2. **If pick was acquired via trade**: Show "from <team>" annotation
3. **If pick was traded away**: Don't show it at all
4. **If pick is user's original pick**: Show without "from" annotation

## Current Implementation

The code uses the Sleeper API `/v1/league/{league_id}/traded_picks` endpoint with these assumptions:

- `roster_id`: Current owner of the pick
- `owner_id`: Original owner (team pick belonged to by default)
- `previous_owner_id`: Previous owner before current trade

**This might be incorrect** - we need real API data to verify.

## Debug Logging Added

Comprehensive debug logging has been added to help diagnose the issue. When you run with `--log=debug`, you'll see:

### 1. Raw Traded Picks Data
```
[DEBUG] ===== TRADED PICKS RAW DATA =====
[DEBUG] Trade 0: map[owner_id:X roster_id:Y previous_owner_id:Z round:1 season:2025]
[DEBUG]   season: 2025
[DEBUG]   round: 1
[DEBUG]   roster_id (current owner): Y
[DEBUG]   owner_id (original owner): X
[DEBUG]   previous_owner_id: Z
[DEBUG] ===================================
```

### 2. Pick Ownership Updates
```
[DEBUG] Processing 5 traded picks
[DEBUG] Trade 0: 2025 Round 1 (originally from roster 3)
[DEBUG]   Current owner (roster_id): 7
[DEBUG]   Previous owner: 3
[DEBUG]   ✓ Updated ownership from roster 3 to roster 7
```

### 3. User's Final Picks
```
[DEBUG] ===== EXTRACTING USER PICKS =====
[DEBUG] Searching for picks owned by roster 7
[DEBUG] Pick 2025 Round 1: ACQUIRED from roster 3 (John Smith)
[DEBUG] Pick 2025 Round 2: YOUR ORIGINAL pick
[DEBUG] ===================================
[DEBUG] FINAL RESULT: User has 12 draft picks total
```

## How to Debug

### Step 1: Run with Debug Logging

```bash
./sleeperpy --log=debug
```

### Step 2: Navigate to a Dynasty League with Traded Picks

Find a dynasty league where you know:
- Which picks you acquired (and from whom)
- Which picks you traded away (and to whom)

### Step 3: Capture the Debug Output

Save the debug output, especially these sections:
1. "TRADED PICKS RAW DATA" - shows what the API returns
2. "EXTRACTING USER PICKS" - shows which picks are displayed

### Step 4: Verify Against Actual Trades

Compare the debug output with your actual trade history. For each pick shown:

**If showing a pick you traded AWAY:**
- This is the bug! Look at the raw API data for that pick
- Check what `roster_id`, `owner_id`, and `previous_owner_id` contain
- This will tell us which field interpretation is wrong

**If showing "from <team>" for a pick you traded AWAY TO that team:**
- This is the specific bug mentioned in the plan
- The logic is likely confusing which direction the pick moved

**If NOT showing a pick you acquired:**
- Check if it appears in the raw traded picks data
- If yes, the extraction logic is wrong
- If no, the API might not have returned it

## Common Scenarios to Test

### Scenario 1: You acquired Team A's 2025 1st round pick

**Expected display:** "2025 Round 1 (from Team A)"

**What to check:**
- Does raw API show this trade?
- What are the field values for roster_id, owner_id, previous_owner_id?
- Is "from Team A" shown correctly?

### Scenario 2: You traded your 2025 2nd round pick to Team B

**Expected display:** Pick should NOT appear at all

**What to check:**
- Does raw API show this trade?
- Is the pick being extracted as "yours"?
- If yes, look at why the ownership check is passing

### Scenario 3: You acquired Team C's pick, then traded it to Team D

**Expected display:** Pick should NOT appear (you don't own it anymore)

**What to check:**
- Are both trades in the raw API data?
- Does the ownership tracking handle multi-hop trades?
- Is the final owner correctly set to Team D (not you)?

## Sleeper API Field Meanings (TO BE VERIFIED)

Based on the debug output, we need to determine the actual meaning of:

### Possibility 1 (Current assumption):
- `roster_id` = current owner after this trade
- `owner_id` = original owner (team it belonged to by default)
- `previous_owner_id` = owner before this specific trade

### Possibility 2 (Alternative):
- `roster_id` = current owner after ALL trades
- `owner_id` = current owner (not original)
- `previous_owner_id` = owner who traded it away

### Possibility 3 (Another alternative):
- Sleeper API might return multiple entries for the same pick if traded multiple times
- Need to process trades in order or use only the latest state

## Example Debug Analysis

### Case: Bug Report - "from gdyche" shown incorrectly

**Reported issue:** User traded their 2025 1st to gdyche, but it shows as "from gdyche"

**Debug output to look for:**
```
[DEBUG] Trade 0: 2025 Round 1 (originally from roster 2)
[DEBUG]   Current owner (roster_id): 5
[DEBUG]   Previous owner: 2
[DEBUG]   User roster ID: 2
```

**Analysis:**
- User is roster 2
- Pick originally from roster 2 (user's own pick)
- Current owner is roster 5 (gdyche)
- Previous owner is roster 2 (user)

**What's happening:**
- The code sees roster_id (5) != originalRosterID (2)
- So it marks it as "acquired from roster 2"
- But user IS roster 2, so this is wrong!

**Fix:**
- Need to check if current owner == user, not just original owner
- The pick should NOT be shown at all because user doesn't own it anymore

## Next Steps After Debugging

1. Share the debug output with the team
2. Identify which API field interpretation is correct
3. Update the ownership logic accordingly
4. Re-test with the same league to verify fix
5. Test with multiple scenarios to ensure robustness

## Code Location

- File: `main.go`
- Lines: 1188-1352 (Draft picks fetching and processing)
- Key logic: Lines 1267-1296 (Apply traded picks)
- Extraction: Lines 1298-1351 (Extract user's picks)
