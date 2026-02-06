// ABOUTME: Dynasty league analysis functions for SleeperPy
// ABOUTME: Includes functions for player valuations, trade targets, power rankings, and breakout candidates

package main

import (
	"fmt"
	"sort"
	"time"
)

func enrichRowsWithDynastyValues(rows []PlayerRow, dynastyValues map[string]DynastyValue, isSuperFlex bool) {
	if dynastyValues == nil {
		return
	}

	for i := range rows {
		// Normalize the player name and lookup dynasty value
		cleanName := stripHTML(rows[i].Name)
		normalizedName := normalizeName(cleanName)

		if val, exists := dynastyValues[normalizedName]; exists {
			// Use 2QB values for superflex leagues, otherwise 1QB
			if isSuperFlex {
				rows[i].DynastyValue = val.Value2QB
			} else {
				rows[i].DynastyValue = val.Value1QB
			}
		}
	}
}

func aggregatePlayerNews(rosterPlayerIDs []string, players map[string]interface{}, startersIDs []string, dynastyValues map[string]DynastyValue, isSuperFlex bool) []PlayerNews {
	newsFeed := []PlayerNews{}

	for _, pid := range rosterPlayerIDs {
		p, ok := players[pid].(map[string]interface{})
		if !ok {
			continue
		}

		name := getPlayerName(p)
		pos, _ := p["position"].(string)

		// Get general news
		newsText := ""
		source := ""
		var timestamp time.Time

		if newsObj, ok := p["news"].(map[string]interface{}); ok {
			if text, ok := newsObj["text"].(string); ok {
				newsText = text
			}
			if src, ok := newsObj["source"].(string); ok {
				source = src
			}
			if ts, ok := newsObj["timestamp"].(float64); ok {
				timestamp = time.Unix(int64(ts), 0)
			}
		}

		// Get injury-related fields
		injuryStatus := ""
		if status, ok := p["injury_status"].(string); ok {
			injuryStatus = status
		}

		injuryBodyPart := ""
		if bodyPart, ok := p["injury_body_part"].(string); ok {
			injuryBodyPart = bodyPart
		}

		injuryNotes := ""
		if notes, ok := p["injury_notes"].(string); ok {
			injuryNotes = notes
		}

		// Fallback to news_updated field if no timestamp from news object (in milliseconds)
		if timestamp.IsZero() {
			if newsUpdated, ok := p["news_updated"].(float64); ok {
				// Convert milliseconds to seconds for Unix timestamp
				timestamp = time.Unix(int64(newsUpdated/1000), 0)
			}
		}

		// Check if starter
		isStarter := false
		for _, sid := range startersIDs {
			if sid == pid {
				isStarter = true
				break
			}
		}

		// Get dynasty value
		dynastyValue := 0
		if dynastyValues != nil {
			cleanName := normalizeName(name)
			if val, exists := dynastyValues[cleanName]; exists {
				if isSuperFlex {
					dynastyValue = val.Value2QB
				} else {
					dynastyValue = val.Value1QB
				}
			}
		}

		// Add to feed if there's news text or injury status
		if newsText != "" || injuryStatus != "" {
			if injuryStatus != "" {
				debugLog("[DEBUG] Injury: %s - status=%s, timestamp=%v, bodypart=%s, notes=%s", name, injuryStatus, timestamp, injuryBodyPart, injuryNotes)
			}
			if newsText != "" {
				debugLog("[DEBUG] News: %s - %s (source: %s, timestamp=%v)", name, newsText, source, timestamp)
			}
			newsFeed = append(newsFeed, PlayerNews{
				PlayerName:     name,
				Position:       pos,
				NewsText:       newsText,
				Source:         source,
				Timestamp:      timestamp,
				InjuryStatus:   injuryStatus,
				InjuryBodyPart: injuryBodyPart,
				InjuryNotes:    injuryNotes,
				IsStarter:      isStarter,
				DynastyValue:   dynastyValue,
			})
		}
	}

	// Sort by timestamp (newest first)
	sort.Slice(newsFeed, func(i, j int) bool {
		return newsFeed[i].Timestamp.After(newsFeed[j].Timestamp)
	})

	return newsFeed
}

func findBreakoutCandidates(benchRows []PlayerRow) []PlayerRow {
	candidates := []PlayerRow{}

	for _, row := range benchRows {
		// Criteria:
		// 1. Age < 25 (young)
		// 2. Dynasty value > 500 (has some value)
		// 3. Currently on bench (not starting)
		// 4. Position is RB/WR/TE (skill positions)

		if row.Age > 0 && row.Age < 25 &&
			row.DynastyValue > 500 &&
			(row.Pos == "RB" || row.Pos == "WR" || row.Pos == "TE") {
			candidates = append(candidates, row)
		}
	}

	// Sort by dynasty value (highest upside first)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].DynastyValue > candidates[j].DynastyValue
	})

	return candidates
}

func findAgingPlayers(startersRows, benchRows []PlayerRow) []PlayerRow {
	aging := []PlayerRow{}
	allPlayers := append([]PlayerRow{}, startersRows...)
	allPlayers = append(allPlayers, benchRows...)

	for _, row := range allPlayers {
		isAging := false

		// Position-specific age thresholds
		switch row.Pos {
		case "RB":
			if row.Age >= 28 {
				isAging = true
			}
		case "WR", "TE":
			if row.Age >= 30 {
				isAging = true
			}
		case "QB":
			if row.Age >= 35 {
				isAging = true
			}
		}

		// Only flag if they still have trade value (>1000) and we identified them as aging
		if isAging && row.DynastyValue > 1000 {
			aging = append(aging, row)
		}
	}

	// Sort by age (oldest first - most urgent)
	sort.Slice(aging, func(i, j int) bool {
		return aging[i].Age > aging[j].Age
	})

	return aging
}

func getTopRookies() []RookieProspect {
	return []RookieProspect{
		// 2025 NFL Draft Results (actual draft picks with NFL teams)
		{Name: "Cam Ward", Position: "QB", College: "Miami (TEN)", Value: 4500, Rank: 1, Year: 2025},
		{Name: "Travis Hunter", Position: "WR", College: "Colorado (JAX)", Value: 7500, Rank: 2, Year: 2025},
		{Name: "Ashton Jeanty", Position: "RB", College: "Boise State (LV)", Value: 6800, Rank: 3, Year: 2025},
		{Name: "Tetairoa McMillan", Position: "WR", College: "Arizona (CAR)", Value: 6500, Rank: 4, Year: 2025},
		{Name: "Colston Loveland", Position: "TE", College: "Michigan (CHI)", Value: 3800, Rank: 5, Year: 2025},
		{Name: "Emeka Egbuka", Position: "WR", College: "Ohio State (TB)", Value: 5500, Rank: 6, Year: 2025},
		{Name: "Omarion Hampton", Position: "RB", College: "North Carolina (LAC)", Value: 5500, Rank: 7, Year: 2025},
		{Name: "Matthew Golden", Position: "WR", College: "Texas (GB)", Value: 5000, Rank: 8, Year: 2025},
		{Name: "Jaxson Dart", Position: "QB", College: "Ole Miss (NYG)", Value: 4200, Rank: 9, Year: 2025},
		{Name: "Luther Burden III", Position: "WR", College: "Missouri", Value: 6000, Rank: 10, Year: 2025},

		// 2026 NFL Draft Prospects (projections - subject to change)
		{Name: "Dante Moore", Position: "QB", College: "Oregon", Value: 4500, Rank: 1, Year: 2026},
		{Name: "Ty Simpson", Position: "QB", College: "Alabama", Value: 4200, Rank: 2, Year: 2026},
		{Name: "Jordyn Tyson", Position: "WR", College: "Arizona State", Value: 6500, Rank: 3, Year: 2026},
		{Name: "Carnell Tate", Position: "WR", College: "Ohio State", Value: 6200, Rank: 4, Year: 2026},
		{Name: "Makai Lemon", Position: "WR", College: "USC", Value: 6000, Rank: 5, Year: 2026},
		{Name: "Fernando Mendoza", Position: "QB", College: "Indiana", Value: 4000, Rank: 6, Year: 2026},
		{Name: "Jeremiyah Love", Position: "RB", College: "Notre Dame", Value: 6800, Rank: 7, Year: 2026},
		{Name: "Denzel Boston", Position: "WR", College: "Washington", Value: 5800, Rank: 8, Year: 2026},
		{Name: "Justice Haynes", Position: "RB", College: "Michigan", Value: 5500, Rank: 9, Year: 2026},
		{Name: "Chris Brazzell II", Position: "WR", College: "Tennessee", Value: 5500, Rank: 10, Year: 2026},
		{Name: "Jonah Coleman", Position: "RB", College: "Washington", Value: 5200, Rank: 11, Year: 2026},
		{Name: "KC Concepcion", Position: "WR", College: "Texas A&M", Value: 5000, Rank: 12, Year: 2026},
		{Name: "Kenyon Sadiq", Position: "TE", College: "Oregon", Value: 3800, Rank: 13, Year: 2026},
		{Name: "Nick Singleton", Position: "RB", College: "Penn State", Value: 4800, Rank: 14, Year: 2026},
		{Name: "Eli Stowers", Position: "TE", College: "Vanderbilt", Value: 3500, Rank: 15, Year: 2026},
	}
}

func calculatePowerRankings(teamAges []TeamAgeData) []PowerRanking {
	rankings := []PowerRanking{}

	// Create power rankings from team data
	for _, team := range teamAges {
		// Determine strategy based on age and record
		strategy := "Contending"
		if team.AvgAge > 27.0 {
			strategy = "Win Now"
		} else if team.AvgAge < 24.5 {
			strategy = "Rebuilding"
		}

		// Get wins/losses from rank (Rank 1 = most wins)
		wins := 15 - team.Rank // Approximate
		losses := team.Rank - 1

		rankings = append(rankings, PowerRanking{
			TeamName:     team.TeamName,
			RosterValue:  team.RosterValue,
			Wins:         wins,
			Losses:       losses,
			AvgAge:       team.AvgAge,
			Strategy:     strategy,
			IsUserTeam:   team.IsUserTeam,
			StandingRank: team.Rank,
		})
	}

	// Sort by roster value (highest first) and assign value ranks
	sort.Slice(rankings, func(i, j int) bool {
		return rankings[i].RosterValue > rankings[j].RosterValue
	})

	for i := range rankings {
		rankings[i].ValueRank = i + 1
		// Overall rank is average of value rank and standing rank
		rankings[i].Rank = (rankings[i].ValueRank + rankings[i].StandingRank) / 2
	}

	// Re-sort by combined rank
	sort.Slice(rankings, func(i, j int) bool {
		return rankings[i].Rank < rankings[j].Rank
	})

	return rankings
}

func calculatePositionalKTC(rows []PlayerRow) PositionalKTC {
	posKTC := PositionalKTC{}
	for _, row := range rows {
		if row.DynastyValue <= 0 {
			continue
		}
		switch row.Pos {
		case "QB":
			posKTC.QB += row.DynastyValue
		case "RB":
			posKTC.RB += row.DynastyValue
		case "WR":
			posKTC.WR += row.DynastyValue
		case "TE":
			posKTC.TE += row.DynastyValue
		}
	}
	return posKTC
}

func findTradeTargets(userRows []PlayerRow, allRosters map[int][]PlayerRow, teamNames map[int]string, userRosterID int) []TradeTarget {
	userKTC := calculatePositionalKTC(userRows)
	userTotal := userKTC.QB + userKTC.RB + userKTC.WR + userKTC.TE

	if userTotal == 0 {
		return nil
	}

	// Calculate user's positional percentages
	userQBPct := float64(userKTC.QB) / float64(userTotal)
	userRBPct := float64(userKTC.RB) / float64(userTotal)
	userWRPct := float64(userKTC.WR) / float64(userTotal)
	userTEPct := float64(userKTC.TE) / float64(userTotal)

	// Determine surplus and deficit positions (>30% is surplus, <15% is deficit)
	type posNeed struct {
		pos   string
		value int
		pct   float64
	}

	userSurplus := []posNeed{}
	userDeficit := []posNeed{}

	if userQBPct > 0.30 {
		userSurplus = append(userSurplus, posNeed{"QB", userKTC.QB, userQBPct})
	} else if userQBPct < 0.15 {
		userDeficit = append(userDeficit, posNeed{"QB", userKTC.QB, userQBPct})
	}

	if userRBPct > 0.30 {
		userSurplus = append(userSurplus, posNeed{"RB", userKTC.RB, userRBPct})
	} else if userRBPct < 0.15 {
		userDeficit = append(userDeficit, posNeed{"RB", userKTC.RB, userRBPct})
	}

	if userWRPct > 0.30 {
		userSurplus = append(userSurplus, posNeed{"WR", userKTC.WR, userWRPct})
	} else if userWRPct < 0.15 {
		userDeficit = append(userDeficit, posNeed{"WR", userKTC.WR, userWRPct})
	}

	if userTEPct > 0.30 {
		userSurplus = append(userSurplus, posNeed{"TE", userKTC.TE, userTEPct})
	} else if userTEPct < 0.15 {
		userDeficit = append(userDeficit, posNeed{"TE", userKTC.TE, userTEPct})
	}

	debugLog("[DEBUG] User positional breakdown: QB=%.1f%%, RB=%.1f%%, WR=%.1f%%, TE=%.1f%%",
		userQBPct*100, userRBPct*100, userWRPct*100, userTEPct*100)
	debugLog("[DEBUG] User surplus positions: %v", userSurplus)
	debugLog("[DEBUG] User deficit positions: %v", userDeficit)

	// If no clear surplus/deficit, no trade recommendations
	if len(userSurplus) == 0 || len(userDeficit) == 0 {
		debugLog("[DEBUG] No trade targets - need both surplus (>30%%) and deficit (<15%%) positions")
		return nil
	}

	type tradeMatch struct {
		rosterID        int
		teamName        string
		complementarity float64
		yourSurplus     string
		theirSurplus    string
		yourSurplusKTC  int
		theirSurplusKTC int
	}

	matches := []tradeMatch{}

	// Analyze each other team
	for rosterID, roster := range allRosters {
		if rosterID == userRosterID {
			continue
		}

		teamKTC := calculatePositionalKTC(roster)
		teamTotal := teamKTC.QB + teamKTC.RB + teamKTC.WR + teamKTC.TE

		if teamTotal == 0 {
			continue
		}

		teamQBPct := float64(teamKTC.QB) / float64(teamTotal)
		teamRBPct := float64(teamKTC.RB) / float64(teamTotal)
		teamWRPct := float64(teamKTC.WR) / float64(teamTotal)
		teamTEPct := float64(teamKTC.TE) / float64(teamTotal)

		// Find complementary matches: user surplus matches team deficit AND team surplus matches user deficit
		var bestMatch *tradeMatch

		for _, userSur := range userSurplus {
			for _, userDef := range userDeficit {
				// Check if team has surplus in user's deficit AND deficit in user's surplus
				teamSurPct := 0.0
				teamDefPct := 0.0
				teamSurValue := 0

				switch userDef.pos {
				case "QB":
					teamSurPct = teamQBPct
					teamSurValue = teamKTC.QB
				case "RB":
					teamSurPct = teamRBPct
					teamSurValue = teamKTC.RB
				case "WR":
					teamSurPct = teamWRPct
					teamSurValue = teamKTC.WR
				case "TE":
					teamSurPct = teamTEPct
					teamSurValue = teamKTC.TE
				}

				switch userSur.pos {
				case "QB":
					teamDefPct = teamQBPct
				case "RB":
					teamDefPct = teamRBPct
				case "WR":
					teamDefPct = teamWRPct
				case "TE":
					teamDefPct = teamTEPct
				}

				// Check for complementarity: they have surplus where you need, you have surplus where they need
				if teamSurPct > 0.30 && teamDefPct < 0.15 {
					complementarity := (userSur.pct - teamDefPct) + (teamSurPct - userDef.pct)

					if bestMatch == nil || complementarity > bestMatch.complementarity {
						bestMatch = &tradeMatch{
							rosterID:        rosterID,
							teamName:        teamNames[rosterID],
							complementarity: complementarity,
							yourSurplus:     userSur.pos,
							theirSurplus:    userDef.pos,
							yourSurplusKTC:  userSur.value,
							theirSurplusKTC: teamSurValue,
						}
					}
				}
			}
		}

		if bestMatch != nil {
			matches = append(matches, *bestMatch)
		}
	}

	// Sort by complementarity score (highest first)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].complementarity > matches[j].complementarity
	})

	// Return top 3 matches
	limit := 3
	if len(matches) < limit {
		limit = len(matches)
	}

	targets := make([]TradeTarget, limit)
	for i := 0; i < limit; i++ {
		m := matches[i]
		reason := fmt.Sprintf("Has %s depth, needs %s", m.theirSurplus, m.yourSurplus)
		targets[i] = TradeTarget{
			TeamName:        m.teamName,
			Reason:          reason,
			YourSurplus:     m.yourSurplus,
			TheirSurplus:    m.theirSurplus,
			YourSurplusKTC:  m.yourSurplusKTC,
			TheirSurplusKTC: m.theirSurplusKTC,
		}
	}

	return targets
}

func calculateLeagueTrends(transactions []Transaction, freeAgentsByPos map[string][]PlayerRow, players map[string]interface{}) LeagueTrends {
	trends := LeagueTrends{
		PositionScarcity: make(map[string]int),
	}

	// Count transactions per team
	teamActivity := make(map[string]*TeamActivity)

	// Count waiver claims per player
	waiverClaims := make(map[string]int) // player name -> claim count
	playerPositions := make(map[string]string)
	lastClaimed := make(map[string]time.Time)

	for _, txn := range transactions {
		// Count transactions for teams
		for _, teamName := range txn.TeamNames {
			if _, exists := teamActivity[teamName]; !exists {
				teamActivity[teamName] = &TeamActivity{
					TeamName: teamName,
				}
			}

			teamActivity[teamName].Transactions++

			if txn.Type == "trade" {
				teamActivity[teamName].Trades++
				trends.TradeVolume++
			} else if txn.Type == "waiver" {
				teamActivity[teamName].WaiverClaims++
				trends.WaiverVolume++

				// Track waiver claims per player
				if txn.AddedPlayer != "" {
					waiverClaims[txn.AddedPlayer]++
					if txn.Timestamp.After(lastClaimed[txn.AddedPlayer]) {
						lastClaimed[txn.AddedPlayer] = txn.Timestamp
					}

					// Try to find player position
					for _, p := range players {
						pMap, ok := p.(map[string]interface{})
						if !ok {
							continue
						}
						playerName := getPlayerName(pMap)
						if playerName == txn.AddedPlayer {
							if pos, ok := pMap["position"].(string); ok {
								playerPositions[txn.AddedPlayer] = pos
							}
							break
						}
					}
				}
			}
		}
	}

	// Determine activity levels for teams
	for _, activity := range teamActivity {
		if activity.Transactions >= 5 {
			activity.ActivityLevel = "Very Active"
		} else if activity.Transactions >= 2 {
			activity.ActivityLevel = "Active"
		} else {
			activity.ActivityLevel = "Quiet"
		}

		trends.MostActiveTeams = append(trends.MostActiveTeams, *activity)
	}

	// Sort teams by transaction count
	sort.Slice(trends.MostActiveTeams, func(i, j int) bool {
		return trends.MostActiveTeams[i].Transactions > trends.MostActiveTeams[j].Transactions
	})

	// Create hot waiver players list
	for playerName, count := range waiverClaims {
		if count < 2 {
			continue // Only show players claimed multiple times
		}

		lastClaimedTime := lastClaimed[playerName]
		timeAgo := formatTimeAgo(lastClaimedTime)

		trends.HotWaiverPlayers = append(trends.HotWaiverPlayers, WaiverActivity{
			PlayerName:  playerName,
			Position:    playerPositions[playerName],
			ClaimCount:  count,
			LastClaimed: timeAgo,
		})
	}

	// Sort by claim count
	sort.Slice(trends.HotWaiverPlayers, func(i, j int) bool {
		return trends.HotWaiverPlayers[i].ClaimCount > trends.HotWaiverPlayers[j].ClaimCount
	})

	// Calculate position scarcity (number of available players per position)
	for pos, players := range freeAgentsByPos {
		trends.PositionScarcity[pos] = len(players)
	}

	return trends
}

func formatTimeAgo(t time.Time) string {
	if t.IsZero() {
		return "Unknown"
	}

	now := time.Now()
	diff := now.Sub(t)

	if diff < time.Hour {
		mins := int(diff.Minutes())
		if mins <= 1 {
			return "just now"
		}
		return fmt.Sprintf("%d mins ago", mins)
	} else if diff < 24*time.Hour {
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else if diff < 7*24*time.Hour {
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
	return fmt.Sprintf("%d days ago", int(diff.Hours()/24))
}

func calculateProjectedDraftPicks(draftPicks []DraftPick, teamAges []TeamAgeData, targetYear int) []ProjectedDraftPick {
	projectedPicks := []ProjectedDraftPick{}

	// Create a map of roster ID to team standing/data
	rosterToStanding := make(map[int]TeamAgeData)
	for _, team := range teamAges {
		rosterToStanding[team.RosterID] = team
	}

	// Get league size from team count
	leagueSize := len(teamAges)

	debugLog("[DEBUG] Calculating projected draft picks for year %d with %d teams", targetYear, leagueSize)

	// Process each draft pick
	for _, pick := range draftPicks {
		// Only process picks for the target year
		if pick.Year != targetYear {
			continue
		}

		// Find the current owner's standing
		// Note: If the pick is owned by user but came from another team, we use the original owner's standing
		teamForStanding := rosterToStanding[pick.RosterID]

		// Find the actual team whose standing determines the pick position
		// If there's an original name, this pick was traded, so we need to find that team's standing
		var pickPositionTeam TeamAgeData
		if pick.OriginalName != "" && pick.OriginalName != "You" {
			// Find the original owner's roster ID and standing
			for _, team := range teamAges {
				if team.TeamName == pick.OriginalName {
					pickPositionTeam = team
					break
				}
			}
			if pickPositionTeam.RosterID == 0 {
				// Couldn't find original team, use current owner
				pickPositionTeam = teamForStanding
			}
		} else {
			pickPositionTeam = teamForStanding
		}

		// Calculate projected position
		// In dynasty drafts, worst team (highest rank) gets pick 1.01
		// Rank 1 = best team (most wins) -> last pick
		// Rank 12 = worst team (fewest wins) -> first pick
		projectedPosition := leagueSize - pickPositionTeam.Rank + 1

		// Calculate overall pick number
		overallPick := (pick.Round-1)*leagueSize + projectedPosition

		// Format team record
		// Rank 1 = most wins, so approximate wins/losses
		wins := leagueSize - pickPositionTeam.Rank
		losses := pickPositionTeam.Rank - 1
		teamRecord := fmt.Sprintf("%d-%d", wins, losses)

		projectedPicks = append(projectedPicks, ProjectedDraftPick{
			Year:              pick.Year,
			Round:             pick.Round,
			OverallPick:       overallPick,
			ProjectedPosition: projectedPosition,
			OwnerName:         pick.OwnerName,
			OriginalOwner:     pick.OriginalName,
			CurrentStanding:   pickPositionTeam.Rank,
			TeamRecord:        teamRecord,
			IsYours:           pick.IsYours,
		})

		debugLog("[DEBUG] Projected pick: Round %d, Position %d (Overall %d), Owner: %s, Original: %s, Current Standing: %d (%s)",
			pick.Round, projectedPosition, overallPick, pick.OwnerName, pick.OriginalName, pickPositionTeam.Rank, teamRecord)
	}

	// Sort by overall pick number
	sort.Slice(projectedPicks, func(i, j int) bool {
		return projectedPicks[i].OverallPick < projectedPicks[j].OverallPick
	})

	return projectedPicks
}
