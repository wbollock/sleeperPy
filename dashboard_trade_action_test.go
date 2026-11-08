package main

import "testing"

// fakeLeagueUsersProvider is a minimal LeagueProvider where only
// FetchLeagueUsers matters -- buildDashboardTradeAction doesn't call any of
// the other methods.
type fakeLeagueUsersProvider struct{}

func (f *fakeLeagueUsersProvider) FetchUser(username string) (map[string]interface{}, error) {
	return nil, nil
}
func (f *fakeLeagueUsersProvider) FetchUserLeagues(userID string, season int) ([]map[string]interface{}, error) {
	return nil, nil
}
func (f *fakeLeagueUsersProvider) FetchNFLState() (map[string]interface{}, error) { return nil, nil }
func (f *fakeLeagueUsersProvider) FetchLeagueRosters(leagueID string) ([]map[string]interface{}, error) {
	return nil, nil
}
func (f *fakeLeagueUsersProvider) FetchLeagueMatchups(leagueID string, week int) ([]map[string]interface{}, error) {
	return nil, nil
}
func (f *fakeLeagueUsersProvider) FetchLeagueUsers(leagueID string) ([]map[string]interface{}, error) {
	return []map[string]interface{}{
		{"user_id": "owner2", "display_name": "Rival Manager"},
	}, nil
}
func (f *fakeLeagueUsersProvider) FetchLeagueTradedPicks(leagueID string) ([]map[string]interface{}, error) {
	return nil, nil
}
func (f *fakeLeagueUsersProvider) FetchLeague(leagueID string) (map[string]interface{}, error) {
	return nil, nil
}
func (f *fakeLeagueUsersProvider) FetchLeagueWinnersBracket(leagueID string) ([]map[string]interface{}, error) {
	return nil, nil
}

func TestBuildDashboardTradeAction(t *testing.T) {
	origProvider := appProvider
	defer func() { appProvider = origProvider }()
	appProvider = &fakeLeagueUsersProvider{}

	userRoster := map[string]interface{}{
		"roster_id": 1.0,
		"owner_id":  "owner1",
		"players":   []interface{}{"RB1", "RB2", "RB3", "QB1"},
	}
	targetRoster := map[string]interface{}{
		"roster_id": 2.0,
		"owner_id":  "owner2",
		"players":   []interface{}{"QB2", "QB3", "RB4"},
	}
	rosters := []map[string]interface{}{userRoster, targetRoster}

	players := map[string]interface{}{
		"RB1": map[string]interface{}{"first_name": "R", "last_name": "B1", "position": "RB", "age": 24.0},
		"RB2": map[string]interface{}{"first_name": "R", "last_name": "B2", "position": "RB", "age": 24.0},
		"RB3": map[string]interface{}{"first_name": "R", "last_name": "B3", "position": "RB", "age": 24.0},
		"QB1": map[string]interface{}{"first_name": "Q", "last_name": "B1", "position": "QB", "age": 28.0},
		"QB2": map[string]interface{}{"first_name": "Q", "last_name": "B2", "position": "QB", "age": 28.0},
		"QB3": map[string]interface{}{"first_name": "Q", "last_name": "B3", "position": "QB", "age": 28.0},
		"RB4": map[string]interface{}{"first_name": "R", "last_name": "B4", "position": "RB", "age": 24.0},
	}

	dynastyValues := map[string]DynastyValue{
		normalizeName("R B1"): {Value1QB: 1000}, normalizeName("R B2"): {Value1QB: 1000}, normalizeName("R B3"): {Value1QB: 1000},
		normalizeName("Q B1"): {Value1QB: 200},
		normalizeName("Q B2"): {Value1QB: 1000}, normalizeName("Q B3"): {Value1QB: 1000},
		normalizeName("R B4"): {Value1QB: 200},
	}

	action := buildDashboardTradeAction("league1", "My League", userRoster, rosters, players, dynastyValues, false)

	if action == nil {
		t.Fatalf("expected a trade action for complementary RB-surplus/QB-deficit rosters, got nil")
	}
	if action.Category != "trade" {
		t.Errorf("expected category 'trade', got %q", action.Category)
	}
	if action.LeagueID != "league1" || action.LeagueName != "My League" {
		t.Errorf("expected league context to be set, got: %+v", action)
	}
	if action.Description == "" || action.Impact == "" {
		t.Errorf("expected non-empty description/impact, got: %+v", action)
	}
	t.Logf("generated trade action: %s | %s", action.Description, action.Impact)
}

func TestBuildDashboardTradeActionNoTargetsReturnsNil(t *testing.T) {
	origProvider := appProvider
	defer func() { appProvider = origProvider }()
	appProvider = &fakeLeagueUsersProvider{}

	// A perfectly balanced roster shouldn't produce a trade action.
	userRoster := map[string]interface{}{
		"roster_id": 1.0,
		"owner_id":  "owner1",
		"players":   []interface{}{"RB1", "WR1", "TE1", "QB1"},
	}
	rosters := []map[string]interface{}{userRoster}
	players := map[string]interface{}{
		"RB1": map[string]interface{}{"first_name": "R", "last_name": "B1", "position": "RB"},
		"WR1": map[string]interface{}{"first_name": "W", "last_name": "R1", "position": "WR"},
		"TE1": map[string]interface{}{"first_name": "T", "last_name": "E1", "position": "TE"},
		"QB1": map[string]interface{}{"first_name": "Q", "last_name": "B1", "position": "QB"},
	}
	dynastyValues := map[string]DynastyValue{
		normalizeName("R B1"): {Value1QB: 500}, normalizeName("W R1"): {Value1QB: 500},
		normalizeName("T E1"): {Value1QB: 500}, normalizeName("Q B1"): {Value1QB: 500},
	}

	action := buildDashboardTradeAction("league1", "My League", userRoster, rosters, players, dynastyValues, false)
	if action != nil {
		t.Errorf("expected nil for a balanced roster with no other teams, got: %+v", action)
	}
}
