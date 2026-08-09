package main

import "testing"

func TestFindByeAlertsSuggestsHealthyReplacement(t *testing.T) {
	league := LeagueData{
		HasMatchups: true,
		Starters: []PlayerRow{
			{Pos: "RB", Name: "On Bye Guy", Tier: 2, OnBye: true},
		},
		Bench: []PlayerRow{
			{Pos: "RB", Name: "Bench Backup", Tier: 5, OnBye: false},
		},
	}

	actions := findByeAlerts(league, "2026-W1")
	if len(actions) != 1 {
		t.Fatalf("expected 1 bye action, got %d: %+v", len(actions), actions)
	}
	want := "Start Bench Backup over On Bye Guy (bye)"
	if actions[0].Description != want {
		t.Errorf("expected description %q, got %q", want, actions[0].Description)
	}
	if actions[0].Category != "bye" {
		t.Errorf("expected category 'bye', got %q", actions[0].Category)
	}
}

func TestFindByeAlertsNoReplacementAvailable(t *testing.T) {
	league := LeagueData{
		HasMatchups: true,
		Starters: []PlayerRow{
			{Pos: "RB", Name: "On Bye Guy", Tier: 2, OnBye: true},
		},
		Bench: []PlayerRow{
			{Pos: "WR", Name: "Wrong Position", Tier: 1, OnBye: false},
		},
	}

	actions := findByeAlerts(league, "2026-W1")
	if len(actions) != 1 {
		t.Fatalf("expected 1 bye action, got %d", len(actions))
	}
	want := "On Bye Guy is on bye this week"
	if actions[0].Description != want {
		t.Errorf("expected description %q, got %q", want, actions[0].Description)
	}
}

func TestFindByeAlertsIgnoresHealthyStarters(t *testing.T) {
	league := LeagueData{
		HasMatchups: true,
		Starters: []PlayerRow{
			{Pos: "RB", Name: "Playing This Week", Tier: 2, OnBye: false},
		},
	}

	actions := findByeAlerts(league, "2026-W1")
	if len(actions) != 0 {
		t.Fatalf("expected no bye actions for a starter not on bye, got %+v", actions)
	}
}

func TestFindInjuryAlertsFiresForOutAndDoubtful(t *testing.T) {
	league := LeagueData{
		Starters: []PlayerRow{
			{Pos: "WR", Name: "Hurt Guy", InjuryStatus: "Out"},
			{Pos: "RB", Name: "Iffy Guy", InjuryStatus: "Doubtful"},
			{Pos: "TE", Name: "Fine Guy", InjuryStatus: "Questionable"},
			{Pos: "QB", Name: "Healthy Guy", InjuryStatus: ""},
		},
	}

	actions := findInjuryAlerts(league, "2026-W1")
	if len(actions) != 2 {
		t.Fatalf("expected 2 injury actions (Out + Doubtful only), got %d: %+v", len(actions), actions)
	}
}

func TestFindHandcuffAlertsMatchesNextDepthChartSlot(t *testing.T) {
	league := LeagueData{
		Starters: []PlayerRow{
			{Pos: "RB", Name: "Starter RB", Team: "SF", InjuryStatus: "Out", DepthChartOrder: 1},
		},
		TopFreeAgentsByValue: []PlayerRow{
			{Pos: "RB", Name: "Handcuff RB", Team: "SF", DepthChartOrder: 2},
			{Pos: "RB", Name: "Deep Backup", Team: "SF", DepthChartOrder: 3},
			{Pos: "RB", Name: "Different Team", Team: "KC", DepthChartOrder: 2},
		},
	}

	actions := findHandcuffAlerts(league, "2026-W1")
	if len(actions) != 1 {
		t.Fatalf("expected exactly 1 handcuff action, got %d: %+v", len(actions), actions)
	}
	if actions[0].Description != "Handcuff RB now next in line behind injured Starter RB" {
		t.Errorf("unexpected description: %q", actions[0].Description)
	}
}

func TestFindHandcuffAlertsIgnoresHealthyStarters(t *testing.T) {
	league := LeagueData{
		Starters: []PlayerRow{
			{Pos: "RB", Name: "Healthy Starter", Team: "SF", DepthChartOrder: 1},
		},
		TopFreeAgentsByValue: []PlayerRow{
			{Pos: "RB", Name: "Handcuff RB", Team: "SF", DepthChartOrder: 2},
		},
	}

	actions := findHandcuffAlerts(league, "2026-W1")
	if len(actions) != 0 {
		t.Fatalf("expected no handcuff actions for a healthy starter, got %+v", actions)
	}
}

func TestFindHandcuffAlertsSkipsAmbiguousFlexSlot(t *testing.T) {
	league := LeagueData{
		Starters: []PlayerRow{
			{Pos: "FLEX", Name: "Flex Starter", Team: "SF", InjuryStatus: "Out", DepthChartOrder: 1},
		},
		TopFreeAgentsByValue: []PlayerRow{
			{Pos: "RB", Name: "Handcuff RB", Team: "SF", DepthChartOrder: 2},
		},
	}

	actions := findHandcuffAlerts(league, "2026-W1")
	if len(actions) != 0 {
		t.Fatalf("expected no handcuff actions when real position is unknown (FLEX slot), got %+v", actions)
	}
}
