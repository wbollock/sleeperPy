package main

import (
	"context"
	"strings"
	"testing"
)

func TestParseESPNLeaguePayload(t *testing.T) {
	payload := espnLeaguePayload{
		ID:       12345,
		SeasonID: 2026,
	}
	payload.Settings.Name = "Test ESPN League"
	payload.Members = []struct {
		ID          string "json:\"id\""
		DisplayName string "json:\"displayName\""
		FirstName   string "json:\"firstName\""
		LastName    string "json:\"lastName\""
	}{
		{ID: "owner-1", DisplayName: "Alice"},
	}
	payload.Teams = []struct {
		ID       int      "json:\"id\""
		Location string   "json:\"location\""
		Nickname string   "json:\"nickname\""
		Abbrev   string   "json:\"abbrev\""
		Owners   []string "json:\"owners\""
		Roster   struct {
			Entries []struct {
				PlayerPoolEntry struct {
					Player struct {
						FullName          string "json:\"fullName\""
						DefaultPositionID int    "json:\"defaultPositionId\""
					} "json:\"player\""
				} "json:\"playerPoolEntry\""
			} "json:\"entries\""
		} "json:\"roster\""
	}{
		{
			ID:       1,
			Location: "Alpha",
			Nickname: "Squad",
			Owners:   []string{"owner-1"},
		},
	}
	payload.Teams[0].Roster.Entries = []struct {
		PlayerPoolEntry struct {
			Player struct {
				FullName          string "json:\"fullName\""
				DefaultPositionID int    "json:\"defaultPositionId\""
			} "json:\"player\""
		} "json:\"playerPoolEntry\""
	}{
		{
			PlayerPoolEntry: struct {
				Player struct {
					FullName          string "json:\"fullName\""
					DefaultPositionID int    "json:\"defaultPositionId\""
				} "json:\"player\""
			}{
				Player: struct {
					FullName          string "json:\"fullName\""
					DefaultPositionID int    "json:\"defaultPositionId\""
				}{
					FullName:          "Josh Allen",
					DefaultPositionID: 1,
				},
			},
		},
	}

	league := parseESPNLeaguePayload(payload)
	if league.Provider != "espn" {
		t.Fatalf("expected espn provider, got %q", league.Provider)
	}
	if league.LeagueName != "Test ESPN League" {
		t.Fatalf("unexpected league name: %q", league.LeagueName)
	}
	if len(league.Teams) != 1 {
		t.Fatalf("expected 1 team, got %d", len(league.Teams))
	}
	if league.Teams[0].Owner != "Alice" {
		t.Fatalf("expected owner Alice, got %q", league.Teams[0].Owner)
	}
	if len(league.Teams[0].Roster) != 1 || league.Teams[0].Roster[0].Position != "QB" {
		t.Fatalf("expected parsed QB roster entry, got %#v", league.Teams[0].Roster)
	}
}

func TestYahooImporterReturnsScaffoldError(t *testing.T) {
	y := &yahooImporter{}
	_, err := y.ImportLeague(context.Background(), ImportOptions{LeagueID: "1"})
	if err == nil {
		t.Fatalf("expected yahoo scaffold error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "scaffold") {
		t.Fatalf("expected scaffold message, got %v", err)
	}
}
