package main

import "testing"

func TestFindCutCandidates(t *testing.T) {
	bench := []PlayerRow{
		{Name: "Low Value Bench", Pos: "WR", DynastyValue: 50},
		{Name: "Trending Sleeper", Pos: "RB", DynastyValue: 80},
		{Name: "Good Enough To Start", Pos: "TE", DynastyValue: 30, ShouldSwapIn: true},
		{Name: "Valuable Stash", Pos: "WR", DynastyValue: 900},
		{Name: "No Value Data", Pos: "QB", DynastyValue: 0},
	}

	trending := map[string]int{
		normalizeName("Trending Sleeper"): 500, // heavily added elsewhere
	}

	got := findCutCandidates(bench, trending)

	names := make(map[string]bool)
	for _, c := range got {
		names[c.Name] = true
	}

	if !names["Low Value Bench"] {
		t.Errorf("expected Low Value Bench to be a cut candidate, got: %+v", got)
	}
	if names["Trending Sleeper"] {
		t.Errorf("Trending Sleeper is trending up elsewhere, should NOT be a cut candidate: %+v", got)
	}
	if names["Good Enough To Start"] {
		t.Errorf("ShouldSwapIn player should NOT be a cut candidate: %+v", got)
	}
	if names["Valuable Stash"] {
		t.Errorf("high dynasty value player should NOT be a cut candidate: %+v", got)
	}
	if names["No Value Data"] {
		t.Errorf("player with zero/no dynasty value data should NOT be a cut candidate: %+v", got)
	}
}

func TestFindCutCandidatesNilTrendingMap(t *testing.T) {
	bench := []PlayerRow{{Name: "Low Value Bench", Pos: "WR", DynastyValue: 50}}

	got := findCutCandidates(bench, nil)
	if len(got) != 1 {
		t.Fatalf("expected 1 candidate when trending data is unavailable (nil), got %d", len(got))
	}
}

func TestBuildTrendingByName(t *testing.T) {
	origFetch := fetchTrendingPlayers
	defer func() { fetchTrendingPlayers = origFetch }()

	fetchTrendingPlayers = func(trendType string) map[string]int {
		if trendType != "add" {
			t.Fatalf("expected trendType=add, got %q", trendType)
		}
		return map[string]int{"P1": 1200}
	}

	players := map[string]interface{}{
		"P1": map[string]interface{}{"first_name": "Waiver", "last_name": "Wonder", "position": "WR"},
	}

	byName := buildTrendingByName(players)
	if byName[normalizeName("Waiver Wonder")] != 1200 {
		t.Fatalf("expected trending count 1200 for Waiver Wonder, got: %+v", byName)
	}
}
