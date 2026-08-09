// Trade negotiation coach - generates trade offers and messages
// Uses LLM to create compelling trade proposals based on team needs

package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
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
	if premiumEnabled && hasOpenRouterKey() {
		proposal.DraftMessage = generateLLMTradeMessage(proposal, userSurplus, targetSurplus)
	} else {
		proposal.DraftMessage = generateTemplateMessage(proposal)
	}

	return proposal
}

// buildRosterValueRows builds skill-position PlayerRows with dynasty value
// attached, for callers that only care about position + value (not tier).
// Deliberately not buildRowsWithPositions: that function routes anything
// without a tier match into an "unranked" bucket that loses the player's
// real position (set to "?"), which is fine for its tier-table UI but wrong
// for a pure value calculation like calculatePositionalKTC.
func buildRosterValueRows(playerIDs []string, players map[string]interface{}, dynastyValues map[string]DynastyValue, isSuperFlex bool) []PlayerRow {
	rows := make([]PlayerRow, 0, len(playerIDs))
	for _, pid := range playerIDs {
		p, ok := players[pid].(map[string]interface{})
		if !ok {
			continue
		}
		pos, _ := p["position"].(string)
		if pos != "QB" && pos != "RB" && pos != "WR" && pos != "TE" {
			continue // calculatePositionalKTC only tracks these positions
		}
		rows = append(rows, PlayerRow{Pos: pos, Name: getPlayerName(p)})
	}
	enrichRowsWithDynastyValues(rows, dynastyValues, isSuperFlex)
	return rows
}

// buildDashboardTradeAction finds the single best trade opportunity for a
// league (if any) and formats it as a dashboard Action. Dynasty-value-based,
// so it works regardless of Boris Chen tier availability.
func buildDashboardTradeAction(leagueID, leagueName string, userRoster map[string]interface{}, rosters []map[string]interface{}, players map[string]interface{}, dynastyValues map[string]DynastyValue, isSuperFlex bool) *Action {
	userRosterID, _ := userRoster["roster_id"].(float64)

	userNames := map[string]string{}
	if leagueUsers, err := appProvider.FetchLeagueUsers(leagueID); err == nil {
		for _, u := range leagueUsers {
			uid, _ := u["user_id"].(string)
			if uid == "" {
				continue
			}
			if dn, ok := u["display_name"].(string); ok && dn != "" {
				userNames[uid] = dn
			} else if un, ok := u["username"].(string); ok {
				userNames[uid] = un
			}
		}
	}

	allRosters := make(map[int][]PlayerRow)
	teamNames := make(map[int]string)
	for _, r := range rosters {
		rosterID, _ := r["roster_id"].(float64)
		rows := buildRosterValueRows(toStringSlice(r["players"]), players, dynastyValues, isSuperFlex)
		allRosters[int(rosterID)] = rows

		teamName := fmt.Sprintf("Team %d", int(rosterID))
		if ownerID, ok := r["owner_id"].(string); ok {
			if name, exists := userNames[ownerID]; exists && name != "" {
				teamName = name
			}
		}
		if metadata, ok := r["metadata"].(map[string]interface{}); ok {
			if tn, ok := metadata["team_name"].(string); ok && tn != "" {
				teamName = tn
			}
		}
		teamNames[int(rosterID)] = teamName
	}

	userFullRoster, ok := allRosters[int(userRosterID)]
	if !ok || len(userFullRoster) == 0 {
		return nil
	}

	targets := findTradeTargets(userFullRoster, allRosters, teamNames, int(userRosterID))
	if len(targets) == 0 {
		return nil
	}
	target := targets[0]

	targetRosterID := -1
	for rosterID, name := range teamNames {
		if name == target.TeamName {
			targetRosterID = rosterID
			break
		}
	}
	targetRoster, ok := allRosters[targetRosterID]
	if !ok {
		return nil
	}

	// premiumEnabled=false: always use the template draft message, never the
	// LLM path -- the dashboard action feed only needs the offer/return/
	// fairness data, not a drafted message, so there's no reason to spend an
	// LLM call (or add a network dependency) generating one.
	proposal := generateTradeProposal(userFullRoster, targetRoster, target.TeamName, target.YourSurplus, target.TheirSurplus, dynastyValues, isSuperFlex, false)
	if len(proposal.YourOffer) == 0 || len(proposal.TheirReturn) == 0 {
		return nil
	}

	return &Action{
		Priority:    3,
		Category:    "trade",
		Title:       "Trade Opportunity",
		Description: fmt.Sprintf("Trade %s for %s (%s)", formatPlayerList(proposal.YourOffer), formatPlayerList(proposal.TheirReturn), target.TeamName),
		Impact:      fmt.Sprintf("%s, %s risk", proposal.Fairness, proposal.RiskLevel),
		Link:        "#trade-targets",
		LeagueID:    leagueID,
		LeagueName:  leagueName,
	}
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

func generateLLMTradeMessage(proposal TradeProposal, userSurplus, targetSurplus string) string {
	yourPlayers := formatPlayerList(proposal.YourOffer)
	theirPlayers := formatPlayerList(proposal.TheirReturn)

	// Safe fallback template if LLM is unavailable or fails.
	fallback := fmt.Sprintf(`Hey! I think we might match up well on a trade.

I can send: %s
I'd be looking for: %s

I have surplus at %s and you have surplus at %s, so this could help both sides.
%s

If this framework works for you, I can adjust details.`,
		yourPlayers,
		theirPlayers,
		emptyToNA(userSurplus),
		emptyToNA(targetSurplus),
		proposal.Rationale,
	)

	if !hasOpenRouterKey() {
		return fallback
	}

	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	messages := []openRouterMessage{
		{
			Role: "system",
			Content: "You are writing a concise fantasy football trade message. " +
				"Be polite, direct, and specific. Do not overhype. Keep it under 120 words.",
		},
		{
			Role: "user",
			Content: fmt.Sprintf(
				"Write a trade message to %s.\n"+
					"My surplus: %s\n"+
					"Their surplus: %s\n"+
					"I send: %s\n"+
					"I receive: %s\n"+
					"Rationale: %s\n"+
					"Fairness: %s\n"+
					"Risk: %s\n"+
					"Use plain language and end with a simple question.",
				proposal.TargetTeamName,
				emptyToNA(userSurplus),
				emptyToNA(targetSurplus),
				yourPlayers,
				theirPlayers,
				proposal.Rationale,
				proposal.Fairness,
				proposal.RiskLevel,
			),
		},
	}

	msg, err := callOpenRouter(ctx, messages, 180)
	if err != nil || strings.TrimSpace(msg) == "" {
		debugLog("[DEBUG] trade message LLM fallback: %v", err)
		return fallback
	}
	return strings.TrimSpace(msg)
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

func emptyToNA(s string) string {
	if strings.TrimSpace(s) == "" {
		return "N/A"
	}
	return s
}
