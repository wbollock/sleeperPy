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

// buildFreeAgentCandidates returns a lightweight top-N-per-position free
// agent list for cross-league action generation. Simpler than the
// per-league page's free-agent builder (no roster-percent tiebreaking or
// upgrade pre-labeling) since findWaiverTargets does the upgrade comparison
// itself -- this only needs to supply candidates with tiers attached.
func buildFreeAgentCandidates(players map[string]interface{}, rostered map[string]bool, borisTiers map[string][][]string, perPositionLimit int) []PlayerRow {
	type candidate struct {
		row  PlayerRow
		tier int
	}
	byPos := make(map[string][]candidate)

	for pid, p := range players {
		if rostered[pid] {
			continue
		}
		pm, ok := p.(map[string]interface{})
		if !ok || pm["active"] == false {
			continue
		}
		pos, _ := pm["position"].(string)
		if pos != "QB" && pos != "RB" && pos != "WR" && pos != "TE" {
			continue // dashboard action feed only covers skill positions
		}
		team, _ := pm["team"].(string)
		if team == "" {
			continue // no NFL team = not realistically addable
		}

		name := getPlayerName(pm)
		tier := findTier(borisTiers[pos], name)
		if tier == 0 {
			continue
		}
		flexTier := 0
		if pos == "RB" || pos == "WR" || pos == "TE" {
			flexTier = findTier(borisTiers["FLX"], name)
		}
		injuryStatus, _ := pm["injury_status"].(string)
		depthChartOrder := 0
		if dco, ok := pm["depth_chart_order"].(float64); ok {
			depthChartOrder = int(dco)
		}

		byPos[pos] = append(byPos[pos], candidate{
			row: PlayerRow{
				Pos: pos, Name: name, Tier: tier, FlexTier: flexTier, IsFreeAgent: true,
				Team: team, InjuryStatus: injuryStatus, DepthChartOrder: depthChartOrder,
			},
			tier: tier,
		})
	}

	var out []PlayerRow
	for _, list := range byPos {
		sort.Slice(list, func(i, j int) bool { return list[i].tier < list[j].tier })
		if len(list) > perPositionLimit {
			list = list[:perPositionLimit]
		}
		for _, c := range list {
			out = append(out, c.row)
		}
	}
	return out
}

func buildWeeklyActions(league LeagueData) []Action {
	var actions []Action
	weekID := getCurrentWeekID()

	// All of the below need an active matchup week to make sense (there's no
	// "this week's lineup" to fix in the offseason).
	if league.HasMatchups {
		// 1. Bye week alerts (highest priority - a bye starter scores 0)
		actions = append(actions, findByeAlerts(league, weekID)...)

		// 2. Injury alerts (highest priority - starters only)
		actions = append(actions, findInjuryAlerts(league, weekID)...)

		// 3. Lineup optimization - swap starters with better bench players
		actions = append(actions, findStarterSwaps(league, weekID)...)

		// 4. Waiver wire opportunities
		actions = append(actions, findWaiverTargets(league, weekID)...)

		// 5. Handcuff alerts - same-team backup for an injured rostered player
		actions = append(actions, findHandcuffAlerts(league, weekID)...)
	}

	// 6. Trade opportunities (dynasty only)
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

// isSignificantInjury reports whether an injury_status value is severe
// enough to warrant surfacing an alert. "Questionable" is deliberately
// excluded -- it covers most of the active player pool on a given week and
// would make the alert noise rather than signal.
func isSignificantInjury(status string) bool {
	switch status {
	case "Out", "Doubtful", "IR":
		return true
	default:
		return false
	}
}

// actualPosOf returns a starter's real position (QB/RB/WR/TE) when it can be
// determined directly from PlayerRow.Pos, or "" when the row is in a FLEX/
// SUPER_FLEX slot and the real position isn't recorded on the row. Bench and
// free-agent rows always carry the real position already.
func actualPosOf(p PlayerRow) string {
	switch p.Pos {
	case "QB", "RB", "WR", "TE":
		return p.Pos
	default:
		return ""
	}
}

// findByeAlerts flags starters whose team has a bye this week. A bye
// starter scores zero regardless of tier, so this fires independent of any
// tier comparison and outranks ordinary swap suggestions.
func findByeAlerts(league LeagueData, weekID string) []Action {
	var actions []Action

	for _, starter := range league.Starters {
		if !starter.OnBye {
			continue
		}
		isFlexSlot := starter.Pos == "FLEX" || starter.Pos == "SUPER_FLEX"

		// Find the best available bench replacement (not also on bye).
		replacement := ""
		var bestTier float64
		for _, bench := range league.Bench {
			if bench.OnBye {
				continue
			}
			var benchTier float64
			if isFlexSlot {
				if bench.FlexTier <= 0 {
					continue
				}
				benchTier = float64(bench.FlexTier)
			} else {
				if bench.Pos != starter.Pos {
					continue
				}
				benchTier = parseTierFloat(bench.Tier)
				if benchTier == 0 {
					continue
				}
			}
			if replacement == "" || benchTier < bestTier {
				replacement = bench.Name
				bestTier = benchTier
			}
		}

		desc := fmt.Sprintf("%s is on bye this week", starter.Name)
		if replacement != "" {
			desc = fmt.Sprintf("Start %s over %s (bye)", replacement, starter.Name)
		}

		actions = append(actions, Action{
			Priority:    1,
			Category:    "bye",
			Title:       "Bye Week",
			Description: desc,
			Impact:      "Scores 0 if left in the lineup",
			Link:        fmt.Sprintf("#player-%s", normalizeAnchor(starter.Name)),
			WeekID:      weekID,
		})
	}

	return actions
}

// findInjuryAlerts flags starters whose injury_status makes them unlikely to
// play. Unlike findStarterSwaps, this isn't tier-driven -- an "Out" starter
// needs a lineup decision even if no tiered bench replacement exists yet.
func findInjuryAlerts(league LeagueData, weekID string) []Action {
	var actions []Action

	for _, starter := range league.Starters {
		if !isSignificantInjury(starter.InjuryStatus) {
			continue
		}
		actions = append(actions, Action{
			Priority:    1,
			Category:    "injury",
			Title:       "Injury Alert",
			Description: fmt.Sprintf("%s is listed as %s", starter.Name, starter.InjuryStatus),
			Impact:      "Confirm status before lineup lock",
			Link:        fmt.Sprintf("#player-%s", normalizeAnchor(starter.Name)),
			WeekID:      weekID,
		})
	}

	return actions
}

func findStarterSwaps(league LeagueData, weekID string) []Action {
	var actions []Action
	seen := make(map[string]bool)

	for _, starter := range league.Starters {
		isFlexSlot := starter.Pos == "FLEX" || starter.Pos == "SUPER_FLEX"

		// Boris Chen tiers only rank within a position (a Tier 4 RB isn't
		// worse than a Tier 3 WR just because 4 > 3 -- RB is deep at the top
		// this year, so plenty of clear starters land in Tier 4). A FLEX
		// starter's Tier is already FLX-based (set during FLEX re-ranking),
		// so it's comparable to bench players' FlexTier; anyone else needs
		// an exact position match.
		starterTier := parseTierFloat(starter.Tier)
		if starterTier == 0 {
			continue // Skip unranked starters
		}

		for _, bench := range league.Bench {
			var benchTier float64
			if isFlexSlot {
				if bench.FlexTier <= 0 {
					continue // not flex-eligible (RB/WR/TE) or unranked on the FLX table
				}
				benchTier = float64(bench.FlexTier)
			} else {
				if starter.Pos != bench.Pos {
					continue
				}
				benchTier = parseTierFloat(bench.Tier)
				if benchTier == 0 {
					continue // Skip unranked bench
				}
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
					Impact:      fmt.Sprintf("Tier %.0f vs Tier %.0f (+%.1f)", benchTier, starterTier, tierDiff),
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
			isFlexSlot := starter.Pos == "FLEX" || starter.Pos == "SUPER_FLEX"

			var faTierForComparison, starterTier float64
			if isFlexSlot {
				// Starter.Tier is already FLX-based for a FLEX slot; compare
				// against the FA's FLX tier, not their positional tier.
				if fa.FlexTier <= 0 {
					continue // FA isn't flex-eligible (RB/WR/TE) or unranked on the FLX table
				}
				faTierForComparison = float64(fa.FlexTier)
				starterTier = parseTierFloat(starter.Tier)
			} else {
				if fa.Pos != starter.Pos {
					continue
				}
				faTierForComparison = faTier
				starterTier = parseTierFloat(starter.Tier)
			}
			if starterTier == 0 {
				continue
			}

			tierDiff := starterTier - faTierForComparison // Lower tier number is better
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
					Impact:      fmt.Sprintf("Tier %.0f vs Tier %.0f (+%.1f)", faTierForComparison, starterTier, tierDiff),
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

// findHandcuffAlerts looks for a same-team, same-position free agent sitting
// one spot behind an injured rostered player on the depth chart -- the
// classic "handcuff" pickup (e.g. a starting RB's direct backup becomes a
// must-add when the starter is out). Requires DepthChartOrder on both sides,
// so it only fires when Sleeper has published a depth chart for that team.
func findHandcuffAlerts(league LeagueData, weekID string) []Action {
	var actions []Action
	seen := make(map[string]bool)

	injured := make([]PlayerRow, 0, len(league.Starters)+len(league.Bench))
	injured = append(injured, league.Starters...)
	injured = append(injured, league.Bench...)

	for _, p := range injured {
		if !isSignificantInjury(p.InjuryStatus) {
			continue
		}
		pos := actualPosOf(p)
		if pos == "" || p.Team == "" || p.DepthChartOrder <= 0 {
			continue
		}

		for _, fa := range league.TopFreeAgentsByValue {
			if fa.Team != p.Team || fa.Pos != pos {
				continue
			}
			if fa.DepthChartOrder != p.DepthChartOrder+1 {
				continue
			}
			key := fmt.Sprintf("handcuff-%s", fa.Name)
			if seen[key] {
				continue
			}
			seen[key] = true

			actions = append(actions, Action{
				Priority:    2,
				Category:    "handcuff",
				Title:       "Handcuff Alert",
				Description: fmt.Sprintf("%s now next in line behind injured %s", fa.Name, p.Name),
				Impact:      fmt.Sprintf("%s (%s) is %s", p.Name, pos, p.InjuryStatus),
				Link:        fmt.Sprintf("#fa-%s", normalizeAnchor(fa.Name)),
				WeekID:      weekID,
			})
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

// mergeActionsAcrossLeagues combines actions from the dashboard's cross-league
// feed that have the same category+description (e.g. the user owns the same
// two players in two leagues, producing the identical swap suggestion in
// both) into one entry citing every applicable league, instead of showing
// the same recommendation once per league.
func mergeActionsAcrossLeagues(actions []Action) []Action {
	type key struct{ category, description string }

	order := []key{}
	merged := make(map[key]Action)
	leagueNames := make(map[key][]string)

	for _, a := range actions {
		k := key{a.Category, a.Description}
		if _, exists := merged[k]; !exists {
			merged[k] = a
			order = append(order, k)
		}
		leagueNames[k] = append(leagueNames[k], a.LeagueName)
	}

	out := make([]Action, 0, len(order))
	for _, k := range order {
		a := merged[k]
		if names := leagueNames[k]; len(names) > 1 {
			a.LeagueName = strings.Join(names, ", ")
		}
		out = append(out, a)
	}
	return out
}

func getCurrentWeekID() string {
	// Get current NFL week (simplified - just use calendar week for now)
	now := time.Now()
	year, week := now.ISOWeek()
	return fmt.Sprintf("%d-W%d", year, week)
}
