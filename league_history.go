// ABOUTME: Resolves a dynasty league's past champions by walking the
// ABOUTME: previous_league_id chain Sleeper uses to link seasons together.

package main

import (
	"sync"
	"time"
)

// maxLeagueHistorySeasons caps how many previous_league_id hops we follow.
// Each hop costs 3 API calls (league, winners_bracket, rosters+users), so
// this bounds worst-case latency for very old leagues.
const maxLeagueHistorySeasons = 10

type leagueHistoryCacheT struct {
	sync.RWMutex
	data      map[string][]LeagueChampion
	timestamp map[string]time.Time
	ttl       time.Duration
}

// findChampionRosterID returns the winning roster_id from a winners_bracket
// response -- the match flagged "p": 1 is Sleeper's championship (1st place)
// game. Returns 0 if the bracket has no decided finals yet (season still in
// progress, or the bracket hasn't been generated).
func findChampionRosterID(bracket []map[string]interface{}) int {
	for _, m := range bracket {
		if p, ok := m["p"].(float64); !ok || p != 1 {
			continue
		}
		if w, ok := m["w"].(float64); ok {
			return int(w)
		}
	}
	return 0
}

// buildChampionRecord resolves a champion roster_id into a display-ready
// LeagueChampion, using the same team-name fallback (metadata.team_name,
// else owner display name) as TeamAgeData elsewhere in dynasty.go.
func buildChampionRecord(season string, championRosterID int, rosters []map[string]interface{}, userNames map[string]string) (LeagueChampion, bool) {
	for _, r := range rosters {
		rosterID, _ := r["roster_id"].(float64)
		if int(rosterID) != championRosterID {
			continue
		}
		ownerID, _ := r["owner_id"].(string)
		ownerName := userNames[ownerID]
		if ownerName == "" {
			ownerName = "Unknown"
		}
		teamName := ownerName
		if metadata, ok := r["metadata"].(map[string]interface{}); ok {
			if tn, ok := metadata["team_name"].(string); ok && tn != "" {
				teamName = tn
			}
		}
		return LeagueChampion{
			Season:    season,
			TeamName:  teamName,
			OwnerName: ownerName,
			OwnerID:   ownerID,
		}, true
	}
	return LeagueChampion{}, false
}

// fetchLeagueHistoryUncached walks the previous_league_id chain starting at
// leagueID, resolving each season's champion. Most recent season first.
// IsUserTeam is left unset here on purpose -- the result is cached and
// shared across whichever Sleeper user is viewing the league, so "is this
// your team" has to be applied per-viewer after the cache lookup, not baked
// into the cached record.
func fetchLeagueHistoryUncached(leagueID string) []LeagueChampion {
	var history []LeagueChampion
	seen := map[string]bool{}

	for i := 0; i < maxLeagueHistorySeasons && leagueID != "" && leagueID != "0" && !seen[leagueID]; i++ {
		seen[leagueID] = true

		league, err := appProvider.FetchLeague(leagueID)
		if err != nil || league == nil {
			break
		}
		season, _ := league["season"].(string)
		prevLeagueID, _ := league["previous_league_id"].(string)

		bracket, err := appProvider.FetchLeagueWinnersBracket(leagueID)
		if err == nil {
			if championRosterID := findChampionRosterID(bracket); championRosterID > 0 {
				rosters, _ := appProvider.FetchLeagueRosters(leagueID)
				leagueUsers, _ := appProvider.FetchLeagueUsers(leagueID)
				if champ, ok := buildChampionRecord(season, championRosterID, rosters, buildUserNameMap(leagueUsers)); ok {
					history = append(history, champ)
				}
			}
		}

		leagueID = prevLeagueID
	}

	return history
}

// fetchLeagueHistory returns fetchLeagueHistoryUncached's result (cached per
// leagueID, since past champions never change) with IsUserTeam filled in for
// the given viewer.
func fetchLeagueHistory(leagueID, viewerUserID string) []LeagueChampion {
	leagueHistoryCache.RLock()
	cached, exists := leagueHistoryCache.data[leagueID]
	fetchedAt, hasTimestamp := leagueHistoryCache.timestamp[leagueID]
	leagueHistoryCache.RUnlock()

	if !exists || !hasTimestamp || time.Since(fetchedAt) > leagueHistoryCache.ttl {
		cached = fetchLeagueHistoryUncached(leagueID)
		leagueHistoryCache.Lock()
		leagueHistoryCache.data[leagueID] = cached
		leagueHistoryCache.timestamp[leagueID] = time.Now()
		leagueHistoryCache.Unlock()
	}

	out := make([]LeagueChampion, len(cached))
	for i, c := range cached {
		c.IsUserTeam = c.OwnerID != "" && c.OwnerID == viewerUserID
		out[i] = c
	}
	return out
}
