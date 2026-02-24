package main

import (
	"os"
	"testing"
	"time"
)

func TestCalculateRetrospectiveWinner(t *testing.T) {
	t.Run("team1 improves over time", func(t *testing.T) {
		s := TradeSnapshot{
			Team1:          "Alpha",
			Team2:          "Beta",
			Team1ValueThen: 1000,
			Team2ValueThen: 1200,
			Team1ValueNow:  800,
			Team2ValueNow:  1700,
		}
		winner, swing := calculateRetrospectiveWinner(s)
		if winner != "Alpha" {
			t.Fatalf("expected Alpha, got %s", winner)
		}
		if swing != 700 {
			t.Fatalf("expected swing 700, got %d", swing)
		}
	})

	t.Run("team2 improves over time", func(t *testing.T) {
		s := TradeSnapshot{
			Team1:          "Alpha",
			Team2:          "Beta",
			Team1ValueThen: 1000,
			Team2ValueThen: 1200,
			Team1ValueNow:  1300,
			Team2ValueNow:  900,
		}
		winner, swing := calculateRetrospectiveWinner(s)
		if winner != "Beta" {
			t.Fatalf("expected Beta, got %s", winner)
		}
		if swing != 600 {
			t.Fatalf("expected swing 600, got %d", swing)
		}
	})

	t.Run("small moves are even", func(t *testing.T) {
		s := TradeSnapshot{
			Team1:          "Alpha",
			Team2:          "Beta",
			Team1ValueThen: 1000,
			Team2ValueThen: 1200,
			Team1ValueNow:  1050,
			Team2ValueNow:  1230,
		}
		winner, swing := calculateRetrospectiveWinner(s)
		if winner != "Even" || swing != 0 {
			t.Fatalf("expected Even/0, got %s/%d", winner, swing)
		}
	})
}

func TestAnalyzeTradeRetrospectiveCreatesSnapshotAndAttachesResult(t *testing.T) {
	// Keep this test isolated from local runtime snapshots.
	_ = os.RemoveAll(tradeCacheDir)
	t.Cleanup(func() {
		_ = os.RemoveAll(tradeCacheDir)
	})

	values := map[string]DynastyValue{
		normalizeName("Player A"): {Value1QB: 900, Value2QB: 900},
		normalizeName("Player B"): {Value1QB: 1800, Value2QB: 1800},
	}

	txns := []Transaction{
		{
			Type:           "trade",
			LeagueID:       "league-123",
			Team1:          "Team One",
			Team2:          "Team Two",
			Timestamp:      time.Now().Add(-48 * time.Hour),
			Team1Gave:      []string{"Player A"},
			Team2Gave:      []string{"Player B"},
			Team1GaveValue: 1200,
			Team2GaveValue: 1300,
		},
	}

	out := analyzeTradeRetrospective(txns, values, false)
	if len(out) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(out))
	}

	if out[0].Retrospective.Winner != "Team One" {
		t.Fatalf("expected Team One winner, got %s", out[0].Retrospective.Winner)
	}
	if out[0].Retrospective.ValueSwing <= 0 {
		t.Fatalf("expected positive swing, got %d", out[0].Retrospective.ValueSwing)
	}
	if out[0].Retrospective.DaysElapsed < 1 {
		t.Fatalf("expected >=1 day elapsed, got %d", out[0].Retrospective.DaysElapsed)
	}
}
