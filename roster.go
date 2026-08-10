// ABOUTME: Roster and player processing functions for SleeperPy
// ABOUTME: Includes functions for building player rows, calculating tiers, and win probabilities

package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// enrichRowsWithByeWeek sets OnBye for any row whose Team has a bye in
// currentWeek. Safe to call with a nil/empty byeWeeks map (e.g. schedule
// fetch failed) -- every row simply stays OnBye=false.
func enrichRowsWithByeWeek(rows []PlayerRow, byeWeeks map[string]int, currentWeek int) {
	if len(byeWeeks) == 0 || currentWeek <= 0 {
		return
	}
	for i := range rows {
		if rows[i].Team == "" {
			continue
		}
		rows[i].OnBye = byeWeeks[rows[i].Team] == currentWeek
	}
}

// enrichRowsWithADPTier sets ADPTier on each row from a secondary,
// independently-sourced tier map (see fetchADPTiers) for side-by-side
// comparison against the primary Boris Chen Tier -- only meaningful when
// the user has TierSource "both" in settings. A FLEX/SUPER_FLEX starter
// uses the FLX bucket, same as the primary Tier does for that slot;
// everything else uses its own positional bucket.
func enrichRowsWithADPTier(rows []PlayerRow, adpTiers map[string][][]string) {
	if len(adpTiers) == 0 {
		return
	}
	for i := range rows {
		pos := rows[i].Pos
		switch pos {
		case "FLEX", "SUPER_FLEX", "SUPERFLEX":
			pos = "FLX"
		case "DEF":
			pos = "DST"
		}
		name := stripHTML(rows[i].Name)
		if tier := findTier(adpTiers[pos], name); tier > 0 {
			rows[i].ADPTier = tier
		}
	}
}

func buildRowsWithPositions(ids []string, players map[string]interface{}, tiers map[string][][]string, isStarter bool, rosterPositions []string, irList []string, bestOtherTier map[string]int) ([]PlayerRow, []PlayerRow, []int) {
	rows := []PlayerRow{}
	unranked := []PlayerRow{}
	tierNums := []int{}
	// For bench, mark as swap candidate if this player is better than any starter at same position
	for idx, pid := range ids {
		p, ok := players[pid].(map[string]interface{})
		if !ok {
			continue
		}

		// Use league roster position if available, otherwise use player's actual position
		pos := ""
		if isStarter && idx < len(rosterPositions) {
			pos = rosterPositions[idx]
			// Handle bench positions
			if pos == "BN" {
				pos, _ = p["position"].(string)
			}
		} else {
			pos, _ = p["position"].(string)
		}
		name := getPlayerName(p)

		// For FLEX and SUPER_FLEX, use actual position for tier lookup
		actualPos, _ := p["position"].(string)
		lookupPos := pos
		if pos == "FLEX" || pos == "SUPER_FLEX" {
			lookupPos = actualPos
		}
		// Always use DST for DEF/DST for Boris Chen mapping
		if lookupPos == "DEF" {
			lookupPos = "DST"
		}

		tier := findTier(tiers[lookupPos], name)

		// FLX tier, keyed by the player's real position (not roster slot), so
		// a FLEX starter and an RB/WR/TE bench player can be compared on the
		// same scale. Positional tiers (e.g. RB Tier 4 vs WR Tier 3) are NOT
		// comparable to each other -- Boris Chen's tiers only rank within a
		// position, not across positions.
		flexTier := 0
		if actualPos == "RB" || actualPos == "WR" || actualPos == "TE" {
			flexTier = findTier(tiers["FLX"], name)
		}

		// IR indicator will be added to displayName
		displayName := name
		// IR indicator: if player is in irList
		for _, irid := range irList {
			if irid == pid {
				displayName += ` <span style="color:#ff7b7b;font-size:0.95em;">(IR)</span>`
				break
			}
		}

		isWorse := false
		shouldSwapIn := false
		if isStarter && bestOtherTier != nil && tier > 0 {
			// For starters, highlight if there is a bench player with a better tier
			if best, exists := bestOtherTier[lookupPos]; exists && best > 0 && best < tier {
				isWorse = true
			}
		}
		if !isStarter && bestOtherTier != nil && tier > 0 {
			// For bench, highlight if this player is better than any starter at same position
			if worst, exists := bestOtherTier[lookupPos]; exists && worst > 0 && tier < worst {
				shouldSwapIn = true
			}
		}
		// Get player age
		age := 0
		if ageVal, ok := p["age"].(float64); ok {
			age = int(ageVal)
		}

		team, _ := p["team"].(string)
		injuryStatus, _ := p["injury_status"].(string)
		depthChartOrder := 0
		if dco, ok := p["depth_chart_order"].(float64); ok {
			depthChartOrder = int(dco)
		}

		if tier > 0 {
			rows = append(rows, PlayerRow{Pos: pos, Name: displayName, Tier: tier, FlexTier: flexTier, IsTierWorseThanBench: isWorse, ShouldSwapIn: shouldSwapIn, Age: age, Team: team, InjuryStatus: injuryStatus, DepthChartOrder: depthChartOrder})
			tierNums = append(tierNums, tier)
		} else {
			unranked = append(unranked, PlayerRow{Pos: pos, Name: displayName, Tier: "Not Ranked", FlexTier: flexTier, IsTierWorseThanBench: false, ShouldSwapIn: false, Age: age, Team: team, InjuryStatus: injuryStatus, DepthChartOrder: depthChartOrder})
		}
	}
	return rows, unranked, tierNums
}

// splitRosterForTiering returns a roster's starter IDs, bench IDs (including
// IR players merged in), and IR IDs, ready for buildTieredRoster.
func splitRosterForTiering(userRoster map[string]interface{}) (starters, bench, irPlayers []string) {
	starters = toStringSlice(userRoster["starters"])
	allPlayers := toStringSlice(userRoster["players"])
	irPlayers = toStringSlice(userRoster["reserve"])
	bench = diff(allPlayers, starters)
	for _, ir := range irPlayers {
		found := false
		for _, b := range bench {
			if b == ir {
				found = true
				break
			}
		}
		if !found {
			bench = append(bench, ir)
		}
	}
	return starters, bench, irPlayers
}

// TieredRoster is a user's starters and bench with Boris Chen tiers assigned,
// including FLEX/SUPERFLEX re-ranking. Shared by the per-league tiers page and
// the cross-league dashboard action feed so tier logic only lives in one place.
type TieredRoster struct {
	StartersRows      []PlayerRow
	UnrankedRows      []PlayerRow
	BenchRows         []PlayerRow
	BenchUnrankedRows []PlayerRow
	StarterTiers      []int
	IsSuperFlex       bool
}

// buildTieredRoster takes an already-computed starters/bench/IR split (see
// splitRosterForTiering) and assigns Boris Chen tiers to it.
func buildTieredRoster(starters, bench, irPlayers []string, players map[string]interface{}, borisTiers map[string][][]string, leagueRosterPositions []string) TieredRoster {
	firstPassBench, _, _ := buildRowsWithPositions(bench, players, borisTiers, false, nil, irPlayers, nil)
	bestBenchTier := make(map[string]int)
	for _, row := range firstPassBench {
		tier, ok := row.Tier.(int)
		if ok && tier > 0 {
			if best, exists := bestBenchTier[row.Pos]; !exists || tier < best {
				bestBenchTier[row.Pos] = tier
			}
		}
	}

	startersRows, unrankedRows, _ := buildRowsWithPositions(starters, players, borisTiers, true, leagueRosterPositions, irPlayers, bestBenchTier)

	// FLEX/SUPERFLEX: re-rank RB/WR/TE occupying a FLEX slot using the FLEX tier table.
	for i, row := range startersRows {
		if row.Pos != "FLEX" && row.Pos != "SUPER_FLEX" {
			continue
		}
		pid := starters[i]
		p, ok := players[pid].(map[string]interface{})
		if !ok {
			continue
		}
		name := getPlayerName(p)
		actualPos, _ := p["position"].(string)
		isFlexEligible := actualPos == "RB" || actualPos == "WR" || actualPos == "TE"

		if row.Pos == "FLEX" {
			startersRows[i].IsFlex = true
		} else {
			startersRows[i].IsSuperflex = true
		}

		flxTier := findTier(borisTiers["FLX"], name)
		if flxTier > 0 && isFlexEligible {
			startersRows[i].Tier = flxTier
		}
	}

	starterTiers := []int{}
	for _, row := range startersRows {
		if t, ok := row.Tier.(int); ok && t > 0 {
			starterTiers = append(starterTiers, t)
		}
	}

	worstStarterTier := make(map[string]int)
	for _, row := range startersRows {
		tier, ok := row.Tier.(int)
		if ok && tier > 0 {
			if worst, exists := worstStarterTier[row.Pos]; !exists || tier > worst {
				worstStarterTier[row.Pos] = tier
			}
		}
	}
	benchRows, benchUnrankedRows, _ := buildRowsWithPositions(bench, players, borisTiers, false, nil, irPlayers, worstStarterTier)

	isSuperFlex := false
	for _, pos := range leagueRosterPositions {
		if pos == "SUPER_FLEX" {
			isSuperFlex = true
			break
		}
	}

	return TieredRoster{
		StartersRows:      startersRows,
		UnrankedRows:      unrankedRows,
		BenchRows:         benchRows,
		BenchUnrankedRows: benchUnrankedRows,
		StarterTiers:      starterTiers,
		IsSuperFlex:       isSuperFlex,
	}
}

func getPlayerName(p map[string]interface{}) string {
	// Handle DST/DEF players
	if pos, ok := p["position"].(string); ok && (pos == "DEF" || pos == "DST") {
		if team, ok := p["team"].(string); ok {
			if fullName, exists := TEAM_MAP[team]; exists {
				return fullName
			}
			return team
		}
		return "Unknown"
	}

	// Regular players
	firstName, _ := p["first_name"].(string)
	lastName, _ := p["last_name"].(string)
	return strings.TrimSpace(firstName + " " + lastName)
}

func avg(arr []int) string {
	if len(arr) == 0 {
		return "-"
	}
	sum := 0
	for _, x := range arr {
		sum += x
	}
	return fmt.Sprintf("%.2f", float64(sum)/float64(len(arr)))
}

func winProbability(avg, opp string) (string, string) {
	if avg == "-" || opp == "-" {
		return "-", "🤝"
	}
	a, _ := strconv.ParseFloat(avg, 64)
	o, _ := strconv.ParseFloat(opp, 64)
	diff := o - a
	prob := 50 + math.Max(-30, math.Min(30, diff*10))
	emoji := "🤝"
	if prob > 60 {
		emoji = "🏆"
	} else if prob < 40 {
		emoji = "💀"
	}

	winner := "Opponent"
	if prob > 50 {
		winner = "You"
	}

	return fmt.Sprintf("%d%% %s", int(prob), winner), emoji
}
