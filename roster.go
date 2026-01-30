// ABOUTME: Roster and player processing functions for SleeperPy
// ABOUTME: Includes functions for building player rows, calculating tiers, and win probabilities

package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

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
		lookupPos := pos
		if pos == "FLEX" || pos == "SUPER_FLEX" {
			if realPos, ok := p["position"].(string); ok {
				lookupPos = realPos
			}
		}
		// Always use DST for DEF/DST for Boris Chen mapping
		if lookupPos == "DEF" {
			lookupPos = "DST"
		}

		tier := findTier(tiers[lookupPos], name)

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

		if tier > 0 {
			rows = append(rows, PlayerRow{Pos: pos, Name: displayName, Tier: tier, IsTierWorseThanBench: isWorse, ShouldSwapIn: shouldSwapIn, Age: age})
			tierNums = append(tierNums, tier)
		} else {
			unranked = append(unranked, PlayerRow{Pos: "?", Name: displayName, Tier: "Not Ranked", IsTierWorseThanBench: false, ShouldSwapIn: false, Age: age})
		}
	}
	return rows, unranked, tierNums
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

func getPos(p map[string]interface{}, idx int, isStarter bool, userRoster map[string]interface{}) string {
	if isStarter && userRoster != nil {
		if slots, ok := userRoster["starter_positions"].([]interface{}); ok && idx < len(slots) {
			if s, ok := slots[idx].(string); ok {
				slot := strings.ToUpper(s)
				if strings.Contains(slot, "SUPER") && strings.Contains(slot, "FLEX") {
					return "SUPERFLEX"
				} else if strings.Contains(slot, "FLEX") {
					return "FLEX"
				}
			}
		}
	}
	if pos, ok := p["position"].(string); ok {
		return pos
	}
	return "?"
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
