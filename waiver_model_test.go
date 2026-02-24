package main

import "testing"

func TestUsageSignal(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{75, "strong usage signal"},
		{55, "stable usage signal"},
		{30, "speculative usage signal"},
		{5, "low usage signal"},
	}

	for _, tc := range cases {
		got := usageSignal(tc.in)
		if got != tc.want {
			t.Fatalf("usageSignal(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestClassifyWaiverRole(t *testing.T) {
	if got := classifyWaiverRole(PlayerRow{RosterPercent: 10}, "Starter Upgrade"); got != "Immediate Starter" {
		t.Fatalf("unexpected starter role: %q", got)
	}
	if got := classifyWaiverRole(PlayerRow{RosterPercent: 10}, "Lottery Ticket"); got != "Upside Stash" {
		t.Fatalf("unexpected lottery role: %q", got)
	}
	if got := classifyWaiverRole(PlayerRow{RosterPercent: 80}, "Depth Add"); got != "High-usage Depth" {
		t.Fatalf("unexpected high-usage role: %q", got)
	}
	if got := classifyWaiverRole(PlayerRow{RosterPercent: 30}, "Depth Add"); got != "Rotational Depth" {
		t.Fatalf("unexpected rotational role: %q", got)
	}
	if got := classifyWaiverRole(PlayerRow{RosterPercent: 5}, "Depth Add"); got != "Bench Stash" {
		t.Fatalf("unexpected stash role: %q", got)
	}
}
