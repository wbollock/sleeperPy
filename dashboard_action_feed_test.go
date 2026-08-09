package main

import (
	"testing"
	"time"
)

// fakeActionFeedProvider is a hand-built LeagueProvider (no HTTP mocking
// needed) that returns one redraft league with an obvious start/sit swap:
// the starting RB is Tier 3, the bench RB is Tier 1.
type fakeActionFeedProvider struct{}

func (f *fakeActionFeedProvider) FetchUser(username string) (map[string]interface{}, error) {
	return map[string]interface{}{"user_id": "u1", "username": username}, nil
}

func (f *fakeActionFeedProvider) FetchUserLeagues(userID string, season int) ([]map[string]interface{}, error) {
	if season != time.Now().Year() {
		// Only return leagues for the "current" season fetch so the
		// dashboard's dedup logic doesn't double them up.
		return nil, nil
	}
	return []map[string]interface{}{
		{
			"league_id": "league1",
			"name":      "Test League",
			"season":    "2026",
			"scoring_settings": map[string]interface{}{
				"rec": 1.0,
			},
			"roster_positions": []interface{}{"QB", "RB", "WR", "TE", "BN", "BN"},
		},
	}, nil
}

func (f *fakeActionFeedProvider) FetchNFLState() (map[string]interface{}, error) {
	return map[string]interface{}{"week": 5.0}, nil
}

func (f *fakeActionFeedProvider) FetchLeagueRosters(leagueID string) ([]map[string]interface{}, error) {
	return []map[string]interface{}{
		{
			"owner_id": "u1",
			"starters": []interface{}{"QB1", "RB_STARTER", "WR1", "TE1"},
			"players":  []interface{}{"QB1", "RB_STARTER", "WR1", "TE1", "RB_BENCH"},
			"reserve":  []interface{}{},
		},
	}, nil
}

func (f *fakeActionFeedProvider) FetchLeagueMatchups(leagueID string, week int) ([]map[string]interface{}, error) {
	return []map[string]interface{}{
		{"roster_id": 1.0, "matchup_id": 1.0, "starters": []interface{}{"QB1", "RB_STARTER", "WR1", "TE1"}},
	}, nil
}

func (f *fakeActionFeedProvider) FetchLeagueUsers(leagueID string) ([]map[string]interface{}, error) {
	return nil, nil
}

func (f *fakeActionFeedProvider) FetchLeagueTradedPicks(leagueID string) ([]map[string]interface{}, error) {
	return nil, nil
}

func TestBuildDashboardPageAggregatesActionFeed(t *testing.T) {
	origProvider := appProvider
	origBorisTiers := fetchBorisTiers
	origPlayersCache := sleeperPlayersCache
	defer func() {
		appProvider = origProvider
		fetchBorisTiers = origBorisTiers
		sleeperPlayersCache = origPlayersCache
	}()

	appProvider = &fakeActionFeedProvider{}

	fetchBorisTiers = func(scoring string) map[string][][]string {
		return map[string][][]string{
			"RB": {
				{"Bench Runner"},        // Tier 1
				{},                      // Tier 2 (empty)
				{"Starter Runningback"}, // Tier 3
			},
			"QB":  {{"Starter Quarterback"}},
			"WR":  {{"Starter Wideout"}},
			"TE":  {{"Starter Tightend"}},
			"FLX": {},
		}
	}

	sleeperPlayersCache = &playersCache{
		data: map[string]interface{}{
			"QB1":        map[string]interface{}{"first_name": "Starter", "last_name": "Quarterback", "position": "QB", "age": 28.0},
			"RB_STARTER": map[string]interface{}{"first_name": "Starter", "last_name": "Runningback", "position": "RB", "age": 27.0},
			"RB_BENCH":   map[string]interface{}{"first_name": "Bench", "last_name": "Runner", "position": "RB", "age": 24.0},
			"WR1":        map[string]interface{}{"first_name": "Starter", "last_name": "Wideout", "position": "WR", "age": 25.0},
			"TE1":        map[string]interface{}{"first_name": "Starter", "last_name": "Tightend", "position": "TE", "age": 26.0},
		},
		timestamp: time.Now(),
		ttl:       time.Hour,
	}

	page, err := buildDashboardPage("testuser")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(page.ActionFeed) == 0 {
		t.Fatalf("expected at least one action in the feed, got none")
	}

	found := false
	for _, a := range page.ActionFeed {
		if a.Category != "swap" {
			continue
		}
		if a.LeagueID != "league1" || a.LeagueName != "Test League" {
			t.Fatalf("action missing league context: %+v", a)
		}
		found = true
	}
	if !found {
		t.Fatalf("expected a swap action for the obviously-better bench RB, got: %+v", page.ActionFeed)
	}

	if len(page.LeagueSummaries) != 1 || page.LeagueSummaries[0].ActionCount == 0 {
		t.Fatalf("expected LeagueSummary.ActionCount to be populated, got: %+v", page.LeagueSummaries)
	}
}
