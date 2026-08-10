package main

import "testing"

func TestBuildAllFreeAgentsByDynastyValue(t *testing.T) {
	players := map[string]interface{}{
		"FA1": map[string]interface{}{"first_name": "Elite", "last_name": "Free Agent", "position": "WR", "team": "KC", "active": true},
		"FA2": map[string]interface{}{"first_name": "Mid", "last_name": "Free Agent", "position": "RB", "team": "SF", "active": true},
		"R1":  map[string]interface{}{"first_name": "Rostered", "last_name": "Runner", "position": "RB", "team": "DAL", "active": true},
		"NT":  map[string]interface{}{"first_name": "No", "last_name": "Team", "position": "WR", "team": "", "active": true},
		"INA": map[string]interface{}{"first_name": "Inactive", "last_name": "Player", "position": "WR", "team": "NYJ", "active": false},
		"DEF": map[string]interface{}{"first_name": "Some", "last_name": "Defense", "position": "DEF", "team": "BUF", "active": true},
		"NV":  map[string]interface{}{"first_name": "No", "last_name": "Value", "position": "TE", "team": "MIA", "active": true},
	}

	dynastyValues := map[string]DynastyValue{
		normalizeName("Elite Free Agent"): {Value1QB: 5000, Value2QB: 6000},
		normalizeName("Mid Free Agent"):   {Value1QB: 2000, Value2QB: 2500},
		normalizeName("Rostered Runner"):  {Value1QB: 9000, Value2QB: 9500},
	}

	rostered := map[string]bool{"R1": true}

	got := buildAllFreeAgentsByDynastyValue(players, rostered, dynastyValues, false, 0, 0)

	if len(got) != 2 {
		t.Fatalf("expected 2 unowned valued free agents, got %d: %+v", len(got), got)
	}
	if got[0].Name != "Elite Free Agent" || got[0].DynastyValue != 5000 {
		t.Errorf("expected highest-value FA first, got %+v", got[0])
	}
	if got[1].Name != "Mid Free Agent" || got[1].DynastyValue != 2000 {
		t.Errorf("expected second-highest-value FA second, got %+v", got[1])
	}
}

func TestBuildAllFreeAgentsByDynastyValueSuperflexUsesValue2QB(t *testing.T) {
	players := map[string]interface{}{
		"FA1": map[string]interface{}{"first_name": "Elite", "last_name": "Free Agent", "position": "QB", "team": "KC", "active": true},
	}
	dynastyValues := map[string]DynastyValue{
		normalizeName("Elite Free Agent"): {Value1QB: 3000, Value2QB: 7000},
	}

	got := buildAllFreeAgentsByDynastyValue(players, map[string]bool{}, dynastyValues, true, 0, 0)
	if len(got) != 1 || got[0].DynastyValue != 7000 {
		t.Fatalf("expected superflex value (7000) to be used, got %+v", got)
	}
}

func TestBuildAllFreeAgentsByDynastyValueRespectsLimit(t *testing.T) {
	players := map[string]interface{}{
		"FA1": map[string]interface{}{"first_name": "A", "last_name": "One", "position": "WR", "team": "KC", "active": true},
		"FA2": map[string]interface{}{"first_name": "B", "last_name": "Two", "position": "WR", "team": "SF", "active": true},
		"FA3": map[string]interface{}{"first_name": "C", "last_name": "Three", "position": "WR", "team": "DAL", "active": true},
	}
	dynastyValues := map[string]DynastyValue{
		normalizeName("A One"):   {Value1QB: 1000},
		normalizeName("B Two"):   {Value1QB: 2000},
		normalizeName("C Three"): {Value1QB: 3000},
	}

	got := buildAllFreeAgentsByDynastyValue(players, map[string]bool{}, dynastyValues, false, 0, 2)
	if len(got) != 2 {
		t.Fatalf("expected limit of 2 to be respected, got %d: %+v", len(got), got)
	}
	if got[0].Name != "C Three" {
		t.Errorf("expected highest-value FA to survive the limit, got %+v", got[0])
	}
}

func TestBuildAllFreeAgentsByDynastyValueTaxiEligibility(t *testing.T) {
	players := map[string]interface{}{
		"ROOKIE": map[string]interface{}{"first_name": "Rookie", "last_name": "Guy", "position": "WR", "team": "KC", "active": true, "years_exp": 0.0},
		"SOPH":   map[string]interface{}{"first_name": "Second", "last_name": "Year", "position": "WR", "team": "SF", "active": true, "years_exp": 1.0},
		"VET":    map[string]interface{}{"first_name": "Grizzled", "last_name": "Vet", "position": "WR", "team": "DAL", "active": true, "years_exp": 8.0},
	}
	dynastyValues := map[string]DynastyValue{
		normalizeName("Rookie Guy"):   {Value1QB: 4000},
		normalizeName("Second Year"):  {Value1QB: 3500},
		normalizeName("Grizzled Vet"): {Value1QB: 3000},
	}

	got := buildAllFreeAgentsByDynastyValue(players, map[string]bool{}, dynastyValues, false, 1, 0)

	byName := map[string]PlayerRow{}
	for _, row := range got {
		byName[row.Name] = row
	}

	if !byName["Rookie Guy"].IsTaxiEligible {
		t.Errorf("expected a rookie (0 years exp) to be taxi eligible with taxiYears=1")
	}
	if !byName["Second Year"].IsTaxiEligible {
		t.Errorf("expected a 1-year player to be taxi eligible with taxiYears=1")
	}
	if byName["Grizzled Vet"].IsTaxiEligible {
		t.Errorf("expected an 8-year veteran to NOT be taxi eligible with taxiYears=1")
	}
}

func TestBuildAllFreeAgentsByDynastyValueTaxiEligibilityDisabled(t *testing.T) {
	players := map[string]interface{}{
		"ROOKIE": map[string]interface{}{"first_name": "Rookie", "last_name": "Guy", "position": "WR", "team": "KC", "active": true, "years_exp": 0.0},
	}
	dynastyValues := map[string]DynastyValue{
		normalizeName("Rookie Guy"): {Value1QB: 4000},
	}

	got := buildAllFreeAgentsByDynastyValue(players, map[string]bool{}, dynastyValues, false, 0, 0)
	if got[0].IsTaxiEligible {
		t.Errorf("expected taxiYears=0 (no taxi squad) to disable eligibility flagging entirely")
	}
}
