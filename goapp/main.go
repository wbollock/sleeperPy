package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

var funcMap = template.FuncMap{
	"safe": func(s string) template.HTML { return template.HTML(s) },
	"parseWinProb": func(s string) int {
		// s is like "62% You 🏆" or "38% Opponent 💀"
		parts := strings.Fields(s)
		if len(parts) > 0 && strings.HasSuffix(parts[0], "%") {
			n, err := strconv.Atoi(strings.TrimSuffix(parts[0], "%"))
			if err == nil {
				return n
			}
		}
		return 50
	},
	"winProbColor": func(s string) string {
		// Green for >60, yellow for 40-60, red for <40
		p := 50
		parts := strings.Fields(s)
		if len(parts) > 0 && strings.HasSuffix(parts[0], "%") {
			n, err := strconv.Atoi(strings.TrimSuffix(parts[0], "%"))
			if err == nil {
				p = n
			}
		}
		if p > 60 {
			return "#3ae87a" // green
		} else if p < 40 {
			return "#e83a3a" // red
		}
		return "#e8c63a" // yellow
	},
	"parseWinEmoji": func(s string) string {
		// s is like "62% You 🏆" or "38% Opponent 💀"
		parts := strings.Fields(s)
		if len(parts) > 2 {
			return parts[2]
		}
		return "🤝"
	},
}

var templates = template.Must(template.New("").Funcs(funcMap).ParseGlob("templates/*.html"))

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/lookup", lookupHandler)

	log.Printf("Server running on http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	templates.ExecuteTemplate(w, "index.html", nil)
}

// --- Data structures for rendering ---
type PlayerRow struct {
	Pos                  string
	Name                 string
	Tier                 interface{}
	IsTierWorseThanBench bool
	ShouldSwapIn         bool
	IsFreeAgent          bool
	IsUpgrade            bool
	UpgradeFor           string // Name of player this FA is better than
	UpgradeType          string // "Starter" or "Bench" or ""
}

type LeagueData struct {
	LeagueName      string
	Scoring         string
	Starters        []PlayerRow
	Unranked        []PlayerRow
	AvgTier         string
	AvgOppTier      string
	WinProb         string
	Bench           []PlayerRow
	BenchUnranked   []PlayerRow
	FreeAgentsByPos map[string][]PlayerRow
}

type TiersPage struct {
	Error    string
	Leagues  []LeagueData
	Username string
}

// Team mapping for DST/DEF
var TEAM_MAP = map[string]string{
	"ARI": "Arizona Cardinals", "ATL": "Atlanta Falcons", "BAL": "Baltimore Ravens", "BUF": "Buffalo Bills",
	"CAR": "Carolina Panthers", "CHI": "Chicago Bears", "CIN": "Cincinnati Bengals", "CLE": "Cleveland Browns",
	"DAL": "Dallas Cowboys", "DEN": "Denver Broncos", "DET": "Detroit Lions", "GB": "Green Bay Packers",
	"HOU": "Houston Texans", "IND": "Indianapolis Colts", "JAX": "Jacksonville Jaguars", "KC": "Kansas City Chiefs",
	"LV": "Las Vegas Raiders", "LAC": "Los Angeles Chargers", "LAR": "Los Angeles Rams", "MIA": "Miami Dolphins",
	"MIN": "Minnesota Vikings", "NE": "New England Patriots", "NO": "New Orleans Saints", "NYG": "New York Giants",
	"NYJ": "New York Jets", "PHI": "Philadelphia Eagles", "PIT": "Pittsburgh Steelers", "SEA": "Seattle Seahawks",
	"SF": "San Francisco 49ers", "TB": "Tampa Bay Buccaneers", "TEN": "Tennessee Titans", "WAS": "Washington Commanders",
}

// --- Handler for form submission ---
func lookupHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("[DEBUG] /lookup handler called")
	r.ParseForm()
	username := r.FormValue("username")
	log.Printf("[DEBUG] Username submitted: %s", username)
	if username == "" {
		log.Println("[DEBUG] No username provided")
		renderError(w, "No username provided")
		return
	}

	// 1. Get user ID
	user, err := fetchJSON(fmt.Sprintf("https://api.sleeper.app/v1/user/%s", username))
	if err != nil || user["user_id"] == nil {
		log.Printf("[DEBUG] User not found or error: %v", err)
		renderError(w, "User not found")
		return
	}
	userID := user["user_id"].(string)

	// 2. Get leagues
	year := time.Now().Year()
	leagues, err := fetchJSONArray(fmt.Sprintf("https://api.sleeper.app/v1/user/%s/leagues/nfl/%d", userID, year))
	if err != nil || len(leagues) == 0 {
		log.Printf("[DEBUG] No leagues found for user %s: %v", userID, err)
		renderError(w, "No leagues found for this user")
		return
	}

	// 3. Get current NFL week from Sleeper API
	state, err := fetchJSON("https://api.sleeper.app/v1/state/nfl")
	if err != nil {
		log.Printf("[DEBUG] Could not get current NFL week: %v", err)
		renderError(w, "Could not get current NFL week")
		return
	}
	week := int(state["week"].(float64))

	// 4. Get players data
	players, err := fetchJSON("https://api.sleeper.app/v1/players/nfl")
	if err != nil {
		log.Printf("[DEBUG] Could not fetch players data: %v", err)
		renderError(w, "Could not fetch players data")
		return
	}

	// 5. Process each league
	var leagueResults []LeagueData
	log.Printf("[DEBUG] Processing %d leagues", len(leagues))
	for _, league := range leagues {
		leagueID := league["league_id"].(string)
		leagueName := league["name"].(string)

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

		// Get rosters and matchups
		rosters, err := fetchJSONArray(fmt.Sprintf("https://api.sleeper.app/v1/league/%s/rosters", leagueID))
		if err != nil {
			log.Printf("[DEBUG] Error fetching rosters for league %s: %v", leagueName, err)
			continue
		}

		matchups, err := fetchJSONArray(fmt.Sprintf("https://api.sleeper.app/v1/league/%s/matchups/%d", leagueID, week))
		if err != nil || len(matchups) == 0 {
			log.Printf("[DEBUG] No matchups found for league %s week %d: %v", leagueName, week, err)
			continue
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
			log.Printf("[DEBUG] No user roster found for league %s", leagueName)
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

		// Find opponent
		var myMatchup, oppMatchup map[string]interface{}
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

		oppStarters := []string{}
		if oppMatchup != nil {
			oppStarters = toStringSlice(oppMatchup["starters"])
		}

		// Fetch Boris Chen tiers
		borisTiers := fetchBorisTiers(scoring)

		// Build rows for roster
		benchRows, _, _ := buildRows(bench, players, borisTiers, false, userRoster, irPlayers, nil)
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
		startersRows, unrankedRows, starterTiers := buildRows(starters, players, borisTiers, true, userRoster, irPlayers, bestBenchTier)
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
		benchRows, _, _ = buildRows(bench, players, borisTiers, false, userRoster, irPlayers, worstStarterTier)
		_, _, oppTiers := buildRows(oppStarters, players, borisTiers, true, nil, nil, nil)

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
		// Find free agents: not rostered, not on user's team, valid tier, and sort by roster_percent
		type faInfo struct {
			pid     string
			percent float64
			tier    int
			pos     string
			name    string
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
			tier := findTier(borisTiers[lookupPos], name)
			if tier <= 0 {
				continue // Only show ranked players
			}
			percent := 0.0
			if v, ok := pm["roster_percent"].(float64); ok {
				percent = v
			} else if v, ok := pm["roster_percent"].(string); ok {
				percent, _ = strconv.ParseFloat(v, 64)
			}
			faList = append(faList, faInfo{pid, percent, tier, pos, name})
		}
		// Sort by roster_percent descending
		sort.Slice(faList, func(i, j int) bool {
			return faList[i].percent > faList[j].percent
		})
		maxFA := 20
		if len(faList) > maxFA {
			faList = faList[:maxFA]
		}
		// Group and limit free agents by position (top 5 per position)
		faByPos := map[string][]faInfo{}
		for _, fa := range faList {
			faByPos[fa.pos] = append(faByPos[fa.pos], fa)
		}
		freeAgentsByPos := map[string][]PlayerRow{}
		// Show K last, after DST
		faOrder := []string{"QB", "RB", "WR", "TE", "DST", "K"}
		for _, pos := range faOrder {
			posList := faByPos[pos]
			if len(posList) > 3 {
				posList = posList[:3]
			}
			rows := []PlayerRow{}
			for _, fa := range posList {
				isUpgrade := false
				upgradeFor := ""
				upgradeType := ""
				// Check starters first
				for _, row := range startersRows {
					if row.Pos == fa.pos {
						t, ok := row.Tier.(int)
						if ok && t > 0 && fa.tier > 0 && fa.tier < t {
							isUpgrade = true
							upgradeFor = row.Name
							upgradeType = "Starter"
							break
						}
					}
				}
				// If not upgrade for starter, check bench
				if !isUpgrade {
					for _, row := range benchRows {
						if row.Pos == fa.pos {
							t, ok := row.Tier.(int)
							if ok && t > 0 && fa.tier > 0 && fa.tier < t {
								isUpgrade = true
								upgradeFor = row.Name
								upgradeType = "Bench"
								break
							}
						}
					}
				}
				rows = append(rows, PlayerRow{
					Pos:         fa.pos,
					Name:        fa.name,
					Tier:        fa.tier,
					IsFreeAgent: true,
					IsUpgrade:   isUpgrade,
					UpgradeFor:  upgradeFor,
					UpgradeType: upgradeType,
				})
			}
			if len(rows) > 0 {
				freeAgentsByPos[pos] = rows
			}
		}

		avgTier := avg(starterTiers)
		avgOppTier := avg(oppTiers)
		winProb, emoji := winProbability(avgTier, avgOppTier)

		leagueData := LeagueData{
			LeagueName:      leagueName,
			Scoring:         scoring,
			Starters:        startersRows,
			Unranked:        unrankedRows,
			AvgTier:         avgTier,
			AvgOppTier:      avgOppTier,
			WinProb:         winProb + " " + emoji,
			Bench:           benchRows,
			FreeAgentsByPos: freeAgentsByPos,
		}

		leagueResults = append(leagueResults, leagueData)
	}

	if len(leagueResults) == 0 {
		log.Printf("[DEBUG] No valid leagues found with matchups for user %s", username)
		renderError(w, "No valid leagues found with matchups")
		return
	}

	username = r.FormValue("username")
	templates.ExecuteTemplate(w, "tiers.html", TiersPage{Leagues: leagueResults, Username: username})
}

func renderError(w http.ResponseWriter, msg string) {
	username := ""
	if u := w.Header().Get("X-Username"); u != "" {
		username = u
	}
	templates.ExecuteTemplate(w, "tiers.html", TiersPage{Error: msg, Username: username})
}

// --- Helper functions ---
func fetchJSON(url string) (map[string]interface{}, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&out)
	return out, err
}

func fetchJSONArray(url string) ([]map[string]interface{}, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&out)
	return out, err
}

func toStringSlice(val interface{}) []string {
	arr := []string{}
	if val == nil {
		return arr
	}
	switch v := val.(type) {
	case []interface{}:
		for _, x := range v {
			if s, ok := x.(string); ok {
				arr = append(arr, s)
			}
		}
	}
	return arr
}

func diff(a, b []string) []string {
	m := make(map[string]bool)
	for _, x := range b {
		m[x] = true
	}
	out := []string{}
	for _, x := range a {
		if !m[x] {
			out = append(out, x)
		}
	}
	return out
}

// --- Boris Chen tier fetching ---
var borisURLs = map[string]map[string]string{
	"PPR": {
		"QB":  "https://s3-us-west-1.amazonaws.com/fftiers/out/text_QB.txt",
		"RB":  "https://s3-us-west-1.amazonaws.com/fftiers/out/text_RB-PPR.txt",
		"WR":  "https://s3-us-west-1.amazonaws.com/fftiers/out/text_WR-PPR.txt",
		"TE":  "https://s3-us-west-1.amazonaws.com/fftiers/out/text_TE-PPR.txt",
		"K":   "https://s3-us-west-1.amazonaws.com/fftiers/out/text_K.txt",
		"DST": "https://s3-us-west-1.amazonaws.com/fftiers/out/text_DST.txt",
	},
	"Half PPR": {
		"QB":  "https://s3-us-west-1.amazonaws.com/fftiers/out/text_QB.txt",
		"RB":  "https://s3-us-west-1.amazonaws.com/fftiers/out/text_RB-HALF.txt",
		"WR":  "https://s3-us-west-1.amazonaws.com/fftiers/out/text_WR-HALF.txt",
		"TE":  "https://s3-us-west-1.amazonaws.com/fftiers/out/text_TE-HALF.txt",
		"K":   "https://s3-us-west-1.amazonaws.com/fftiers/out/text_K.txt",
		"DST": "https://s3-us-west-1.amazonaws.com/fftiers/out/text_DST.txt",
	},
	"Standard": {
		"QB":  "https://s3-us-west-1.amazonaws.com/fftiers/out/text_QB.txt",
		"RB":  "https://s3-us-west-1.amazonaws.com/fftiers/out/text_RB.txt",
		"WR":  "https://s3-us-west-1.amazonaws.com/fftiers/out/text_WR.txt",
		"TE":  "https://s3-us-west-1.amazonaws.com/fftiers/out/text_TE.txt",
		"K":   "https://s3-us-west-1.amazonaws.com/fftiers/out/text_K.txt",
		"DST": "https://s3-us-west-1.amazonaws.com/fftiers/out/text_DST.txt",
	},
}

func fetchBorisTiers(scoring string) map[string][][]string {
	urls := borisURLs[scoring]
	out := make(map[string][][]string)
	for pos, url := range urls {
		resp, err := http.Get(url)
		if err != nil {
			continue
		}
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
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
				out[pos] = append(out[pos], names)
			}
		}
	}
	return out
}

// --- Build rows for starters/bench ---
func buildRows(ids []string, players map[string]interface{}, tiers map[string][][]string, isStarter bool, userRoster map[string]interface{}, irList []string, bestOtherTier map[string]int) ([]PlayerRow, []PlayerRow, []int) {
	rows := []PlayerRow{}
	unranked := []PlayerRow{}
	tierNums := []int{}
	// For bench, mark as swap candidate if this player is better than any starter at same position
	for idx, pid := range ids {
		p, ok := players[pid].(map[string]interface{})
		if !ok {
			continue
		}

		pos := getPos(p, idx, isStarter, userRoster)
		name := getPlayerName(p)

		// For FLEX, use actual position for tier lookup
		lookupPos := pos
		if pos == "FLEX" {
			if realPos, ok := p["position"].(string); ok {
				lookupPos = realPos
			}
		}
		// Always use DST for DEF/DST for Boris Chen mapping
		if lookupPos == "DEF" {
			lookupPos = "DST"
		}

		tier := findTier(tiers[lookupPos], name)

		// Add FLEX/SUPERFLEX/IR indicator to name if applicable
		displayName := name
		if pos == "FLEX" {
			displayName += ` <span style="color:#7bb0ff;font-size:0.95em;">(FLEX)</span>`
		} else if pos == "SUPERFLEX" {
			displayName += ` <span style="color:#7bb0ff;font-size:0.95em;">(SUPERFLEX)</span>`
		}
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
		if tier > 0 {
			rows = append(rows, PlayerRow{Pos: pos, Name: displayName, Tier: tier, IsTierWorseThanBench: isWorse, ShouldSwapIn: shouldSwapIn})
			tierNums = append(tierNums, tier)
		} else {
			unranked = append(unranked, PlayerRow{Pos: "?", Name: displayName, Tier: "Not Ranked", IsTierWorseThanBench: false, ShouldSwapIn: false})
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

func findTier(tiers [][]string, name string) int {
	norm := normalizeName(name)
	for i, names := range tiers {
		for _, n := range names {
			if normalizeName(n) == norm {
				return i + 1
			}
		}
	}
	return 0
}

func normalizeName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, ".", "")
	name = strings.ReplaceAll(name, ",", "")
	for _, suf := range []string{" jr", " sr", " ii", " iii", " iv", " v"} {
		name = strings.TrimSuffix(name, suf)
	}
	// Remove non-alphanumeric except spaces
	var result strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == ' ' {
			result.WriteRune(r)
		}
	}
	name = strings.Join(strings.Fields(result.String()), " ")
	return name
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
