package main

import "testing"

func TestMergeActionsAcrossLeagues(t *testing.T) {
	input := []Action{
		{Category: "swap", Description: "Start Bench Guy over Starter Guy", LeagueName: "League A"},
		{Category: "swap", Description: "Start Bench Guy over Starter Guy", LeagueName: "League B"},
		{Category: "swap", Description: "Start Someone Else over Another Guy", LeagueName: "League A"},
	}

	got := mergeActionsAcrossLeagues(input)

	if len(got) != 2 {
		t.Fatalf("expected 2 merged actions, got %d: %+v", len(got), got)
	}

	if got[0].LeagueName != "League A, League B" {
		t.Errorf("expected merged league names 'League A, League B', got %q", got[0].LeagueName)
	}
	if got[1].LeagueName != "League A" {
		t.Errorf("expected unmerged single-league action to keep its league name, got %q", got[1].LeagueName)
	}
}

func TestMergeActionsAcrossLeaguesDifferentCategorySameDescription(t *testing.T) {
	// Same description text but different category should NOT merge --
	// they're different kinds of recommendations that happen to read the same.
	input := []Action{
		{Category: "swap", Description: "Same text", LeagueName: "League A"},
		{Category: "waiver", Description: "Same text", LeagueName: "League B"},
	}

	got := mergeActionsAcrossLeagues(input)
	if len(got) != 2 {
		t.Fatalf("expected different categories to stay separate, got %d: %+v", len(got), got)
	}
}

func TestMergeActionsAcrossLeaguesEmpty(t *testing.T) {
	got := mergeActionsAcrossLeagues(nil)
	if len(got) != 0 {
		t.Fatalf("expected empty input to produce empty output, got %+v", got)
	}
}
