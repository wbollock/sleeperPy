// Trade retrospective analysis - tracks who wins trades over time
// Compares dynasty values at trade time vs current values

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type TradeSnapshot struct {
	TradeID         string    // Unique ID for this trade
	Timestamp       time.Time
	LeagueID        string
	Team1           string
	Team2           string
	Team1Assets     []string // Player names
	Team2Assets     []string
	Team1ValueThen  int // KTC value at trade time
	Team2ValueThen  int
	Team1ValueNow   int // Current KTC value
	Team2ValueNow   int
	Winner          string // "Team1", "Team2", or "Even"
	ValueSwing      int    // Absolute change in delta
	DaysElapsed     int
}

// Cache directory for trade snapshots
const tradeCacheDir = "/tmp/sleeperpy_trades"

// Analyze past trades to see who won over time
func analyzeTradeRetrospective(transactions []Transaction, currentDynastyValues map[string]DynastyValue, isSuperFlex bool) []Transaction {
	// Ensure cache directory exists
	os.MkdirAll(tradeCacheDir, 0755)

	// For each trade, check if we have a snapshot
	// If not, create one with current values
	// If yes, compare current values to snapshot to see who won

	for i := range transactions {
		if transactions[i].Type != "trade" {
			continue
		}

		// Generate trade ID (simple hash of teams + timestamp)
		tradeID := fmt.Sprintf("%s-%s-%d",
			transactions[i].Team1,
			transactions[i].Team2,
			transactions[i].Timestamp.Unix())

		// Check if snapshot exists
		snapshotPath := filepath.Join(tradeCacheDir, tradeID+".json")

		var snapshot TradeSnapshot
		if fileData, err := os.ReadFile(snapshotPath); err == nil {
			// Snapshot exists - load it
			json.Unmarshal(fileData, &snapshot)

			// Update current values
			snapshot.Team1ValueNow = calculateAssetValue(transactions[i].Team1Gave, currentDynastyValues, isSuperFlex)
			snapshot.Team2ValueNow = calculateAssetValue(transactions[i].Team2Gave, currentDynastyValues, isSuperFlex)
			snapshot.DaysElapsed = int(time.Since(snapshot.Timestamp).Hours() / 24)

			// Determine winner
			team1NetChange := snapshot.Team2ValueNow - snapshot.Team1ValueNow
			team2NetChange := snapshot.Team1ValueNow - snapshot.Team2ValueNow

			if team1NetChange > team2NetChange+200 {
				snapshot.Winner = transactions[i].Team1
				snapshot.ValueSwing = team1NetChange - team2NetChange
			} else if team2NetChange > team1NetChange+200 {
				snapshot.Winner = transactions[i].Team2
				snapshot.ValueSwing = team2NetChange - team1NetChange
			} else {
				snapshot.Winner = "Even"
				snapshot.ValueSwing = 0
			}

			// Attach retrospective to transaction
			transactions[i].Retrospective = TradeRetrospective{
				Winner:      snapshot.Winner,
				ValueSwing:  snapshot.ValueSwing,
				DaysElapsed: snapshot.DaysElapsed,
				WinnerGain:  fmt.Sprintf("+%d value", snapshot.ValueSwing),
			}

			// Save updated snapshot
			data, _ := json.Marshal(snapshot)
			os.WriteFile(snapshotPath, data, 0644)

		} else {
			// No snapshot - create initial one
			snapshot = TradeSnapshot{
				TradeID:        tradeID,
				Timestamp:      transactions[i].Timestamp,
				Team1:          transactions[i].Team1,
				Team2:          transactions[i].Team2,
				Team1Assets:    transactions[i].Team1Gave,
				Team2Assets:    transactions[i].Team2Gave,
				Team1ValueThen: transactions[i].Team1GaveValue,
				Team2ValueThen: transactions[i].Team2GaveValue,
				Team1ValueNow:  transactions[i].Team1GaveValue,
				Team2ValueNow:  transactions[i].Team2GaveValue,
				Winner:         "TBD",
				ValueSwing:     0,
				DaysElapsed:    0,
			}

			// Save snapshot
			data, _ := json.Marshal(snapshot)
			os.WriteFile(snapshotPath, data, 0644)
		}
	}

	return transactions
}

func calculateAssetValue(assets []string, dynastyValues map[string]DynastyValue, isSuperFlex bool) int {
	total := 0
	for _, asset := range assets {
		// Skip draft picks (they contain "Round")
		if len(asset) > 0 && (asset[0] == '2' || asset[0] == '3') {
			// Likely a draft pick like "2025 Round 1"
			continue
		}
		normName := normalizeName(asset)
		if dv, exists := dynastyValues[normName]; exists {
			if isSuperFlex {
				total += dv.Value2QB
			} else {
				total += dv.Value1QB
			}
		}
	}
	return total
}
