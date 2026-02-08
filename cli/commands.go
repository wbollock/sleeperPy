package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// APIClient interface for dependency injection
type APIClient interface {
	FetchUser(ctx context.Context, username string) (map[string]interface{}, error)
	FetchUserLeagues(ctx context.Context, userID string) ([]map[string]interface{}, error)
	FetchBorisChenTiers(ctx context.Context, scoring string) (map[string][][]string, error)
	FetchDynastyValues(ctx context.Context) (map[string]interface{}, string, error)
	FetchPlayers(ctx context.Context) (map[string]interface{}, error)
}

// Global API client instance
var API APIClient

func cmdUser(ctx *Context) int {
	if len(ctx.Args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: cli user <username>")
		return 1
	}

	if API == nil {
		fmt.Fprintln(os.Stderr, "Error: API client not initialized")
		return 1
	}

	username := ctx.Args[0]
	goCtx := context.Background()

	// Fetch user
	user, err := API.FetchUser(goCtx, username)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching user: %v\n", err)
		return 1
	}

	userID, ok := user["user_id"].(string)
	if !ok {
		fmt.Fprintln(os.Stderr, "Error: could not get user ID")
		return 1
	}

	// Fetch leagues
	leagues, err := API.FetchUserLeagues(goCtx, userID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching leagues: %v\n", err)
		return 1
	}

	// Output
	if ctx.JSON {
		output := map[string]interface{}{
			"user":    user,
			"leagues": leagues,
		}
		json.NewEncoder(os.Stdout).Encode(output)
	} else {
		displayName := username
		if name, ok := user["display_name"].(string); ok && name != "" {
			displayName = name
		}

		fmt.Printf("User: %s (ID: %s)\n", displayName, userID)
		fmt.Printf("Found %d leagues:\n\n", len(leagues))

		for _, league := range leagues {
			name := league["name"]
			leagueID := league["league_id"]

			leagueType := "unknown"
			scoring := "unknown"
			teams := 0

			if settings, ok := league["settings"].(map[string]interface{}); ok {
				if t, ok := settings["type"].(float64); ok {
					if t == 2 {
						leagueType = "dynasty"
					} else {
						leagueType = "redraft"
					}
				}
				if numTeams, ok := settings["num_teams"].(float64); ok {
					teams = int(numTeams)
				}
			}

			if scoringSettings, ok := league["scoring_settings"].(map[string]interface{}); ok {
				if rec, ok := scoringSettings["rec"].(float64); ok {
					if rec == 1.0 {
						scoring = "ppr"
					} else if rec == 0.5 {
						scoring = "half-ppr"
					} else {
						scoring = "standard"
					}
				} else {
					scoring = "standard"
				}
			}

			fmt.Printf("  - %s (%s)\n", name, leagueID)
			fmt.Printf("    Type: %s, Scoring: %s, Teams: %d\n\n", leagueType, scoring, teams)
		}
	}

	return 0
}

func cmdLeague(ctx *Context) int {
	if len(ctx.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: cli league <league_id> <username>")
		return 1
	}

	leagueID := ctx.Args[0]
	username := ctx.Args[1]

	// This would need full integration with roster analysis logic from main package
	// For now, provide a placeholder
	fmt.Printf("League Analysis: %s\n", leagueID)
	fmt.Printf("User: %s\n", username)
	fmt.Println()
	fmt.Println("Note: Full league analysis requires integration with roster analysis logic.")
	fmt.Println("      This feature will be completed in the next iteration.")

	if ctx.JSON {
		output := map[string]interface{}{
			"status":    "partial",
			"league_id": leagueID,
			"username":  username,
			"message":   "Full integration pending",
		}
		json.NewEncoder(os.Stdout).Encode(output)
	}

	return 0
}

func cmdTiers(ctx *Context) int {
	format := "ppr"
	if len(ctx.Args) >= 1 {
		format = strings.ToLower(ctx.Args[0])
	}

	validFormats := map[string]bool{
		"ppr": true, "half-ppr": true, "standard": true, "superflex": true,
	}

	if !validFormats[format] {
		fmt.Fprintf(os.Stderr, "Invalid format: %s\n", format)
		fmt.Fprintln(os.Stderr, "Valid formats: ppr, half-ppr, standard, superflex")
		return 1
	}

	if API == nil {
		fmt.Fprintln(os.Stderr, "Error: API client not initialized")
		return 1
	}

	goCtx := context.Background()
	tiers, err := API.FetchBorisChenTiers(goCtx, format)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching tiers: %v\n", err)
		return 1
	}

	if ctx.JSON {
		json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"format": format,
			"tiers":  tiers,
		})
	} else {
		printTiersSummary(tiers, format)
	}

	return 0
}

func cmdDynastyValues(ctx *Context) int {
	if API == nil {
		fmt.Fprintln(os.Stderr, "Error: API client not initialized")
		return 1
	}

	goCtx := context.Background()
	values, scrapeDate, err := API.FetchDynastyValues(goCtx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching dynasty values: %v\n", err)
		return 1
	}

	if ctx.JSON {
		json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"scrape_date": scrapeDate,
			"values":      values,
		})
	} else {
		printDynastyValuesSummary(values, scrapeDate)
	}

	return 0
}

func cmdPlayer(ctx *Context) int {
	if len(ctx.Args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: cli player <name>")
		return 1
	}

	if API == nil {
		fmt.Fprintln(os.Stderr, "Error: API client not initialized")
		return 1
	}

	playerName := strings.Join(ctx.Args, " ")
	goCtx := context.Background()

	// Fetch all players
	players, err := API.FetchPlayers(goCtx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching players: %v\n", err)
		return 1
	}

	// Search for player by name (case-insensitive)
	var foundPlayer map[string]interface{}
	searchName := strings.ToLower(playerName)

	for _, p := range players {
		if playerData, ok := p.(map[string]interface{}); ok {
			if fullName, ok := playerData["full_name"].(string); ok {
				if strings.Contains(strings.ToLower(fullName), searchName) {
					foundPlayer = playerData
					break
				}
			}
			if firstName, ok := playerData["first_name"].(string); ok {
				if lastName, ok := playerData["last_name"].(string); ok {
					fullName := firstName + " " + lastName
					if strings.Contains(strings.ToLower(fullName), searchName) {
						foundPlayer = playerData
						break
					}
				}
			}
		}
	}

	if foundPlayer == nil {
		fmt.Fprintf(os.Stderr, "Player not found: %s\n", playerName)
		return 1
	}

	if ctx.JSON {
		json.NewEncoder(os.Stdout).Encode(foundPlayer)
	} else {
		printPlayerInfo(foundPlayer)
	}

	return 0
}
