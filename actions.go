// ABOUTME: Weekly action list generation for lineup optimization
// ABOUTME: Detects starter swaps, waiver targets, injury alerts, and trade opportunities

package main

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	TIER_THRESHOLD_SWAP   = 1.0 // Minimum tier difference to suggest swap
	TIER_THRESHOLD_WAIVER = 1.0 // Minimum tier difference to suggest waiver pickup
	MAX_ACTIONS           = 5   // Maximum number of actions to return
)

func buildWeeklyActions(league LeagueData) []Action {
	var actions []Action
	weekID := getCurrentWeekID()

	// 1. Injury alerts (highest priority - starters only)
	// TODO: Add injury status to PlayerRow in future
	// for _, starter := range league.Starters {
	//     if starter.InjuryStatus == "Out" || starter.InjuryStatus == "Doubtful" {
	//         // Create injury alert action
	//     }
	// }

	// 2. Lineup optimization (in-season only) - swap starters with better bench players
	if league.HasMatchups {
		actions = append(actions, findStarterSwaps(league, weekID)...)
	}

	// 3. Waiver wire opportunities
	if league.HasMatchups {
		actions = append(actions, findWaiverTargets(league, weekID)...)
	}

	// 4. Trade opportunities (dynasty only)
	if league.IsDynasty && len(league.TradeTargets) > 0 {
		action := findTradeOpportunity(league, weekID)
		if action.Title != "" {
			actions = append(actions, action)
		}
	}

	// Sort by priority (lower number = higher priority)
	sort.Slice(actions, func(i, j int) bool {
		return actions[i].Priority < actions[j].Priority
	})

	// Deduplicate: if same player mentioned multiple times, keep highest priority
	actions = deduplicateActions(actions)

	// Limit to max actions
	if len(actions) > MAX_ACTIONS {
		actions = actions[:MAX_ACTIONS]
	}

	return actions
}

func findStarterSwaps(league LeagueData, weekID string) []Action {
	var actions []Action
	seen := make(map[string]bool)

	for _, starter := range league.Starters {
		starterTier := parseTierFloat(starter.Tier)
		if starterTier == 0 {
			continue // Skip unranked starters
		}

		for _, bench := range league.Bench {
			benchTier := parseTierFloat(bench.Tier)
			if benchTier == 0 {
				continue // Skip unranked bench
			}

			// Only suggest if bench player is significantly better
			tierDiff := starterTier - benchTier // Lower tier number is better
			if tierDiff >= TIER_THRESHOLD_SWAP {
				// Check if we've already suggested this swap
				key := fmt.Sprintf("swap-%s-%s", bench.Name, starter.Name)
				if seen[key] {
					continue
				}
				seen[key] = true

				actions = append(actions, Action{
					Priority:    1,
					Category:    "swap",
					Title:       "Swap Starter",
					Description: fmt.Sprintf("Start %s over %s", bench.Name, starter.Name),
					Impact:      fmt.Sprintf("+%.1f tier upgrade", tierDiff),
					Link:        fmt.Sprintf("#player-%s", normalizeAnchor(bench.Name)),
					WeekID:      weekID,
				})
			}
		}
	}

	return actions
}

func findWaiverTargets(league LeagueData, weekID string) []Action {
	var actions []Action
	seen := make(map[string]bool)

	// Look at top free agents by value (already sorted in the league data)
	for _, fa := range league.TopFreeAgentsByValue {
		faTier := parseTierFloat(fa.Tier)
		if faTier == 0 {
			continue // Skip unranked free agents
		}

		// Check if this FA would upgrade any starter
		for _, starter := range league.Starters {
			// Must be same position
			if fa.Pos != starter.Pos {
				continue
			}

			starterTier := parseTierFloat(starter.Tier)
			if starterTier == 0 {
				continue
			}

			tierDiff := starterTier - faTier // Lower tier number is better
			if tierDiff >= TIER_THRESHOLD_WAIVER {
				key := fmt.Sprintf("waiver-%s", fa.Name)
				if seen[key] {
					continue
				}
				seen[key] = true

				actions = append(actions, Action{
					Priority:    2,
					Category:    "waiver",
					Title:       "Check Waiver Wire",
					Description: fmt.Sprintf("%s available (would start over %s)", fa.Name, starter.Name),
					Impact:      fmt.Sprintf("+%.1f tier upgrade", tierDiff),
					Link:        fmt.Sprintf("#fa-%s", normalizeAnchor(fa.Name)),
					WeekID:      weekID,
				})

				// Only suggest one waiver pickup per free agent
				break
			}
		}
	}

	return actions
}

func findTradeOpportunity(league LeagueData, weekID string) Action {
	// Check if there's a positional imbalance
	pb := league.PositionalBreakdown
	total := pb.QB + pb.RB + pb.WR + pb.TE
	if total == 0 {
		return Action{}
	}

	// Calculate percentage of each position
	rbPct := float64(pb.RB) / float64(total) * 100
	wrPct := float64(pb.WR) / float64(total) * 100
	tePct := float64(pb.TE) / float64(total) * 100

	// Identify surplus and deficit positions
	var surplus, deficit string

	if wrPct > 40 {
		surplus = "WR"
	} else if rbPct > 40 {
		surplus = "RB"
	} else if tePct > 25 {
		surplus = "TE"
	}

	if rbPct < 20 {
		deficit = "RB"
	} else if wrPct < 25 {
		deficit = "WR"
	} else if tePct < 10 {
		deficit = "TE"
	}

	if surplus != "" && deficit != "" && surplus != deficit {
		return Action{
			Priority:    3,
			Category:    "trade",
			Title:       "Consider Trade",
			Description: fmt.Sprintf("Surplus: %s depth, Need: %s upgrade", surplus, deficit),
			Impact:      fmt.Sprintf("%d trade targets available", len(league.TradeTargets)),
			Link:        "#trade-targets",
			WeekID:      weekID,
		}
	}

	return Action{}
}

func parseTierFloat(tier interface{}) float64 {
	switch v := tier.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case string:
		// Try to parse string tier like "2.1"
		var f float64
		fmt.Sscanf(v, "%f", &f)
		return f
	default:
		return 0
	}
}

func normalizeAnchor(name string) string {
	// Convert name to anchor format (lowercase, no spaces)
	return strings.ToLower(strings.ReplaceAll(name, " ", "-"))
}

func deduplicateActions(actions []Action) []Action {
	seen := make(map[string]bool)
	var result []Action

	for _, action := range actions {
		// Create key from category + description
		key := action.Category + ":" + action.Description
		if !seen[key] {
			seen[key] = true
			result = append(result, action)
		}
	}

	return result
}

func getCurrentWeekID() string {
	// Get current NFL week (simplified - just use calendar week for now)
	now := time.Now()
	year, week := now.ISOWeek()
	return fmt.Sprintf("%d-W%d", year, week)
}
