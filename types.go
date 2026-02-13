// ABOUTME: Type definitions for SleeperPy application
// ABOUTME: Includes data structures for players, leagues, transactions, and caching

package main

import (
	"sync"
	"time"
)

// Cache for Boris Chen tiers with TTL
type tiersCache struct {
	sync.RWMutex
	data      map[string]map[string][][]string
	timestamp map[string]time.Time
	ttl       time.Duration
}

// Dynasty value data structure
type DynastyValue struct {
	Name       string
	Position   string
	Value1QB   int
	Value2QB   int
	ScrapeDate string
}

// Cache for dynasty values
type dynastyCache struct {
	sync.RWMutex
	data      map[string]DynastyValue // key: normalized player name
	timestamp time.Time
	ttl       time.Duration
}

// Cache for Sleeper players API response
type playersCache struct {
	sync.RWMutex
	data      map[string]interface{}
	timestamp time.Time
	ttl       time.Duration
}

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

type ProjectedDraftPick struct {
	Year              int
	Round             int
	OverallPick       int    // Projected overall pick number (e.g., 1.01 = 1, 1.12 = 12, 2.01 = 13)
	ProjectedPosition int    // Projected position within round (1-12)
	OwnerName         string // Team that currently owns this pick
	OriginalOwner     string // Original owner if traded
	CurrentStanding   int    // Current standing of the team (1 = worst record, 12 = best)
	TeamRecord        string // e.g., "3-11"
	IsYours           bool
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
	PlayerName      string
	Position        string
	NewsText        string
	Source          string
	Timestamp       time.Time
	InjuryStatus    string
	InjuryBodyPart  string
	InjuryNotes     string
	IsStarter       bool
	DynastyValue    int
	ImportanceScore int // Feature #4: 0-200 range for news prioritization
}

type CompressedNews struct {
	TimeWindow   string       // "This Week" or "Last 3 Months"
	TopHeadlines []PlayerNews // Max 3 items
	TotalItems   int          // Total news items for user's players
}

type TradeFairness struct {
	Winner        string  // "TeamA" or "TeamB" or "Fair"
	ValueDelta    int     // Absolute KTC difference
	ValueDeltaPct float64 // % of smaller team's roster value
	Fleeced       bool    // Extreme gap flag
	Context       string  // "Competing strategy", "Rebuilding", "Extreme value gap"
	WinnerTeam    string  // Team that got better value
	DisplayBadge  string  // "🟢 +12%", "🔴 FLEECED", "→ Fair"
}

type Transaction struct {
	Type           string // "trade", "waiver", "free_agent"
	Timestamp      time.Time
	Description    string
	TeamNames      []string
	PlayerNames    []string
	Team1          string
	Team2          string
	Team1Gave      []string
	Team2Gave      []string
	Team1GaveValue int   // Total dynasty value of Team1's players
	Team2GaveValue int   // Total dynasty value of Team2's players
	NetValue       int   // Net value difference (positive = Team1 gained value)
	AddedPlayer    string
	DroppedPlayer  string
	Fairness       TradeFairness // Feature #3: Trade fairness detection
}

type RookieProspect struct {
	Name     string
	Position string
	College  string
	Value    int
	Rank     int
	Year     int // Draft year
}

type LeagueTrends struct {
	MostActiveTeams  []TeamActivity
	HotWaiverPlayers []WaiverActivity
	TradeVolume      int
	WaiverVolume     int
	PositionScarcity map[string]int // Position -> number of available players
}

type TeamActivity struct {
	TeamName      string
	Transactions  int
	Trades        int
	WaiverClaims  int
	ActivityLevel string // "Very Active", "Active", "Quiet"
}

type WaiverActivity struct {
	PlayerName  string
	Position    string
	ClaimCount  int
	LastClaimed string // Time ago
}

type Action struct {
	Priority    int    // 1-5 (1 = highest)
	Category    string // "swap", "waiver", "trade", "injury", "lineup"
	Title       string // "Swap Starter"
	Description string // "Start Jahmyr Gibbs over James Conner"
	Impact      string // "+1.2 tier upgrade"
	Link        string // "#player-name" anchor link
	Completed   bool   // User checked it off
	WeekID      string // "2026-W14" for persistence
}

type LeagueData struct {
	LeagueName           string
	Scoring              string
	IsDynasty            bool
	HasMatchups          bool
	DynastyValueDate     string
	LeagueSize           int
	RosterSlots          string
	Starters             []PlayerRow
	Unranked             []PlayerRow
	AvgTier              string
	AvgOppTier           string
	WinProb              string
	Bench                []PlayerRow
	BenchUnranked        []PlayerRow
	FreeAgentsByPos      map[string][]PlayerRow
	TopFreeAgents        []PlayerRow
	TopFreeAgentsByValue []PlayerRow
	TotalRosterValue     int
	UserAvgAge           float64
	TeamAges             []TeamAgeData
	PowerRankings        []PowerRanking
	DraftPicks           []DraftPick
	ProjectedDraftPicks  []ProjectedDraftPick
	TradeTargets         []TradeTarget
	PositionalBreakdown  PositionalKTC
	PlayerNewsFeed       []PlayerNews
	BreakoutCandidates   []PlayerRow
	AgingPlayers         []PlayerRow
	RecentTransactions   []Transaction
	TopRookies           []RookieProspect
	LeagueTrends         LeagueTrends
	PremiumTeamTalk      string
	WeeklyActions        []Action       // Feature #2: Weekly Action List
	CompressedNews       CompressedNews // Feature #4: News Signal Compression
}

type TiersPage struct {
	Error           string
	Leagues         []LeagueData
	Username        string
	IsPremium       bool
	PremiumEnabled  bool
	PremiumOverview string
}

type IndexPage struct {
	SavedUsername string
}

// Dashboard types for cross-league overview
type LeagueSummary struct {
	LeagueID          string
	LeagueName        string
	Season            string // "2024", "2025" - year of the league
	Scoring           string
	IsDynasty         bool
	IsSuperFlex       bool
	LeagueSize        int

	// Dynasty metrics
	TotalRosterValue  int
	ValueRank         int     // 1-12
	ValueTrend        string  // "↗ +5%", "↘ -3%", "→ stable"
	AvgAge            float64
	AgeRank           int
	DraftPicksSummary string // "2026 1st, 2027 1st, 2nd"

	// Season metrics
	Record         string // "8-5" or empty if offseason
	PlayoffStatus  string // "Clinched", "In Hunt", "Eliminated", ""

	// Action items (for Feature #2)
	ActionCount int

	LastUpdated time.Time
}

type DashboardPage struct {
	Username        string
	LeagueSummaries []LeagueSummary
	TotalLeagues    int
	DynastyCount    int
	RedraftCount    int
}

// Cache for roster value trends (24h comparison)
type CachedRosterValue struct {
	RosterValue int
	Timestamp   time.Time
}

type valueTrendCache struct {
	sync.RWMutex
	data map[string]CachedRosterValue // key: "username:leagueID"
	ttl  time.Duration
}
