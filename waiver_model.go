// Advanced waiver wire model - scores free agents and suggests FAAB bids
// Considers tier delta, positional scarcity, and playoff schedule

package main

import (
	"fmt"
	"sort"
)

// Generate waiver recommendations for a league
func generateWaiverRecommendations(
	league LeagueData,
	freeAgentsByPos map[string][]PlayerRow,
	maxRecommendations int,
	isPremium bool,
) []WaiverRecommendation {

	if !isPremium {
		return nil // Feature gated to premium users
	}

	recommendations := []WaiverRecommendation{}

	// Analyze each free agent
	for pos, freeAgents := range freeAgentsByPos {
		for _, fa := range freeAgents {
			if fa.Tier == nil || fa.Tier == "" {
				continue // Skip unranked players
			}

			// Calculate score for this free agent
			rec := scoreWaiverTarget(fa, league, pos, len(freeAgents))
			if rec.Score > 0 {
				recommendations = append(recommendations, rec)
			}
		}
	}

	// Sort by score (descending)
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].Score > recommendations[j].Score
	})

	// Limit to top N
	if len(recommendations) > maxRecommendations {
		recommendations = recommendations[:maxRecommendations]
	}

	return recommendations
}

func scoreWaiverTarget(fa PlayerRow, league LeagueData, position string, availableAtPos int) WaiverRecommendation {
	score := 0
	tierDelta := 0.0
	impactType := "Depth Add"
	rationale := ""
	role := classifyWaiverRole(fa, impactType)
	usageSignal := usageSignal(fa.RosterPercent)

	faTier := parseTierFloat(fa.Tier)
	if faTier == 0 {
		return WaiverRecommendation{} // Skip unranked
	}

	// 1. Check if would upgrade a starter (biggest impact)
	worstStarterTier := 99.0
	worstStarterName := ""
	for _, starter := range league.Starters {
		if starter.Pos == position || (fa.IsFlex && starter.IsFlex) {
			starterTier := parseTierFloat(starter.Tier)
			if starterTier > worstStarterTier {
				worstStarterTier = starterTier
				worstStarterName = starter.Name
			}
		}
	}

	if worstStarterTier < 90 && faTier < worstStarterTier {
		tierDelta = worstStarterTier - faTier
		score += int(tierDelta * 30) // Major points for starter upgrades
		impactType = "Starter Upgrade"
		rationale = fmt.Sprintf("Would replace %s (tier %.1f → %.1f)", worstStarterName, worstStarterTier, faTier)
	}

	// 2. Check if would upgrade bench
	if score == 0 {
		worstBenchTier := 99.0
		worstBenchName := ""
		for _, bench := range league.Bench {
			if bench.Pos == position {
				benchTier := parseTierFloat(bench.Tier)
				if benchTier == 0 {
					benchTier = 99.0 // Unranked bench
				}
				if benchTier > worstBenchTier {
					worstBenchTier = benchTier
					worstBenchName = bench.Name
				}
			}
		}

		if worstBenchTier < 90 && faTier < worstBenchTier {
			tierDelta = worstBenchTier - faTier
			score += int(tierDelta * 15) // Moderate points for bench upgrades
			impactType = "Depth Add"
			rationale = fmt.Sprintf("Better than bench player %s", worstBenchName)
		}
	}

	// 3. Positional scarcity bonus
	scarcityScore := 0
	if availableAtPos < 5 {
		scarcityScore = 20 // Very scarce position
	} else if availableAtPos < 10 {
		scarcityScore = 10 // Somewhat scarce
	} else if availableAtPos < 15 {
		scarcityScore = 5 // Slight scarcity
	}
	score += scarcityScore

	// 3b. Usage bonus from roster percentage (proxy for role/market confidence)
	if fa.RosterPercent >= 70 {
		score += 15
	} else if fa.RosterPercent >= 40 {
		score += 8
	} else if fa.RosterPercent >= 20 {
		score += 4
	}

	// 4. Dynasty value bonus (if dynasty league)
	if league.IsDynasty && fa.DynastyValue > 0 {
		if fa.DynastyValue > 1000 {
			score += 20 // High dynasty value
		} else if fa.DynastyValue > 500 {
			score += 10 // Moderate dynasty value
		}
	}

	// 5. Tier quality bonus (elite players worth more)
	if faTier <= 1.5 {
		score += 25 // Elite tier
	} else if faTier <= 3.0 {
		score += 15 // Good tier
	} else if faTier <= 5.0 {
		score += 5 // Decent tier
	}

	// 6. Check for breakout potential (young + value)
	if league.IsDynasty && fa.Age > 0 && fa.Age < 25 && fa.DynastyValue > 300 {
		score += 15
		if impactType == "Depth Add" {
			impactType = "Lottery Ticket"
			rationale = fmt.Sprintf("Young breakout candidate (age %d, value %d)", fa.Age, fa.DynastyValue)
		}
	}

	// Determine priority based on score
	priority := "Low"
	if score >= 70 {
		priority = "High"
	} else if score >= 40 {
		priority = "Medium"
	}

	// Calculate suggested FAAB bid (percentage of budget)
	suggestedBid := calculateFAABBid(score, impactType, league.HasMatchups)

	// Build rationale if not set
	if rationale == "" {
		if faTier <= 3.0 {
			rationale = fmt.Sprintf("Strong tier ranking (%.1f), %s", faTier, usageSignal)
		} else {
			rationale = fmt.Sprintf("Depth option at %s, %s", position, usageSignal)
		}
	}
	if role == "" {
		role = classifyWaiverRole(fa, impactType)
	}

	return WaiverRecommendation{
		Player:           fa,
		Score:            score,
		Priority:         priority,
		SuggestedBid:     suggestedBid,
		Rationale:        rationale,
		ImpactType:       impactType,
		Role:             role,
		UsageSignal:      usageSignal,
		TierDelta:        tierDelta,
		PositionScarcity: availableAtPos,
	}
}

func calculateFAABBid(score int, impactType string, inSeason bool) int {
	// Base bid percentage
	bid := 0

	switch impactType {
	case "Starter Upgrade":
		if score >= 80 {
			bid = 40 // 40% of budget for elite starter upgrade
		} else if score >= 70 {
			bid = 25
		} else {
			bid = 15
		}
	case "Depth Add":
		if score >= 60 {
			bid = 10
		} else if score >= 40 {
			bid = 5
		} else {
			bid = 2
		}
	case "Lottery Ticket":
		bid = 8 // Young upside play
	}

	// Reduce bids in offseason (less urgency)
	if !inSeason {
		bid = bid / 2
		if bid < 1 && score > 30 {
			bid = 1
		}
	}

	return bid
}

func classifyWaiverRole(fa PlayerRow, impactType string) string {
	if impactType == "Starter Upgrade" {
		return "Immediate Starter"
	}
	if impactType == "Lottery Ticket" {
		return "Upside Stash"
	}
	if fa.RosterPercent >= 60 {
		return "High-usage Depth"
	}
	if fa.RosterPercent >= 25 {
		return "Rotational Depth"
	}
	return "Bench Stash"
}

func usageSignal(rosterPercent float64) string {
	if rosterPercent >= 70 {
		return "strong usage signal"
	}
	if rosterPercent >= 40 {
		return "stable usage signal"
	}
	if rosterPercent >= 20 {
		return "speculative usage signal"
	}
	return "low usage signal"
}

// Count starters at each position for depth analysis
func countStartersByPosition(starters []PlayerRow) map[string]int {
	counts := make(map[string]int)
	for _, s := range starters {
		counts[s.Pos]++
	}
	return counts
}

// Calculate positional need score (higher = more need)
func calculatePositionalNeed(position string, league LeagueData) int {
	need := 0

	// Count how many starters at this position
	starterCount := 0
	benchCount := 0
	for _, s := range league.Starters {
		if s.Pos == position {
			starterCount++
		}
	}
	for _, b := range league.Bench {
		if b.Pos == position {
			benchCount++
		}
	}

	totalDepth := starterCount + benchCount

	// Shallow depth = higher need
	if totalDepth <= 2 {
		need = 30
	} else if totalDepth <= 3 {
		need = 20
	} else if totalDepth <= 4 {
		need = 10
	}

	// Dynasty: RB is always scarce
	if league.IsDynasty && position == "RB" {
		need += 15
	}

	return need
}
