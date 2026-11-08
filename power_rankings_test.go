package main

import "testing"

// A superflex team that stashes four QBs on the bench should not out-value a
// team with two great QBs, because only 2 QB-eligible slots (QB + SUPER_FLEX)
// actually start.
func TestCalculateStarterValueCapsQBsAtSuperflexSlotCount(t *testing.T) {
	players := map[string]interface{}{
		"QB1": map[string]interface{}{"first_name": "Ace", "last_name": "One", "position": "QB"},
		"QB2": map[string]interface{}{"first_name": "Ace", "last_name": "Two", "position": "QB"},
		"QB3": map[string]interface{}{"first_name": "Ace", "last_name": "Three", "position": "QB"},
		"QB4": map[string]interface{}{"first_name": "Ace", "last_name": "Four", "position": "QB"},
		"RB1": map[string]interface{}{"first_name": "Back", "last_name": "One", "position": "RB"},
		"WR1": map[string]interface{}{"first_name": "Wide", "last_name": "One", "position": "WR"},
	}
	dynastyValues := map[string]DynastyValue{
		normalizeName("Ace One"):   {Value2QB: 9000},
		normalizeName("Ace Two"):   {Value2QB: 8000},
		normalizeName("Ace Three"): {Value2QB: 7000}, // rostered but shouldn't start
		normalizeName("Ace Four"):  {Value2QB: 6000}, // rostered but shouldn't start
		normalizeName("Back One"):  {Value2QB: 3000},
		normalizeName("Wide One"):  {Value2QB: 2000},
	}
	rosterPositions := []string{"QB", "RB", "WR", "TE", "SUPER_FLEX", "FLEX", "BN", "BN"}
	rosterPlayers := []string{"QB1", "QB2", "QB3", "QB4", "RB1", "WR1"}

	got := calculateStarterValue(rosterPlayers, players, dynastyValues, true, rosterPositions)

	// QB slot + SUPER_FLEX slot can only hold 2 QBs: the top two (9000+8000).
	// RB fills the RB slot (3000), WR fills the FLEX slot (2000) since no
	// eligible WR/TE is left for a dedicated WR slot beyond that FLEX use.
	want := 9000 + 8000 + 3000 + 2000
	if got != want {
		t.Errorf("expected starter value %d (top-2 QBs + RB + WR), got %d -- bench QBs must not count", want, got)
	}
}

// Fixed single-position slots should be filled before flex slots consume
// the same eligible players.
func TestCalculateStarterValueFillsFixedSlotsBeforeFlex(t *testing.T) {
	players := map[string]interface{}{
		"RB1": map[string]interface{}{"first_name": "Best", "last_name": "Back", "position": "RB"},
		"RB2": map[string]interface{}{"first_name": "Good", "last_name": "Back", "position": "RB"},
		"RB3": map[string]interface{}{"first_name": "Third", "last_name": "Back", "position": "RB"},
	}
	dynastyValues := map[string]DynastyValue{
		normalizeName("Best Back"):  {Value1QB: 5000},
		normalizeName("Good Back"):  {Value1QB: 4000},
		normalizeName("Third Back"): {Value1QB: 3000},
	}
	rosterPositions := []string{"RB", "FLEX", "BN"}
	rosterPlayers := []string{"RB1", "RB2", "RB3"}

	got := calculateStarterValue(rosterPlayers, players, dynastyValues, false, rosterPositions)

	want := 5000 + 4000 // RB slot + FLEX slot; third RB stays on bench, uncounted
	if got != want {
		t.Errorf("expected %d, got %d", want, got)
	}
}

// The Contending/Win Now/Rebuilding label must track starter value, not raw
// roster value -- a team that hoards bench value shouldn't rank as
// "Contending" over a team with a genuinely stronger starting lineup.
func TestCalculatePowerRankingsStrategyUsesStarterValueNotRosterValue(t *testing.T) {
	teamAges := []TeamAgeData{
		{
			TeamName:     "Bench Hoarder",
			RosterID:     1,
			AvgAge:       26.0,
			Wins:         5,
			Losses:       5,
			RosterValue:  50000, // inflated by bench depth
			StarterValue: 8000,  // weak actual lineup
		},
		{
			TeamName:     "Lean Contender",
			RosterID:     2,
			AvgAge:       26.0,
			Wins:         5,
			Losses:       5,
			RosterValue:  15000, // much lower total roster value
			StarterValue: 20000, // but a much stronger starting lineup
		},
	}

	rankings := calculatePowerRankings(teamAges)

	byName := map[string]PowerRanking{}
	for _, r := range rankings {
		byName[r.TeamName] = r
	}

	if byName["Lean Contender"].ValueRank != 1 {
		t.Errorf("expected Lean Contender to rank #1 by starter value, got #%d", byName["Lean Contender"].ValueRank)
	}
	if byName["Lean Contender"].Strategy != "Contending" {
		t.Errorf("expected Lean Contender to be Contending, got %q", byName["Lean Contender"].Strategy)
	}
	if byName["Bench Hoarder"].Strategy == "Contending" {
		t.Errorf("Bench Hoarder has a weak starting lineup and should not be labeled Contending")
	}
}

func TestCalculatePowerRankingsYoungTeamAlwaysRebuilding(t *testing.T) {
	teamAges := []TeamAgeData{
		{TeamName: "Young Studs", RosterID: 1, AvgAge: 22.0, StarterValue: 30000}, // best value in league, but very young
		{TeamName: "Vets", RosterID: 2, AvgAge: 29.0, StarterValue: 10000},
	}

	rankings := calculatePowerRankings(teamAges)
	for _, r := range rankings {
		if r.TeamName == "Young Studs" && r.Strategy != "Rebuilding" {
			t.Errorf("expected a very young roster to be Rebuilding regardless of value, got %q", r.Strategy)
		}
	}
}
