// ABOUTME: HTTP handlers for SleeperPy web application
// ABOUTME: Includes route handlers for index, lookup, error rendering, and static pages

package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

func visitorLogging(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		if ipHeader := r.Header.Get("X-Forwarded-For"); ipHeader != "" {
			ip = ipHeader
		}
		if logLevel == "debug" {
			log.Printf("[VISITOR] IP: %s Path: %s", ip, r.URL.Path)
		}
		totalVisitors.Inc()
		next(w, r)
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	trackPageView(r.URL.Path)
	trackUserAgent(r.UserAgent())

	savedUsername := ""
	if cookie, err := r.Cookie("sleeper_username"); err == nil {
		savedUsername = cookie.Value
	}
	templates.ExecuteTemplate(w, "index.html", IndexPage{SavedUsername: savedUsername})
}

func signoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie := &http.Cookie{
		Name:     "sleeper_username",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func privacyHandler(w http.ResponseWriter, r *http.Request) {
	trackPageView(r.URL.Path)
	tmpl := template.Must(template.ParseFiles("templates/privacy.html"))
	if err := tmpl.Execute(w, nil); err != nil {
		http.Error(w, "Error rendering privacy policy", http.StatusInternalServerError)
		log.Printf("Error rendering privacy.html: %v", err)
	}
}

func termsHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("templates/terms.html"))
	if err := tmpl.Execute(w, nil); err != nil {
		http.Error(w, "Error rendering terms of service", http.StatusInternalServerError)
		log.Printf("Error rendering terms.html: %v", err)
	}
}

func pricingHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("templates/pricing.html"))
	if err := tmpl.Execute(w, nil); err != nil {
		http.Error(w, "Error rendering pricing page", http.StatusInternalServerError)
		log.Printf("Error rendering pricing.html: %v", err)
	}
}

func aboutHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("templates/about.html"))
	if err := tmpl.Execute(w, nil); err != nil {
		http.Error(w, "Error rendering about page", http.StatusInternalServerError)
		log.Printf("Error rendering about.html: %v", err)
	}
}

func faqHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("templates/faq.html"))
	if err := tmpl.Execute(w, nil); err != nil {
		http.Error(w, "Error rendering FAQ page", http.StatusInternalServerError)
		log.Printf("Error rendering faq.html: %v", err)
	}
}

func robotsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprint(w, `User-agent: *
Allow: /
Disallow: /lookup
Disallow: /metrics
Disallow: /signout

Sitemap: https://sleeperpy.com/sitemap.xml
`)
}

func sitemapHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/xml")
	fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url>
    <loc>https://sleeperpy.com/</loc>
    <priority>1.0</priority>
  </url>
  <url>
    <loc>https://sleeperpy.com/about</loc>
    <priority>0.8</priority>
  </url>
  <url>
    <loc>https://sleeperpy.com/faq</loc>
    <priority>0.7</priority>
  </url>
  <url>
    <loc>https://sleeperpy.com/pricing</loc>
    <priority>0.7</priority>
  </url>
  <url>
    <loc>https://sleeperpy.com/demo</loc>
    <priority>0.6</priority>
  </url>
  <url>
    <loc>https://sleeperpy.com/privacy</loc>
    <priority>0.3</priority>
  </url>
  <url>
    <loc>https://sleeperpy.com/terms</loc>
    <priority>0.3</priority>
  </url>
</urlset>
`)
}

func demoHandler(w http.ResponseWriter, r *http.Request) {
	trackPageView(r.URL.Path)
	demoLeague := LeagueData{
		LeagueName: "Example League (Demo)",
		Scoring:    "PPR",
		IsDynasty:  false,
		HasMatchups: true,
		LeagueSize: 12,
		RosterSlots: "1 QB, 2 RB, 3 WR, 1 TE, 1 FLEX, 1 K, 1 DEF, 5 BN",
		Starters: []PlayerRow{
			{Pos: "QB", Name: "Patrick Mahomes", Tier: 1},
			{Pos: "RB", Name: "Saquon Barkley", Tier: 2},
			{Pos: "RB", Name: "Derrick Henry", Tier: 5, IsTierWorseThanBench: true},
			{Pos: "WR", Name: "CeeDee Lamb", Tier: 1},
			{Pos: "WR", Name: "Garrett Wilson", Tier: 4},
			{Pos: "WR", Name: "Terry McLaurin", Tier: 5},
			{Pos: "TE", Name: "Travis Kelce", Tier: 1},
			{Pos: "FLEX", Name: "Josh Jacobs", Tier: 6, IsFlex: true},
		},
		Bench: []PlayerRow{
			{Pos: "QB", Name: "Justin Herbert", Tier: 3},
			{Pos: "RB", Name: "Aaron Jones", Tier: 4, ShouldSwapIn: true},
			{Pos: "WR", Name: "Brandon Aiyuk", Tier: 3},
			{Pos: "WR", Name: "Amari Cooper", Tier: 6},
			{Pos: "TE", Name: "Sam LaPorta", Tier: 2},
		},
		TopFreeAgents: []PlayerRow{
			{Pos: "RB", Name: "Brian Robinson", Tier: 5, IsFreeAgent: true, IsUpgrade: true, UpgradeFor: "Josh Jacobs", UpgradeType: "Starter"},
			{Pos: "WR", Name: "Zay Flowers", Tier: 4, IsFreeAgent: true, IsUpgrade: true, UpgradeFor: "Amari Cooper", UpgradeType: "Bench"},
		},
		AvgTier:    "3.1",
		AvgOppTier: "2.5",
		WinProb:    "38%",
	}

	page := TiersPage{
		Leagues:  []LeagueData{demoLeague},
		Username: "demo",
	}

	if err := templates.ExecuteTemplate(w, "tiers.html", page); err != nil {
		log.Printf("[ERROR] Demo template error: %v", err)
		http.Error(w, "Error rendering demo", http.StatusInternalServerError)
	}
}

func lookupHandler(w http.ResponseWriter, r *http.Request) {
	totalLookups.Inc()
	debugLog("[DEBUG] /lookup handler called")
	r.ParseForm()
	username := r.FormValue("username")
	llmMode := strings.ToLower(strings.TrimSpace(r.FormValue("llm")))
	debugLog("[DEBUG] Username submitted: %s", username)
	if username == "" {
		debugLog("[DEBUG] No username provided")
		totalErrors.Inc()
		renderError(w, "No username provided. Please enter your Sleeper username on the homepage and try again.")
		return
	}

	// 1. Get user ID
	user, err := fetchJSON(fmt.Sprintf("https://api.sleeper.app/v1/user/%s", username))
	if err != nil || user["user_id"] == nil {
		log.Printf("[ERROR] User not found or error: %v", err)
		totalErrors.Inc()
		renderError(w, fmt.Sprintf("User \"%s\" not found on Sleeper. Double-check your username (it's case-sensitive) — you can find it in the Sleeper app under Settings.", username))
		return
	}
	userID := user["user_id"].(string)

	// 2. Get leagues (check current year and previous year for dynasty leagues)
	year := time.Now().Year()
	leagues, err := fetchJSONArray(fmt.Sprintf("https://api.sleeper.app/v1/user/%s/leagues/nfl/%d", userID, year))
	if err != nil {
		debugLog("[DEBUG] Error fetching leagues for year %d: %v", year, err)
	}

	// Also fetch previous year leagues (dynasty leagues often stay on previous season)
	previousYear := year - 1
	previousYearLeagues, err := fetchJSONArray(fmt.Sprintf("https://api.sleeper.app/v1/user/%s/leagues/nfl/%d", userID, previousYear))
	if err != nil {
		debugLog("[DEBUG] Error fetching leagues for year %d: %v", previousYear, err)
	} else {
		// Append previous year leagues to current year leagues
		leagues = append(leagues, previousYearLeagues...)
	}

	if len(leagues) == 0 {
		log.Printf("[ERROR] No leagues found for user %s", userID)
		totalErrors.Inc()
		renderError(w, fmt.Sprintf("No leagues found for \"%s\". Make sure you've joined a Sleeper league for the current or previous NFL season. Dynasty leagues from last season are also checked.", username))
		return
	}

	// 3. Get current NFL week from Sleeper API
	state, err := fetchJSON("https://api.sleeper.app/v1/state/nfl")
	if err != nil {
		log.Printf("[ERROR] Could not get current NFL week: %v", err)
		totalErrors.Inc()
		renderError(w, "The Sleeper API is temporarily unavailable. Please try again in a few minutes.")
		return
	}
	week := int(state["week"].(float64))

	// 4. Get players data (cached for 1 hour)
	players, err := fetchPlayers()
	if err != nil {
		log.Printf("[ERROR] Could not fetch players data: %v", err)
		totalErrors.Inc()
		renderError(w, "Could not fetch player data from Sleeper. This usually means the Sleeper API is under heavy load — please try again in a few minutes.")
		return
	}

	// 4.5. Fetch dynasty values if any dynasty leagues exist
	var dynastyValues map[string]DynastyValue
	var dynastyValueDate string
	hasDynasty := false
	for _, league := range leagues {
		if isDynastyLeague(league) {
			hasDynasty = true
			break
		}
	}
	if hasDynasty {
		dynastyValues, dynastyValueDate = fetchDynastyValues()
	}

	// Check if user has premium access and if premium features are enabled
	isPremium := isPremiumUsername(username)
	premiumEnabled := hasOpenRouterKey()

	// 5. Process each league
	var leagueResults []LeagueData
	log.Printf("[INFO] Processed %s with %d leagues", username, len(leagues))
	totalLeagues.Add(float64(len(leagues)))

	// Sort leagues: dynasty leagues first, then by name
	sort.Slice(leagues, func(i, j int) bool {
		isDynastyI := isDynastyLeague(leagues[i])
		isDynastyJ := isDynastyLeague(leagues[j])
		if isDynastyI != isDynastyJ {
			return isDynastyI // Dynasty leagues come first
		}
		nameI, _ := leagues[i]["name"].(string)
		nameJ, _ := leagues[j]["name"].(string)
		return nameI < nameJ
	})

	for _, league := range leagues {
		leagueID := league["league_id"].(string)
		leagueName := league["name"].(string)

		// Check if this is a dynasty league
		isDynasty := isDynastyLeague(league)

		// Determine scoring type
		scoring := "PPR"
		if scoringSettings, ok := league["scoring_settings"].(map[string]interface{}); ok {
			if rec, ok := scoringSettings["rec"].(float64); ok {
				if rec == 0.5 {
					scoring = "Half PPR"
				} else if rec == 0.0 {
					scoring = "Standard"
				}
			}
		}

		// Debug: Check league roster_positions
		if rosterPositions, ok := league["roster_positions"].([]interface{}); ok {
			debugLog("[DEBUG] League roster_positions: %v", rosterPositions)
		} else {
			debugLog("[DEBUG] roster_positions not found in league settings")
		}

		// Get rosters and matchups
		rosters, err := fetchJSONArray(fmt.Sprintf("https://api.sleeper.app/v1/league/%s/rosters", leagueID))
		if err != nil {
			log.Printf("[ERROR] Error fetching rosters for league %s: %v", leagueName, err)
			totalErrors.Inc()
			continue
		}
		totalTeams.Add(float64(len(rosters)))

		matchups, err := fetchJSONArray(fmt.Sprintf("https://api.sleeper.app/v1/league/%s/matchups/%d", leagueID, week))
		hasMatchups := (err == nil && len(matchups) > 0)
		if !hasMatchups {
			if isDynasty {
				log.Printf("[INFO] No matchups for league %s week %d (dynasty league in offseason)", leagueName, week)
			} else {
				log.Printf("[ERROR] No matchups found for league %s week %d: %v", leagueName, week, err)
				totalErrors.Inc()
				continue
			}
		}

		// Find user roster
		var userRoster map[string]interface{}
		for _, r := range rosters {
			if r["owner_id"] == userID {
				userRoster = r
				break
			}
		}
		if userRoster == nil {
			log.Printf("[ERROR] No user roster found for league %s", leagueName)
			totalErrors.Inc()
			continue
		}

		starters := toStringSlice(userRoster["starters"])
		allPlayers := toStringSlice(userRoster["players"])
		irPlayers := toStringSlice(userRoster["reserve"])
		bench := diff(allPlayers, starters)
		// Add IR players to bench if not already present
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
		debugLog("[DEBUG] After IR merge, Bench: %v", bench)

		// Find opponent (only if we have matchups)
		var myMatchup, oppMatchup map[string]interface{}
		oppStarters := []string{}
		if hasMatchups {
			for _, m := range matchups {
				if m["roster_id"] == userRoster["roster_id"] {
					myMatchup = m
					break
				}
			}

			if myMatchup != nil {
				for _, m := range matchups {
					if m["matchup_id"] == myMatchup["matchup_id"] && m["roster_id"] != userRoster["roster_id"] {
						oppMatchup = m
						break
					}
				}
			}
			debugLog("[DEBUG] MyMatchup: %v | OppMatchup: %v", myMatchup, oppMatchup)

			if oppMatchup != nil {
				oppStarters = toStringSlice(oppMatchup["starters"])
			}
			debugLog("[DEBUG] Opponent Starters: %v", oppStarters)
		}

		// Fetch Boris Chen tiers
		borisTiers := fetchBorisTiers(scoring)
		debugLog("[DEBUG] Boris Tiers loaded for scoring: %s", scoring)

		// Get roster positions from league settings
		var leagueRosterPositions []string
		if rp, ok := league["roster_positions"].([]interface{}); ok {
			for _, pos := range rp {
				if posStr, ok := pos.(string); ok {
					leagueRosterPositions = append(leagueRosterPositions, posStr)
				}
			}
		}
		debugLog("[DEBUG] Parsed roster positions for league: %v", leagueRosterPositions)

		// Build rows for roster
		var benchUnrankedRows []PlayerRow
		benchRows, _, _ := buildRowsWithPositions(bench, players, borisTiers, false, nil, irPlayers, nil)
		debugLog("[DEBUG] Built benchRows: %v", benchRows)
		bestBenchTier := make(map[string]int)
		for _, row := range benchRows {
			pos := row.Pos
			tier, ok := row.Tier.(int)
			if ok && tier > 0 {
				if best, exists := bestBenchTier[pos]; !exists || tier < best {
					bestBenchTier[pos] = tier
				}
			}
		}
		debugLog("[DEBUG] Best bench tier map: %v", bestBenchTier)

		// Debug: Log starters with their designated positions
		for i, pid := range starters {
			if i < len(leagueRosterPositions) {
				if p, ok := players[pid].(map[string]interface{}); ok {
					name := getPlayerName(p)
					debugLog("[DEBUG] Starter %d: %s -> Designated position: %s", i, name, leagueRosterPositions[i])
				}
			}
		}

		startersRows, unrankedRows, starterTiers := buildRowsWithPositions(starters, players, borisTiers, true, leagueRosterPositions, irPlayers, bestBenchTier)
		debugLog("[DEBUG] Built startersRows: %v", startersRows)
		for i, row := range startersRows {
			debugLog("[DEBUG]   Row %d: Pos=%s, Name=%s, Tier=%v", i, row.Pos, row.Name, row.Tier)
		}

		// --- FLEX/SUPERFLEX MARKING ---
		// Mark FLEX and SUPERFLEX positions based on league roster configuration
		// and re-rank them using FLEX tiers (only for FLEX-eligible positions: RB/WR/TE)
		for i, row := range startersRows {
			if row.Pos == "FLEX" || row.Pos == "SUPER_FLEX" {
				pid := starters[i]
				if p, ok := players[pid].(map[string]interface{}); ok {
					name := getPlayerName(p)
					actualPos, _ := p["position"].(string)
					isFlexEligible := actualPos == "RB" || actualPos == "WR" || actualPos == "TE"

					flxTier := findTier(borisTiers["FLX"], name)
					if flxTier > 0 && isFlexEligible {
						// Re-rank FLEX-eligible positions (RB/WR/TE) using FLEX tier
						if row.Pos == "FLEX" {
							startersRows[i].IsFlex = true
							debugLog("[DEBUG] FLEX position: %s, re-ranking from tier %v to FLX tier %d", name, row.Tier, flxTier)
						} else {
							startersRows[i].IsSuperflex = true
							debugLog("[DEBUG] SUPERFLEX position: %s, re-ranking from tier %v to FLX tier %d", name, row.Tier, flxTier)
						}
						startersRows[i].Tier = flxTier
					} else {
						// For QBs or players without FLEX tier, just mark the slot but keep position tier
						if row.Pos == "FLEX" {
							startersRows[i].IsFlex = true
							debugLog("[DEBUG] FLEX position: %s, keeping position tier %v (pos: %s)", name, row.Tier, actualPos)
						} else {
							startersRows[i].IsSuperflex = true
							debugLog("[DEBUG] SUPERFLEX position: %s, keeping position tier %v (pos: %s)", name, row.Tier, actualPos)
						}
					}
				}
			}
		}

		// Recalculate starterTiers after FLEX re-ranking (whether from API or heuristic)
		starterTiers = []int{}
		for _, row := range startersRows {
			if t, ok := row.Tier.(int); ok && t > 0 {
				starterTiers = append(starterTiers, t)
			}
		}

		worstStarterTier := make(map[string]int)
		for _, row := range startersRows {
			pos := row.Pos
			tier, ok := row.Tier.(int)
			if ok && tier > 0 {
				if worst, exists := worstStarterTier[pos]; !exists || tier > worst {
					worstStarterTier[pos] = tier
				}
			}
		}
		debugLog("[DEBUG] Worst starter tier map: %v", worstStarterTier)
		benchRows, benchUnrankedRows, _ = buildRowsWithPositions(bench, players, borisTiers, false, nil, irPlayers, worstStarterTier)

		// Detect if this is a superflex league
		isSuperFlex := false
		for _, pos := range leagueRosterPositions {
			if pos == "SUPER_FLEX" {
				isSuperFlex = true
				break
			}
		}
		debugLog("[DEBUG] League is superflex: %v", isSuperFlex)

		// Enrich rows with dynasty values (if this is a dynasty league)
		if isDynasty && dynastyValues != nil {
			enrichRowsWithDynastyValues(startersRows, dynastyValues, isSuperFlex)
			enrichRowsWithDynastyValues(unrankedRows, dynastyValues, isSuperFlex)
			enrichRowsWithDynastyValues(benchRows, dynastyValues, isSuperFlex)
			enrichRowsWithDynastyValues(benchUnrankedRows, dynastyValues, isSuperFlex)
			debugLog("[DEBUG] Enriched starter and bench rows with dynasty values")
		}

		// Don't re-rank bench TEs to FLEX tier - keep them at their position-specific tier for display
		// Bench RB/WR/TE comparison with FLEX starters happens in free agent logic using FLEX tier lookup,
		// but we don't change the display tier here
		_, _, oppTiers := buildRowsWithPositions(oppStarters, players, borisTiers, true, nil, nil, nil)

		// --- FREE AGENTS LOGIC ---
		// Find all rostered player IDs
		rostered := map[string]bool{}
		for _, r := range rosters {
			for _, pid := range toStringSlice(r["players"]) {
				rostered[pid] = true
			}
			for _, pid := range toStringSlice(r["reserve"]) {
				rostered[pid] = true
			}
		}
		debugLog("[DEBUG] Rostered player IDs: %v", rostered)
		// Find free agents: not rostered, not on user's team, valid tier
		type faInfo struct {
			pid         string
			percent     float64
			tier        int
			pos         string
			name        string
			isUpgrade   bool
			upgradeFor  string
			upgradeType string
			tierDiff    int // How much better this FA is than the player it replaces
		}
		faList := []faInfo{}
		for pid, p := range players {
			if _, ok := rostered[pid]; ok {
				continue
			}
			pm, ok := p.(map[string]interface{})
			if !ok || pm["active"] == false {
				continue
			}
			pos, _ := pm["position"].(string)
			if pos == "" || (pos != "QB" && pos != "RB" && pos != "WR" && pos != "TE" && pos != "K" && pos != "DEF" && pos != "DST") {
				continue
			}
			name := getPlayerName(pm)
			lookupPos := pos
			if lookupPos == "DEF" {
				lookupPos = "DST"
			}

			// Determine which tier to use and how to compare
			// For RB/WR/TE: try FLEX tier first
			flexTier := 0
			posTier := 0
			isFlexEligible := pos == "RB" || pos == "WR" || pos == "TE"

			if isFlexEligible {
				flexTier = findTier(borisTiers["FLX"], name)
				posTier = findTier(borisTiers[lookupPos], name)
			} else {
				posTier = findTier(borisTiers[lookupPos], name)
			}

			// Skip if player has no valid tiers at all
			if flexTier <= 0 && posTier <= 0 {
				continue
			}

			percent := 0.0
			if v, ok := pm["roster_percent"].(float64); ok {
				percent = v
			} else if v, ok := pm["roster_percent"].(string); ok {
				percent, _ = strconv.ParseFloat(v, 64)
			}

			// Check if this FA is an upgrade for any position on the team
			isUpgrade := false
			upgradeFor := ""
			upgradeType := ""
			tierDiff := 0
			finalTier := 0

			// For RB/WR/TE: use position-specific tier for display and comparison
			// (Bench TEs now display their TE tier, not FLEX tier)
			if isFlexEligible && posTier > 0 {
				finalTier = posTier
				for _, row := range startersRows {
					// Skip non-flex positions like QB/K/DST
					if row.Pos == "QB" || row.Pos == "K" || row.Pos == "DST" {
						continue
					}
					// TEs only compare to TE starters (not FLEX slots)
					// RB/WR can compare to same-position starters OR FLEX/SUPERFLEX starters
					var canReplace bool
					if pos == "TE" {
						// TE only compares to TE position
						canReplace = (row.Pos == pos)
					} else {
						// RB/WR can compare to same position OR FLEX slots
						canReplace = (row.IsFlex || row.IsSuperflex) || (row.Pos == pos)
					}
					if !canReplace {
						continue
					}
					t, ok := row.Tier.(int)
					if ok && t > 0 && posTier < t {
						diff := t - posTier
						if diff > tierDiff {
							isUpgrade = true
							upgradeFor = stripHTML(row.Name)
							if row.IsFlex || row.IsSuperflex {
								upgradeType = "Starter (FLEX)"
							} else {
								upgradeType = "Starter"
							}
							tierDiff = diff
						}
					}
				}
				// Also check bench RB/WR/TE (but skip IR players)
				if !isUpgrade {
					for _, row := range benchRows {
						if strings.Contains(row.Name, "(IR)") {
							continue
						}
						if row.Pos != "RB" && row.Pos != "WR" && row.Pos != "TE" {
							continue
						}
						// Only compare if same position (e.g., RB FA vs RB bench, not RB vs TE)
						if row.Pos != pos {
							continue
						}
						t, ok := row.Tier.(int)
						if ok && t > 0 && posTier < t {
							diff := t - posTier
							if diff > tierDiff {
								isUpgrade = true
								upgradeFor = stripHTML(row.Name)
								upgradeType = "Bench"
								tierDiff = diff
							}
						}
					}
				}
			} else if posTier > 0 {
				// For QB/K/DST or RB/WR/TE without position tier: use position-specific comparison
				finalTier = posTier
				for _, row := range startersRows {
					if row.Pos == pos {
						t, ok := row.Tier.(int)
						if ok && t > 0 && posTier < t {
							diff := t - posTier
							if diff > tierDiff {
								isUpgrade = true
								upgradeFor = stripHTML(row.Name)
								upgradeType = "Starter"
								tierDiff = diff
							}
						}
					}
				}
				// Check bench (but skip IR players)
				if !isUpgrade {
					for _, row := range benchRows {
						if strings.Contains(row.Name, "(IR)") {
							continue
						}
						if row.Pos == pos {
							t, ok := row.Tier.(int)
							if ok && t > 0 && posTier < t {
								diff := t - posTier
								if diff > tierDiff {
									isUpgrade = true
									upgradeFor = stripHTML(row.Name)
									upgradeType = "Bench"
									tierDiff = diff
								}
							}
						}
					}
				}
			} else {
				continue // No valid tier to use
			}

			debugLog("[DEBUG] FA: %s | Pos: %s | PosTier: %d | FlexTier: %d | FinalTier: %d | IsUpgrade: %v | UpgradeFor: %s | UpgradeType: %s | TierDiff: %d", name, pos, posTier, flexTier, finalTier, isUpgrade, upgradeFor, upgradeType, tierDiff)
			faList = append(faList, faInfo{pid, percent, finalTier, pos, name, isUpgrade, upgradeFor, upgradeType, tierDiff})
		}
		debugLog("[DEBUG] Free agent candidates: %d total", len(faList))

		// Group by position first
		faByPos := map[string][]faInfo{}
		for _, fa := range faList {
			faByPos[fa.pos] = append(faByPos[fa.pos], fa)
		}

		// For each position, sort by: upgrades first (by tier diff), then by tier quality, then by roster %
		freeAgentsByPos := map[string][]PlayerRow{}
		faOrder := []string{"QB", "RB", "WR", "TE", "DST", "K"}
		for _, pos := range faOrder {
			posList := faByPos[pos]
			if len(posList) == 0 {
				continue
			}

			// Sort: upgrades first (by tier diff desc), then by tier asc (better tier), then by roster % desc
			sort.Slice(posList, func(i, j int) bool {
				// Upgrades before non-upgrades
				if posList[i].isUpgrade != posList[j].isUpgrade {
					return posList[i].isUpgrade
				}
				// Among upgrades, sort by tier difference (bigger improvement first)
				if posList[i].isUpgrade && posList[j].isUpgrade {
					if posList[i].tierDiff != posList[j].tierDiff {
						return posList[i].tierDiff > posList[j].tierDiff
					}
				}
				// Then by tier (better tier first)
				if posList[i].tier != posList[j].tier {
					return posList[i].tier < posList[j].tier
				}
				// Finally by roster percentage
				return posList[i].percent > posList[j].percent
			})

			// Take top 3, but prioritize upgrades - if we have upgrades, show up to 5
			// Exception: only show 2 kickers max
			limit := 3
			upgradeCount := 0
			for _, fa := range posList {
				if fa.isUpgrade {
					upgradeCount++
				}
			}
			if upgradeCount > 3 {
				limit = 5 // Show more if we have many upgrade options
			}
			if pos == "K" {
				limit = 2 // Only show 2 kickers max
			}
			if len(posList) > limit {
				posList = posList[:limit]
			}

			rows := []PlayerRow{}
			for _, fa := range posList {
				rows = append(rows, PlayerRow{
					Pos:         fa.pos,
					Name:        fa.name,
					Tier:        fa.tier,
					IsFreeAgent: true,
					IsUpgrade:   fa.isUpgrade,
					UpgradeFor:  fa.upgradeFor,
					UpgradeType: fa.upgradeType,
				})
			}
			if len(rows) > 0 {
				freeAgentsByPos[pos] = rows
			}
		}

		// Enrich free agents with dynasty values
		if isDynasty && dynastyValues != nil {
			for pos, faRows := range freeAgentsByPos {
				enrichRowsWithDynastyValues(faRows, dynastyValues, isSuperFlex)
				freeAgentsByPos[pos] = faRows // Update the map with enriched rows
			}
			debugLog("[DEBUG] Enriched free agents by position with dynasty values")
		}

		debugLog("[DEBUG] Final freeAgentsByPos: %v", freeAgentsByPos)

		// Create a combined prioritized list of top free agents across all positions
		allFAs := []PlayerRow{}
		for _, rows := range freeAgentsByPos {
			allFAs = append(allFAs, rows...)
		}
		debugLog("[DEBUG] Combined FA list has %d players", len(allFAs))

		// Sort by: upgrades first, then FLEX-eligible positions (RB/WR/TE), then by tier quality
		sort.Slice(allFAs, func(i, j int) bool {
			// Upgrades before non-upgrades
			if allFAs[i].IsUpgrade != allFAs[j].IsUpgrade {
				return allFAs[i].IsUpgrade
			}
			// Among same upgrade status, prioritize FLEX-eligible positions (RB/WR/TE)
			isFlex_i := allFAs[i].Pos == "RB" || allFAs[i].Pos == "WR" || allFAs[i].Pos == "TE"
			isFlex_j := allFAs[j].Pos == "RB" || allFAs[j].Pos == "WR" || allFAs[j].Pos == "TE"
			if isFlex_i != isFlex_j {
				return isFlex_i
			}
			// Then by tier (better tier first)
			ti, _ := allFAs[i].Tier.(int)
			tj, _ := allFAs[j].Tier.(int)
			return ti < tj
		})

		// Take top 12 most relevant FAs (more to ensure we show FLEX options)
		limit := 12
		if len(allFAs) < limit {
			limit = len(allFAs)
		}
		var topFreeAgents []PlayerRow
		if limit > 0 {
			topFreeAgents = allFAs[:limit]
			// Enrich top free agents with dynasty values
			if isDynasty && dynastyValues != nil {
				enrichRowsWithDynastyValues(topFreeAgents, dynastyValues, isSuperFlex)
				debugLog("[DEBUG] Enriched top free agents with dynasty values")
			}
		}
		debugLog("[DEBUG] Top free agents: %d selected from %d", len(topFreeAgents), len(allFAs))

		// Dynasty mode: generate value-based free agent recommendations and calculate total roster value
		var topFreeAgentsByValue []PlayerRow
		var totalRosterValue int
		if isDynasty && dynastyValues != nil {
			// Calculate total roster value (starters + bench)
			for _, row := range startersRows {
				totalRosterValue += row.DynastyValue
			}
			for _, row := range benchRows {
				totalRosterValue += row.DynastyValue
			}
			debugLog("[DEBUG] Total roster value: %d", totalRosterValue)

			// Find lowest dynasty values on current roster (to identify upgrade targets)
			lowestRosterValue := make(map[string]int) // pos -> lowest value on roster
			for _, row := range startersRows {
				if row.DynastyValue > 0 {
					if lowest, exists := lowestRosterValue[row.Pos]; !exists || row.DynastyValue < lowest {
						lowestRosterValue[row.Pos] = row.DynastyValue
					}
				}
			}
			for _, row := range benchRows {
				actualPos := row.Pos
				// Skip positions like FLEX, use actual player position for comparison
				if actualPos != "FLEX" && actualPos != "SUPER_FLEX" && row.DynastyValue > 0 {
					if lowest, exists := lowestRosterValue[actualPos]; !exists || row.DynastyValue < lowest {
						lowestRosterValue[actualPos] = row.DynastyValue
					}
				}
			}
			debugLog("[DEBUG] Lowest roster values by position: %v", lowestRosterValue)

			// Find free agents with higher dynasty values than current roster
			valueUpgrades := []PlayerRow{}
			for pid, p := range players {
				if _, ok := rostered[pid]; ok {
					continue
				}
				pm, ok := p.(map[string]interface{})
				if !ok || pm["active"] == false {
					continue
				}
				pos, _ := pm["position"].(string)
				if pos == "" || (pos != "QB" && pos != "RB" && pos != "WR" && pos != "TE") {
					continue // Only consider skill positions for dynasty
				}

				name := getPlayerName(pm)
				cleanName := normalizeName(name)

				// Get dynasty value for this FA
				if val, exists := dynastyValues[cleanName]; exists {
					faValue := val.Value1QB
					if isSuperFlex {
						faValue = val.Value2QB
					}

					// Check if this FA is an upgrade over anyone on the roster
					if lowestValue, exists := lowestRosterValue[pos]; exists && faValue > lowestValue {
						valueDiff := faValue - lowestValue
						debugLog("[DEBUG] Dynasty upgrade found: %s (pos: %s, value: %d, diff: +%d)", name, pos, faValue, valueDiff)

						// Find the player they would replace
						upgradeFor := ""
						upgradeType := ""
						for _, row := range startersRows {
							if (row.Pos == pos || row.IsFlex || row.IsSuperflex) && row.DynastyValue > 0 && row.DynastyValue < faValue {
								if row.DynastyValue == lowestValue {
									upgradeFor = stripHTML(row.Name)
									upgradeType = "Starter"
									break
								}
							}
						}
						if upgradeFor == "" {
							for _, row := range benchRows {
								if row.Pos == pos && row.DynastyValue > 0 && row.DynastyValue < faValue {
									if row.DynastyValue == lowestValue {
										upgradeFor = stripHTML(row.Name)
										upgradeType = "Bench"
										break
									}
								}
							}
						}

						valueUpgrades = append(valueUpgrades, PlayerRow{
							Pos:          pos,
							Name:         name,
							DynastyValue: faValue,
							IsFreeAgent:  true,
							IsUpgrade:    true,
							UpgradeFor:   upgradeFor,
							UpgradeType:  upgradeType,
						})
					}
				}
			}

			// Sort by dynasty value (highest first)
			sort.Slice(valueUpgrades, func(i, j int) bool {
				return valueUpgrades[i].DynastyValue > valueUpgrades[j].DynastyValue
			})

			// Take top 30 most valuable available players (increased for dynasty mode)
			limit := 30
			if len(valueUpgrades) < limit {
				limit = len(valueUpgrades)
			}
			if limit > 0 {
				topFreeAgentsByValue = valueUpgrades[:limit]
			}
			debugLog("[DEBUG] Dynasty mode: found %d value-based free agent upgrades", len(topFreeAgentsByValue))
		}

		// Calculate user's average age (for dynasty mode)
		var userAvgAge float64
		if isDynasty {
			totalAge := 0
			ageCount := 0
			for _, row := range startersRows {
				if row.Age > 0 {
					totalAge += row.Age
					ageCount++
				}
			}
			for _, row := range benchRows {
				if row.Age > 0 {
					totalAge += row.Age
					ageCount++
				}
			}
			if ageCount > 0 {
				userAvgAge = float64(totalAge) / float64(ageCount)
			}
			debugLog("[DEBUG] User's average roster age: %.2f (%d players)", userAvgAge, ageCount)
		}

		// Get league users for team names (for dynasty mode - used by both team ages and draft picks)
		var userNames map[string]string
		if isDynasty {
			leagueUsers, err := fetchJSONArray(fmt.Sprintf("https://api.sleeper.app/v1/league/%s/users", leagueID))
			if err != nil {
				debugLog("[DEBUG] Could not fetch league users: %v", err)
			}

			// Create a map of user_id -> display_name
			userNames = make(map[string]string)
			if leagueUsers != nil {
				for _, u := range leagueUsers {
					if uid, ok := u["user_id"].(string); ok {
						displayName := ""
						if dn, ok := u["display_name"].(string); ok && dn != "" {
							displayName = dn
						} else if un, ok := u["username"].(string); ok {
							displayName = un
						}
						userNames[uid] = displayName
					}
				}
			}
		}

		// Calculate average age for all teams in the league (for dynasty mode)
		var teamAges []TeamAgeData
		if isDynasty {

			// Calculate average age and roster value for each roster
			for _, r := range rosters {
				rosterID, _ := r["roster_id"].(float64)
				ownerID, _ := r["owner_id"].(string)
				ownerName := userNames[ownerID]
				if ownerName == "" {
					ownerName = "Unknown"
				}

				// Get team name from metadata (if available)
				teamName := ownerName
				if metadata, ok := r["metadata"].(map[string]interface{}); ok {
					if tn, ok := metadata["team_name"].(string); ok && tn != "" {
						teamName = tn
					}
				}

				// Calculate average age and total roster value
				rosterPlayers := toStringSlice(r["players"])
				totalAge := 0
				ageCount := 0
				rosterValue := 0

				for _, pid := range rosterPlayers {
					if p, ok := players[pid].(map[string]interface{}); ok {
						// Age
						if ageVal, ok := p["age"].(float64); ok && ageVal > 0 {
							totalAge += int(ageVal)
							ageCount++
						}

						// Dynasty value
						if dynastyValues != nil {
							name := getPlayerName(p)
							cleanName := normalizeName(name)
							if val, exists := dynastyValues[cleanName]; exists {
								if isSuperFlex {
									rosterValue += val.Value2QB
								} else {
									rosterValue += val.Value1QB
								}
							}
						}
					}
				}

				avgAge := 0.0
				if ageCount > 0 {
					avgAge = float64(totalAge) / float64(ageCount)
				}

				// Get standings rank (wins)
				rank := 0
				if settings, ok := r["settings"].(map[string]interface{}); ok {
					if wins, ok := settings["wins"].(float64); ok {
						rank = int(wins)
					}
				}

				teamAges = append(teamAges, TeamAgeData{
					TeamName:    teamName,
					OwnerName:   ownerName,
					AvgAge:      avgAge,
					Rank:        rank,
					RosterID:    int(rosterID),
					IsUserTeam:  (ownerID == userID),
					RosterValue: rosterValue,
				})
			}

			// Sort teams by age (oldest to youngest)
			sort.Slice(teamAges, func(i, j int) bool {
				return teamAges[i].AvgAge > teamAges[j].AvgAge
			})

			debugLog("[DEBUG] Calculated ages for %d teams", len(teamAges))
		}

		// Fetch draft picks for dynasty leagues
		var draftPicks []DraftPick
		var projectedDraftPicks []ProjectedDraftPick
		if isDynasty {
			// Fetch traded picks from Sleeper API
			tradedPicks, err := fetchJSONArray(fmt.Sprintf("https://api.sleeper.app/v1/league/%s/traded_picks", leagueID))
			if err != nil {
				debugLog("[DEBUG] Could not fetch traded picks: %v", err)
			}

			// Debug: Log raw traded picks data to understand API response
			if tradedPicks != nil {
				debugLog("[DEBUG] ===== TRADED PICKS RAW DATA =====")
				for i, trade := range tradedPicks {
					debugLog("[DEBUG] Trade %d: %+v", i, trade)
					if season, ok := trade["season"].(string); ok {
						debugLog("[DEBUG]   season: %s", season)
					}
					if round, ok := trade["round"].(float64); ok {
						debugLog("[DEBUG]   round: %.0f", round)
					}
					if rosterID, ok := trade["roster_id"].(float64); ok {
						debugLog("[DEBUG]   roster_id (original owner): %.0f", rosterID)
					}
					if ownerID, ok := trade["owner_id"].(float64); ok {
						debugLog("[DEBUG]   owner_id (current owner): %.0f", ownerID)
					}
					if prevOwner, ok := trade["previous_owner_id"].(float64); ok {
						debugLog("[DEBUG]   previous_owner_id: %.0f", prevOwner)
					}
				}
				debugLog("[DEBUG] ===================================")
			}

			// Get league settings to determine number of rounds
			numRounds := 3 // Default to 3 rounds
			if settings, ok := league["settings"].(map[string]interface{}); ok {
				if rounds, ok := settings["draft_rounds"].(float64); ok && rounds > 0 {
					numRounds = int(rounds)
				}
			}
			debugLog("[DEBUG] League has %d draft rounds", numRounds)

			// Create map of roster_id -> user info for owner names
			rosterOwners := make(map[int]string)
			for _, r := range rosters {
				rosterID, _ := r["roster_id"].(float64)
				ownerID, _ := r["owner_id"].(string)

				// Get owner name from league users (already fetched for team ages)
				ownerName := "Unknown"
				if userNames != nil {
					if name, exists := userNames[ownerID]; exists {
						ownerName = name
					}
				}
				rosterOwners[int(rosterID)] = ownerName
			}
			debugLog("[DEBUG] Roster owners map: %+v", rosterOwners)

			// Calculate which picks each team has
			currentYear := time.Now().Year()
			userRosterIDFloat, _ := userRoster["roster_id"].(float64)
			userRosterID := int(userRosterIDFloat)
			debugLog("[DEBUG] User roster ID: %d", userRosterID)

			draftPicks = buildDraftPicks(tradedPicks, rosters, rosterOwners, numRounds, userRosterID, currentYear, debugLog)

			// Calculate projected draft order for 2026
			if len(draftPicks) > 0 && len(teamAges) > 0 {
				currentYear := time.Now().Year()
				// Project for next year's draft (2026 in February 2026)
				targetYear := currentYear
				projectedDraftPicks = calculateProjectedDraftPicks(draftPicks, teamAges, targetYear)
				debugLog("[DEBUG] Calculated %d projected draft picks for %d", len(projectedDraftPicks), targetYear)
			}
		}

		// Aggregate player news for dynasty leagues
		var playerNewsFeed []PlayerNews
		var breakoutCandidates []PlayerRow
		var agingPlayers []PlayerRow
		var recentTransactions []Transaction
		if isDynasty {
			playerNewsFeed = aggregatePlayerNews(allPlayers, players, starters, dynastyValues, isSuperFlex)
			debugLog("[DEBUG] Aggregated %d player news items", len(playerNewsFeed))

			// Find breakout candidates from bench (only if we have dynasty values)
			if dynastyValues != nil {
				breakoutCandidates = findBreakoutCandidates(benchRows)
				debugLog("[DEBUG] Found %d breakout candidates", len(breakoutCandidates))

				// Find aging players from starters and bench
				agingPlayers = findAgingPlayers(startersRows, benchRows)
				debugLog("[DEBUG] Found %d aging players", len(agingPlayers))
			}

			// Fetch recent league transactions
			recentTransactions = fetchRecentTransactions(leagueID, week, players, rosters, userNames, dynastyValues, isSuperFlex)
			debugLog("[DEBUG] Found %d recent transactions", len(recentTransactions))

			// Analyze trade retrospectives (Feature #5)
			if dynastyValues != nil && len(recentTransactions) > 0 {
				recentTransactions = analyzeTradeRetrospective(recentTransactions, dynastyValues, isSuperFlex)
			}
		}

		// Get top rookies for dynasty leagues
		var topRookies []RookieProspect
		if isDynasty {
			topRookies = getTopRookies()
			// Filter out defensive players (they have 0 fantasy value)
			fantasyRookies := []RookieProspect{}
			for _, r := range topRookies {
				if r.Value > 0 {
					fantasyRookies = append(fantasyRookies, r)
				}
			}
			topRookies = fantasyRookies
			debugLog("[DEBUG] Loaded %d top rookie prospects", len(topRookies))
		}

		// Calculate league trends for dynasty leagues
		var leagueTrends LeagueTrends
		if isDynasty {
			leagueTrends = calculateLeagueTrends(recentTransactions, freeAgentsByPos, players)
			debugLog("[DEBUG] Calculated league trends: %d active teams, %d hot waiver players", len(leagueTrends.MostActiveTeams), len(leagueTrends.HotWaiverPlayers))
		}

		// Calculate trade targets for dynasty leagues
		var tradeTargets []TradeTarget
		var positionalBreakdown PositionalKTC
		if isDynasty && dynastyValues != nil {
			// Build map of all rosters with enriched player rows
			allRosters := make(map[int][]PlayerRow)
			teamNamesMap := make(map[int]string)
			userRosterID, _ := userRoster["roster_id"].(float64)

			for _, r := range rosters {
				rosterID, _ := r["roster_id"].(float64)
				ownerID, _ := r["owner_id"].(string)

				// Get team name
				teamName := ""
				if userNames != nil {
					if name, exists := userNames[ownerID]; exists {
						teamName = name
					}
				}
				if teamName == "" {
					teamName = "Unknown"
				}
				// Try to get custom team name from metadata
				if metadata, ok := r["metadata"].(map[string]interface{}); ok {
					if tn, ok := metadata["team_name"].(string); ok && tn != "" {
						teamName = tn
					}
				}
				teamNamesMap[int(rosterID)] = teamName

				// Build full roster for this team
				rosterPlayers := toStringSlice(r["players"])
				rosterRows, _, _ := buildRowsWithPositions(rosterPlayers, players, borisTiers, false, nil, nil, nil)

				// Enrich with dynasty values
				enrichRowsWithDynastyValues(rosterRows, dynastyValues, isSuperFlex)

				allRosters[int(rosterID)] = rosterRows
			}

			// Combine user's starters and bench for trade analysis
			userFullRoster := append([]PlayerRow{}, startersRows...)
			userFullRoster = append(userFullRoster, benchRows...)

			// Calculate user's positional breakdown
			positionalBreakdown = calculatePositionalKTC(userFullRoster)

			// Find trade targets
			tradeTargets = findTradeTargets(userFullRoster, allRosters, teamNamesMap, int(userRosterID))
			debugLog("[DEBUG] Found %d trade targets", len(tradeTargets))

			// Generate trade proposals for each target (Feature #9)
			for i := range tradeTargets {
				target := &tradeTargets[i]
				targetRosterID := 0

				// Find the roster ID for this target team
				for rosterID, teamName := range teamNamesMap {
					if teamName == target.TeamName {
						targetRosterID = rosterID
						break
					}
				}

				if targetRosterID != 0 {
					if targetRoster, ok := allRosters[targetRosterID]; ok {
						proposal := generateTradeProposal(
							userFullRoster,
							targetRoster,
							target.TeamName,
							target.YourSurplus,
							target.TheirSurplus,
							dynastyValues,
							isSuperFlex,
							premiumEnabled,
						)
						target.Proposal = &proposal
						debugLog("[DEBUG] Generated trade proposal for %s", target.TeamName)
					}
				}
			}
		}

		avgTier := avg(starterTiers)
		avgOppTier := avg(oppTiers)
		winProb, emoji := winProbability(avgTier, avgOppTier)

		// Build roster slots summary
		rosterSlots := ""
		if len(leagueRosterPositions) > 0 {
			posCounts := make(map[string]int)
			posOrder := []string{}
			for _, pos := range leagueRosterPositions {
				if posCounts[pos] == 0 {
					posOrder = append(posOrder, pos)
				}
				posCounts[pos]++
			}
			parts := []string{}
			for _, pos := range posOrder {
				displayName := pos
				switch pos {
				case "SUPER_FLEX":
					displayName = "SF"
				case "REC_FLEX":
					displayName = "RF"
				case "IDP_FLEX":
					displayName = "IDP"
				case "BN":
					displayName = "BN"
				}
				parts = append(parts, fmt.Sprintf("%d %s", posCounts[pos], displayName))
			}
			rosterSlots = strings.Join(parts, ", ")
		}

		leagueData := LeagueData{
			LeagueName:           leagueName,
			Scoring:              scoring,
			IsDynasty:            isDynasty,
			HasMatchups:          hasMatchups,
			DynastyValueDate:     dynastyValueDate,
			LeagueSize:           len(rosters),
			RosterSlots:          rosterSlots,
			Starters:             startersRows,
			Unranked:             unrankedRows,
			AvgTier:              avgTier,
			AvgOppTier:           avgOppTier,
			WinProb:              winProb + " " + emoji,
			Bench:                benchRows,
			BenchUnranked:        benchUnrankedRows,
			FreeAgentsByPos:      freeAgentsByPos,
			TopFreeAgents:        topFreeAgents,
			TopFreeAgentsByValue: topFreeAgentsByValue,
			TotalRosterValue:     totalRosterValue,
			UserAvgAge:           userAvgAge,
			TeamAges:             teamAges,
			PowerRankings:        calculatePowerRankings(teamAges),
			DraftPicks:           draftPicks,
			ProjectedDraftPicks:  projectedDraftPicks,
			TradeTargets:         tradeTargets,
			PositionalBreakdown:  positionalBreakdown,
			PlayerNewsFeed:       playerNewsFeed,
			BreakoutCandidates:   breakoutCandidates,
			AgingPlayers:         agingPlayers,
			RecentTransactions:   recentTransactions,
			TopRookies:           topRookies,
			LeagueTrends:         leagueTrends,
		}

		// Generate weekly actions (Feature #2)
		leagueData.WeeklyActions = buildWeeklyActions(leagueData)

		// Compress player news (Feature #4)
		if len(playerNewsFeed) > 0 {
			userPlayerNames := extractPlayerNames(startersRows, benchRows)
			leagueData.CompressedNews = compressPlayerNews(playerNewsFeed, userPlayerNames, isDynasty)
		}

		// Build context cards (Feature #6)
		leagueData.ContextCards = buildContextCards(leagueData, totalRosterValue, userAvgAge)

		// Track value changes (Feature #7)
		if isDynasty && dynastyValues != nil && len(dynastyValues) > 0 {
			userPlayerNames := extractPlayerNames(startersRows, benchRows)
			valueChanges, _ := getValueChanges(dynastyValues, userPlayerNames, isSuperFlex)
			leagueData.ValueChanges = valueChanges
		}

		// Generate waiver recommendations (Feature #10)
		if len(freeAgentsByPos) > 0 {
			waiverRecs := generateWaiverRecommendations(leagueData, freeAgentsByPos, 10, isPremium)
			leagueData.WaiverRecommendations = waiverRecs
			debugLog("[DEBUG] Generated %d waiver recommendations", len(waiverRecs))
		}

		// Generate season plan (Feature #11)
		if isDynasty {
			seasonPlan := generateSeasonPlan(leagueData, isPremium)
			leagueData.SeasonPlan = seasonPlan
			debugLog("[DEBUG] Generated season plan: %s strategy", seasonPlan.Strategy)
		}

		leagueResults = append(leagueResults, leagueData)
	}

	if len(leagueResults) == 0 {
		debugLog("[DEBUG] No valid leagues found with matchups for user %s", username)
		renderError(w, "No active matchups found for your leagues. During the offseason, only dynasty leagues are available. If you have dynasty leagues, they should appear automatically — if not, please report this on GitHub.")
		return
	}

	username = r.FormValue("username")

	// Set cookie to remember username for 30 days
	cookie := &http.Cookie{
		Name:     "sleeper_username",
		Value:    username,
		Path:     "/",
		MaxAge:   30 * 24 * 60 * 60, // 30 days
		HttpOnly: false,             // Allow JavaScript to read for UI logic
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)

	premiumOverview := ""

	if isPremium && premiumEnabled && llmMode != "" {
		if llmMode == "overview" || llmMode == "all" || llmMode == "1" {
			ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
			overview, err := generateOverview(ctx, leagueResults)
			cancel()
			if err != nil {
				debugLog("[DEBUG] OpenRouter overview error: %v", err)
			} else {
				premiumOverview = overview
			}
		}

		if llmMode == "team" || llmMode == "all" || llmMode == "1" {
			leagueResults = applyTeamTalks(r.Context(), leagueResults)
		}
	}

	if err = templates.ExecuteTemplate(w, "tiers.html", TiersPage{
		Leagues:         leagueResults,
		Username:        username,
		IsPremium:       isPremium,
		PremiumEnabled:  premiumEnabled,
		PremiumOverview: premiumOverview,
	}); err != nil {
		log.Printf("[ERROR] Template execution error: %v", err)
	}
}

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	debugLog("[DEBUG] /dashboard handler called")

	// Get username from query param or cookie
	username := r.URL.Query().Get("user")
	if username == "" {
		if cookie, err := r.Cookie("sleeper_username"); err == nil {
			username = cookie.Value
		}
	}

	if username == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Build dashboard data
	dashboardPage, err := buildDashboardPage(username)
	if err != nil {
		log.Printf("[ERROR] Failed to build dashboard: %v", err)
		renderError(w, fmt.Sprintf("Failed to load dashboard: %v", err))
		return
	}

	// Render dashboard template
	if err := templates.ExecuteTemplate(w, "dashboard.html", dashboardPage); err != nil {
		log.Printf("[ERROR] Template execution error: %v", err)
		http.Error(w, fmt.Sprintf("Error rendering dashboard: %v", err), http.StatusInternalServerError)
	}
}

func buildDashboardPage(username string) (*DashboardPage, error) {
	// 1. Get user ID
	user, err := fetchJSON(fmt.Sprintf("https://api.sleeper.app/v1/user/%s", username))
	if err != nil || user["user_id"] == nil {
		return nil, fmt.Errorf("user not found")
	}
	userID := user["user_id"].(string)

	// 2. Get leagues (current + previous year)
	year := time.Now().Year()
	leagues, err := fetchJSONArray(fmt.Sprintf("https://api.sleeper.app/v1/user/%s/leagues/nfl/%d", userID, year))
	if err != nil {
		debugLog("[DEBUG] Error fetching leagues for year %d: %v", year, err)
	}

	previousYear := year - 1
	previousYearLeagues, err := fetchJSONArray(fmt.Sprintf("https://api.sleeper.app/v1/user/%s/leagues/nfl/%d", userID, previousYear))
	if err == nil {
		leagues = append(leagues, previousYearLeagues...)
	}

	if len(leagues) == 0 {
		return nil, fmt.Errorf("no leagues found")
	}

	// 3. Get players data (for dynasty values and age)
	players, err := fetchPlayers()
	if err != nil {
		return nil, fmt.Errorf("could not fetch player data: %v", err)
	}

	// 4. Fetch dynasty values if needed
	var dynastyValues map[string]DynastyValue
	hasDynasty := false
	for _, league := range leagues {
		if isDynastyLeague(league) {
			hasDynasty = true
			break
		}
	}
	if hasDynasty {
		dynastyValues, _ = fetchDynastyValues()
	}

	// 5. Build summary for each league
	var summaries []LeagueSummary
	dynastyCount := 0
	redraftCount := 0

	for _, league := range leagues {
		leagueID := league["league_id"].(string)
		leagueName := league["name"].(string)
		isDynasty := isDynastyLeague(league)

		// Get season year
		season := ""
		if seasonStr, ok := league["season"].(string); ok {
			season = seasonStr
			debugLog("[DEBUG] League %s has season: %s", leagueName, season)
		} else {
			debugLog("[DEBUG] League %s has no season field", leagueName)
		}

		if isDynasty {
			dynastyCount++
		} else {
			redraftCount++
		}

		// Get league size
		rosters, err := fetchJSONArray(fmt.Sprintf("https://api.sleeper.app/v1/league/%s/rosters", leagueID))
		if err != nil {
			debugLog("[DEBUG] Error fetching rosters for league %s: %v", leagueName, err)
			continue
		}
		leagueSize := len(rosters)

		// Determine scoring type
		scoring := "PPR"
		if scoringSettings, ok := league["scoring_settings"].(map[string]interface{}); ok {
			if rec, ok := scoringSettings["rec"].(float64); ok {
				if rec == 0.5 {
					scoring = "Half PPR"
				} else if rec == 0.0 {
					scoring = "Standard"
				}
			}
		}

		// Check if superflex
		isSuperFlex := false
		if rosterPositions, ok := league["roster_positions"].([]interface{}); ok {
			for _, pos := range rosterPositions {
				if posStr, ok := pos.(string); ok && posStr == "SUPER_FLEX" {
					isSuperFlex = true
					break
				}
			}
		}

		summary := LeagueSummary{
			LeagueID:    leagueID,
			LeagueName:  leagueName,
			Season:      season,
			Scoring:     scoring,
			IsDynasty:   isDynasty,
			IsSuperFlex: isSuperFlex,
			LeagueSize:  leagueSize,
			LastUpdated: time.Now(),
		}

		// Find user's roster
		var userRoster map[string]interface{}
		for _, r := range rosters {
			if r["owner_id"] == userID {
				userRoster = r
				break
			}
		}

		if userRoster == nil {
			debugLog("[DEBUG] User roster not found in league %s", leagueName)
			summaries = append(summaries, summary)
			continue
		}

		// Get user's record (wins/losses)
		if settings, ok := userRoster["settings"].(map[string]interface{}); ok {
			wins := 0
			losses := 0
			if w, ok := settings["wins"].(float64); ok {
				wins = int(w)
			}
			if l, ok := settings["losses"].(float64); ok {
				losses = int(l)
			}
			if wins > 0 || losses > 0 {
				summary.Record = fmt.Sprintf("%d-%d", wins, losses)

				// Simple playoff status (need >50% win rate and >6 wins)
				totalGames := wins + losses
				if totalGames > 0 {
					winPct := float64(wins) / float64(totalGames)
					if winPct >= 0.6 && wins >= 6 {
						summary.PlayoffStatus = "Clinched ✓"
					} else if winPct >= 0.45 {
						summary.PlayoffStatus = "In Hunt"
					} else {
						summary.PlayoffStatus = "Eliminated"
					}
				}
			}
		}

		// Dynasty-specific metrics
		if isDynasty && dynastyValues != nil && len(dynastyValues) > 0 {
			// Calculate total roster value and rank
			playerIDs, _ := userRoster["players"].([]interface{})
			totalValue := 0
			totalAge := 0.0
			playerCount := 0

			for _, pid := range playerIDs {
				playerID := pid.(string)
				if player, ok := players[playerID].(map[string]interface{}); ok {
					playerName, _ := player["full_name"].(string)
					normName := normalizeName(playerName)

					if dv, exists := dynastyValues[normName]; exists {
						value := dv.Value1QB
						if isSuperFlex {
							value = dv.Value2QB
						}
						totalValue += value
					}

					// Calculate age
					if ageFloat, ok := player["age"].(float64); ok {
						totalAge += ageFloat
						playerCount++
					}
				}
			}

			summary.TotalRosterValue = totalValue
			if playerCount > 0 {
				summary.AvgAge = totalAge / float64(playerCount)
			}

			// Calculate value rank
			var allRosterValues []int
			for _, roster := range rosters {
				pids, _ := roster["players"].([]interface{})
				rosterValue := 0
				for _, pid := range pids {
					playerID := pid.(string)
					if player, ok := players[playerID].(map[string]interface{}); ok {
						playerName, _ := player["full_name"].(string)
						normName := normalizeName(playerName)
						if dv, exists := dynastyValues[normName]; exists {
							value := dv.Value1QB
							if isSuperFlex {
								value = dv.Value2QB
							}
							rosterValue += value
						}
					}
				}
				allRosterValues = append(allRosterValues, rosterValue)
			}

			// Sort and find rank
			sort.Sort(sort.Reverse(sort.IntSlice(allRosterValues)))
			for i, val := range allRosterValues {
				if val == totalValue {
					summary.ValueRank = i + 1
					break
				}
			}

			// Calculate value trend
			summary.ValueTrend = getValueTrend(username, leagueID, totalValue)

			// Get draft picks summary
			summary.DraftPicksSummary = getDraftPicksSummary(leagueID, userRoster)
		}

		summaries = append(summaries, summary)
	}

	// Sort: dynasty first, then by name
	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].IsDynasty != summaries[j].IsDynasty {
			return summaries[i].IsDynasty
		}
		return summaries[i].LeagueName < summaries[j].LeagueName
	})

	return &DashboardPage{
		Username:        username,
		LeagueSummaries: summaries,
		TotalLeagues:    len(summaries),
		DynastyCount:    dynastyCount,
		RedraftCount:    redraftCount,
	}, nil
}

func getValueTrend(username, leagueID string, currentValue int) string {
	cacheKey := fmt.Sprintf("%s:%s", username, leagueID)

	rosterValueTrendCache.RLock()
	cached, exists := rosterValueTrendCache.data[cacheKey]
	rosterValueTrendCache.RUnlock()

	// If no cached value or too old, cache current and return stable
	if !exists || time.Since(cached.Timestamp) > rosterValueTrendCache.ttl {
		rosterValueTrendCache.Lock()
		rosterValueTrendCache.data[cacheKey] = CachedRosterValue{
			RosterValue: currentValue,
			Timestamp:   time.Now(),
		}
		rosterValueTrendCache.Unlock()
		return "→ stable"
	}

	// Calculate trend
	delta := currentValue - cached.RosterValue
	if cached.RosterValue == 0 {
		return "→ stable"
	}

	deltaPct := float64(delta) / float64(cached.RosterValue) * 100

	// Update cache with current value
	rosterValueTrendCache.Lock()
	rosterValueTrendCache.data[cacheKey] = CachedRosterValue{
		RosterValue: currentValue,
		Timestamp:   time.Now(),
	}
	rosterValueTrendCache.Unlock()

	if deltaPct >= 1.0 {
		return fmt.Sprintf("↗ +%.0f%%", deltaPct)
	} else if deltaPct <= -1.0 {
		return fmt.Sprintf("↘ %.0f%%", deltaPct)
	}
	return "→ stable"
}

func getDraftPicksSummary(leagueID string, userRoster map[string]interface{}) string {
	// Fetch traded picks from API
	tradedPicks, err := fetchJSONArray(fmt.Sprintf("https://api.sleeper.app/v1/league/%s/traded_picks", leagueID))
	if err != nil {
		debugLog("[DEBUG] Error fetching traded picks: %v", err)
		return ""
	}

	rosterID := int(userRoster["roster_id"].(float64))
	year := time.Now().Year()

	// Count user's picks for next 2 years
	type PickCount struct {
		Year  int
		Round int
	}
	userPickMap := make(map[PickCount]bool)

	// Start with default picks (user owns their own picks by default)
	for y := year; y < year+2; y++ {
		userPickMap[PickCount{Year: y, Round: 1}] = true
		userPickMap[PickCount{Year: y, Round: 2}] = true
	}

	// Apply traded picks
	for _, trade := range tradedPicks {
		season, seasonOk := trade["season"].(string)
		round, roundOk := trade["round"].(float64)
		ownerID, ownerOk := trade["owner_id"].(float64)
		originalRosterID, origOk := trade["roster_id"].(float64)

		if !seasonOk || !roundOk || !ownerOk || !origOk {
			continue
		}

		tradeYear := 0
		fmt.Sscanf(season, "%d", &tradeYear)
		if tradeYear < year || tradeYear >= year+2 {
			continue
		}

		pc := PickCount{Year: tradeYear, Round: int(round)}

		// If user traded away their pick
		if int(originalRosterID) == rosterID && int(ownerID) != rosterID {
			delete(userPickMap, pc)
		}

		// If user acquired someone else's pick
		if int(originalRosterID) != rosterID && int(ownerID) == rosterID {
			userPickMap[pc] = true
		}
	}

	// Build summary string
	var picks []string
	for pc := range userPickMap {
		roundStr := "th"
		if pc.Round == 1 {
			roundStr = "st"
		} else if pc.Round == 2 {
			roundStr = "nd"
		} else if pc.Round == 3 {
			roundStr = "rd"
		}
		picks = append(picks, fmt.Sprintf("%d %d%s", pc.Year, pc.Round, roundStr))
	}

	if len(picks) == 0 {
		return "None"
	}

	sort.Strings(picks)

	// Limit to first 3
	if len(picks) > 3 {
		return strings.Join(picks[:3], ", ") + "..."
	}

	return strings.Join(picks, ", ")
}

func renderError(w http.ResponseWriter, msg string) {
	username := ""
	if u := w.Header().Get("X-Username"); u != "" {
		username = u
	}
	templates.ExecuteTemplate(w, "tiers.html", TiersPage{Error: msg, Username: username})
}
