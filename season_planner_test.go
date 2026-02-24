package main

import "testing"

func TestParseWinProbPct(t *testing.T) {
	if got := parseWinProbPct("62% You"); got != 62 {
		t.Fatalf("expected 62, got %d", got)
	}
	if got := parseWinProbPct(""); got != 0 {
		t.Fatalf("expected 0 for empty, got %d", got)
	}
	if got := parseWinProbPct("bad"); got != 0 {
		t.Fatalf("expected 0 for invalid, got %d", got)
	}
}

func TestBuildScheduleDifficultyProfile(t *testing.T) {
	league := LeagueData{
		HasMatchups: true,
		LeagueSize:  12,
		WinProb:     "35% You",
		PowerRankings: []PowerRanking{
			{IsUserTeam: true, StandingRank: 2, ValueRank: 9},
		},
	}
	diff, _ := buildScheduleDifficultyProfile(league, "Contending")
	if diff != "Hard" {
		t.Fatalf("expected Hard difficulty, got %s", diff)
	}

	off, _ := buildScheduleDifficultyProfile(LeagueData{HasMatchups: false}, "Build")
	if off != "Offseason" {
		t.Fatalf("expected Offseason difficulty, got %s", off)
	}
}
