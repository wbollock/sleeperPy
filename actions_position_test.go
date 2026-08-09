package main

import "testing"

// TestFindStarterSwapsIgnoresCrossPositionTiers reproduces the exact bug
// reported live: a Tier 4 RB starter (a clear starter -- RB is just deep at
// the top this season) got suggested for a swap in favor of a Tier 3 bench
// WR, purely because 4 > 3. Boris Chen tiers only rank within a position;
// comparing raw tier numbers across positions is meaningless.
func TestFindStarterSwapsIgnoresCrossPositionTiers(t *testing.T) {
	league := LeagueData{
		HasMatchups: true,
		Starters: []PlayerRow{
			{Pos: "RB", Name: "Kyren Williams", Tier: 4},
		},
		Bench: []PlayerRow{
			{Pos: "WR", Name: "Jakobi Meyers", Tier: 3},
		},
	}

	actions := findStarterSwaps(league, "2026-W1")
	for _, a := range actions {
		if a.Description == "Start Jakobi Meyers over Kyren Williams" {
			t.Fatalf("cross-position swap should never be suggested, got: %+v", actions)
		}
	}
	if len(actions) != 0 {
		t.Fatalf("expected no actions for a cross-position-only scenario, got: %+v", actions)
	}
}

func TestFindStarterSwapsSamePositionStillWorks(t *testing.T) {
	league := LeagueData{
		HasMatchups: true,
		Starters: []PlayerRow{
			{Pos: "RB", Name: "Worse Starter", Tier: 4},
		},
		Bench: []PlayerRow{
			{Pos: "RB", Name: "Better Bench", Tier: 1},
		},
	}

	actions := findStarterSwaps(league, "2026-W1")
	if len(actions) != 1 {
		t.Fatalf("expected 1 same-position swap suggestion, got %d: %+v", len(actions), actions)
	}
	if actions[0].Description != "Start Better Bench over Worse Starter" {
		t.Errorf("unexpected description: %q", actions[0].Description)
	}
}

func TestFindStarterSwapsFlexSlotUsesFlexTier(t *testing.T) {
	league := LeagueData{
		HasMatchups: true,
		Starters: []PlayerRow{
			// A FLEX starter's Tier is already FLX-based by the time it
			// reaches this function (buildTieredRoster's FLEX re-ranking).
			{Pos: "FLEX", Name: "Mediocre Flex Starter", Tier: 5, IsFlex: true},
		},
		Bench: []PlayerRow{
			// Positional Tier looks bad (8), but FlexTier (the comparable
			// scale for a FLEX slot) is excellent (1).
			{Pos: "TE", Name: "Great Flex Option", Tier: 8, FlexTier: 1},
			// Flex-ineligible position (no FlexTier) should never be
			// suggested for a FLEX slot even with a great positional tier.
			{Pos: "QB", Name: "Backup QB", Tier: 1, FlexTier: 0},
		},
	}

	actions := findStarterSwaps(league, "2026-W1")
	if len(actions) != 1 {
		t.Fatalf("expected exactly 1 FLEX-tier-based swap, got %d: %+v", len(actions), actions)
	}
	if actions[0].Description != "Start Great Flex Option over Mediocre Flex Starter" {
		t.Errorf("unexpected description: %q", actions[0].Description)
	}
}

func TestFindWaiverTargetsFlexSlotUsesFlexTier(t *testing.T) {
	league := LeagueData{
		HasMatchups: true,
		Starters: []PlayerRow{
			{Pos: "FLEX", Name: "Mediocre Flex Starter", Tier: 5, IsFlex: true},
		},
		TopFreeAgentsByValue: []PlayerRow{
			{Pos: "RB", Name: "Hot Waiver Add", Tier: 6, FlexTier: 1},
		},
	}

	actions := findWaiverTargets(league, "2026-W1")
	if len(actions) != 1 {
		t.Fatalf("expected 1 FLEX-tier-based waiver suggestion, got %d: %+v", len(actions), actions)
	}
}

func TestFindWaiverTargetsRequiresExactPositionOutsideFlex(t *testing.T) {
	league := LeagueData{
		HasMatchups: true,
		Starters: []PlayerRow{
			{Pos: "RB", Name: "Literal RB Starter", Tier: 4},
		},
		TopFreeAgentsByValue: []PlayerRow{
			{Pos: "WR", Name: "Unrelated WR", Tier: 3},
		},
	}

	actions := findWaiverTargets(league, "2026-W1")
	if len(actions) != 0 {
		t.Fatalf("cross-position waiver suggestion should never fire outside a FLEX slot, got: %+v", actions)
	}
}
