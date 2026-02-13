// Value change tracker - identifies dynasty value risers and fallers
// Caches daily snapshots and computes deltas

package main

import (
	"encoding/json"
	"os"
	"sort"
)

const valueCacheFile = "/tmp/sleeperpy_value_snapshot.json"

// Get top risers and fallers by comparing to cached snapshot
func getValueChanges(currentValues map[string]DynastyValue, userPlayerNames []string, isSuperFlex bool) ([]ValueChange, error) {
	changes := []ValueChange{}

	// Load previous snapshot if exists
	oldSnapshot := make(map[string]int)
	if data, err := os.ReadFile(valueCacheFile); err == nil {
		json.Unmarshal(data, &oldSnapshot)
	}

	// If no old snapshot or very old, create new one and return empty
	if len(oldSnapshot) == 0 {
		saveSnapshot(currentValues, isSuperFlex)
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

		oldValue, existed := oldSnapshot[name]
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
				PlayerName:  dv.Name,
				Position:    dv.Position,
				OldValue:    oldValue,
				NewValue:    currentValue,
				Delta:       delta,
				DeltaPct:    deltaPct,
				IsRiser:     delta > 0,
				IsOwned:     ownedPlayers[name],
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

	// Update snapshot (daily)
	saveSnapshot(currentValues, isSuperFlex)

	return changes, nil
}

func saveSnapshot(values map[string]DynastyValue, isSuperFlex bool) {
	snapshot := make(map[string]int)
	for name, dv := range values {
		if isSuperFlex {
			snapshot[name] = dv.Value2QB
		} else {
			snapshot[name] = dv.Value1QB
		}
	}

	data, _ := json.Marshal(snapshot)
	os.WriteFile(valueCacheFile, data, 0644)
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
