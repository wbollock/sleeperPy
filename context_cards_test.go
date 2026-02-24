package main

import "testing"

func TestBuildContextCardsUsesYoungestFirstAgeRank(t *testing.T) {
	league := LeagueData{
		IsDynasty:  true,
		LeagueSize: 3,
		TeamAges: []TeamAgeData{
			{TeamName: "Old Team", AvgAge: 29.0, IsUserTeam: false},
			{TeamName: "User Team", AvgAge: 24.2, IsUserTeam: true},
			{TeamName: "Mid Team", AvgAge: 26.5, IsUserTeam: false},
		},
		PowerRankings: []PowerRanking{
			{IsUserTeam: true, ValueRank: 2},
		},
	}

	cards := buildContextCards(league, 10000, 24.2)
	found := false
	for _, c := range cards {
		if c.Title == "Roster Age" {
			found = true
			if c.Value != "24.2 yrs (#1st)" {
				t.Fatalf("expected youngest rank in value, got %q", c.Value)
			}
			if c.Trend != "Youngest" {
				t.Fatalf("expected Youngest trend, got %q", c.Trend)
			}
		}
	}
	if !found {
		t.Fatal("expected Roster Age card")
	}
}

func TestBuildContextCardsCountsAcquiredFirstRoundPicks(t *testing.T) {
	league := LeagueData{
		IsDynasty:  true,
		LeagueSize: 12,
		DraftPicks: []DraftPick{
			{Year: 2026, Round: 1, IsYours: false}, // acquired 1st
			{Year: 2027, Round: 1, IsYours: true},  // original 1st
			{Year: 2026, Round: 2, IsYours: true},
		},
		TeamAges: []TeamAgeData{
			{TeamName: "User Team", AvgAge: 26.0, IsUserTeam: true},
		},
		PowerRankings: []PowerRanking{
			{IsUserTeam: true, ValueRank: 4},
		},
	}

	cards := buildContextCards(league, 8000, 26.0)
	found := false
	for _, c := range cards {
		if c.Title == "Draft Capital" {
			found = true
			if c.Value != "2 1sts" {
				t.Fatalf("expected acquired+original 1sts counted, got %q", c.Value)
			}
		}
	}
	if !found {
		t.Fatal("expected Draft Capital card")
	}
}
