// Rookie draft needs analyzer - suggests positions and archetypes to target
// Based on roster age curves, depth, positional scarcity, and pick inventory

package main

import (
	"fmt"
	"sort"
)

// Generate rookie draft needs analysis
func generateRookieDraftNeeds(league LeagueData, isPremium bool) DraftStrategy {
	if !isPremium || !league.IsDynasty {
		return DraftStrategy{}
	}

	strategy := DraftStrategy{
		Needs:           []RookieDraftNeed{},
		Recommendations: []string{},
	}

	// Summarize pick inventory
	strategy.PickInventory, strategy.CapitalLevel = summarizePickInventory(league.DraftPicks)

	// Analyze positional needs
	needs := analyzePositionalNeeds(league)

	// Sort by priority
	sort.Slice(needs, func(i, j int) bool {
		return needs[i].Priority < needs[j].Priority
	})

	strategy.Needs = needs

	// Determine overall approach
	strategy.OverallApproach = determineOverallApproach(league, needs)

	// Generate recommendations
	strategy.Recommendations = generateDraftRecommendations(league, strategy.OverallApproach, needs)

	return strategy
}

func summarizePickInventory(picks []DraftPick) (string, string) {
	if len(picks) == 0 {
		return "No picks", "Low"
	}

	firstRoundCount := 0
	pickSummary := []string{}

	for _, pick := range picks {
		if pick.Year == 2026 || pick.Year == 2027 {
			label := fmt.Sprintf("%d.%02d", pick.Round, 1) // Simplified
			pickSummary = append(pickSummary, label)

			if pick.Round == 1 {
				firstRoundCount++
			}
		}
	}

	// Determine capital level
	capitalLevel := "Low"
	if firstRoundCount >= 3 {
		capitalLevel = "High"
	} else if firstRoundCount >= 2 || len(picks) >= 4 {
		capitalLevel = "Medium"
	}

	summary := ""
	if len(pickSummary) > 0 {
		summary = fmt.Sprintf("%d picks", len(picks))
		if firstRoundCount > 0 {
			summary = fmt.Sprintf("%d 1st", firstRoundCount)
			if len(picks) > firstRoundCount {
				summary += fmt.Sprintf(", %d total", len(picks))
			}
		}
	}

	return summary, capitalLevel
}

func analyzePositionalNeeds(league LeagueData) []RookieDraftNeed {
	needs := []RookieDraftNeed{}

	// Count young assets by position (age < 25)
	youngAssets := make(map[string]int)
	totalAssets := make(map[string]int)
	positions := []string{"RB", "WR", "TE", "QB"}

	// Combine starters and bench
	allPlayers := append([]PlayerRow{}, league.Starters...)
	allPlayers = append(allPlayers, league.Bench...)

	for _, player := range allPlayers {
		if player.DynastyValue > 0 {
			totalAssets[player.Pos]++
			if player.Age > 0 && player.Age < 25 {
				youngAssets[player.Pos]++
			}
		}
	}

	// Analyze each position
	for _, pos := range positions {
		need := analyzePositionNeed(pos, youngAssets[pos], totalAssets[pos], league)
		if need.Priority > 0 {
			needs = append(needs, need)
		}
	}

	return needs
}

func analyzePositionNeed(position string, youngCount, totalCount int, league LeagueData) RookieDraftNeed {
	need := RookieDraftNeed{
		Position: position,
		Examples: []string{},
	}

	// Get positional value percentage
	pb := league.PositionalBreakdown
	totalValue := pb.QB + pb.RB + pb.WR + pb.TE
	var posValue int
	var posTarget float64 // Target percentage

	switch position {
	case "RB":
		posValue = pb.RB
		posTarget = 0.25 // 25% target
	case "WR":
		posValue = pb.WR
		posTarget = 0.35 // 35% target
	case "TE":
		posValue = pb.TE
		posTarget = 0.12 // 12% target
	case "QB":
		posValue = pb.QB
		posTarget = 0.15 // 15% target
	}

	posPercent := 0.0
	if totalValue > 0 {
		posPercent = float64(posValue) / float64(totalValue)
	}

	// Priority scoring (1-5, 1 = highest)
	priority := 5 // Default low priority

	// Factor 1: Low total depth
	if totalCount <= 2 {
		priority = min(priority, 1) // Critical need
	} else if totalCount <= 3 {
		priority = min(priority, 2)
	}

	// Factor 2: Lack of young assets
	if youngCount == 0 && totalCount > 0 {
		priority = min(priority, 2) // Aging position group
	} else if youngCount <= 1 {
		priority = min(priority, 3)
	}

	// Factor 3: Value percentage below target
	if posPercent < posTarget*0.7 {
		priority = min(priority, 2) // Significantly underweight
	} else if posPercent < posTarget*0.85 {
		priority = min(priority, 3)
	}

	// Factor 4: RB is always important in dynasty
	if position == "RB" && priority > 2 {
		priority = 3
	}

	// Build reasoning
	reasoning := buildReasoning(position, youngCount, totalCount, posPercent, posTarget)

	// Determine target archetype
	archetype, draftRange := determineArchetype(position, priority, league)

	// Count picks in range
	picksAvailable := countPicksInRange(league.DraftPicks, draftRange)

	// Add examples
	examples := getPositionExamples(position, archetype)

	need.Priority = priority
	need.Reasoning = reasoning
	need.TargetArchetype = archetype
	need.DraftRange = draftRange
	need.PicksAvailable = picksAvailable
	need.Examples = examples

	return need
}

func buildReasoning(pos string, young, total int, pct, target float64) string {
	reasons := []string{}

	if total <= 2 {
		reasons = append(reasons, "shallow depth")
	}

	if young == 0 && total > 0 {
		reasons = append(reasons, "no young assets")
	} else if young <= 1 {
		reasons = append(reasons, "aging position group")
	}

	if pct < target*0.7 {
		reasons = append(reasons, fmt.Sprintf("%.0f%% below target", (target-pct)*100))
	}

	if len(reasons) == 0 {
		return fmt.Sprintf("%d assets, %d under 25", total, young)
	}

	return fmt.Sprintf("%s (%d total, %d young)",
		concatenateWithCommas(reasons), total, young)
}

func determineArchetype(position string, priority int, league LeagueData) (string, string) {
	// Default archetypes and draft ranges
	archetypes := map[string]string{
		"RB": "Bell Cow RB",
		"WR": "Alpha WR",
		"TE": "Pass-Catching TE",
		"QB": "Dual-Threat QB",
	}

	ranges := map[int]string{
		1: "Early 1st",
		2: "Mid 1st",
		3: "Late 1st",
		4: "2nd Round",
		5: "3rd+",
	}

	archetype := archetypes[position]
	draftRange := ranges[priority]

	// Customize based on need
	if position == "RB" && priority <= 2 {
		archetype = "3-Down Workhorse"
	} else if position == "WR" && priority == 1 {
		archetype = "WR1 Upside"
	}

	return archetype, draftRange
}

func countPicksInRange(picks []DraftPick, draftRange string) int {
	count := 0
	for _, pick := range picks {
		if pick.Year == 2026 || pick.Year == 2027 {
			switch draftRange {
			case "Early 1st":
				if pick.Round == 1 {
					count++
				}
			case "Mid 1st", "Late 1st", "Mid-Late 1st":
				if pick.Round == 1 {
					count++
				}
			case "2nd Round", "2nd+":
				if pick.Round >= 2 {
					count++
				}
			}
		}
	}
	return count
}

func getPositionExamples(position, archetype string) []string {
	// Return example player types/archetypes
	examples := map[string][]string{
		"RB": {"Bijan Robinson", "Jahmyr Gibbs", "Blake Corum"},
		"WR": {"Marvin Harrison Jr", "Rome Odunze", "Malik Nabers"},
		"TE": {"Brock Bowers", "Sam LaPorta", "Dalton Kincaid"},
		"QB": {"Caleb Williams", "Jayden Daniels", "Bo Nix"},
	}

	if ex, ok := examples[position]; ok {
		return ex
	}
	return []string{}
}

func determineOverallApproach(league LeagueData, needs []RookieDraftNeed) string {
	// Determine draft strategy based on team context

	// Get user's rank
	userRank := 0
	avgAge := 0.0
	for _, pr := range league.PowerRankings {
		if pr.IsUserTeam {
			userRank = pr.ValueRank
			avgAge = pr.AvgAge
			break
		}
	}

	// Competing teams: BPA or position of need
	if userRank <= 4 {
		if len(needs) > 0 && needs[0].Priority <= 2 {
			return "Address Critical Need"
		}
		return "Best Player Available"
	}

	// Rebuilding: accumulate picks
	if userRank > 8 || avgAge > 28 {
		return "Trade Down for Capital"
	}

	// Middle teams: target RBs (short shelf life)
	if len(needs) > 0 {
		for _, need := range needs {
			if need.Position == "RB" && need.Priority <= 3 {
				return "RB Heavy"
			}
		}
	}

	return "Best Player Available"
}

func generateDraftRecommendations(league LeagueData, approach string, needs []RookieDraftNeed) []string {
	recs := []string{}

	// Approach-specific recommendations
	switch approach {
	case "Best Player Available":
		recs = append(recs, "Rank prospects independently of roster needs")
		recs = append(recs, "Take highest-value player at each pick")

	case "Address Critical Need":
		if len(needs) > 0 {
			recs = append(recs, fmt.Sprintf("Prioritize %s in early rounds", needs[0].Position))
			recs = append(recs, "Ensure you land at least one impact player at need position")
		}

	case "Trade Down for Capital":
		recs = append(recs, "Look to trade back from early picks")
		recs = append(recs, "Accumulate 2027 picks for future draft class")
		recs = append(recs, "Target volume over individual prospects")

	case "RB Heavy":
		recs = append(recs, "RB scarcity makes them premium assets")
		recs = append(recs, "Target 2-3 RBs across all rounds")
		recs = append(recs, "Don't reach, but prioritize RB in tiers")
	}

	// Need-specific recommendations
	if len(needs) > 0 {
		topNeed := needs[0]
		if topNeed.Priority == 1 {
			recs = append(recs, fmt.Sprintf("Critical: %s depth is a major roster hole", topNeed.Position))
		}

		// Multiple high-priority needs
		highPriorityCount := 0
		for _, need := range needs {
			if need.Priority <= 2 {
				highPriorityCount++
			}
		}

		if highPriorityCount >= 2 {
			recs = append(recs, "Multiple needs - consider trading for extra picks")
		}
	}

	return recs
}

func concatenateWithCommas(items []string) string {
	if len(items) == 0 {
		return ""
	}
	if len(items) == 1 {
		return items[0]
	}
	if len(items) == 2 {
		return items[0] + " and " + items[1]
	}

	result := ""
	for i, item := range items {
		if i == len(items)-1 {
			result += "and " + item
		} else {
			result += item + ", "
		}
	}
	return result
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
