package main

import (
	"testing"
	"time"
)

func TestParseBorisTierText(t *testing.T) {
	body := []byte("Tier 1: Ja'Marr Chase, CeeDee Lamb\nTier 2: Malik Nabers\nnot a tier line\nTier 3: A.J. Brown, Brock Bowers\n")

	tiers := parseBorisTierText(body)
	if len(tiers) != 3 {
		t.Fatalf("expected 3 tiers, got %d: %+v", len(tiers), tiers)
	}
	if len(tiers[0]) != 2 || tiers[0][0] != "Ja'Marr Chase" || tiers[0][1] != "CeeDee Lamb" {
		t.Errorf("unexpected tier 1 contents: %+v", tiers[0])
	}
	if len(tiers[1]) != 1 || tiers[1][0] != "Malik Nabers" {
		t.Errorf("unexpected tier 2 contents: %+v", tiers[1])
	}
}

func TestParseBorisTierTextEmpty(t *testing.T) {
	if got := parseBorisTierText([]byte("")); got != nil {
		t.Errorf("expected nil for empty input, got %+v", got)
	}
	if got := parseBorisTierText([]byte("no tiers here\njust text")); got != nil {
		t.Errorf("expected nil when no 'Tier ' lines present, got %+v", got)
	}
}

func TestBuildPositionLookupMapsNormalizedNames(t *testing.T) {
	origPlayersCache := sleeperPlayersCache
	defer func() { sleeperPlayersCache = origPlayersCache }()
	sleeperPlayersCache = &playersCache{
		data: map[string]interface{}{
			"1": map[string]interface{}{"first_name": "Ja'Marr", "last_name": "Chase", "position": "WR"},
			"2": map[string]interface{}{"first_name": "No", "last_name": "Position"},
		},
		timestamp: time.Now(),
		ttl:       time.Hour,
	}

	lookup := buildPositionLookup()
	if lookup[normalizeName("Ja'Marr Chase")] != "WR" {
		t.Errorf("expected Ja'Marr Chase to map to WR, got %+v", lookup)
	}
	if _, exists := lookup[normalizeName("No Position")]; exists {
		t.Errorf("expected a player with no position to be skipped, got %+v", lookup)
	}
}
