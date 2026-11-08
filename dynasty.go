// ABOUTME: Dynasty league analysis functions for SleeperPy
// ABOUTME: Includes functions for player valuations, trade targets, power rankings, and breakout candidates

package main

import (
	"fmt"
	"math"
	"sort"
	"time"
)

const dynastyValueDisagreementThreshold = 30.0 // percent

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
			rows[i].DynastyValueLowConf = dynastyValueSourcesDisagree(val, isSuperFlex)
		}
	}
}

// dynastyValueSourcesDisagree reports whether DynastyProcess and FantasyCalc
// disagree by 30%+ on a player's value. Only meaningful when both sources
// have data for this player -- with a single source there's nothing to
// compare, so this returns false rather than flagging every player.
func dynastyValueSourcesDisagree(val DynastyValue, isSuperFlex bool) bool {
	dp, fc := val.DynastyProcessValue1QB, val.FantasyCalcValue1QB
	if isSuperFlex {
		dp, fc = val.DynastyProcessValue2QB, val.FantasyCalcValue2QB
	}
	if dp <= 0 || fc <= 0 {
		return false
	}
	avg := float64(dp+fc) / 2
	diffPct := math.Abs(float64(dp-fc)) / avg * 100
	return diffPct >= dynastyValueDisagreementThreshold
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

// buildAllFreeAgentsByDynastyValue ranks every unowned skill-position player
// with a known dynasty value, regardless of whether they'd actually improve
// the user's roster. This is the full waiver-wire board ("who's the best
// asset out there") as opposed to the narrower upgrade-only add/drop
// suggestions, which only surface players that beat something already
// rostered. taxiYears marks each player IsTaxiEligible when their NFL
// experience is within the league's taxi squad window (see
// resolveTaxiSettings) -- pass 0 to skip taxi eligibility entirely (e.g. the
// league has no taxi squad).
func buildAllFreeAgentsByDynastyValue(players map[string]interface{}, rostered map[string]bool, dynastyValues map[string]DynastyValue, isSuperFlex bool, taxiYears int, limit int) []PlayerRow {
	all := []PlayerRow{}
	for pid, p := range players {
		if rostered[pid] {
			continue
		}
		pm, ok := p.(map[string]interface{})
		if !ok || pm["active"] == false {
			continue
		}
		pos, _ := pm["position"].(string)
		if pos != "QB" && pos != "RB" && pos != "WR" && pos != "TE" {
			continue // Only skill positions carry dynasty value
		}
		team, _ := pm["team"].(string)
		if team == "" {
			continue // Undrafted rookies / players without an NFL team
		}

		name := getPlayerName(pm)
		val, exists := dynastyValues[normalizeName(name)]
		if !exists {
			continue
		}
		value := val.Value1QB
		if isSuperFlex {
			value = val.Value2QB
		}
		if value <= 0 {
			continue
		}

		yearsExp := 0
		if ye, ok := pm["years_exp"].(float64); ok {
			yearsExp = int(ye)
		}

		all = append(all, PlayerRow{
			Pos:            pos,
			Name:           name,
			Team:           team,
			DynastyValue:   value,
			IsFreeAgent:    true,
			YearsExp:       yearsExp,
			IsTaxiEligible: taxiYears > 0 && yearsExp <= taxiYears,
		})
	}

	sort.Slice(all, func(i, j int) bool {
		return all[i].DynastyValue > all[j].DynastyValue
	})

	if limit > 0 && len(all) > limit {
		all = all[:limit]
	}
	return all
}

// buildTaxiSquadNameSet converts a roster's taxi squad player IDs into a set
// of normalized names -- the same lookup key PlayerRow enrichment functions
// already use, since rows only carry a display name, not the player_id they
// were built from.
func buildTaxiSquadNameSet(taxiIDs []string, players map[string]interface{}) map[string]bool {
	if len(taxiIDs) == 0 {
		return nil
	}
	set := make(map[string]bool, len(taxiIDs))
	for _, pid := range taxiIDs {
		p, ok := players[pid].(map[string]interface{})
		if !ok {
			continue
		}
		name := getPlayerName(p)
		if name == "" {
			continue
		}
		set[normalizeName(name)] = true
	}
	return set
}

// enrichRowsWithTaxiSquad flags rows currently stashed on the taxi squad.
// Dynasty leagues treat taxi players differently from the rest of the
// bench -- they're intentionally low-value prospect stashes, not roster
// mistakes -- so drop/cut value comparisons need to be able to exclude them.
func enrichRowsWithTaxiSquad(rows []PlayerRow, taxiNames map[string]bool) {
	if len(taxiNames) == 0 {
		return
	}
	for i := range rows {
		if taxiNames[normalizeName(stripHTML(rows[i].Name))] {
			rows[i].IsTaxiSquad = true
		}
	}
}

// buildTaxiSquadRows builds PlayerRows directly from a roster's taxi squad
// player IDs and dynasty values, rather than filtering already-tiered
// bench rows. Taxi stashes are frequently un-drafted-by-Boris-Chen rookies,
// which land in the "unranked" bucket with Pos hardcoded to "?" -- fine for
// the tier tables, but useless for a real position-based value comparison.
// Going straight to the Sleeper player record sidesteps that entirely.
func buildTaxiSquadRows(taxiIDs []string, players map[string]interface{}, dynastyValues map[string]DynastyValue, isSuperFlex bool) []PlayerRow {
	rows := make([]PlayerRow, 0, len(taxiIDs))
	for _, pid := range taxiIDs {
		p, ok := players[pid].(map[string]interface{})
		if !ok {
			continue
		}
		name := getPlayerName(p)
		if name == "" {
			continue
		}
		pos, _ := p["position"].(string)

		row := PlayerRow{Pos: pos, Name: name, IsTaxiSquad: true}
		if val, exists := dynastyValues[normalizeName(name)]; exists {
			if isSuperFlex {
				row.DynastyValue = val.Value2QB
			} else {
				row.DynastyValue = val.Value1QB
			}
		}
		rows = append(rows, row)
	}
	return rows
}

// findLowestValueRosterPlayers picks the user's `limit` weakest dynasty
// assets on the active roster (starters + bench, excluding taxi squad
// stashes and anything with no dynasty value at all), plus every player
// currently on the taxi squad, and pairs each with the single best unowned
// player at the same position. Taxi squad players are always included in
// full (not just the weakest ones) since the whole point of a taxi stash is
// to track whether it's still the best available prospect at its position.
func findLowestValueRosterPlayers(startersRows, benchRows, taxiRows, allFreeAgentsByValue []PlayerRow, limit int) []LowestValuePlayer {
	owned := make([]PlayerRow, 0, len(startersRows)+len(benchRows))
	for _, row := range append(append([]PlayerRow{}, startersRows...), benchRows...) {
		if row.IsTaxiSquad || row.DynastyValue <= 0 {
			continue
		}
		owned = append(owned, row)
	}

	sort.Slice(owned, func(i, j int) bool {
		return owned[i].DynastyValue < owned[j].DynastyValue
	})
	if limit > 0 && len(owned) > limit {
		owned = owned[:limit]
	}

	taxi := append([]PlayerRow{}, taxiRows...)
	sort.Slice(taxi, func(i, j int) bool {
		return taxi[i].DynastyValue < taxi[j].DynastyValue
	})

	bestByPos := make(map[string]PlayerRow, 4)
	for _, fa := range allFreeAgentsByValue {
		if _, exists := bestByPos[fa.Pos]; !exists {
			bestByPos[fa.Pos] = fa
		}
	}

	comparisons := make([]LowestValuePlayer, 0, len(owned)+len(taxi))
	for _, row := range append(owned, taxi...) {
		best := bestByPos[row.Pos]
		gap := 0
		if best.Name != "" {
			gap = best.DynastyValue - row.DynastyValue
		}
		comparisons = append(comparisons, LowestValuePlayer{
			YourPlayer:    row,
			BestAvailable: best,
			ValueGap:      gap,
		})
	}
	return comparisons
}

// buildTaxiSquadSuggestions recommends the best taxi-eligible free agents to
// stash, in order of dynasty value. While openSlots remain, prospects go
// straight into an unused slot; once slots are full, a prospect is only
// suggested if it clearly beats the weakest player currently on the taxi
// squad (as a swap). allFreeAgentsByValue must already be sorted by dynasty
// value descending (see buildAllFreeAgentsByDynastyValue) and have
// IsTaxiEligible set.
func buildTaxiSquadSuggestions(allFreeAgentsByValue, currentTaxiRows []PlayerRow, openSlots int, limit int) []TaxiSuggestion {
	var worstTaxi PlayerRow
	for _, row := range currentTaxiRows {
		if worstTaxi.Name == "" || row.DynastyValue < worstTaxi.DynastyValue {
			worstTaxi = row
		}
	}

	suggestions := []TaxiSuggestion{}
	remainingOpen := openSlots
	for _, fa := range allFreeAgentsByValue {
		if !fa.IsTaxiEligible {
			continue
		}
		if limit > 0 && len(suggestions) >= limit {
			break
		}

		if remainingOpen > 0 {
			suggestions = append(suggestions, TaxiSuggestion{Prospect: fa, HasOpenSlot: true})
			remainingOpen--
			continue
		}

		if worstTaxi.Name != "" && fa.DynastyValue > worstTaxi.DynastyValue {
			suggestions = append(suggestions, TaxiSuggestion{Prospect: fa, DropFromTaxi: worstTaxi})
		}
	}
	return suggestions
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

// buildTrendingByName converts Sleeper's player_id-keyed trending-add counts
// into a normalized-name-keyed map, so it can be looked up the same way
// dynasty values are (PlayerRow only carries a display name, not the
// player_id it was built from).
func buildTrendingByName(players map[string]interface{}) map[string]int {
	adds := fetchTrendingPlayers("add")
	if len(adds) == 0 {
		return nil
	}
	byName := make(map[string]int, len(adds))
	for playerID, count := range adds {
		p, ok := players[playerID].(map[string]interface{})
		if !ok {
			continue
		}
		name := getPlayerName(p)
		if name == "" {
			continue
		}
		byName[normalizeName(name)] = count
	}
	return byName
}

// findCutCandidates identifies bench players who are reasonable drop
// candidates: low dynasty value, not good enough to actually start, and not
// currently trending up on waivers elsewhere. A spike in adds usually means
// something changed (injury ahead of them, role change) that's worth
// checking before cutting, so trending players are excluded even if their
// dynasty value is low.
func findCutCandidates(benchRows []PlayerRow, trendingAddsByName map[string]int) []PlayerRow {
	const lowValueThreshold = 200

	candidates := []PlayerRow{}
	for _, row := range benchRows {
		if row.ShouldSwapIn {
			continue // good enough to start over someone -- not a cut candidate
		}
		if row.DynastyValue <= 0 || row.DynastyValue >= lowValueThreshold {
			continue
		}
		if trendingAddsByName != nil {
			normName := normalizeName(stripHTML(row.Name))
			if trendingAddsByName[normName] > 0 {
				continue
			}
		}
		candidates = append(candidates, row)
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].DynastyValue < candidates[j].DynastyValue
	})
	if len(candidates) > 10 {
		candidates = candidates[:10]
	}
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

// buildTopRookies surfaces this year's actual rookies (years_exp == 0, i.e.
// currently in their first NFL season) with a known dynasty value, ranked
// highest first. A hardcoded draft-class snapshot goes stale the moment the
// real draft happens each April -- pulling straight from the live Sleeper
// player pool means this never needs a manual yearly update.
func buildTopRookies(players map[string]interface{}, dynastyValues map[string]DynastyValue, isSuperFlex bool, limit int) []RookieProspect {
	rookies := []RookieProspect{}
	for _, p := range players {
		pm, ok := p.(map[string]interface{})
		if !ok || pm["active"] == false {
			continue
		}
		yearsExp, ok := pm["years_exp"].(float64)
		if !ok || yearsExp != 0 {
			continue
		}
		pos, _ := pm["position"].(string)
		if pos != "QB" && pos != "RB" && pos != "WR" && pos != "TE" {
			continue // Only skill positions carry dynasty value
		}

		name := getPlayerName(pm)
		val, exists := dynastyValues[normalizeName(name)]
		if !exists {
			continue
		}
		value := val.Value1QB
		if isSuperFlex {
			value = val.Value2QB
		}
		if value <= 0 {
			continue
		}

		college, _ := pm["college"].(string)
		rookies = append(rookies, RookieProspect{Name: name, Position: pos, College: college, Value: value})
	}

	sort.Slice(rookies, func(i, j int) bool {
		return rookies[i].Value > rookies[j].Value
	})
	if limit > 0 && len(rookies) > limit {
		rookies = rookies[:limit]
	}
	for i := range rookies {
		rookies[i].Rank = i + 1
	}
	return rookies
}

// calculateStarterValue returns the KTC value of the best starting lineup a
// roster can field under the league's actual slot counts, instead of the
// sum of every rostered player. Using full-roster value for power rankings
// lets a team inflate its score just by hoarding bench depth at one
// position (e.g. stashing four extra QBs in a superflex league that only
// starts two) -- starter value can't be padded that way.
//
// Slot eligibility is a strict nesting -- REC_FLEX (WR/TE) is a subset of
// FLEX (RB/WR/TE), which is a subset of SUPER_FLEX (QB/RB/WR/TE) -- so
// filling fixed single-position slots first, then flex slots narrowest to
// widest with players sorted by value descending, is optimal (a standard
// greedy-on-a-laminar-family argument): a player only "wastes" a wide slot
// when no narrower slot can use them, which is exactly what this order
// avoids.
func calculateStarterValue(rosterPlayerIDs []string, players map[string]interface{}, dynastyValues map[string]DynastyValue, isSuperFlex bool, rosterPositions []string) int {
	type candidate struct {
		pos   string
		value int
	}

	byPos := map[string][]candidate{}
	for _, pid := range rosterPlayerIDs {
		p, ok := players[pid].(map[string]interface{})
		if !ok {
			continue
		}
		pos, _ := p["position"].(string)
		if pos != "QB" && pos != "RB" && pos != "WR" && pos != "TE" {
			continue
		}
		name := getPlayerName(p)
		val, exists := dynastyValues[normalizeName(name)]
		if !exists {
			continue
		}
		value := val.Value1QB
		if isSuperFlex {
			value = val.Value2QB
		}
		if value <= 0 {
			continue
		}
		byPos[pos] = append(byPos[pos], candidate{pos: pos, value: value})
	}
	for pos := range byPos {
		sort.Slice(byPos[pos], func(i, j int) bool { return byPos[pos][i].value > byPos[pos][j].value })
	}

	fixedSlots := map[string]int{}
	var flexSlots, superFlexSlots, recFlexSlots int
	for _, slot := range rosterPositions {
		switch slot {
		case "QB", "RB", "WR", "TE":
			fixedSlots[slot]++
		case "FLEX", "WRRB_FLEX", "WR_RB_FLEX":
			flexSlots++
		case "SUPER_FLEX":
			superFlexSlots++
		case "REC_FLEX", "WR_TE_FLEX":
			recFlexSlots++
		}
	}

	used := map[string]int{}
	take := func(pos string) (int, bool) {
		idx := used[pos]
		list := byPos[pos]
		if idx >= len(list) {
			return 0, false
		}
		used[pos] = idx + 1
		return list[idx].value, true
	}

	total := 0
	for pos, n := range fixedSlots {
		for i := 0; i < n; i++ {
			if v, ok := take(pos); ok {
				total += v
			}
		}
	}

	// Remaining candidates across all positions, best value first, greedily
	// claimed by the narrowest flex slot type they still fit.
	var remaining []candidate
	for pos, list := range byPos {
		remaining = append(remaining, list[used[pos]:]...)
	}
	sort.Slice(remaining, func(i, j int) bool { return remaining[i].value > remaining[j].value })

	for _, c := range remaining {
		switch c.pos {
		case "WR", "TE":
			if recFlexSlots > 0 {
				total += c.value
				recFlexSlots--
			} else if flexSlots > 0 {
				total += c.value
				flexSlots--
			} else if superFlexSlots > 0 {
				total += c.value
				superFlexSlots--
			}
		case "RB":
			if flexSlots > 0 {
				total += c.value
				flexSlots--
			} else if superFlexSlots > 0 {
				total += c.value
				superFlexSlots--
			}
		case "QB":
			if superFlexSlots > 0 {
				total += c.value
				superFlexSlots--
			}
		}
	}

	return total
}

func calculatePowerRankings(teamAges []TeamAgeData) []PowerRanking {
	rankings := []PowerRanking{}

	for _, team := range teamAges {
		rankings = append(rankings, PowerRanking{
			TeamName:     team.TeamName,
			RosterID:     team.RosterID,
			RosterValue:  team.RosterValue,
			StarterValue: team.StarterValue,
			Wins:         team.Wins,
			Losses:       team.Losses,
			AvgAge:       team.AvgAge,
			IsUserTeam:   team.IsUserTeam,
			StandingRank: team.Rank,
		})
	}

	// Rank by starter value (highest first), not full roster value -- this
	// is the number superflex teams can't inflate by hoarding backup QBs on
	// the bench, since bench players never make it into calculateStarterValue.
	sort.Slice(rankings, func(i, j int) bool {
		return rankings[i].StarterValue > rankings[j].StarterValue
	})

	for i := range rankings {
		rankings[i].ValueRank = i + 1
	}

	// A team is only "contending" if its starting lineup is actually
	// top-half in the league by value -- age alone used to decide this,
	// which let a young team full of weak starters get called "Contending"
	// just for not being old, and let a strong veteran-laden team that
	// wasn't literally the oldest miss "Win Now".
	topHalf := (len(rankings) + 1) / 2
	for i := range rankings {
		valueStrong := rankings[i].ValueRank <= topHalf
		switch {
		case rankings[i].AvgAge < 24.5:
			rankings[i].Strategy = "Rebuilding"
		case valueStrong && rankings[i].AvgAge > 27.0:
			rankings[i].Strategy = "Win Now"
		case valueStrong:
			rankings[i].Strategy = "Contending"
		default:
			rankings[i].Strategy = "Rebuilding"
		}
	}

	for i := range rankings {
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

// A perfectly bidirectional 30%/15% surplus/deficit match (both sides) is
// rare enough in practice that it can produce zero trade targets for a
// perfectly normal roster. 25%/20% still requires a real imbalance, just not
// an extreme one.
const (
	tradeSurplusThreshold = 0.25
	tradeDeficitThreshold = 0.20
)

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
		pos string
		pct float64
	}

	userSurplus := []posNeed{}
	userDeficit := []posNeed{}

	if userQBPct > tradeSurplusThreshold {
		userSurplus = append(userSurplus, posNeed{"QB", userQBPct})
	} else if userQBPct < tradeDeficitThreshold {
		userDeficit = append(userDeficit, posNeed{"QB", userQBPct})
	}

	if userRBPct > tradeSurplusThreshold {
		userSurplus = append(userSurplus, posNeed{"RB", userRBPct})
	} else if userRBPct < tradeDeficitThreshold {
		userDeficit = append(userDeficit, posNeed{"RB", userRBPct})
	}

	if userWRPct > tradeSurplusThreshold {
		userSurplus = append(userSurplus, posNeed{"WR", userWRPct})
	} else if userWRPct < tradeDeficitThreshold {
		userDeficit = append(userDeficit, posNeed{"WR", userWRPct})
	}

	if userTEPct > tradeSurplusThreshold {
		userSurplus = append(userSurplus, posNeed{"TE", userTEPct})
	} else if userTEPct < tradeDeficitThreshold {
		userDeficit = append(userDeficit, posNeed{"TE", userTEPct})
	}

	debugLog("[DEBUG] User positional breakdown: QB=%.1f%%, RB=%.1f%%, WR=%.1f%%, TE=%.1f%%",
		userQBPct*100, userRBPct*100, userWRPct*100, userTEPct*100)
	debugLog("[DEBUG] User surplus positions: %v", userSurplus)
	debugLog("[DEBUG] User deficit positions: %v", userDeficit)

	// If no clear surplus/deficit, no trade recommendations
	if len(userSurplus) == 0 || len(userDeficit) == 0 {
		debugLog("[DEBUG] No trade targets - need both surplus (>25%%) and deficit (<20%%) positions")
		return nil
	}

	type tradeMatch struct {
		rosterID        int
		teamName        string
		complementarity float64
		yourSurplus     string
		theirSurplus    string
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

				switch userDef.pos {
				case "QB":
					teamSurPct = teamQBPct
				case "RB":
					teamSurPct = teamRBPct
				case "WR":
					teamSurPct = teamWRPct
				case "TE":
					teamSurPct = teamTEPct
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
				if teamSurPct > tradeSurplusThreshold && teamDefPct < tradeDeficitThreshold {
					complementarity := (userSur.pct - teamDefPct) + (teamSurPct - userDef.pct)

					if bestMatch == nil || complementarity > bestMatch.complementarity {
						bestMatch = &tradeMatch{
							rosterID:        rosterID,
							teamName:        teamNames[rosterID],
							complementarity: complementarity,
							yourSurplus:     userSur.pos,
							theirSurplus:    userDef.pos,
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
			TeamName:     m.teamName,
			Reason:       reason,
			YourSurplus:  m.yourSurplus,
			TheirSurplus: m.theirSurplus,
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

		teamRecord := fmt.Sprintf("%d-%d", pickPositionTeam.Wins, pickPositionTeam.Losses)

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
