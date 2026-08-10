package main

import "testing"

func TestResolveTaxiSettings(t *testing.T) {
	cases := []struct {
		name        string
		league      map[string]interface{}
		expectSlots int
		expectYears int
	}{
		{
			name:        "no settings at all",
			league:      map[string]interface{}{},
			expectSlots: 0,
			expectYears: 1,
		},
		{
			name: "taxi slots configured, years unset defaults to 1",
			league: map[string]interface{}{
				"settings": map[string]interface{}{"taxi_slots": 4.0},
			},
			expectSlots: 4,
			expectYears: 1,
		},
		{
			name: "explicit taxi_years overrides default",
			league: map[string]interface{}{
				"settings": map[string]interface{}{"taxi_slots": 4.0, "taxi_years": 2.0},
			},
			expectSlots: 4,
			expectYears: 2,
		},
		{
			name: "no taxi squad at all",
			league: map[string]interface{}{
				"settings": map[string]interface{}{"taxi_slots": 0.0},
			},
			expectSlots: 0,
			expectYears: 1,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			slots, years := resolveTaxiSettings(c.league)
			if slots != c.expectSlots {
				t.Errorf("expected slots=%d, got %d", c.expectSlots, slots)
			}
			if years != c.expectYears {
				t.Errorf("expected years=%d, got %d", c.expectYears, years)
			}
		})
	}
}

func TestBuildTaxiSquadNameSet(t *testing.T) {
	players := map[string]interface{}{
		"T1": map[string]interface{}{"first_name": "Taxi", "last_name": "Rookie"},
		"T2": map[string]interface{}{"first_name": "Another", "last_name": "Stash"},
	}

	set := buildTaxiSquadNameSet([]string{"T1", "T2"}, players)
	if !set[normalizeName("Taxi Rookie")] || !set[normalizeName("Another Stash")] {
		t.Errorf("expected both taxi players in the name set, got %+v", set)
	}

	if buildTaxiSquadNameSet(nil, players) != nil {
		t.Errorf("expected nil set for no taxi IDs")
	}
}

func TestEnrichRowsWithTaxiSquad(t *testing.T) {
	rows := []PlayerRow{{Name: "Taxi Rookie"}, {Name: "Active Bench Guy"}}
	taxiNames := map[string]bool{normalizeName("Taxi Rookie"): true}

	enrichRowsWithTaxiSquad(rows, taxiNames)

	if !rows[0].IsTaxiSquad {
		t.Errorf("expected Taxi Rookie flagged as taxi squad")
	}
	if rows[1].IsTaxiSquad {
		t.Errorf("expected Active Bench Guy to NOT be flagged as taxi squad")
	}
}

func TestFindLowestValueRosterPlayers(t *testing.T) {
	starters := []PlayerRow{
		{Pos: "QB", Name: "Starter QB", DynastyValue: 8000},
	}
	bench := []PlayerRow{
		{Pos: "WR", Name: "Weak WR", DynastyValue: 500},
		{Pos: "RB", Name: "Weak RB", DynastyValue: 300},
		{Pos: "TE", Name: "Zero Value TE", DynastyValue: 0},
	}
	taxi := []PlayerRow{
		{Pos: "WR", Name: "Stashed Rookie", DynastyValue: 100, IsTaxiSquad: true},
	}
	freeAgents := []PlayerRow{
		{Pos: "RB", Name: "Best Available RB", DynastyValue: 2000},
		{Pos: "WR", Name: "Best Available WR", DynastyValue: 1500},
	}

	got := findLowestValueRosterPlayers(starters, bench, taxi, freeAgents, 3)

	if len(got) != 4 {
		t.Fatalf("expected 3 active-roster comparisons + 1 taxi comparison, got %d: %+v", len(got), got)
	}
	// Lowest first among active roster: Weak RB (300), Weak WR (500), Starter QB (8000)
	if got[0].YourPlayer.Name != "Weak RB" {
		t.Errorf("expected lowest-value active-roster player first, got %+v", got[0])
	}
	if got[0].BestAvailable.Name != "Best Available RB" || got[0].ValueGap != 1700 {
		t.Errorf("expected Weak RB paired with Best Available RB (+1700 gap), got %+v", got[0])
	}
	if got[2].YourPlayer.Name != "Starter QB" {
		t.Errorf("expected the 3 active-roster comparisons to come first, got %+v", got)
	}

	// Taxi squad player appended after the 3 active-roster comparisons.
	taxiComparison := got[3]
	if taxiComparison.YourPlayer.Name != "Stashed Rookie" || !taxiComparison.YourPlayer.IsTaxiSquad {
		t.Fatalf("expected taxi squad player included in comparisons, got %+v", taxiComparison)
	}
	if taxiComparison.BestAvailable.Name != "Best Available WR" || taxiComparison.ValueGap != 1400 {
		t.Errorf("expected taxi player compared against Best Available WR (+1400 gap), got %+v", taxiComparison)
	}

	for _, c := range got {
		if c.YourPlayer.Name == "Zero Value TE" {
			t.Errorf("zero-dynasty-value active-roster player should have been excluded")
		}
	}
}

func TestFindLowestValueRosterPlayersIncludesEntireTaxiSquadRegardlessOfLimit(t *testing.T) {
	taxi := []PlayerRow{
		{Pos: "WR", Name: "Taxi One", DynastyValue: 100, IsTaxiSquad: true},
		{Pos: "RB", Name: "Taxi Two", DynastyValue: 200, IsTaxiSquad: true},
		{Pos: "TE", Name: "Taxi Three", DynastyValue: 300, IsTaxiSquad: true},
		{Pos: "QB", Name: "Taxi Four", DynastyValue: 400, IsTaxiSquad: true},
	}

	got := findLowestValueRosterPlayers(nil, nil, taxi, nil, 3)

	if len(got) != 4 {
		t.Fatalf("expected all 4 taxi squad players included even though limit=3, got %d: %+v", len(got), got)
	}
	if got[0].YourPlayer.Name != "Taxi One" {
		t.Errorf("expected taxi comparisons sorted lowest-value first, got %+v", got[0])
	}
}

func TestFindLowestValueRosterPlayersNoAvailableAtPosition(t *testing.T) {
	bench := []PlayerRow{{Pos: "K", Name: "Weak Kicker", DynastyValue: 50}}
	got := findLowestValueRosterPlayers(nil, bench, nil, nil, 3)

	if len(got) != 1 {
		t.Fatalf("expected 1 comparison, got %d", len(got))
	}
	if got[0].BestAvailable.Name != "" || got[0].ValueGap != 0 {
		t.Errorf("expected empty BestAvailable/zero gap when no free agent exists at the position, got %+v", got[0])
	}
}

func TestBuildTaxiSquadSuggestionsFillsOpenSlotsFirst(t *testing.T) {
	freeAgents := []PlayerRow{
		{Pos: "WR", Name: "Best Prospect", DynastyValue: 5000, IsTaxiEligible: true},
		{Pos: "RB", Name: "Non-Taxi Vet", DynastyValue: 4000, IsTaxiEligible: false},
		{Pos: "TE", Name: "Second Prospect", DynastyValue: 2000, IsTaxiEligible: true},
	}

	got := buildTaxiSquadSuggestions(freeAgents, nil, 2, 5)

	if len(got) != 2 {
		t.Fatalf("expected 2 suggestions (non-taxi-eligible skipped), got %d: %+v", len(got), got)
	}
	if got[0].Prospect.Name != "Best Prospect" || !got[0].HasOpenSlot {
		t.Errorf("expected highest-value taxi-eligible prospect to fill first open slot, got %+v", got[0])
	}
	if got[1].Prospect.Name != "Second Prospect" || !got[1].HasOpenSlot {
		t.Errorf("expected second prospect to fill second open slot, got %+v", got[1])
	}
}

func TestBuildTaxiSquadSuggestionsSwapWhenFull(t *testing.T) {
	freeAgents := []PlayerRow{
		{Pos: "WR", Name: "Great Prospect", DynastyValue: 5000, IsTaxiEligible: true},
		{Pos: "TE", Name: "Mediocre Prospect", DynastyValue: 100, IsTaxiEligible: true},
	}
	currentTaxi := []PlayerRow{
		{Pos: "RB", Name: "Current Stash", DynastyValue: 1000, IsTaxiSquad: true},
	}

	got := buildTaxiSquadSuggestions(freeAgents, currentTaxi, 0, 5)

	if len(got) != 1 {
		t.Fatalf("expected only the clearly-better prospect to be suggested as a swap, got %d: %+v", len(got), got)
	}
	if got[0].Prospect.Name != "Great Prospect" {
		t.Errorf("expected Great Prospect suggested, got %+v", got[0])
	}
	if got[0].HasOpenSlot {
		t.Errorf("expected HasOpenSlot=false when taxi squad is full")
	}
	if got[0].DropFromTaxi.Name != "Current Stash" {
		t.Errorf("expected swap target to be the weakest current taxi player, got %+v", got[0].DropFromTaxi)
	}
}

func TestBuildTaxiSquadSuggestionsNoneWhenNoImprovementAndFull(t *testing.T) {
	freeAgents := []PlayerRow{
		{Pos: "WR", Name: "Worse Prospect", DynastyValue: 500, IsTaxiEligible: true},
	}
	currentTaxi := []PlayerRow{
		{Pos: "RB", Name: "Current Stash", DynastyValue: 1000, IsTaxiSquad: true},
	}

	got := buildTaxiSquadSuggestions(freeAgents, currentTaxi, 0, 5)
	if len(got) != 0 {
		t.Errorf("expected no suggestions when no available prospect beats the weakest current taxi player, got %+v", got)
	}
}

func TestBuildTaxiSquadSuggestionsRespectsLimit(t *testing.T) {
	freeAgents := []PlayerRow{
		{Pos: "WR", Name: "P1", DynastyValue: 5000, IsTaxiEligible: true},
		{Pos: "RB", Name: "P2", DynastyValue: 4000, IsTaxiEligible: true},
		{Pos: "TE", Name: "P3", DynastyValue: 3000, IsTaxiEligible: true},
	}

	got := buildTaxiSquadSuggestions(freeAgents, nil, 10, 2)
	if len(got) != 2 {
		t.Errorf("expected limit of 2 to be respected, got %d: %+v", len(got), got)
	}
}

func TestBuildTaxiSquadRows(t *testing.T) {
	players := map[string]interface{}{
		"T1": map[string]interface{}{"first_name": "Taxi", "last_name": "Rookie", "position": "WR"},
		"T2": map[string]interface{}{"first_name": "No", "last_name": "Value", "position": "RB"},
		"T3": map[string]interface{}{"first_name": "Missing", "last_name": "Player"}, // not in players map lookup below
	}
	dynastyValues := map[string]DynastyValue{
		normalizeName("Taxi Rookie"): {Value1QB: 900, Value2QB: 1200},
	}

	got := buildTaxiSquadRows([]string{"T1", "T2", "T404"}, players, dynastyValues, false)

	if len(got) != 2 {
		t.Fatalf("expected 2 rows (unknown player_id skipped), got %d: %+v", len(got), got)
	}
	if got[0].Name != "Taxi Rookie" || got[0].Pos != "WR" || got[0].DynastyValue != 900 || !got[0].IsTaxiSquad {
		t.Errorf("expected real position and dynasty value for Taxi Rookie, got %+v", got[0])
	}
	if got[1].Name != "No Value" || got[1].DynastyValue != 0 {
		t.Errorf("expected zero value for a player with no dynasty value entry, got %+v", got[1])
	}
}

func TestBuildTaxiSquadRowsSuperflexUsesValue2QB(t *testing.T) {
	players := map[string]interface{}{
		"T1": map[string]interface{}{"first_name": "Taxi", "last_name": "Rookie", "position": "QB"},
	}
	dynastyValues := map[string]DynastyValue{
		normalizeName("Taxi Rookie"): {Value1QB: 900, Value2QB: 1200},
	}

	got := buildTaxiSquadRows([]string{"T1"}, players, dynastyValues, true)
	if len(got) != 1 || got[0].DynastyValue != 1200 {
		t.Fatalf("expected superflex value (1200) to be used, got %+v", got)
	}
}
