// ABOUTME: Trade fairness detection for dynasty leagues
// ABOUTME: Flags extreme value gaps ("fleeced" trades) while recognizing valid rebuild/compete strategies

package main

import (
	"fmt"
	"math"
)

const (
	FLEECED_THRESHOLD = 15.0 // Q8: ~15% value delta (aggressive detection, iterate as needed)
	FAIR_THRESHOLD    = 5.0  // Below this is considered fair
)

func calculateTradeFairness(transaction Transaction, allRosterValues map[string]int) TradeFairness {
	// Skip if no dynasty values
	if transaction.Team1GaveValue == 0 && transaction.Team2GaveValue == 0 {
		return TradeFairness{
			Winner:       "Fair",
			DisplayBadge: "",
		}
	}

	// Calculate delta
	team1Gave := transaction.Team1GaveValue
	team2Gave := transaction.Team2GaveValue
	delta := int(math.Abs(float64(team1Gave - team2Gave)))

	// Determine winner
	winner := "Fair"
	winnerTeam := ""
	var deltaPct float64

	if team1Gave == 0 && team2Gave == 0 {
		// No values, can't determine fairness
		return TradeFairness{
			Winner:       "Fair",
			DisplayBadge: "",
		}
	}

	if team1Gave > team2Gave {
		winner = "Team2"
		winnerTeam = transaction.Team2
	} else if team2Gave > team1Gave {
		winner = "Team1"
		winnerTeam = transaction.Team1
	}

	// Calculate delta as % of smaller trade side (more generous)
	// This prevents small trades from being flagged as unfair
	smallerSide := team1Gave
	if team2Gave > 0 && team2Gave < team1Gave {
		smallerSide = team2Gave
	}
	if smallerSide > 0 {
		deltaPct = float64(delta) / float64(smallerSide) * 100
	}

	// Determine if fleeced
	fleeced := deltaPct >= FLEECED_THRESHOLD

	// Context detection (Q7: hybrid approach)
	context := ""
	if deltaPct < FAIR_THRESHOLD {
		context = "Fair trade"
	} else if deltaPct < FLEECED_THRESHOLD {
		// Moderate gap - could be strategic
		context = fmt.Sprintf("%s won on value", winnerTeam)

		// TODO: Add rebuild/compete detection
		// if isRebuildingStrategy(transaction) {
		//     context = fmt.Sprintf("%s rebuilding - acquiring future value", winnerTeam)
		// } else if isCompetingStrategy(transaction) {
		//     context = fmt.Sprintf("%s competing - acquired win-now pieces", winnerTeam)
		// }
	} else {
		// Fleeced
		context = "Extreme value gap - verify trade validity"
	}

	// Display badge (Q9: subtle)
	badge := ""
	if fleeced {
		badge = fmt.Sprintf("%s +%.0f%% 🔴", winnerTeam, deltaPct)
	} else if deltaPct >= FAIR_THRESHOLD {
		badge = fmt.Sprintf("%s +%.0f%%", winnerTeam, deltaPct)
	} else {
		badge = "Fair trade"
	}

	return TradeFairness{
		Winner:        winner,
		ValueDelta:    delta,
		ValueDeltaPct: deltaPct,
		Fleeced:       fleeced,
		Context:       context,
		WinnerTeam:    winnerTeam,
		DisplayBadge:  badge,
	}
}

// Helper: detect rebuilding strategy
// TODO: Implement heuristics:
// - Winner received mostly picks + young players (age < 24)
// - Winner gave away older players (age > 26)
// - Winner's roster avg age decreased significantly
func isRebuildingStrategy(transaction Transaction) bool {
	// Placeholder for future implementation
	return false
}

// Helper: detect competing strategy
// TODO: Implement heuristics:
// - Winner received proven starters (top-24 positional players)
// - Winner gave away picks + young players
// - Winner's roster avg age increased
func isCompetingStrategy(transaction Transaction) bool {
	// Placeholder for future implementation
	return false
}
