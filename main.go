package main

import (
	"encoding/json"
	"flag"
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
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var logLevel string
var testMode bool

// HTTP client with connection pooling
var httpClient = &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
	},
}

// Cache for Boris Chen tiers with TTL
type tiersCache struct {
	sync.RWMutex
	data      map[string]map[string][][]string
	timestamp map[string]time.Time
	ttl       time.Duration
}

var borisTiersCache = &tiersCache{
	data:      make(map[string]map[string][][]string),
	timestamp: make(map[string]time.Time),
	ttl:       15 * time.Minute, // Cache tiers for 15 minutes
}

// Dynasty value data structure
type DynastyValue struct {
	Name         string
	Position     string
	Value1QB     int
	Value2QB     int
	ScrapeDate   string
}

// Cache for dynasty values
type dynastyCache struct {
	sync.RWMutex
	data      map[string]DynastyValue // key: normalized player name
	timestamp time.Time
	ttl       time.Duration
}

var dynastyValuesCache = &dynastyCache{
	data: make(map[string]DynastyValue),
	ttl:  24 * time.Hour, // Cache for 24 hours (values don't change frequently)
}

func debugLog(format string, v ...interface{}) {
	if logLevel == "debug" {
		log.Printf(format, v...)
	}
}

// --- Prometheus metrics ---
var (
	totalVisitors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "sleeperpy_total_visitors",
		Help: "Total number of unique visitors to the site.",
	})
	totalLookups = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "sleeperpy_total_lookups",
		Help: "Total number of /lookup requests.",
	})
	totalLeagues = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "sleeperpy_total_leagues",
		Help: "Total number of leagues processed.",
	})
	totalTeams = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "sleeperpy_total_teams",
		Help: "Total number of teams processed.",
	})
	totalErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "sleeperpy_total_errors",
		Help: "Total number of errors encountered.",
	})
)

func init() {
	prometheus.MustRegister(totalVisitors)
	prometheus.MustRegister(totalLookups)
	prometheus.MustRegister(totalLeagues)
	prometheus.MustRegister(totalTeams)
	prometheus.MustRegister(totalErrors)
}

var funcMap = template.FuncMap{
	"safe":    func(s string) template.HTML { return template.HTML(s) },
	"float64": func(i int) float64 { return float64(i) },
	"mul":     func(a, b float64) float64 { return a * b },
	"div":     func(a, b float64) float64 { if b == 0 { return 0 }; return a / b },
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
	"formatTime": func(t time.Time) string {
		if t.IsZero() {
			return "Unknown"
		}
		// Format as relative time
		now := time.Now()
		diff := now.Sub(t)

		if diff < time.Minute {
			return "just now"
		} else if diff < time.Hour {
			mins := int(diff.Minutes())
			if mins == 1 {
				return "1 min ago"
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
		return t.Format("Jan 2, 2006")
	},
	"add": func(a, b int) int { return a + b },
}

var templates = template.Must(template.New("").Funcs(funcMap).ParseGlob("templates/*.html"))

func main() {
	flag.StringVar(&logLevel, "log", "info", "Log level: info or debug")
	flag.BoolVar(&testMode, "test", false, "Run in test mode with mock data")
	flag.Parse()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Initialize test mode if enabled
	if testMode {
		initTestMode()
		log.Printf("[TEST MODE] Mock API endpoints registered")
		http.HandleFunc("/api/mock/", mockAPIHandler)
		http.HandleFunc("/boris/mock/", mockBorisTiersHandler)
	}

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	http.HandleFunc("/", visitorLogging(indexHandler))
	http.HandleFunc("/lookup", lookupHandler)
	http.HandleFunc("/signout", signoutHandler)
	http.Handle("/metrics", promhttp.Handler())

	if testMode {
		log.Printf("Server running on http://localhost:%s (log level: %s, TEST MODE ENABLED)", port, logLevel)
		log.Printf("  → Use username 'testuser' to see mock data")
		log.Printf("  → 3 test leagues will be loaded with mock tiers")
	} else {
		log.Printf("Server running on http://localhost:%s (log level: %s)", port, logLevel)
	}
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// Middleware to log and count unique visitors (by IP)
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
	savedUsername := ""
	if cookie, err := r.Cookie("sleeper_username"); err == nil {
		savedUsername = cookie.Value
	}
	templates.ExecuteTemplate(w, "index.html", IndexPage{SavedUsername: savedUsername})
}

func signoutHandler(w http.ResponseWriter, r *http.Request) {
	// Clear the username cookie
	cookie := &http.Cookie{
		Name:     "sleeper_username",
		Value:    "",
		Path:     "/",
		MaxAge:   -1, // Delete cookie
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)

	// Redirect to home page
	http.Redirect(w, r, "/", http.StatusSeeOther)
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
	IsFlex               bool   // Heuristic FLEX indicator
	IsSuperflex          bool   // Heuristic SUPERFLEX indicator
	DynastyValue         int    // Dynasty value from DynastyProcess (0-10000 scale)
	Age                  int    // Player age from Sleeper API
}

type TeamAgeData struct {
	TeamName    string
	OwnerName   string
	AvgAge      float64
	Rank        int // Standings rank (wins)
	RosterID    int
	IsUserTeam  bool
	RosterValue int // Total dynasty value of roster
}

type PowerRanking struct {
	Rank         int
	TeamName     string
	RosterValue  int
	Wins         int
	Losses       int
	AvgAge       float64
	Strategy     string // "Win Now", "Contending", "Rebuilding"
	IsUserTeam   bool
	ValueRank    int // Rank by dynasty value
	StandingRank int // Rank by wins
}

type DraftPick struct {
	Round        int
	Year         int
	OwnerName    string // "You" or team name
	OriginalName string // Original owner if traded, empty if not traded
	RosterID     int
	IsYours      bool
}

type PositionalKTC struct {
	QB int
	RB int
	WR int
	TE int
}

type TradeTarget struct {
	TeamName        string
	Reason          string
	YourSurplus     string
	TheirSurplus    string
	YourSurplusKTC  int
	TheirSurplusKTC int
}

type PlayerNews struct {
	PlayerName       string
	Position         string
	NewsText         string
	Source           string
	Timestamp        time.Time
	InjuryStatus     string
	InjuryBodyPart   string
	InjuryNotes      string
	IsStarter        bool
	DynastyValue     int
}

type Transaction struct {
	Type        string    // "trade", "waiver", "free_agent"
	Timestamp   time.Time
	Description string
	TeamNames   []string
	PlayerNames []string
	// For trades: better structure
	Team1        string
	Team2        string
	Team1Gave    []string
	Team2Gave    []string
	AddedPlayer  string // For waivers/FA
	DroppedPlayer string // For waivers/FA
}

type RookieProspect struct {
	Name     string
	Position string
	College  string
	Value    int
	Rank     int
	Year     int // Draft year
}

type LeagueData struct {
	LeagueName           string
	Scoring              string
	IsDynasty            bool
	HasMatchups          bool
	DynastyValueDate     string // Date dynasty values were last updated
	Starters             []PlayerRow
	Unranked             []PlayerRow
	AvgTier              string
	AvgOppTier           string
	WinProb              string
	Bench                []PlayerRow
	BenchUnranked        []PlayerRow
	FreeAgentsByPos      map[string][]PlayerRow
	TopFreeAgents        []PlayerRow // Combined prioritized list (tier-based)
	TopFreeAgentsByValue []PlayerRow // Dynasty mode: value-based recommendations
	TotalRosterValue     int         // Sum of all dynasty values on roster
	UserAvgAge           float64     // Average age of user's roster
	TeamAges             []TeamAgeData // All teams' ages for dynasty chart
	PowerRankings        []PowerRanking // League-wide power rankings (dynasty only)
	DraftPicks           []DraftPick // User's draft picks (dynasty only)
	TradeTargets         []TradeTarget // Potential trade partners (dynasty only)
	PositionalBreakdown  PositionalKTC // User's positional value breakdown (dynasty only)
	PlayerNewsFeed       []PlayerNews      // Player news for all roster players (dynasty only)
	BreakoutCandidates   []PlayerRow       // Young players with upside (dynasty only)
	AgingPlayers         []PlayerRow       // Players approaching decline (dynasty only)
	RecentTransactions   []Transaction     // Recent league transactions (dynasty only)
	TopRookies           []RookieProspect  // Top rookie prospects for upcoming draft (dynasty only)
}

type TiersPage struct {
	Error    string
	Leagues  []LeagueData
	Username string
}

type IndexPage struct {
	SavedUsername string
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
	totalLookups.Inc()
	debugLog("[DEBUG] /lookup handler called")
	r.ParseForm()
	username := r.FormValue("username")
	debugLog("[DEBUG] Username submitted: %s", username)
	if username == "" {
		debugLog("[DEBUG] No username provided")
		totalErrors.Inc()
		renderError(w, "No username provided")
		return
	}

	// 1. Get user ID
	user, err := fetchJSON(fmt.Sprintf("https://api.sleeper.app/v1/user/%s", username))
	if err != nil || user["user_id"] == nil {
		log.Printf("[ERROR] User not found or error: %v", err)
		totalErrors.Inc()
		renderError(w, "User not found")
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
		renderError(w, "No leagues found for this user")
		return
	}

	// 3. Get current NFL week from Sleeper API
	state, err := fetchJSON("https://api.sleeper.app/v1/state/nfl")
	if err != nil {
		log.Printf("[ERROR] Could not get current NFL week: %v", err)
		totalErrors.Inc()
		renderError(w, "Could not get current NFL week")
		return
	}
	week := int(state["week"].(float64))

	// 4. Get players data
	players, err := fetchJSON("https://api.sleeper.app/v1/players/nfl")
	if err != nil {
		log.Printf("[ERROR] Could not fetch players data: %v", err)
		totalErrors.Inc()
		renderError(w, "Could not fetch players data")
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
		// (info log removed, only debug available)
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
		if isDynasty {
			// Fetch traded picks from Sleeper API
			// KNOWN ISSUE: This logic may show incorrect pick ownership
			// Expected behavior:
			//   - Only show picks currently owned by the user
			//   - If pick was acquired via trade, show "from <team>"
			//   - If pick was traded away, DON'T show it at all
			//
			// Current implementation uses:
			//   - roster_id: assumed to be current owner
			//   - owner_id: assumed to be original owner (team pick belonged to by default)
			//   - previous_owner_id: assumed to be previous owner before current trade
			//
			// Debug logging will help verify these field meanings with real API data
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
						debugLog("[DEBUG]   roster_id (current owner): %.0f", rosterID)
					}
					if ownerID, ok := trade["owner_id"].(float64); ok {
						debugLog("[DEBUG]   owner_id (original owner): %.0f", ownerID)
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
			userRosterID, _ := userRoster["roster_id"].(float64)
			debugLog("[DEBUG] User roster ID: %.0f", userRosterID)

			// Start with default picks (each team has their own picks by default)
			pickOwnership := make(map[string]int) // key: "year-round-original_roster_id" -> current_owner_roster_id

			// Initialize with default picks for next 3 years
			for year := currentYear; year < currentYear+3; year++ {
				for round := 1; round <= numRounds; round++ {
					for _, r := range rosters {
						rosterID, _ := r["roster_id"].(float64)
						key := fmt.Sprintf("%d-%d-%d", year, round, int(rosterID))
						pickOwnership[key] = int(rosterID) // Initially owned by the original team
					}
				}
			}
			debugLog("[DEBUG] Initialized %d default picks", len(pickOwnership))

			// Apply traded picks
			if tradedPicks != nil {
				debugLog("[DEBUG] Processing %d traded picks", len(tradedPicks))
				for i, trade := range tradedPicks {
					season, _ := trade["season"].(string)
					round, _ := trade["round"].(float64)
					rosterID, _ := trade["roster_id"].(float64)          // Current owner
					originalRosterID, _ := trade["owner_id"].(float64)  // Original owner (who the pick belonged to)
					previousOwnerID, _ := trade["previous_owner_id"].(float64)

					year, _ := strconv.Atoi(season)
					key := fmt.Sprintf("%d-%d-%d", year, int(round), int(originalRosterID))

					debugLog("[DEBUG] Trade %d: %d Round %d (originally from roster %.0f)", i, year, int(round), originalRosterID)
					debugLog("[DEBUG]   Current owner (roster_id): %.0f", rosterID)
					debugLog("[DEBUG]   Previous owner: %.0f", previousOwnerID)
					debugLog("[DEBUG]   Key: %s", key)

					// Update ownership
					oldOwner := pickOwnership[key]
					if rosterID > 0 {
						pickOwnership[key] = int(rosterID)
						debugLog("[DEBUG]   ✓ Updated ownership from roster %d to roster %.0f", oldOwner, rosterID)
					} else if previousOwnerID > 0 {
						// If roster_id is 0, the pick was traded away from previous owner
						delete(pickOwnership, key)
						debugLog("[DEBUG]   ✗ Deleted pick (roster_id is 0, pick traded away)")
					}
				}
			}

			// Extract user's picks
			debugLog("[DEBUG] ===== EXTRACTING USER PICKS =====")
			debugLog("[DEBUG] Searching for picks owned by roster %.0f", userRosterID)
			for key, ownerRosterID := range pickOwnership {
				if ownerRosterID == int(userRosterID) {
					parts := strings.Split(key, "-")
					if len(parts) != 3 {
						continue
					}
					year, _ := strconv.Atoi(parts[0])
					round, _ := strconv.Atoi(parts[1])
					originalRosterID, _ := strconv.Atoi(parts[2])

					ownerName := "You"
					originalName := ""

					// If this pick was traded (original owner != current owner)
					if originalRosterID != int(userRosterID) {
						if origOwner, exists := rosterOwners[originalRosterID]; exists {
							originalName = origOwner
						}
						debugLog("[DEBUG] Pick %d Round %d: ACQUIRED from roster %d (%s)", year, round, originalRosterID, originalName)
					} else {
						debugLog("[DEBUG] Pick %d Round %d: YOUR ORIGINAL pick", year, round)
					}

					draftPicks = append(draftPicks, DraftPick{
						Round:        round,
						Year:         year,
						OwnerName:    ownerName,
						OriginalName: originalName,
						RosterID:     int(userRosterID),
						IsYours:      true,
					})
				}
			}

			// Sort picks by year, then by round
			sort.Slice(draftPicks, func(i, j int) bool {
				if draftPicks[i].Year != draftPicks[j].Year {
					return draftPicks[i].Year < draftPicks[j].Year
				}
				return draftPicks[i].Round < draftPicks[j].Round
			})

			debugLog("[DEBUG] ===================================")
			debugLog("[DEBUG] FINAL RESULT: User has %d draft picks total", len(draftPicks))
			for _, pick := range draftPicks {
				if pick.OriginalName != "" {
					debugLog("[DEBUG]   - %d Round %d (from %s)", pick.Year, pick.Round, pick.OriginalName)
				} else {
					debugLog("[DEBUG]   - %d Round %d (original)", pick.Year, pick.Round)
				}
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
			recentTransactions = fetchRecentTransactions(leagueID, week, players, rosters, userNames)
			debugLog("[DEBUG] Found %d recent transactions", len(recentTransactions))
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
		}

		avgTier := avg(starterTiers)
		avgOppTier := avg(oppTiers)
		winProb, emoji := winProbability(avgTier, avgOppTier)

		leagueData := LeagueData{
			LeagueName:           leagueName,
			Scoring:              scoring,
			IsDynasty:            isDynasty,
			HasMatchups:          hasMatchups,
			DynastyValueDate:     dynastyValueDate,
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
			TradeTargets:         tradeTargets,
			PositionalBreakdown:  positionalBreakdown,
			PlayerNewsFeed:       playerNewsFeed,
			BreakoutCandidates:   breakoutCandidates,
			AgingPlayers:         agingPlayers,
			RecentTransactions:   recentTransactions,
			TopRookies:           topRookies,
		}

		leagueResults = append(leagueResults, leagueData)
	}

	if len(leagueResults) == 0 {
		debugLog("[DEBUG] No valid leagues found with matchups for user %s", username)
		renderError(w, "No valid leagues found with matchups")
		return
	}

	username = r.FormValue("username")

	// Set cookie to remember username for 30 days
	cookie := &http.Cookie{
		Name:     "sleeper_username",
		Value:    username,
		Path:     "/",
		MaxAge:   30 * 24 * 60 * 60, // 30 days
		HttpOnly: false,              // Allow JavaScript to read for UI logic
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)

	if err = templates.ExecuteTemplate(w, "tiers.html", TiersPage{Leagues: leagueResults, Username: username}); err != nil {
		log.Printf("[ERROR] Template execution error: %v", err)
	}
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

func isDynastyLeague(league map[string]interface{}) bool {
	// Check type field - type 2 indicates dynasty league
	if settings, ok := league["settings"].(map[string]interface{}); ok {
		if leagueType, ok := settings["type"].(float64); ok && leagueType == 2 {
			debugLog("[DEBUG] League detected as dynasty via type: %v", leagueType)
			return true
		}
	}

	// Check for taxi squad (dynasty-specific feature)
	if settings, ok := league["settings"].(map[string]interface{}); ok {
		if taxiSlots, ok := settings["taxi_slots"].(float64); ok && taxiSlots > 0 {
			debugLog("[DEBUG] League detected as dynasty via taxi_slots: %v", taxiSlots)
			return true
		}
	}

	// Fallback: check league name for "dynasty" keyword
	if name, ok := league["name"].(string); ok {
		nameLower := strings.ToLower(name)
		if strings.Contains(nameLower, "dynasty") {
			debugLog("[DEBUG] League detected as dynasty via name: %s", name)
			return true
		}
	}

	return false
}

// --- Dynasty Values fetching ---
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

func parseCSVLine(line string) []string {
	var fields []string
	var current strings.Builder
	inQuotes := false

	for _, char := range line {
		switch char {
		case '"':
			inQuotes = !inQuotes
			current.WriteRune(char)
		case ',':
			if inQuotes {
				current.WriteRune(char)
			} else {
				fields = append(fields, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(char)
		}
	}
	fields = append(fields, current.String())
	return fields
}

// --- Boris Chen tier fetching ---
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

// --- Build rows for starters/bench with roster positions ---
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

func stripHTML(s string) string {
	// Strip HTML tags like <span...>...</span>
	if idx := strings.Index(s, "<span"); idx >= 0 {
		return strings.TrimSpace(s[:idx])
	}
	return s
}

// enrichRowsWithDynastyValues adds dynasty values to player rows
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

// aggregatePlayerNews collects news for all players on the user's roster
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
				PlayerName:       name,
				Position:         pos,
				NewsText:         newsText,
				Source:           source,
				Timestamp:        timestamp,
				InjuryStatus:     injuryStatus,
				InjuryBodyPart:   injuryBodyPart,
				InjuryNotes:      injuryNotes,
				IsStarter:        isStarter,
				DynastyValue:     dynastyValue,
			})
		}
	}

	// Sort by timestamp (newest first)
	sort.Slice(newsFeed, func(i, j int) bool {
		return newsFeed[i].Timestamp.After(newsFeed[j].Timestamp)
	})

	return newsFeed
}

// findBreakoutCandidates identifies young players with high upside on the bench
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

// fetchRecentTransactions gets recent trades, adds, and drops from the league
func fetchRecentTransactions(leagueID string, currentWeek int, players map[string]interface{}, rosters []map[string]interface{}, userNames map[string]string) []Transaction {
	transactions := []Transaction{}

	// Fetch last 4 weeks of transactions
	weeksToFetch := 4
	startWeek := currentWeek - weeksToFetch
	if startWeek < 1 {
		startWeek = 1
	}

	// Build roster ID to team name map
	rosterIDToName := make(map[int]string)
	for _, r := range rosters {
		rosterID, _ := r["roster_id"].(float64)
		ownerID, _ := r["owner_id"].(string)

		// Get display name from userNames map, fall back to metadata team_name
		teamName := ""
		if ownerID != "" && userNames != nil {
			if displayName, exists := userNames[ownerID]; exists && displayName != "" {
				teamName = displayName
			}
		}

		// If no display name, try metadata team_name
		if teamName == "" {
			if metadata, ok := r["metadata"].(map[string]interface{}); ok {
				if tn, ok := metadata["team_name"].(string); ok && tn != "" {
					teamName = tn
				}
			}
		}

		// Last resort: use owner ID
		if teamName == "" && ownerID != "" {
			teamName = ownerID
		}

		if teamName != "" {
			rosterIDToName[int(rosterID)] = teamName
		}
	}

	for week := startWeek; week <= currentWeek; week++ {
		url := fmt.Sprintf("https://api.sleeper.app/v1/league/%s/transactions/%d", leagueID, week)
		transactionsData, err := fetchJSONArray(url)
		if err != nil {
			debugLog("[DEBUG] Could not fetch transactions for week %d: %v", week, err)
			continue
		}

		for _, t := range transactionsData {
			transType, _ := t["type"].(string)
			if transType == "" {
				continue
			}

			// Parse timestamp (in milliseconds)
			var timestamp time.Time
			if created, ok := t["created"].(float64); ok {
				timestamp = time.Unix(int64(created/1000), 0)
			}

			// Build description based on transaction type
			description := ""
			teamNames := []string{}
			playerNames := []string{}

			switch transType {
			case "trade":
				// Get roster IDs involved
				rosterIDs, _ := t["roster_ids"].([]interface{})
				for _, rid := range rosterIDs {
					if rosterID, ok := rid.(float64); ok {
						if name, exists := rosterIDToName[int(rosterID)]; exists {
							teamNames = append(teamNames, name)
						}
					}
				}

				// Parse adds/drops to determine who gave what
				adds, _ := t["adds"].(map[string]interface{})

				// Map to track which roster got which players
				team1Gave := []string{}
				team2Gave := []string{}

				if len(rosterIDs) >= 2 {
					roster1ID := int(rosterIDs[0].(float64))
					roster2ID := int(rosterIDs[1].(float64))

					// Players in adds are who RECEIVED them
					// So if roster1 received a player, roster2 gave it up
					for playerID, rosterIDVal := range adds {
						if rosterID, ok := rosterIDVal.(float64); ok {
							if p, ok := players[playerID].(map[string]interface{}); ok {
								playerName := getPlayerName(p)
								playerNames = append(playerNames, playerName)

								if int(rosterID) == roster1ID {
									// Roster 1 got this player, so Roster 2 gave it
									team2Gave = append(team2Gave, playerName)
								} else if int(rosterID) == roster2ID {
									// Roster 2 got this player, so Roster 1 gave it
									team1Gave = append(team1Gave, playerName)
								}
							}
						}
					}
				} else {
					// Fallback for trades without clear roster IDs
					for playerID := range adds {
						if p, ok := players[playerID].(map[string]interface{}); ok {
							playerNames = append(playerNames, getPlayerName(p))
						}
					}
				}

				if len(teamNames) >= 2 {
					description = fmt.Sprintf("Trade between %s and %s", teamNames[0], teamNames[1])
					// Store structured trade data
					if len(teamNames) >= 2 {
						transactions = append(transactions, Transaction{
							Type:        transType,
							Timestamp:   timestamp,
							Description: description,
							TeamNames:   teamNames,
							PlayerNames: playerNames,
							Team1:       teamNames[0],
							Team2:       teamNames[1],
							Team1Gave:   team1Gave,
							Team2Gave:   team2Gave,
						})
						continue // Skip the generic append at the end
					}
				} else {
					description = "Trade completed"
				}

			case "waiver":
				// Waiver claim
				adds, _ := t["adds"].(map[string]interface{})
				drops, _ := t["drops"].(map[string]interface{})

				addedPlayer := ""
				droppedPlayer := ""
				var teamName string

				for playerID, rosterIDVal := range adds {
					if rosterID, ok := rosterIDVal.(float64); ok {
						teamName = rosterIDToName[int(rosterID)]
						if p, ok := players[playerID].(map[string]interface{}); ok {
							addedPlayer = getPlayerName(p)
							playerNames = append(playerNames, addedPlayer)
						}
					}
				}

				for playerID := range drops {
					if p, ok := players[playerID].(map[string]interface{}); ok {
						droppedPlayer = getPlayerName(p)
					}
				}

				if addedPlayer != "" && droppedPlayer != "" {
					description = fmt.Sprintf("%s claimed %s (dropped %s)", teamName, addedPlayer, droppedPlayer)
				} else if addedPlayer != "" {
					description = fmt.Sprintf("%s claimed %s", teamName, addedPlayer)
				}

				// Store structured waiver data
				if description != "" {
					transactions = append(transactions, Transaction{
						Type:          transType,
						Timestamp:     timestamp,
						Description:   description,
						TeamNames:     []string{teamName},
						PlayerNames:   playerNames,
						AddedPlayer:   addedPlayer,
						DroppedPlayer: droppedPlayer,
					})
					continue
				}

			case "free_agent":
				// Free agent add/drop
				adds, _ := t["adds"].(map[string]interface{})
				drops, _ := t["drops"].(map[string]interface{})

				addedPlayer := ""
				droppedPlayer := ""
				var teamName string

				for playerID, rosterIDVal := range adds {
					if rosterID, ok := rosterIDVal.(float64); ok {
						teamName = rosterIDToName[int(rosterID)]
						if p, ok := players[playerID].(map[string]interface{}); ok {
							addedPlayer = getPlayerName(p)
							playerNames = append(playerNames, addedPlayer)
						}
					}
				}

				for playerID, rosterIDVal := range drops {
					if _, ok := rosterIDVal.(float64); ok {
						if p, ok := players[playerID].(map[string]interface{}); ok {
							droppedPlayer = getPlayerName(p)
						}
					}
				}

				if addedPlayer != "" && droppedPlayer != "" {
					description = fmt.Sprintf("%s added %s (dropped %s)", teamName, addedPlayer, droppedPlayer)
				} else if addedPlayer != "" {
					description = fmt.Sprintf("%s added %s", teamName, addedPlayer)
				}

				// Store structured FA data
				if description != "" {
					transactions = append(transactions, Transaction{
						Type:          transType,
						Timestamp:     timestamp,
						Description:   description,
						TeamNames:     []string{teamName},
						PlayerNames:   playerNames,
						AddedPlayer:   addedPlayer,
						DroppedPlayer: droppedPlayer,
					})
					continue
				}
			}

			if description != "" {
				transactions = append(transactions, Transaction{
					Type:        transType,
					Timestamp:   timestamp,
					Description: description,
					TeamNames:   teamNames,
					PlayerNames: playerNames,
				})
			}
		}
	}

	// Sort by timestamp (newest first)
	sort.Slice(transactions, func(i, j int) bool {
		return transactions[i].Timestamp.After(transactions[j].Timestamp)
	})

	// Limit to 20 most recent
	if len(transactions) > 20 {
		transactions = transactions[:20]
	}

	return transactions
}

// findAgingPlayers flags players approaching the end of their fantasy relevance
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

// getTopRookies returns top rookie prospects for the 2025 and 2026 NFL drafts
func getTopRookies() []RookieProspect {
	return []RookieProspect{
		// 2025 NFL Draft
		{Name: "Shedeur Sanders", Position: "QB", College: "Colorado", Value: 4500, Rank: 1, Year: 2025},
		{Name: "Travis Hunter", Position: "WR", College: "Colorado", Value: 7500, Rank: 2, Year: 2025},
		{Name: "Ashton Jeanty", Position: "RB", College: "Boise State", Value: 6800, Rank: 3, Year: 2025},
		{Name: "Abdul Carter", Position: "LB", College: "Penn State", Value: 0, Rank: 4, Year: 2025},
		{Name: "Tetairoa McMillan", Position: "WR", College: "Arizona", Value: 6500, Rank: 5, Year: 2025},
		{Name: "Will Johnson", Position: "CB", College: "Michigan", Value: 0, Rank: 6, Year: 2025},
		{Name: "Mason Graham", Position: "DT", College: "Michigan", Value: 0, Rank: 7, Year: 2025},
		{Name: "Cam Ward", Position: "QB", College: "Miami", Value: 4200, Rank: 8, Year: 2025},
		{Name: "Malaki Starks", Position: "S", College: "Georgia", Value: 0, Rank: 9, Year: 2025},
		{Name: "Luther Burden III", Position: "WR", College: "Missouri", Value: 6000, Rank: 10, Year: 2025},
		{Name: "Kelvin Banks Jr.", Position: "OT", College: "Texas", Value: 0, Rank: 11, Year: 2025},
		{Name: "Tyler Warren", Position: "TE", College: "Penn State", Value: 3800, Rank: 12, Year: 2025},
		{Name: "Will Campbell", Position: "OT", College: "LSU", Value: 0, Rank: 13, Year: 2025},
		{Name: "Omarion Hampton", Position: "RB", College: "North Carolina", Value: 5500, Rank: 14, Year: 2025},
		{Name: "Mykel Williams", Position: "DE", College: "Georgia", Value: 0, Rank: 15, Year: 2025},

		// 2026 NFL Draft (Early projections)
		{Name: "Quinn Ewers", Position: "QB", College: "Texas", Value: 4000, Rank: 1, Year: 2026},
		{Name: "Jalen Milroe", Position: "QB", College: "Alabama", Value: 3500, Rank: 2, Year: 2026},
		{Name: "Jeremiah Smith", Position: "WR", College: "Ohio State", Value: 6500, Rank: 3, Year: 2026},
		{Name: "TreVeyon Henderson", Position: "RB", College: "Ohio State", Value: 5500, Rank: 4, Year: 2026},
		{Name: "Kelvin Banks III", Position: "OT", College: "Texas", Value: 0, Rank: 5, Year: 2026},
		{Name: "James Pearce Jr.", Position: "DE", College: "Tennessee", Value: 0, Rank: 6, Year: 2026},
		{Name: "Colston Loveland", Position: "TE", College: "Michigan", Value: 3200, Rank: 7, Year: 2026},
		{Name: "Jahdae Barron", Position: "CB", College: "Texas", Value: 0, Rank: 8, Year: 2026},
		{Name: "Quinshon Judkins", Position: "RB", College: "Ohio State", Value: 5000, Rank: 9, Year: 2026},
		{Name: "Emeka Egbuka", Position: "WR", College: "Ohio State", Value: 5800, Rank: 10, Year: 2026},
	}
}

// calculatePowerRankings creates league-wide power rankings based on dynasty value and record
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
		// For now, use rank as wins estimate (will be replaced with actual W-L when available)
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

// calculatePositionalKTC calculates total dynasty value by position
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

// findTradeTargets identifies potential trade partners based on complementary positional needs
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
