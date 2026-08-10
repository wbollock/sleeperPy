package main

import "testing"

func TestBlendValue3(t *testing.T) {
	cases := []struct {
		name     string
		a, b, c  int
		expected int
	}{
		{"all three sources", 1000, 1200, 1400, 1200},
		{"two sources", 1000, 0, 1400, 1200},
		{"one source", 0, 0, 900, 900},
		{"no sources", 0, 0, 0, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := blendValue3(c.a, c.b, c.c); got != c.expected {
				t.Errorf("blendValue3(%d, %d, %d) = %d, want %d", c.a, c.b, c.c, got, c.expected)
			}
		})
	}
}

func TestBlendDynastyADPValues(t *testing.T) {
	origADP := fetchDynastyADPValues
	defer func() { fetchDynastyADPValues = origADP }()

	fetchDynastyADPValues = func() map[string]int {
		return map[string]int{
			normalizeName("Star Player"): 9000,
			normalizeName("Unknown Guy"): 500, // Not in DynastyProcess/FantasyCalc -- should be skipped
		}
	}

	values := map[string]DynastyValue{
		normalizeName("Star Player"): {
			Name: "Star Player", Value1QB: 1100, Value2QB: 1700,
			DynastyProcessValue1QB: 1000, DynastyProcessValue2QB: 1600,
			FantasyCalcValue1QB: 1200, FantasyCalcValue2QB: 1800,
		},
	}

	blendDynastyADPValues(values)

	star := values[normalizeName("Star Player")]
	if star.DynastyADPValue != 9000 {
		t.Errorf("expected DynastyADPValue=9000, got %d", star.DynastyADPValue)
	}
	// (1000+1200+9000)/3 = 3733
	if star.Value1QB != 3733 {
		t.Errorf("expected blended Value1QB=3733, got %d", star.Value1QB)
	}
	// (1600+1800+9000)/3 = 4133
	if star.Value2QB != 4133 {
		t.Errorf("expected blended Value2QB=4133, got %d", star.Value2QB)
	}

	if _, exists := values[normalizeName("Unknown Guy")]; exists {
		t.Errorf("ADP-only player with no DynastyProcess/FantasyCalc data should not be added")
	}
}

func TestBlendDynastyADPValuesGracefulWhenUnavailable(t *testing.T) {
	origADP := fetchDynastyADPValues
	defer func() { fetchDynastyADPValues = origADP }()

	fetchDynastyADPValues = func() map[string]int { return nil }

	values := map[string]DynastyValue{
		"x": {Name: "X", Value1QB: 100, Value2QB: 200, DynastyProcessValue1QB: 100, DynastyProcessValue2QB: 200},
	}
	blendDynastyADPValues(values)

	if values["x"].Value1QB != 100 || values["x"].Value2QB != 200 {
		t.Errorf("values should be unchanged when dynasty ADP is unavailable, got %+v", values["x"])
	}
}
