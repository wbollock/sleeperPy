package main

import "testing"

func TestBuildDraftPicksOriginalOnly(t *testing.T) {
	rosters := []map[string]interface{}{
		{"roster_id": float64(1)},
		{"roster_id": float64(2)},
		{"roster_id": float64(3)},
	}
	rosterOwners := map[int]string{
		1: "TeamA",
		2: "TeamB",
		3: "TeamC",
	}

	picks := buildDraftPicks(nil, rosters, rosterOwners, 1, 2, 2026, nil)
	if len(picks) != 3 {
		t.Fatalf("expected 3 picks (2026-2028), got %d", len(picks))
	}
	for _, pick := range picks {
		if pick.OriginalName != "" {
			t.Fatalf("expected original pick with empty OriginalName, got %q", pick.OriginalName)
		}
		if pick.RosterID != 2 || !pick.IsYours {
			t.Fatalf("expected pick owned by roster 2, got roster %d isYours=%v", pick.RosterID, pick.IsYours)
		}
	}
}

func TestBuildDraftPicksAcquiredPick(t *testing.T) {
	rosters := []map[string]interface{}{
		{"roster_id": float64(1)},
		{"roster_id": float64(2)},
		{"roster_id": float64(3)},
	}
	rosterOwners := map[int]string{
		1: "TeamA",
		2: "TeamB",
		3: "TeamC",
	}
	traded := []map[string]interface{}{
		{"season": "2026", "round": float64(1), "roster_id": float64(1), "owner_id": float64(2), "previous_owner_id": float64(1)},
	}

	picks := buildDraftPicks(traded, rosters, rosterOwners, 1, 2, 2026, nil)
	foundFromA := false
	foundOwn2026 := false
	for _, pick := range picks {
		if pick.Year == 2026 && pick.Round == 1 && pick.OriginalName == "TeamA" {
			foundFromA = true
		}
		if pick.Year == 2026 && pick.Round == 1 && pick.OriginalName == "" {
			foundOwn2026 = true
		}
	}
	if !foundFromA {
		t.Fatalf("expected acquired 2026 R1 from TeamA")
	}
	if !foundOwn2026 {
		t.Fatalf("expected to retain own 2026 R1 pick as separate entry")
	}
}

func TestBuildDraftPicksTradedAway(t *testing.T) {
	rosters := []map[string]interface{}{
		{"roster_id": float64(1)},
		{"roster_id": float64(2)},
		{"roster_id": float64(3)},
	}
	rosterOwners := map[int]string{
		1: "TeamA",
		2: "TeamB",
		3: "TeamC",
	}
	traded := []map[string]interface{}{
		{"season": "2026", "round": float64(1), "roster_id": float64(2), "owner_id": float64(3), "previous_owner_id": float64(2)},
	}

	picks := buildDraftPicks(traded, rosters, rosterOwners, 1, 2, 2026, nil)
	for _, pick := range picks {
		if pick.Year == 2026 && pick.Round == 1 && pick.OriginalName == "" {
			t.Fatalf("did not expect user's 2026 R1 original pick after trade away")
		}
	}
}

func TestBuildDraftPicksMultiHop(t *testing.T) {
	rosters := []map[string]interface{}{
		{"roster_id": float64(1)},
		{"roster_id": float64(2)},
		{"roster_id": float64(3)},
	}
	rosterOwners := map[int]string{
		1: "TeamA",
		2: "TeamB",
		3: "TeamC",
	}
	traded := []map[string]interface{}{
		{"season": "2027", "round": float64(2), "roster_id": float64(1), "owner_id": float64(2), "previous_owner_id": float64(3)},
	}

	picks := buildDraftPicks(traded, rosters, rosterOwners, 2, 2, 2026, nil)
	foundFromA := false
	for _, pick := range picks {
		if pick.Year == 2027 && pick.Round == 2 && pick.OriginalName == "TeamA" {
			foundFromA = true
		}
	}
	if !foundFromA {
		t.Fatalf("expected multi-hop pick to show from TeamA")
	}
}
