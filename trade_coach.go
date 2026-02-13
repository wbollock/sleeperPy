// Trade negotiation coach - generates trade offers and messages
// Uses LLM to create compelling trade proposals based on team needs

package main

import (
	"fmt"
	"sort"
	"strings"
)

// Generate trade proposal for a specific target team
func generateTradeProposal(
	userRoster []PlayerRow,
	targetRoster []PlayerRow,
	targetTeamName string,
	userSurplus string,
	targetSurplus string,
	dynastyValues map[string]DynastyValue,
	isSuperFlex bool,
	premiumEnabled bool,
) TradeProposal {

	// Find user's surplus players
	yourOfferCandidates := findSurplusPlayers(userRoster, userSurplus, dynastyValues, isSuperFlex)

	// Find target's surplus players
	theirOfferCandidates := findSurplusPlayers(targetRoster, targetSurplus, dynastyValues, isSuperFlex)

	// Build balanced trade (within 10% value)
	yourOffer, theirReturn := buildBalancedTrade(yourOfferCandidates, theirOfferCandidates, 0.10)

	// Calculate value delta
	yourOfferValue := sumPlayerValues(yourOffer)
	theirReturnValue := sumPlayerValues(theirReturn)
	valueDelta := theirReturnValue - yourOfferValue

	// Determine fairness
	fairness := "Fair"
	if valueDelta > yourOfferValue/10 {
		fairness = "Big win"
	} else if valueDelta > yourOfferValue/20 {
		fairness = "Slight win"
	} else if valueDelta < -yourOfferValue/10 {
		fairness = "Overpay"
	}

	// Calculate impact scores
	winNowImpact := calculateWinNowImpact(yourOffer, theirReturn, dynastyValues)
	futureImpact := calculateFutureImpact(yourOffer, theirReturn, dynastyValues)

	// Risk assessment
	riskLevel := assessRisk(yourOffer, theirReturn)

	proposal := TradeProposal{
		TargetTeamName: targetTeamName,
		YourOffer:      yourOffer,
		TheirReturn:    theirReturn,
		ValueDelta:     valueDelta,
		Fairness:       fairness,
		RiskLevel:      riskLevel,
		WinNowImpact:   winNowImpact,
		FutureImpact:   futureImpact,
	}

	// Generate rationale
	proposal.Rationale = buildRationale(proposal, userSurplus, targetSurplus)

	// Generate draft message (LLM if premium, template otherwise)
	if premiumEnabled {
		proposal.DraftMessage = generateLLMTradeMessage(proposal)
	} else {
		proposal.DraftMessage = generateTemplateMessage(proposal)
	}

	return proposal
}

func findSurplusPlayers(roster []PlayerRow, position string, dynastyValues map[string]DynastyValue, isSuperFlex bool) []ProposalPlayer {
	candidates := []ProposalPlayer{}

	for _, player := range roster {
		if position == "" || player.Pos == position {
			dv := getDynastyValue(player.Name, dynastyValues, isSuperFlex)
			if dv > 0 {
				candidates = append(candidates, ProposalPlayer{
					Name:         player.Name,
					Position:     player.Pos,
					DynastyValue: dv,
					Tier:         fmt.Sprintf("%v", player.Tier),
				})
			}
		}
	}

	// Sort by value (descending)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].DynastyValue > candidates[j].DynastyValue
	})

	return candidates
}

func buildBalancedTrade(yourCandidates, theirCandidates []ProposalPlayer, tolerance float64) ([]ProposalPlayer, []ProposalPlayer) {
	if len(yourCandidates) == 0 || len(theirCandidates) == 0 {
		return nil, nil
	}

	// Simple 1-for-1 trade first
	your := yourCandidates[0]
	their := theirCandidates[0]

	// Try to find close match in value
	bestDiff := abs(your.DynastyValue - their.DynastyValue)
	bestTheir := their

	for _, candidate := range theirCandidates {
		diff := abs(your.DynastyValue - candidate.DynastyValue)
		if diff < bestDiff {
			bestDiff = diff
			bestTheir = candidate
		}
	}

	// Check if within tolerance
	avgValue := (your.DynastyValue + bestTheir.DynastyValue) / 2
	if float64(bestDiff)/float64(avgValue) <= tolerance {
		return []ProposalPlayer{your}, []ProposalPlayer{bestTheir}
	}

	// If no good 1-for-1, return top from each
	return []ProposalPlayer{your}, []ProposalPlayer{their}
}

func sumPlayerValues(players []ProposalPlayer) int {
	total := 0
	for _, p := range players {
		total += p.DynastyValue
	}
	return total
}

func calculateWinNowImpact(yourOffer, theirReturn []ProposalPlayer, dynastyValues map[string]DynastyValue) int {
	// Simplified: younger players hurt win-now, older help
	yourAge := estimateAverageAge(yourOffer)
	theirAge := estimateAverageAge(theirReturn)

	// If you're trading away young for old = win-now boost
	ageDiff := int((theirAge - yourAge) * 10)

	return clamp(ageDiff, -100, 100)
}

func calculateFutureImpact(yourOffer, theirReturn []ProposalPlayer, dynastyValues map[string]DynastyValue) int {
	// Simplified: value delta represents future impact
	valueDelta := sumPlayerValues(theirReturn) - sumPlayerValues(yourOffer)

	// Scale to -100 to +100
	impact := valueDelta / 50
	return clamp(impact, -100, 100)
}

func estimateAverageAge(players []ProposalPlayer) float64 {
	if len(players) == 0 {
		return 26.0
	}
	// Simplified estimation based on position
	totalAge := 0.0
	for _, p := range players {
		switch p.Position {
		case "QB":
			totalAge += 28.0
		case "RB":
			totalAge += 25.0
		case "WR":
			totalAge += 26.5
		case "TE":
			totalAge += 27.0
		default:
			totalAge += 26.0
		}
	}
	return totalAge / float64(len(players))
}

func assessRisk(yourOffer, theirReturn []ProposalPlayer) string {
	// High risk if trading away more valuable players
	yourValue := sumPlayerValues(yourOffer)
	theirValue := sumPlayerValues(theirReturn)

	if theirValue > yourValue*2 {
		return "High" // Big gamble
	} else if float64(yourValue) > float64(theirValue)*1.2 {
		return "Medium" // Overpaying
	}
	return "Low"
}

func buildRationale(proposal TradeProposal, userSurplus, targetSurplus string) string {
	parts := []string{}

	if userSurplus != "" && targetSurplus != "" {
		parts = append(parts, fmt.Sprintf("Trade your %s depth for their %s upgrade", userSurplus, targetSurplus))
	}

	if proposal.Fairness == "Big win" || proposal.Fairness == "Slight win" {
		parts = append(parts, fmt.Sprintf("Value advantage: +%d (%s)", proposal.ValueDelta, proposal.Fairness))
	}

	if proposal.WinNowImpact > 30 {
		parts = append(parts, "Boosts playoff chances this year")
	} else if proposal.FutureImpact > 30 {
		parts = append(parts, "Builds long-term value")
	}

	if len(parts) == 0 {
		return "Addresses positional needs"
	}

	return strings.Join(parts, ". ")
}

func generateTemplateMessage(proposal TradeProposal) string {
	yourPlayers := formatPlayerList(proposal.YourOffer)
	theirPlayers := formatPlayerList(proposal.TheirReturn)

	template := fmt.Sprintf(`Hey! I'm looking to upgrade at %s and noticed you have strong depth there.

Would you be interested in trading:
- You get: %s
- I get: %s

%s

Let me know if you'd like to discuss!`,
		proposal.TheirReturn[0].Position,
		yourPlayers,
		theirPlayers,
		proposal.Rationale,
	)

	return template
}

func generateLLMTradeMessage(proposal TradeProposal) string {
	// TODO: Integrate with actual LLM API in future
	// For now, return enhanced template

	yourPlayers := formatPlayerList(proposal.YourOffer)
	theirPlayers := formatPlayerList(proposal.TheirReturn)

	// More personalized message based on trade context
	opener := "Hey! I've been looking at both our rosters and think we could help each other out."

	if proposal.WinNowImpact > 30 {
		opener = "Hey! I'm making a push for the playoffs and wanted to reach out about a potential trade."
	} else if proposal.FutureImpact > 30 {
		opener = "Hey! I'm building for the future and think we might have a mutually beneficial trade."
	}

	message := fmt.Sprintf(`%s

Trade proposal:
- You receive: %s
- I receive: %s

%s

This looks like a fair deal for both sides (within %d dynasty points). Interested in discussing?`,
		opener,
		yourPlayers,
		theirPlayers,
		proposal.Rationale,
		abs(proposal.ValueDelta),
	)

	return message
}

func formatPlayerList(players []ProposalPlayer) string {
	names := []string{}
	for _, p := range players {
		names = append(names, p.Name)
	}
	return strings.Join(names, ", ")
}

func getDynastyValue(playerName string, dynastyValues map[string]DynastyValue, isSuperFlex bool) int {
	normalized := normalizeName(playerName)
	if dv, ok := dynastyValues[normalized]; ok {
		if isSuperFlex {
			return dv.Value2QB
		}
		return dv.Value1QB
	}
	return 0
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
