// Trade retrospective analysis - tracks who wins trades over time
// Compares dynasty values at trade time vs current values

package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type TradeSnapshot struct {
	TradeID        string // Unique ID for this trade
	Timestamp      time.Time
	LeagueID       string
	Team1          string
	Team2          string
	Team1Assets    []string // Player names
	Team2Assets    []string
	Team1ValueThen int // KTC value at trade time
	Team2ValueThen int
	Team1ValueNow  int // Current KTC value
	Team2ValueNow  int
	Winner         string // "Team1", "Team2", or "Even"
	ValueSwing     int    // Absolute change in delta
	DaysElapsed    int
}

// Cache directory for trade snapshots
const tradeCacheDir = "/tmp/sleeperpy_trades"
const retrospectiveSwingThreshold = 200

// Analyze past trades to see who won over time
func analyzeTradeRetrospective(transactions []Transaction, currentDynastyValues map[string]DynastyValue, isSuperFlex bool) []Transaction {
	// Ensure cache directory exists
	if err := os.MkdirAll(tradeCacheDir, 0755); err != nil {
		log.Printf("[ERROR] Failed to create trade snapshot cache dir: %v", err)
		return transactions
	}

	// For each trade, check if we have a snapshot
	// If not, create one with current values
	// If yes, compare current values to snapshot to see who won

	for i := range transactions {
		if transactions[i].Type != "trade" {
			continue
		}
		leagueID := transactions[i].LeagueID
		if leagueID == "" {
			leagueID = "unknown"
		}

		tradeID := buildTradeSnapshotID(transactions[i])
		leagueDir := filepath.Join(tradeCacheDir, sanitizePathSegment(leagueID))
		if err := os.MkdirAll(leagueDir, 0755); err != nil {
			log.Printf("[ERROR] Failed to create league trade snapshot dir: %v", err)
			continue
		}
		snapshotPath := filepath.Join(leagueDir, tradeID+".json")

		var snapshot TradeSnapshot
		if fileData, err := os.ReadFile(snapshotPath); err == nil {
			if err := json.Unmarshal(fileData, &snapshot); err != nil {
				log.Printf("[ERROR] Failed to unmarshal trade snapshot %s: %v", snapshotPath, err)
				continue
			}
		} else {
			// First sighting: capture trade-time baseline values.
			snapshot = TradeSnapshot{
				TradeID:        tradeID,
				Timestamp:      transactions[i].Timestamp,
				LeagueID:       leagueID,
				Team1:          transactions[i].Team1,
				Team2:          transactions[i].Team2,
				Team1Assets:    transactions[i].Team1Gave,
				Team2Assets:    transactions[i].Team2Gave,
				Team1ValueThen: transactions[i].Team1GaveValue,
				Team2ValueThen: transactions[i].Team2GaveValue,
			}
		}

		// Recalculate "now" values every request to evaluate change over time.
		snapshot.Team1ValueNow = calculateAssetValue(snapshot.Team1Assets, currentDynastyValues, isSuperFlex)
		snapshot.Team2ValueNow = calculateAssetValue(snapshot.Team2Assets, currentDynastyValues, isSuperFlex)
		snapshot.DaysElapsed = int(time.Since(snapshot.Timestamp).Hours() / 24)
		snapshot.Winner, snapshot.ValueSwing = calculateRetrospectiveWinner(snapshot)

		if snapshot.DaysElapsed > 0 && snapshot.Winner != "" {
			winnerGain := "No significant change"
			if snapshot.Winner != "Even" {
				winnerGain = fmt.Sprintf("+%d value vs trade day", snapshot.ValueSwing)
			}
			transactions[i].Retrospective = TradeRetrospective{
				Winner:      snapshot.Winner,
				ValueSwing:  snapshot.ValueSwing,
				DaysElapsed: snapshot.DaysElapsed,
				WinnerGain:  winnerGain,
			}
		}

		data, err := json.Marshal(snapshot)
		if err != nil {
			log.Printf("[ERROR] Failed to marshal trade snapshot: %v", err)
			continue
		}
		if err := os.WriteFile(snapshotPath, data, 0644); err != nil {
			log.Printf("[ERROR] Failed to write trade snapshot %s: %v", snapshotPath, err)
		}
	}

	return transactions
}

func calculateAssetValue(assets []string, dynastyValues map[string]DynastyValue, isSuperFlex bool) int {
	total := 0
	for _, asset := range assets {
		// Skip draft picks; they are tracked separately and do not map cleanly to player KTC.
		if strings.Contains(asset, "Round") {
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

func calculateRetrospectiveWinner(snapshot TradeSnapshot) (string, int) {
	// Positive means Team1 benefited more.
	advantageThen := snapshot.Team2ValueThen - snapshot.Team1ValueThen
	advantageNow := snapshot.Team2ValueNow - snapshot.Team1ValueNow
	swingForTeam1 := advantageNow - advantageThen

	if swingForTeam1 > retrospectiveSwingThreshold {
		return snapshot.Team1, swingForTeam1
	}
	if swingForTeam1 < -retrospectiveSwingThreshold {
		return snapshot.Team2, -swingForTeam1
	}
	return "Even", 0
}

func buildTradeSnapshotID(txn Transaction) string {
	team1Assets := append([]string(nil), txn.Team1Gave...)
	team2Assets := append([]string(nil), txn.Team2Gave...)
	sort.Strings(team1Assets)
	sort.Strings(team2Assets)

	base := strings.Join([]string{
		txn.LeagueID,
		txn.Team1,
		txn.Team2,
		fmt.Sprintf("%d", txn.Timestamp.Unix()),
		strings.Join(team1Assets, "|"),
		strings.Join(team2Assets, "|"),
	}, "||")

	hash := sha1.Sum([]byte(base))
	return hex.EncodeToString(hash[:10])
}

func sanitizePathSegment(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "unknown"
	}

	replacer := strings.NewReplacer("/", "_", "\\", "_", " ", "_", ":", "_")
	return replacer.Replace(s)
}
