// Value change tracker - identifies dynasty value risers and fallers
// Caches daily snapshots and computes deltas

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"sync"
	"time"
)

const valueCacheFilePattern = "/tmp/sleeperpy_value_snapshot_%s.json"

type valueSnapshotFile struct {
	Date   string         `json:"date"`
	Values map[string]int `json:"values"`
}

type valueSnapshotState struct {
	sync.Mutex
	loaded              bool
	filePath            string
	snapshot            valueSnapshotFile
	rotatedDate         string
	rotationBaseline    map[string]int
	rotationBaselineSet bool
}

var valueTrackerState = &valueSnapshotState{}

// Get top risers and fallers by comparing to cached snapshot
func getValueChanges(currentValues map[string]DynastyValue, userPlayerNames []string, isSuperFlex bool) ([]ValueChange, error) {
	changes := []ValueChange{}
	today := time.Now().Format("2006-01-02")
	filePath := getValueSnapshotPath(isSuperFlex)

	// Load snapshot once per mode and keep in memory.
	baseline, snapshotDate, err := getValueBaseline(filePath, today)
	if err != nil {
		return changes, err
	}

	// First run for this mode: initialize and return no changes.
	if len(baseline) == 0 {
		if err := saveSnapshot(filePath, currentValues, isSuperFlex, today); err != nil {
			log.Printf("[ERROR] Failed to initialize value snapshot: %v", err)
		}
		return changes, nil
	}

	// Build user player map for owned check
	ownedPlayers := make(map[string]bool)
	for _, name := range userPlayerNames {
		ownedPlayers[normalizeName(name)] = true
	}

	// Compare current values to old snapshot
	for name, dv := range currentValues {
		currentValue := dv.Value1QB
		if isSuperFlex {
			currentValue = dv.Value2QB
		}

		oldValue, existed := baseline[name]
		if !existed || currentValue == 0 || oldValue == 0 {
			continue
		}

		delta := currentValue - oldValue
		if delta == 0 {
			continue
		}

		deltaPct := float64(delta) / float64(oldValue) * 100

		// Only track significant changes (>5% or >100 value)
		if deltaPct < -5 || deltaPct > 5 || delta > 100 || delta < -100 {
			changes = append(changes, ValueChange{
				PlayerName: dv.Name,
				Position:   dv.Position,
				OldValue:   oldValue,
				NewValue:   currentValue,
				Delta:      delta,
				DeltaPct:   deltaPct,
				IsRiser:    delta > 0,
				IsOwned:    ownedPlayers[name],
			})
		}
	}

	// Sort by absolute delta (biggest changes first)
	sort.Slice(changes, func(i, j int) bool {
		absI := changes[i].Delta
		if absI < 0 {
			absI = -absI
		}
		absJ := changes[j].Delta
		if absJ < 0 {
			absJ = -absJ
		}
		return absI > absJ
	})

	// Rotate snapshot once per day, while keeping the prior-day baseline in memory
	// for all calls during this process day.
	if snapshotDate != today {
		if err := saveSnapshot(filePath, currentValues, isSuperFlex, today); err != nil {
			log.Printf("[ERROR] Failed to rotate value snapshot: %v", err)
		}
	}

	return changes, nil
}

func saveSnapshot(filePath string, values map[string]DynastyValue, isSuperFlex bool, date string) error {
	out := make(map[string]int)
	for name, dv := range values {
		if isSuperFlex {
			out[name] = dv.Value2QB
		} else {
			out[name] = dv.Value1QB
		}
	}

	payload := valueSnapshotFile{
		Date:   date,
		Values: out,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return err
	}

	// Keep in-memory state consistent with persisted state.
	valueTrackerState.Lock()
	defer valueTrackerState.Unlock()
	valueTrackerState.loaded = true
	valueTrackerState.filePath = filePath
	valueTrackerState.snapshot = payload
	return nil
}

func getValueBaseline(filePath, today string) (map[string]int, string, error) {
	valueTrackerState.Lock()
	defer valueTrackerState.Unlock()

	// Load from disk on first use or when switching mode file.
	if !valueTrackerState.loaded || valueTrackerState.filePath != filePath {
		loaded, err := loadSnapshotFromFile(filePath)
		if err != nil {
			return nil, "", err
		}
		valueTrackerState.loaded = true
		valueTrackerState.filePath = filePath
		valueTrackerState.snapshot = loaded
		valueTrackerState.rotationBaseline = nil
		valueTrackerState.rotationBaselineSet = false
		valueTrackerState.rotatedDate = ""
	}

	snapshot := valueTrackerState.snapshot
	if len(snapshot.Values) == 0 {
		return map[string]int{}, snapshot.Date, nil
	}

	// If we already rotated today, keep using the pre-rotation baseline for this process.
	if valueTrackerState.rotatedDate == today && valueTrackerState.rotationBaselineSet {
		return valueTrackerState.rotationBaseline, snapshot.Date, nil
	}

	// If we're about to rotate, remember prior baseline for the rest of today.
	if snapshot.Date != today && !valueTrackerState.rotationBaselineSet {
		valueTrackerState.rotationBaseline = cloneIntMap(snapshot.Values)
		valueTrackerState.rotationBaselineSet = true
		valueTrackerState.rotatedDate = today
	}

	if valueTrackerState.rotationBaselineSet && valueTrackerState.rotatedDate == today {
		return valueTrackerState.rotationBaseline, snapshot.Date, nil
	}

	return snapshot.Values, snapshot.Date, nil
}

func loadSnapshotFromFile(filePath string) (valueSnapshotFile, error) {
	out := valueSnapshotFile{
		Values: map[string]int{},
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return out, err
	}

	// Backward compatibility for old raw map snapshot format.
	var oldFormat map[string]int
	if err := json.Unmarshal(data, &oldFormat); err == nil && len(oldFormat) > 0 {
		out.Date = ""
		out.Values = oldFormat
		return out, nil
	}

	if err := json.Unmarshal(data, &out); err != nil {
		return valueSnapshotFile{}, err
	}
	if out.Values == nil {
		out.Values = map[string]int{}
	}
	return out, nil
}

func getValueSnapshotPath(isSuperFlex bool) string {
	mode := "1qb"
	if isSuperFlex {
		mode = "sf"
	}
	return fmt.Sprintf(valueCacheFilePattern, mode)
}

func cloneIntMap(in map[string]int) map[string]int {
	out := make(map[string]int, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// Get top N risers from user's roster
func getTopRisers(changes []ValueChange, limit int) []ValueChange {
	risers := []ValueChange{}
	for _, c := range changes {
		if c.IsRiser && c.IsOwned {
			risers = append(risers, c)
			if len(risers) >= limit {
				break
			}
		}
	}
	return risers
}

// Get top N fallers from user's roster
func getTopFallers(changes []ValueChange, limit int) []ValueChange {
	fallers := []ValueChange{}
	for _, c := range changes {
		if !c.IsRiser && c.IsOwned {
			fallers = append(fallers, c)
			if len(fallers) >= limit {
				break
			}
		}
	}
	return fallers
}

// Get league-wide risers (for waiver targets)
func getLeagueRisers(changes []ValueChange, limit int) []ValueChange {
	risers := []ValueChange{}
	for _, c := range changes {
		if c.IsRiser {
			risers = append(risers, c)
			if len(risers) >= limit {
				break
			}
		}
	}
	return risers
}
