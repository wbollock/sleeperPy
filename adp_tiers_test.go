package main

import "testing"

func TestClusterByADPGapSplitsOnLargeGap(t *testing.T) {
	players := []adpPlayer{
		{Name: "Elite RB", ADP: 1.5, StDev: 0.5},
		{Name: "Also Elite RB", ADP: 2.0, StDev: 0.5},
		{Name: "Tier 2 RB", ADP: 15.0, StDev: 1.0}, // big gap -- new tier
		{Name: "Tier 2 RB Also", ADP: 16.0, StDev: 1.0},
	}

	tiers := clusterByADPGap(players)
	if len(tiers) != 2 {
		t.Fatalf("expected 2 tiers, got %d: %+v", len(tiers), tiers)
	}
	if len(tiers[0]) != 2 || len(tiers[1]) != 2 {
		t.Errorf("expected 2 players per tier, got %+v", tiers)
	}
	if tiers[0][0] != "Elite RB" || tiers[1][0] != "Tier 2 RB" {
		t.Errorf("unexpected tier contents: %+v", tiers)
	}
}

func TestClusterByADPGapEmptyInput(t *testing.T) {
	if got := clusterByADPGap(nil); got != nil {
		t.Errorf("expected nil for empty input, got %+v", got)
	}
}

func TestClusterByADPGapSinglePlayer(t *testing.T) {
	tiers := clusterByADPGap([]adpPlayer{{Name: "Solo", ADP: 5.0, StDev: 1.0}})
	if len(tiers) != 1 || len(tiers[0]) != 1 || tiers[0][0] != "Solo" {
		t.Fatalf("expected a single one-player tier, got %+v", tiers)
	}
}

func TestBuildADPTiersMapsPositionsAndBuildsFlex(t *testing.T) {
	players := []adpPlayer{
		{Name: "QB One", Position: "QB", ADP: 20, StDev: 2},
		{Name: "RB One", Position: "RB", ADP: 1, StDev: 1},
		{Name: "WR One", Position: "WR", ADP: 3, StDev: 1},
		{Name: "TE One", Position: "TE", ADP: 30, StDev: 2},
		{Name: "Kicker One", Position: "PK", ADP: 180, StDev: 5},
		{Name: "Defense One", Position: "DEF", ADP: 170, StDev: 5},
	}

	tiers := buildADPTiers(players)

	if _, ok := tiers["K"]; !ok {
		t.Errorf("expected PK to be normalized to K, got keys: %v", keysOf(tiers))
	}
	if _, ok := tiers["DST"]; !ok {
		t.Errorf("expected DEF to be normalized to DST, got keys: %v", keysOf(tiers))
	}
	flex, ok := tiers["FLX"]
	if !ok {
		t.Fatalf("expected an FLX bucket, got keys: %v", keysOf(tiers))
	}
	flexNames := map[string]bool{}
	for _, tier := range flex {
		for _, name := range tier {
			flexNames[name] = true
		}
	}
	if !flexNames["RB One"] || !flexNames["WR One"] || !flexNames["TE One"] {
		t.Errorf("expected FLX to contain RB/WR/TE players, got %+v", flex)
	}
	if flexNames["QB One"] || flexNames["Kicker One"] {
		t.Errorf("FLX should never contain QB or K, got %+v", flex)
	}
}

func keysOf(m map[string][][]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
