package main

import "testing"

func TestBuildFreeAgentCandidates(t *testing.T) {
	tiers := map[string][][]string{
		"WR": {{"Elite Free Agent"}, {}, {"Mid Free Agent"}},
		"RB": {{"Rostered Runner"}},
	}

	players := map[string]interface{}{
		"FA1": map[string]interface{}{"first_name": "Elite", "last_name": "Free Agent", "position": "WR", "team": "KC", "active": true},
		"FA2": map[string]interface{}{"first_name": "Mid", "last_name": "Free Agent", "position": "WR", "team": "SF", "active": true},
		"R1":  map[string]interface{}{"first_name": "Rostered", "last_name": "Runner", "position": "RB", "team": "DAL", "active": true},
		"NT":  map[string]interface{}{"first_name": "No", "last_name": "Team", "position": "WR", "team": "", "active": true},
		"INA": map[string]interface{}{"first_name": "Inactive", "last_name": "Player", "position": "WR", "team": "NYJ", "active": false},
		"DEF": map[string]interface{}{"first_name": "Some", "last_name": "Defense", "position": "DEF", "team": "BUF", "active": true},
	}

	rostered := map[string]bool{"R1": true}

	got := buildFreeAgentCandidates(players, rostered, tiers, 5)

	names := make(map[string]bool)
	for _, c := range got {
		names[c.Name] = true
	}

	if !names["Elite Free Agent"] {
		t.Errorf("expected unrostered, tiered, active WR with a team to be a candidate, got: %+v", got)
	}
	if names["Rostered Runner"] {
		t.Errorf("rostered player should be excluded, got: %+v", got)
	}
	if names["No Team"] {
		t.Errorf("player with no NFL team should be excluded, got: %+v", got)
	}
	if names["Inactive Player"] {
		t.Errorf("inactive player should be excluded, got: %+v", got)
	}
	if names["Some Defense"] {
		t.Errorf("non-skill position (DEF) should be excluded, got: %+v", got)
	}
}

func TestBuildFreeAgentCandidatesRespectsPerPositionLimit(t *testing.T) {
	tiers := map[string][][]string{
		"WR": {{"WR One"}, {"WR Two"}, {"WR Three"}},
	}
	players := map[string]interface{}{
		"P1": map[string]interface{}{"first_name": "WR", "last_name": "One", "position": "WR", "team": "KC", "active": true},
		"P2": map[string]interface{}{"first_name": "WR", "last_name": "Two", "position": "WR", "team": "SF", "active": true},
		"P3": map[string]interface{}{"first_name": "WR", "last_name": "Three", "position": "WR", "team": "DAL", "active": true},
	}

	got := buildFreeAgentCandidates(players, map[string]bool{}, tiers, 2)
	if len(got) != 2 {
		t.Fatalf("expected per-position limit of 2 to be respected, got %d: %+v", len(got), got)
	}
}
