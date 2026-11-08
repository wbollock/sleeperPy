package main

import (
	"testing"
	"time"
)

func TestFindChampionRosterID(t *testing.T) {
	cases := []struct {
		name    string
		bracket []map[string]interface{}
		want    int
	}{
		{
			name:    "no bracket",
			bracket: nil,
			want:    0,
		},
		{
			name: "season in progress, no finals decided",
			bracket: []map[string]interface{}{
				{"r": 1.0, "m": 1.0, "t1": 1.0, "t2": 2.0},
			},
			want: 0,
		},
		{
			name: "finals slot exists but not yet played",
			bracket: []map[string]interface{}{
				{"r": 2.0, "m": 3.0, "p": 1.0},
			},
			want: 0,
		},
		{
			name: "finals decided",
			bracket: []map[string]interface{}{
				{"r": 1.0, "m": 1.0, "t1": 1.0, "t2": 4.0, "w": 1.0, "l": 4.0},
				{"r": 2.0, "m": 3.0, "p": 1.0, "t1": 1.0, "t2": 3.0, "w": 3.0, "l": 1.0},
			},
			want: 3,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := findChampionRosterID(c.bracket)
			if got != c.want {
				t.Errorf("expected %d, got %d", c.want, got)
			}
		})
	}
}

func TestBuildChampionRecord(t *testing.T) {
	rosters := []map[string]interface{}{
		{"roster_id": 1.0, "owner_id": "userA"},
		{"roster_id": 3.0, "owner_id": "userB", "metadata": map[string]interface{}{"team_name": "Dynasty Destroyers"}},
	}
	userNames := map[string]string{"userA": "Alice", "userB": "Bob"}

	champ, ok := buildChampionRecord("2025", 3, rosters, userNames)
	if !ok {
		t.Fatal("expected champion to be found")
	}
	if champ.TeamName != "Dynasty Destroyers" {
		t.Errorf("expected team_name override, got %q", champ.TeamName)
	}
	if champ.OwnerName != "Bob" || champ.OwnerID != "userB" {
		t.Errorf("expected owner Bob/userB, got %q/%q", champ.OwnerName, champ.OwnerID)
	}

	champNoTeamName, ok := buildChampionRecord("2024", 1, rosters, userNames)
	if !ok {
		t.Fatal("expected champion to be found")
	}
	if champNoTeamName.TeamName != "Alice" {
		t.Errorf("expected team_name to fall back to owner display name, got %q", champNoTeamName.TeamName)
	}

	if _, ok := buildChampionRecord("2023", 99, rosters, userNames); ok {
		t.Error("expected no match for an unknown roster id")
	}
}

// fakeHistoryProvider simulates a 2-season previous_league_id chain:
// L2025 (current) -> L2024 -> "" (chain end). Counts calls per league so
// tests can assert the cache actually avoids refetching.
type fakeHistoryProvider struct {
	fetchLeagueCalls int
}

func (f *fakeHistoryProvider) FetchUser(username string) (map[string]interface{}, error) { return nil, nil }
func (f *fakeHistoryProvider) FetchUserLeagues(userID string, season int) ([]map[string]interface{}, error) {
	return nil, nil
}
func (f *fakeHistoryProvider) FetchNFLState() (map[string]interface{}, error) { return nil, nil }
func (f *fakeHistoryProvider) FetchLeagueMatchups(leagueID string, week int) ([]map[string]interface{}, error) {
	return nil, nil
}
func (f *fakeHistoryProvider) FetchLeagueTradedPicks(leagueID string) ([]map[string]interface{}, error) {
	return nil, nil
}

func (f *fakeHistoryProvider) FetchLeague(leagueID string) (map[string]interface{}, error) {
	f.fetchLeagueCalls++
	switch leagueID {
	case "L2025":
		return map[string]interface{}{"league_id": "L2025", "season": "2025", "previous_league_id": "L2024"}, nil
	case "L2024":
		return map[string]interface{}{"league_id": "L2024", "season": "2024", "previous_league_id": ""}, nil
	}
	return nil, nil
}

func (f *fakeHistoryProvider) FetchLeagueWinnersBracket(leagueID string) ([]map[string]interface{}, error) {
	switch leagueID {
	case "L2025":
		return []map[string]interface{}{
			{"r": 2.0, "p": 1.0, "w": 1.0}, // roster 1 (userA) wins 2025
		}, nil
	case "L2024":
		return []map[string]interface{}{
			{"r": 2.0, "p": 1.0, "w": 2.0}, // roster 2 (userB) wins 2024
		}, nil
	}
	return nil, nil
}

func (f *fakeHistoryProvider) FetchLeagueRosters(leagueID string) ([]map[string]interface{}, error) {
	return []map[string]interface{}{
		{"roster_id": 1.0, "owner_id": "userA"},
		{"roster_id": 2.0, "owner_id": "userB"},
	}, nil
}

func (f *fakeHistoryProvider) FetchLeagueUsers(leagueID string) ([]map[string]interface{}, error) {
	return []map[string]interface{}{
		{"user_id": "userA", "display_name": "Alice"},
		{"user_id": "userB", "display_name": "Bob"},
	}, nil
}

func TestFetchLeagueHistoryWalksChainMostRecentFirst(t *testing.T) {
	origProvider := appProvider
	origCache := leagueHistoryCache
	defer func() {
		appProvider = origProvider
		leagueHistoryCache = origCache
	}()

	fake := &fakeHistoryProvider{}
	appProvider = fake
	leagueHistoryCache = &leagueHistoryCacheT{
		data:      make(map[string][]LeagueChampion),
		timestamp: make(map[string]time.Time),
		ttl:       time.Hour,
	}

	history := fetchLeagueHistory("L2025", "userA")

	if len(history) != 2 {
		t.Fatalf("expected 2 seasons of history, got %d: %+v", len(history), history)
	}
	if history[0].Season != "2025" || !history[0].IsUserTeam {
		t.Errorf("expected 2025 champion to be the viewer's team, got %+v", history[0])
	}
	if history[1].Season != "2024" || history[1].IsUserTeam {
		t.Errorf("expected 2024 champion to not be the viewer's team, got %+v", history[1])
	}

	// A second call (even for a different viewer) should be served from
	// cache -- FetchLeague should not be called again.
	callsAfterFirst := fake.fetchLeagueCalls
	_ = fetchLeagueHistory("L2025", "userB")
	if fake.fetchLeagueCalls != callsAfterFirst {
		t.Errorf("expected cached result to avoid refetching, calls went from %d to %d", callsAfterFirst, fake.fetchLeagueCalls)
	}

	// IsUserTeam must be recomputed per viewer even when served from cache.
	historyForB := fetchLeagueHistory("L2025", "userB")
	if historyForB[0].IsUserTeam {
		t.Error("expected 2025 champion to not be userB's team")
	}
	if !historyForB[1].IsUserTeam {
		t.Error("expected 2024 champion to be userB's team")
	}
}
