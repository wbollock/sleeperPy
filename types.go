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
	PlayerName     string
	Position       string
	NewsText       string
	Source         string
	Timestamp      time.Time
	InjuryStatus   string
	InjuryBodyPart string
	InjuryNotes    string
	IsStarter      bool
	DynastyValue   int
}

type Transaction struct {
	Type          string // "trade", "waiver", "free_agent"
	Timestamp     time.Time
	Description   string
	TeamNames     []string
	PlayerNames   []string
	Team1         string
	Team2         string
	Team1Gave     []string
	Team2Gave     []string
	AddedPlayer   string
	DroppedPlayer string
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

type LeagueData struct {
	LeagueName           string
	Scoring              string
	IsDynasty            bool
	HasMatchups          bool
	DynastyValueDate     string
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
}

type TiersPage struct {
	Error    string
	Leagues  []LeagueData
	Username string
}

type IndexPage struct {
	SavedUsername string
}
