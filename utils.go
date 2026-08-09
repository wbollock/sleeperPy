// ABOUTME: Utility functions for SleeperPy application
// ABOUTME: Includes string manipulation, name normalization, and helper functions

package main

import (
	"strings"
)

// isOffseasonMode reports whether we're in the window from the end of the
// previous fantasy season through about a week before regular-season
// kickoff. During this window, weekly matchup-dependent content (tiers,
// swap/waiver suggestions) has nothing to show; dynasty tools (trade
// targets, season plan, draft strategy) are the focus instead.
//
// Deliberately uses season_type/week, not season_start_date: verified
// against the live API that season_start_date marks the start of Sleeper's
// fantasy year (roster resets, ~early August), not the regular-season
// opener (~early September) -- using it directly would end offseason mode
// a month early, right as preseason starts. NFL preseason runs 3 weeks;
// treat the last one (kickoff about a week out) as no longer offseason.
func isOffseasonMode(nflState map[string]interface{}) bool {
	seasonType, _ := nflState["season_type"].(string)
	switch seasonType {
	case "off":
		return true
	case "pre":
		week, ok := nflState["week"].(float64)
		return !ok || week < 3
	default: // "regular", "post", or unrecognized
		return false
	}
}

// toStringSlice converts interface{} to []string safely
func toStringSlice(val interface{}) []string {
	arr := []string{}
	if val == nil {
		return arr
	}
	switch v := val.(type) {
	case []interface{}:
		for _, x := range v {
			if s, ok := x.(string); ok {
				arr = append(arr, s)
			}
		}
	}
	return arr
}

// diff returns elements in a that are not in b
func diff(a, b []string) []string {
	m := make(map[string]bool)
	for _, x := range b {
		m[x] = true
	}
	out := []string{}
	for _, x := range a {
		if !m[x] {
			out = append(out, x)
		}
	}
	return out
}

// findTier locates the tier number for a player by name
func findTier(tiers [][]string, name string) int {
	norm := normalizeName(name)
	for i, names := range tiers {
		for _, n := range names {
			if normalizeName(n) == norm {
				return i + 1
			}
		}
	}
	return 0
}

// normalizeName standardizes player names for comparison
func normalizeName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, ".", "")
	name = strings.ReplaceAll(name, ",", "")
	for _, suf := range []string{" jr", " sr", " ii", " iii", " iv", " v"} {
		name = strings.TrimSuffix(name, suf)
	}
	// Remove non-alphanumeric except spaces
	var result strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == ' ' {
			result.WriteRune(r)
		}
	}
	name = strings.Join(strings.Fields(result.String()), " ")
	return name
}

// stripHTML removes HTML tags from strings
func stripHTML(s string) string {
	if idx := strings.Index(s, "<span"); idx >= 0 {
		return strings.TrimSpace(s[:idx])
	}
	return s
}

// isDynastyLeague determines if a league is dynasty format
func isDynastyLeague(league map[string]interface{}) bool {
	// Check type field - type 2 indicates dynasty league
	if settings, ok := league["settings"].(map[string]interface{}); ok {
		if leagueType, ok := settings["type"].(float64); ok && leagueType == 2 {
			debugLog("[DEBUG] League detected as dynasty via type: %v", leagueType)
			return true
		}
	}

	// Check for taxi squad (dynasty-specific feature)
	if settings, ok := league["settings"].(map[string]interface{}); ok {
		if taxiSlots, ok := settings["taxi_slots"].(float64); ok && taxiSlots > 0 {
			debugLog("[DEBUG] League detected as dynasty via taxi_slots: %v", taxiSlots)
			return true
		}
	}

	// Fallback: check league name for "dynasty" keyword
	if name, ok := league["name"].(string); ok {
		nameLower := strings.ToLower(name)
		if strings.Contains(nameLower, "dynasty") {
			debugLog("[DEBUG] League detected as dynasty via name: %s", name)
			return true
		}
	}

	return false
}

// detectScoringFormat auto-derives a league's scoring format ("PPR", "Half
// PPR", or "Standard") from its Sleeper reception-points setting. This is
// the auto-detection fallback; callers should prefer a user's manual
// per-league override (see UserSettings.ScoringOverrides) when one is set.
func detectScoringFormat(league map[string]interface{}) string {
	scoring := "PPR"
	if scoringSettings, ok := league["scoring_settings"].(map[string]interface{}); ok {
		if rec, ok := scoringSettings["rec"].(float64); ok {
			if rec == 0.5 {
				scoring = "Half PPR"
			} else if rec == 0.0 {
				scoring = "Standard"
			}
		}
	}
	return scoring
}

// parseCSVLine parses a CSV line handling quoted fields
func parseCSVLine(line string) []string {
	var fields []string
	var current strings.Builder
	inQuotes := false

	for i := 0; i < len(line); i++ {
		char := line[i]
		if char == '"' {
			if inQuotes && i+1 < len(line) && line[i+1] == '"' {
				current.WriteByte('"')
				i++
			} else {
				inQuotes = !inQuotes
			}
		} else if char == ',' && !inQuotes {
			fields = append(fields, current.String())
			current.Reset()
		} else {
			current.WriteByte(char)
		}
	}
	fields = append(fields, current.String())
	return fields
}
