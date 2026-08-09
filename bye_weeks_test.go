package main

import "testing"

func TestDeriveByeWeeks(t *testing.T) {
	// 3 teams, 2 weeks. Week 1: A vs B (C has bye). Week 2: A vs C (B has bye).
	games := []scheduleGame{
		{Week: 1, Home: "A", Away: "B"},
		{Week: 2, Home: "A", Away: "C"},
	}

	got := deriveByeWeeks(games)

	if got["C"] != 1 {
		t.Errorf("expected team C to have bye week 1, got %d", got["C"])
	}
	if got["B"] != 2 {
		t.Errorf("expected team B to have bye week 2, got %d", got["B"])
	}
	if _, exists := got["A"]; exists {
		t.Errorf("team A plays every week, should have no bye entry, got %d", got["A"])
	}
}

func TestDeriveByeWeeksEmpty(t *testing.T) {
	got := deriveByeWeeks(nil)
	if len(got) != 0 {
		t.Errorf("expected empty input to produce empty output, got %+v", got)
	}
}
