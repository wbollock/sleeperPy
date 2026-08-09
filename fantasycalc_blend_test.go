package main

import "testing"

func TestBlendValue(t *testing.T) {
	cases := []struct {
		name     string
		a, b     int
		expected int
	}{
		{"both sources", 1000, 1200, 1100},
		{"only DynastyProcess", 1000, 0, 1000},
		{"only FantasyCalc", 0, 1200, 1200},
		{"neither", 0, 0, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := blendValue(c.a, c.b); got != c.expected {
				t.Errorf("blendValue(%d, %d) = %d, want %d", c.a, c.b, got, c.expected)
			}
		})
	}
}

func TestBlendFantasyCalcValues(t *testing.T) {
	origFC := fetchFantasyCalcValues
	defer func() { fetchFantasyCalcValues = origFC }()

	fetchFantasyCalcValues = func(numQBs int) map[string]int {
		if numQBs == 1 {
			return map[string]int{"P1": 1200, "P2": 500} // P2: FantasyCalc-only player
		}
		return map[string]int{"P1": 1800}
	}

	players := map[string]interface{}{
		"P1": map[string]interface{}{"full_name": "Star Player", "position": "RB"},
		"P2": map[string]interface{}{"full_name": "Waiver Guy", "position": "WR"},
	}

	values := map[string]DynastyValue{
		normalizeName("Star Player"): {
			Name:                   "Star Player",
			Position:               "RB",
			Value1QB:               1000,
			Value2QB:               1600,
			DynastyProcessValue1QB: 1000,
			DynastyProcessValue2QB: 1600,
		},
	}

	blendFantasyCalcValues(values, players)

	star := values[normalizeName("Star Player")]
	if star.FantasyCalcValue1QB != 1200 || star.FantasyCalcValue2QB != 1800 {
		t.Fatalf("expected FantasyCalc values recorded, got %+v", star)
	}
	if star.Value1QB != 1100 { // (1000+1200)/2
		t.Errorf("expected blended Value1QB=1100, got %d", star.Value1QB)
	}
	if star.Value2QB != 1700 { // (1600+1800)/2
		t.Errorf("expected blended Value2QB=1700, got %d", star.Value2QB)
	}

	// A player DynastyProcess had no row for at all should still get added,
	// using FantasyCalc alone.
	waiver, exists := values[normalizeName("Waiver Guy")]
	if !exists {
		t.Fatalf("expected a new entry created for a FantasyCalc-only player")
	}
	if waiver.Value1QB != 500 {
		t.Errorf("expected Value1QB=500 (FantasyCalc only), got %d", waiver.Value1QB)
	}
}

func TestFetchDynastyValuesGracefulWithoutFantasyCalc(t *testing.T) {
	origFC := fetchFantasyCalcValues
	defer func() { fetchFantasyCalcValues = origFC }()

	fetchFantasyCalcValues = func(numQBs int) map[string]int { return nil }

	values := map[string]DynastyValue{
		"x": {Name: "X", Value1QB: 100, Value2QB: 200, DynastyProcessValue1QB: 100, DynastyProcessValue2QB: 200},
	}
	blendFantasyCalcValues(values, map[string]interface{}{})

	if values["x"].Value1QB != 100 || values["x"].Value2QB != 200 {
		t.Errorf("values should be unchanged when FantasyCalc is unavailable, got %+v", values["x"])
	}
}

func TestDynastyValueSourcesDisagree(t *testing.T) {
	cases := []struct {
		name     string
		val      DynastyValue
		expected bool
	}{
		{
			name:     "sources agree closely",
			val:      DynastyValue{DynastyProcessValue1QB: 1000, FantasyCalcValue1QB: 1050},
			expected: false,
		},
		{
			name:     "sources disagree by 40%",
			val:      DynastyValue{DynastyProcessValue1QB: 1000, FantasyCalcValue1QB: 1500},
			expected: true,
		},
		{
			name:     "only DynastyProcess has data",
			val:      DynastyValue{DynastyProcessValue1QB: 1000, FantasyCalcValue1QB: 0},
			expected: false,
		},
		{
			name:     "only FantasyCalc has data",
			val:      DynastyValue{DynastyProcessValue1QB: 0, FantasyCalcValue1QB: 1000},
			expected: false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := dynastyValueSourcesDisagree(c.val, false); got != c.expected {
				t.Errorf("dynastyValueSourcesDisagree(%+v) = %v, want %v", c.val, got, c.expected)
			}
		})
	}

	// Superflex should compare the 2QB fields, not 1QB.
	superflexVal := DynastyValue{
		DynastyProcessValue1QB: 1000, FantasyCalcValue1QB: 1050, // agree in 1QB
		DynastyProcessValue2QB: 1000, FantasyCalcValue2QB: 2000, // disagree in 2QB
	}
	if dynastyValueSourcesDisagree(superflexVal, false) {
		t.Errorf("expected 1QB comparison to agree")
	}
	if !dynastyValueSourcesDisagree(superflexVal, true) {
		t.Errorf("expected 2QB (superflex) comparison to disagree")
	}
}

func TestEnrichRowsWithDynastyValuesSetsLowConfidence(t *testing.T) {
	rows := []PlayerRow{{Name: "Disputed Player"}, {Name: "Agreed Player"}}
	dynastyValues := map[string]DynastyValue{
		normalizeName("Disputed Player"): {
			Value1QB: 1250, DynastyProcessValue1QB: 1000, FantasyCalcValue1QB: 1500,
		},
		normalizeName("Agreed Player"): {
			Value1QB: 1025, DynastyProcessValue1QB: 1000, FantasyCalcValue1QB: 1050,
		},
	}

	enrichRowsWithDynastyValues(rows, dynastyValues, false)

	if !rows[0].DynastyValueLowConf {
		t.Errorf("expected Disputed Player to be flagged low-confidence")
	}
	if rows[1].DynastyValueLowConf {
		t.Errorf("expected Agreed Player to NOT be flagged low-confidence")
	}
}
