package main

import (
	"os"
	"testing"
)

func resetValueTrackerStateForTest(t *testing.T) {
	t.Helper()
	_ = os.Remove(getValueSnapshotPath(false))
	_ = os.Remove(getValueSnapshotPath(true))

	valueTrackerState.Lock()
	valueTrackerState.loaded = false
	valueTrackerState.filePath = ""
	valueTrackerState.snapshot = valueSnapshotFile{}
	valueTrackerState.rotatedDate = ""
	valueTrackerState.rotationBaseline = nil
	valueTrackerState.rotationBaselineSet = false
	valueTrackerState.Unlock()
}

func TestGetValueChangesInitializesSnapshot(t *testing.T) {
	resetValueTrackerStateForTest(t)

	values := map[string]DynastyValue{
		normalizeName("Player One"): {
			Name:     "Player One",
			Position: "WR",
			Value1QB: 1200,
			Value2QB: 1200,
		},
	}

	changes, err := getValueChanges(values, []string{"Player One"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(changes) != 0 {
		t.Fatalf("expected no changes on first snapshot, got %d", len(changes))
	}
}

func TestGetValueChangesDetectsDeltaAndOwnedStatus(t *testing.T) {
	resetValueTrackerStateForTest(t)

	key := normalizeName("Player One")
	initial := map[string]DynastyValue{
		key: {
			Name:     "Player One",
			Position: "WR",
			Value1QB: 1000,
			Value2QB: 1000,
		},
	}
	_, err := getValueChanges(initial, []string{"Player One"}, false)
	if err != nil {
		t.Fatalf("unexpected error initializing snapshot: %v", err)
	}

	updated := map[string]DynastyValue{
		key: {
			Name:     "Player One",
			Position: "WR",
			Value1QB: 1200,
			Value2QB: 1200,
		},
	}
	changes, err := getValueChanges(updated, []string{"Player One"}, false)
	if err != nil {
		t.Fatalf("unexpected error reading changes: %v", err)
	}
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}
	if !changes[0].IsOwned {
		t.Fatalf("expected owned player change")
	}
	if !changes[0].IsRiser {
		t.Fatalf("expected riser")
	}
	if changes[0].Delta != 200 {
		t.Fatalf("expected delta 200, got %d", changes[0].Delta)
	}
}

func TestGetValueChangesSeparatesModes(t *testing.T) {
	resetValueTrackerStateForTest(t)

	key := normalizeName("Mode Player")
	values := map[string]DynastyValue{
		key: {
			Name:     "Mode Player",
			Position: "QB",
			Value1QB: 900,
			Value2QB: 1400,
		},
	}

	if _, err := getValueChanges(values, []string{"Mode Player"}, false); err != nil {
		t.Fatalf("unexpected error for 1qb init: %v", err)
	}
	if _, err := getValueChanges(values, []string{"Mode Player"}, true); err != nil {
		t.Fatalf("unexpected error for sf init: %v", err)
	}

	if _, err := os.Stat(getValueSnapshotPath(false)); err != nil {
		t.Fatalf("missing 1qb snapshot file: %v", err)
	}
	if _, err := os.Stat(getValueSnapshotPath(true)); err != nil {
		t.Fatalf("missing sf snapshot file: %v", err)
	}
}
