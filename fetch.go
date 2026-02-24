// ABOUTME: External API fetching functions for SleeperPy
// ABOUTME: Handles fetching data from Sleeper API, Boris Chen tiers, and DynastyProcess values

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

func fetchPlayers() (map[string]interface{}, error) {
	// Check cache first
	sleeperPlayersCache.RLock()
	if time.Since(sleeperPlayersCache.timestamp) < sleeperPlayersCache.ttl && sleeperPlayersCache.data != nil {
		debugLog("[DEBUG] Using cached Sleeper players data")
		sleeperPlayersCache.RUnlock()
		return sleeperPlayersCache.data, nil
	}
	sleeperPlayersCache.RUnlock()

	debugLog("[DEBUG] Fetching fresh Sleeper players data")
	players, err := fetchJSON("https://api.sleeper.app/v1/players/nfl")
	if err != nil {
		return nil, err
	}

	// Update cache
	sleeperPlayersCache.Lock()
	sleeperPlayersCache.data = players
	sleeperPlayersCache.timestamp = time.Now()
	sleeperPlayersCache.Unlock()

	debugLog("[DEBUG] Cached %d Sleeper players", len(players))
	return players, nil
}

func fetchJSON(url string) (map[string]interface{}, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&out)
	return out, err
}

func fetchJSONArray(url string) ([]map[string]interface{}, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&out)
	return out, err
}

func fetchDynastyValues() (map[string]DynastyValue, string) {
	// Check cache first
	dynastyValuesCache.RLock()
	if time.Since(dynastyValuesCache.timestamp) < dynastyValuesCache.ttl && len(dynastyValuesCache.data) > 0 {
		debugLog("[DEBUG] Using cached dynasty values")
		scrapeDate := ""
		for _, v := range dynastyValuesCache.data {
			scrapeDate = v.ScrapeDate
			break
		}
		dynastyValuesCache.RUnlock()
		return dynastyValuesCache.data, scrapeDate
	}
	dynastyValuesCache.RUnlock()

	debugLog("[DEBUG] Fetching fresh dynasty values from DynastyProcess")

	resp, err := httpClient.Get("https://raw.githubusercontent.com/dynastyprocess/data/master/files/values-players.csv")
	if err != nil {
		log.Printf("[ERROR] Failed to fetch dynasty values: %v", err)
		return nil, ""
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[ERROR] Failed to read dynasty values: %v", err)
		return nil, ""
	}

	// Parse CSV
	lines := strings.Split(string(body), "\n")
	values := make(map[string]DynastyValue)
	scrapeDate := ""

	for i, line := range lines {
		if i == 0 || line == "" {
			continue // Skip header and empty lines
		}

		// Parse CSV line - handle quoted fields
		fields := parseCSVLine(line)
		if len(fields) < 12 {
			continue
		}

		playerName := strings.Trim(fields[0], "\"")
		position := strings.Trim(fields[1], "\"")
		value1QB, _ := strconv.Atoi(strings.Trim(fields[8], "\""))
		value2QB, _ := strconv.Atoi(strings.Trim(fields[9], "\""))
		date := strings.Trim(fields[10], "\"")

		if scrapeDate == "" {
			scrapeDate = date
		}

		// Store with normalized name as key
		normalizedName := normalizeName(playerName)
		values[normalizedName] = DynastyValue{
			Name:       playerName,
			Position:   position,
			Value1QB:   value1QB,
			Value2QB:   value2QB,
			ScrapeDate: date,
		}
	}

	// Update cache
	dynastyValuesCache.Lock()
	dynastyValuesCache.data = values
	dynastyValuesCache.timestamp = time.Now()
	dynastyValuesCache.Unlock()

	debugLog("[DEBUG] Loaded %d dynasty values (last updated: %s)", len(values), scrapeDate)
	return values, scrapeDate
}

var borisURLs = map[string]map[string]string{
	"PPR": {
		"QB":  "https://s3-us-west-1.amazonaws.com/fftiers/out/text_QB.txt",
		"RB":  "https://s3-us-west-1.amazonaws.com/fftiers/out/text_RB-PPR.txt",
		"WR":  "https://s3-us-west-1.amazonaws.com/fftiers/out/text_WR-PPR.txt",
		"TE":  "https://s3-us-west-1.amazonaws.com/fftiers/out/text_TE-PPR.txt",
		"FLX": "https://s3-us-west-1.amazonaws.com/fftiers/out/text_FLX-PPR.txt",
		"K":   "https://s3-us-west-1.amazonaws.com/fftiers/out/text_K.txt",
		"DST": "https://s3-us-west-1.amazonaws.com/fftiers/out/text_DST.txt",
	},
	"Half PPR": {
		"QB":  "https://s3-us-west-1.amazonaws.com/fftiers/out/text_QB.txt",
		"RB":  "https://s3-us-west-1.amazonaws.com/fftiers/out/text_RB-HALF.txt",
		"WR":  "https://s3-us-west-1.amazonaws.com/fftiers/out/text_WR-HALF.txt",
		"TE":  "https://s3-us-west-1.amazonaws.com/fftiers/out/text_TE-HALF.txt",
		"FLX": "https://s3-us-west-1.amazonaws.com/fftiers/out/text_FLX-HALF.txt",
		"K":   "https://s3-us-west-1.amazonaws.com/fftiers/out/text_K.txt",
		"DST": "https://s3-us-west-1.amazonaws.com/fftiers/out/text_DST.txt",
	},
	"Standard": {
		"QB":  "https://s3-us-west-1.amazonaws.com/fftiers/out/text_QB.txt",
		"RB":  "https://s3-us-west-1.amazonaws.com/fftiers/out/text_RB.txt",
		"WR":  "https://s3-us-west-1.amazonaws.com/fftiers/out/text_WR.txt",
		"TE":  "https://s3-us-west-1.amazonaws.com/fftiers/out/text_TE.txt",
		"FLX": "https://s3-us-west-1.amazonaws.com/fftiers/out/text_FLX.txt",
		"K":   "https://s3-us-west-1.amazonaws.com/fftiers/out/text_K.txt",
		"DST": "https://s3-us-west-1.amazonaws.com/fftiers/out/text_DST.txt",
	},
}

// fetchBorisTiers is a variable to allow mocking in tests
var fetchBorisTiers = func(scoring string) map[string][][]string {
	return fetchBorisTiersImpl(scoring)
}

func fetchBorisTiersImpl(scoring string) map[string][][]string {
	// Check cache first
	borisTiersCache.RLock()
	if cached, exists := borisTiersCache.data[scoring]; exists {
		if time.Since(borisTiersCache.timestamp[scoring]) < borisTiersCache.ttl {
			debugLog("[DEBUG] Using cached Boris tiers for %s", scoring)
			borisTiersCache.RUnlock()
			return cached
		}
	}
	borisTiersCache.RUnlock()

	debugLog("[DEBUG] Fetching fresh Boris tiers for %s", scoring)

	urls := borisURLs[scoring]
	out := make(map[string][][]string)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Fetch all position tiers concurrently
	for pos, url := range urls {
		wg.Add(1)
		go func(position, tierURL string) {
			defer wg.Done()

			resp, err := httpClient.Get(tierURL)
			if err != nil {
				debugLog("[DEBUG] Error fetching %s tiers: %v", position, err)
				return
			}
			defer resp.Body.Close()

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				debugLog("[DEBUG] Error reading %s tier data: %v", position, err)
				return
			}

			var posTiers [][]string
			lines := strings.Split(string(body), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "Tier ") {
					tierContent := strings.TrimPrefix(line, "Tier ")
					parts := strings.SplitN(tierContent, ":", 2)
					if len(parts) < 2 {
						continue
					}
					tier := strings.TrimSpace(parts[1])
					names := strings.Split(tier, ",")
					for j := range names {
						names[j] = strings.TrimSpace(names[j])
					}
					posTiers = append(posTiers, names)
				}
			}

			mu.Lock()
			out[position] = posTiers
			mu.Unlock()
		}(pos, url)
	}

	wg.Wait()

	// Cache the result
	borisTiersCache.Lock()
	borisTiersCache.data[scoring] = out
	borisTiersCache.timestamp[scoring] = time.Now()
	borisTiersCache.Unlock()

	return out
}

func fetchRecentTransactions(leagueID string, currentWeek int, players map[string]interface{}, rosters []map[string]interface{}, userNames map[string]string, dynastyValues map[string]DynastyValue, isSuperFlex bool) []Transaction {
	transactions := []Transaction{}

	// Fetch transactions from multiple weeks (last 3 weeks)
	startWeek := currentWeek - 2
	if startWeek < 1 {
		startWeek = 1
	}

	debugLog("[DEBUG] Fetching transactions from week %d to %d", startWeek, currentWeek)

	// Fetch all weeks concurrently
	type weekTxns struct {
		week int
		data []map[string]interface{}
		err  error
	}
	results := make(chan weekTxns, 3)

	for week := startWeek; week <= currentWeek; week++ {
		go func(w int) {
			url := fmt.Sprintf("https://api.sleeper.app/v1/league/%s/transactions/%d", leagueID, w)
			txnData, err := fetchJSONArray(url)
			results <- weekTxns{week: w, data: txnData, err: err}
		}(week)
	}

	// Collect results
	allTxns := []map[string]interface{}{}
	for i := 0; i < (currentWeek - startWeek + 1); i++ {
		result := <-results
		if result.err != nil {
			debugLog("[DEBUG] Error fetching transactions for week %d: %v", result.week, result.err)
			continue
		}
		allTxns = append(allTxns, result.data...)
	}
	close(results)

	debugLog("[DEBUG] Fetched %d total transactions", len(allTxns))

	// Create map of roster_id -> team name for easier lookup
	rosterTeams := make(map[float64]string)
	for _, r := range rosters {
		rosterID, _ := r["roster_id"].(float64)
		ownerID, _ := r["owner_id"].(string)

		teamName := "Unknown"
		if userNames != nil {
			if name, exists := userNames[ownerID]; exists {
				teamName = name
			}
		}

		// Try to get custom team name from metadata
		if metadata, ok := r["metadata"].(map[string]interface{}); ok {
			if tn, ok := metadata["team_name"].(string); ok && tn != "" {
				teamName = tn
			}
		}

		rosterTeams[rosterID] = teamName
	}

	// Process each transaction
	for _, txn := range allTxns {
		txnType, _ := txn["type"].(string)
		if txnType == "" {
			continue
		}

		// Parse timestamp
		created, _ := txn["created"].(float64)
		timestamp := time.Unix(int64(created/1000), 0)

		// Handle trades
		if txnType == "trade" {
			rosterIDs, _ := txn["roster_ids"].([]interface{})
			if len(rosterIDs) < 2 {
				continue
			}

			roster1, _ := rosterIDs[0].(float64)
			roster2, _ := rosterIDs[1].(float64)

			team1 := rosterTeams[roster1]
			team2 := rosterTeams[roster2]

			// Get draft picks involved (map of roster_id -> pick details)
			draftPicks, _ := txn["draft_picks"].([]interface{})

			// Get players involved
			adds, _ := txn["adds"].(map[string]interface{})
			_, _ = txn["drops"].(map[string]interface{})

			// Determine what each team gave
			team1Gave := []string{}
			team2Gave := []string{}

			// Players
			for playerID, rosterID := range adds {
				recipientRosterID, _ := rosterID.(float64)
				if p, ok := players[playerID].(map[string]interface{}); ok {
					playerName := getPlayerName(p)
					if recipientRosterID == roster1 {
						team2Gave = append(team2Gave, playerName)
					} else {
						team1Gave = append(team1Gave, playerName)
					}
				}
			}

			// Draft picks
			for _, pickData := range draftPicks {
				pick, _ := pickData.(map[string]interface{})
				ownerID, _ := pick["owner_id"].(float64)   // Who originally owned this pick
				rosterID, _ := pick["roster_id"].(float64) // Who now owns this pick
				season, _ := pick["season"].(string)
				round, _ := pick["round"].(float64)

				pickDesc := fmt.Sprintf("%s Round %d", season, int(round))

				// Determine who gave this pick
				if rosterID == roster1 {
					// roster1 now has the pick, so roster2 (or original owner) gave it
					if ownerID == roster2 {
						team2Gave = append(team2Gave, pickDesc)
					} else {
						// Pick was traded to roster2 before, now going to roster1
						origTeam := rosterTeams[ownerID]
						team2Gave = append(team2Gave, fmt.Sprintf("%s (from %s)", pickDesc, origTeam))
					}
				} else {
					// roster2 now has the pick, so roster1 (or original owner) gave it
					if ownerID == roster1 {
						team1Gave = append(team1Gave, pickDesc)
					} else {
						origTeam := rosterTeams[ownerID]
						team1Gave = append(team1Gave, fmt.Sprintf("%s (from %s)", pickDesc, origTeam))
					}
				}
			}

			// Calculate dynasty values if available
			team1GaveValue := 0
			team2GaveValue := 0
			if dynastyValues != nil {
				for _, playerName := range team1Gave {
					// Skip draft picks (they contain "Round")
					if !strings.Contains(playerName, "Round") {
						normalizedName := normalizeName(playerName)
						if value, exists := dynastyValues[normalizedName]; exists {
							if isSuperFlex {
								team1GaveValue += value.Value2QB
							} else {
								team1GaveValue += value.Value1QB
							}
						}
					}
				}
				for _, playerName := range team2Gave {
					// Skip draft picks
					if !strings.Contains(playerName, "Round") {
						normalizedName := normalizeName(playerName)
						if value, exists := dynastyValues[normalizedName]; exists {
							if isSuperFlex {
								team2GaveValue += value.Value2QB
							} else {
								team2GaveValue += value.Value1QB
							}
						}
					}
				}
			}

			// Net value: positive means team1 gained value (got more than they gave)
			netValue := team2GaveValue - team1GaveValue

			// Build description
			desc := fmt.Sprintf("%s traded with %s", team1, team2)

			txn := Transaction{
				Type:           "trade",
				LeagueID:       leagueID,
				Timestamp:      timestamp,
				Description:    desc,
				TeamNames:      []string{team1, team2},
				Team1:          team1,
				Team2:          team2,
				Team1Gave:      team1Gave,
				Team2Gave:      team2Gave,
				Team1GaveValue: team1GaveValue,
				Team2GaveValue: team2GaveValue,
				NetValue:       netValue,
			}

			// Calculate fairness (Feature #3) - only for dynasty leagues
			if dynastyValues != nil && (team1GaveValue > 0 || team2GaveValue > 0) {
				txn.Fairness = calculateTradeFairness(txn, nil)
			}

			transactions = append(transactions, txn)
		} else if txnType == "waiver" || txnType == "free_agent" {
			// Handle waivers and free agent pickups
			rosterIDs, _ := txn["roster_ids"].([]interface{})
			if len(rosterIDs) == 0 {
				continue
			}

			rosterID, _ := rosterIDs[0].(float64)
			teamName := rosterTeams[rosterID]

			adds, _ := txn["adds"].(map[string]interface{})
			drops, _ := txn["drops"].(map[string]interface{})

			addedPlayer := ""
			droppedPlayer := ""

			for playerID := range adds {
				if p, ok := players[playerID].(map[string]interface{}); ok {
					addedPlayer = getPlayerName(p)
					break
				}
			}

			for playerID := range drops {
				if p, ok := players[playerID].(map[string]interface{}); ok {
					droppedPlayer = getPlayerName(p)
					break
				}
			}

			desc := ""
			if txnType == "waiver" {
				if droppedPlayer != "" {
					desc = fmt.Sprintf("%s claimed %s (dropped %s)", teamName, addedPlayer, droppedPlayer)
				} else {
					desc = fmt.Sprintf("%s claimed %s", teamName, addedPlayer)
				}
			} else {
				if droppedPlayer != "" {
					desc = fmt.Sprintf("%s added %s (dropped %s)", teamName, addedPlayer, droppedPlayer)
				} else {
					desc = fmt.Sprintf("%s added %s", teamName, addedPlayer)
				}
			}

			transactions = append(transactions, Transaction{
				Type:          txnType,
				LeagueID:      leagueID,
				Timestamp:     timestamp,
				Description:   desc,
				TeamNames:     []string{teamName},
				AddedPlayer:   addedPlayer,
				DroppedPlayer: droppedPlayer,
			})
		}
	}

	// Sort by timestamp (most recent first)
	sort.Slice(transactions, func(i, j int) bool {
		return transactions[i].Timestamp.After(transactions[j].Timestamp)
	})

	// Limit to 20 most recent
	if len(transactions) > 20 {
		transactions = transactions[:20]
	}

	return transactions
}
