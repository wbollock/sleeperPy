package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type debugLogger func(format string, args ...interface{})

func buildDraftPicks(
	tradedPicks []map[string]interface{},
	rosters []map[string]interface{},
	rosterOwners map[int]string,
	numRounds int,
	userRosterID int,
	currentYear int,
	debug debugLogger,
) []DraftPick {
	if userRosterID == 0 {
		return nil
	}

	debugf := func(format string, args ...interface{}) {
		if debug != nil {
			debug(format, args...)
		}
	}

	// Build year set from next 3 years plus any years present in traded picks
	years := make(map[int]struct{})
	for year := currentYear; year < currentYear+3; year++ {
		years[year] = struct{}{}
	}
	for _, trade := range tradedPicks {
		if year := parsePickYear(trade["season"]); year > 0 {
			years[year] = struct{}{}
		}
	}

	// Initialize default picks (each team has their own picks by default)
	// key: "year-round-original_roster_id" -> current_owner_roster_id
	pickOwnership := make(map[string]int)
	for year := range years {
		for round := 1; round <= numRounds; round++ {
			for _, r := range rosters {
				rosterID, ok := parsePickInt(r["roster_id"])
				if !ok || rosterID == 0 {
					continue
				}
				key := fmt.Sprintf("%d-%d-%d", year, round, rosterID)
				pickOwnership[key] = rosterID
			}
		}
	}
	debugf("[DEBUG] Initialized %d default picks across %d years", len(pickOwnership), len(years))

	// Apply traded picks
	if tradedPicks != nil {
		debugf("[DEBUG] ===== APPLYING TRADED PICKS =====")
		debugf("[DEBUG] Processing %d traded picks", len(tradedPicks))
		for i, trade := range tradedPicks {
			seasonYear := parsePickYear(trade["season"])
			round, _ := parsePickInt(trade["round"])
			// Sleeper API field meanings (verified with real data):
			// roster_id = original owner (default owner)
			// owner_id = current owner
			// previous_owner_id = previous owner before current owner
			originalRosterID, _ := parsePickInt(trade["roster_id"])
			ownerID, _ := parsePickInt(trade["owner_id"])
			previousOwnerID, _ := parsePickInt(trade["previous_owner_id"])

			// Validate data
			if seasonYear == 0 || round == 0 || originalRosterID == 0 {
				debugf("[DEBUG] Trade %d: SKIPPING invalid trade data", i)
				continue
			}

			key := fmt.Sprintf("%d-%d-%d", seasonYear, round, originalRosterID)

			debugf("[DEBUG] Trade %d: %d Round %d (originally roster %d's pick)", i, seasonYear, round, originalRosterID)
			debugf("[DEBUG]   owner_id (current owner): %d", ownerID)
			debugf("[DEBUG]   previous_owner_id: %d", previousOwnerID)
			debugf("[DEBUG]   Key: %s", key)

			// Defensive check: Verify pick exists in ownership map
			oldOwner, pickExists := pickOwnership[key]
			if !pickExists {
				debugf("[DEBUG]   ⚠️  WARNING: Pick not found in ownership map, skipping")
				continue
			}

			// Update ownership
			if ownerID > 0 {
				if previousOwnerID > 0 && previousOwnerID != oldOwner {
					debugf("[DEBUG]   ⚠️  WARNING: previous_owner_id %d does not match expected owner %d", previousOwnerID, oldOwner)
				}

				pickOwnership[key] = ownerID
				debugf("[DEBUG]   ✓ Updated: roster %d → roster %d", oldOwner, ownerID)

				// Extra logging for user's picks
				if ownerID == userRosterID {
					debugf("[DEBUG]   📥 USER ACQUIRED this pick")
				} else if oldOwner == userRosterID {
					debugf("[DEBUG]   📤 USER TRADED AWAY this pick to roster %d", ownerID)
				}
			} else {
				// owner_id is 0 or invalid - pick might be in limbo or deleted
				delete(pickOwnership, key)
				debugf("[DEBUG]   ✗ Deleted pick (owner_id is 0 or invalid)")
			}
		}
		debugf("[DEBUG] ===================================")
	}

	// Extract user's picks
	debugf("[DEBUG] ===== EXTRACTING USER PICKS =====")
	debugf("[DEBUG] Searching for picks owned by roster %d", userRosterID)
	debugf("[DEBUG] Total picks in ownership map: %d", len(pickOwnership))

	draftPicks := make([]DraftPick, 0)
	for key, ownerRosterID := range pickOwnership {
		if ownerRosterID != userRosterID {
			debugf("[DEBUG] Skipping pick %s: owned by roster %d (not user)", key, ownerRosterID)
			continue
		}

		parts := strings.Split(key, "-")
		if len(parts) != 3 {
			debugf("[DEBUG] Invalid pick key format: %s", key)
			continue
		}
		year, _ := strconv.Atoi(parts[0])
		round, _ := strconv.Atoi(parts[1])
		originalRosterID, _ := strconv.Atoi(parts[2])

		ownerName := "You"
		originalName := ""

		if originalRosterID != userRosterID {
			// User acquired this pick from another team
			if origOwner, exists := rosterOwners[originalRosterID]; exists {
				originalName = origOwner
			} else {
				originalName = fmt.Sprintf("Team %d", originalRosterID)
			}
			debugf("[DEBUG] ✓ Pick %d Round %d: ACQUIRED from roster %d (%s)", year, round, originalRosterID, originalName)
		} else {
			debugf("[DEBUG] ✓ Pick %d Round %d: YOUR ORIGINAL pick", year, round)
		}

		draftPicks = append(draftPicks, DraftPick{
			Round:        round,
			Year:         year,
			OwnerName:    ownerName,
			OriginalName: originalName,
			RosterID:     userRosterID,
			IsYours:      true,
		})
	}

	debugf("[DEBUG] ===================================")
	debugf("[DEBUG] FINAL RESULT: User has %d draft picks total", len(draftPicks))

	sort.Slice(draftPicks, func(i, j int) bool {
		if draftPicks[i].Year != draftPicks[j].Year {
			return draftPicks[i].Year < draftPicks[j].Year
		}
		return draftPicks[i].Round < draftPicks[j].Round
	})

	return draftPicks
}

func parsePickYear(v interface{}) int {
	switch val := v.(type) {
	case string:
		year, _ := strconv.Atoi(val)
		return year
	case float64:
		return int(val)
	case int:
		return val
	default:
		return 0
	}
}

func parsePickInt(v interface{}) (int, bool) {
	switch val := v.(type) {
	case float64:
		return int(val), true
	case int:
		return val, true
	case string:
		if val == "" {
			return 0, false
		}
		num, err := strconv.Atoi(val)
		if err != nil {
			return 0, false
		}
		return num, true
	default:
		return 0, false
	}
}
