package cli

import (
	"fmt"
	"sort"
	"strings"
)

// printLeagueAnalysis prints formatted league analysis
func printLeagueAnalysis(data map[string]interface{}) {
	if league, ok := data["league"].(map[string]interface{}); ok {
		if name, ok := league["name"].(string); ok {
			fmt.Printf("League: %s\n", name)
		}
		if leagueID, ok := league["league_id"].(string); ok {
			fmt.Printf("League ID: %s\n", leagueID)
		}
	}

	if username, ok := data["username"].(string); ok {
		fmt.Printf("User: %s\n", username)
	}

	if rosterID, ok := data["user_roster_id"].(float64); ok {
		fmt.Printf("Roster ID: %d\n", int(rosterID))
	}

	if record, ok := data["record"].(string); ok && record != "" {
		fmt.Printf("Record: %s\n", record)
	}

	fmt.Println()

	// Print starters
	if starters, ok := data["starters"].([]interface{}); ok && len(starters) > 0 {
		fmt.Println("Starters:")
		for _, s := range starters {
			if starter, ok := s.(map[string]interface{}); ok {
				pos := getStringField(starter, "pos")
				name := getStringField(starter, "name")
				tier := getIntField(starter, "tier")
				value := getIntField(starter, "dynasty_value")
				fmt.Printf("  %-4s %-25s Tier %-2d  Value: %d\n", pos, name, tier, value)
			}
		}
		fmt.Println()
	}

	// Print bench summary
	if bench, ok := data["bench"].([]interface{}); ok {
		fmt.Printf("Bench: %d players\n", len(bench))
	}

	// Print total roster value
	if totalValue, ok := data["total_roster_value"].(float64); ok {
		fmt.Printf("Total Roster Value: %s\n", formatValue(int(totalValue)))
	}

	// Print average age
	if avgAge, ok := data["avg_age"].(float64); ok {
		fmt.Printf("Average Age: %.1f years\n", avgAge)
	}

	fmt.Println()

	// Print free agent upgrades
	if upgrades, ok := data["free_agent_upgrades"].([]interface{}); ok && len(upgrades) > 0 {
		fmt.Printf("Free Agent Upgrades: %d available\n", len(upgrades))
	}
}

// printTiersSummary prints a summary of Boris Chen tiers
func printTiersSummary(tiers map[string][][]string, format string) {
	fmt.Printf("Boris Chen Tiers (%s)\n", format)
	fmt.Println(strings.Repeat("=", 60))

	positions := []string{"QB", "RB", "WR", "TE", "FLEX", "K", "DST"}
	for _, pos := range positions {
		if tierData, ok := tiers[pos]; ok {
			fmt.Printf("\n%s Tiers:\n", pos)
			for i, tier := range tierData {
				fmt.Printf("  Tier %d: %d players\n", i+1, len(tier))
			}
		}
	}
}

// printDynastyValuesSummary prints top dynasty values
func printDynastyValuesSummary(values map[string]interface{}, scrapeDate string) {
	fmt.Println("Dynasty Values (KeepTradeCut)")
	fmt.Printf("Scrape Date: %s\n", scrapeDate)
	fmt.Printf("Total players: %d\n", len(values))
	fmt.Println(strings.Repeat("=", 60))

	// Convert to sorted list
	type playerValue struct {
		name  string
		value int
		pos   string
	}
	var players []playerValue

	for name, val := range values {
		if v, ok := val.(map[string]interface{}); ok {
			value := getIntField(v, "value_1qb")
			pos := getStringField(v, "position")
			if value > 0 {
				players = append(players, playerValue{name, value, pos})
			}
		}
	}

	sort.Slice(players, func(i, j int) bool {
		return players[i].value > players[j].value
	})

	fmt.Println("\nTop 10 Most Valuable:")
	for i := 0; i < 10 && i < len(players); i++ {
		p := players[i]
		fmt.Printf("  %2d. %-30s (%s)  %s\n", i+1, p.name, p.pos, formatValue(p.value))
	}
}

// printPlayerInfo prints detailed player information
func printPlayerInfo(player map[string]interface{}) {
	if name, ok := player["name"].(string); ok {
		fmt.Printf("Player: %s\n", name)
	}

	if pos, ok := player["position"].(string); ok {
		fmt.Printf("Position: %s\n", pos)
	}

	if age, ok := player["age"].(float64); ok {
		fmt.Printf("Age: %d\n", int(age))
	}

	if team, ok := player["team"].(string); ok {
		fmt.Printf("Team: %s\n", team)
	}

	fmt.Println()

	if tier, ok := player["tier"].(float64); ok {
		fmt.Printf("Boris Chen Tier: %d\n", int(tier))
	}

	if value, ok := player["dynasty_value"].(float64); ok {
		fmt.Printf("Dynasty Value: %s\n", formatValue(int(value)))
	}

	if status, ok := player["injury_status"].(string); ok && status != "" {
		fmt.Printf("Status: %s\n", status)
	}
}

// Helper functions

func getStringField(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getIntField(m map[string]interface{}, key string) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	if v, ok := m[key].(int); ok {
		return v
	}
	return 0
}

func formatValue(value int) string {
	if value >= 1000 {
		return fmt.Sprintf("%d,%03d", value/1000, value%1000)
	}
	return fmt.Sprintf("%d", value)
}
