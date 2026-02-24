// Season planner - generates strategic roadmap for dynasty leagues
// Shows key dates, trade deadlines, and optimal timing for moves

package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Generate season plan for a league
func generateSeasonPlan(league LeagueData, isPremium bool) SeasonPlan {
	if !isPremium {
		return SeasonPlan{} // Premium feature
	}

	now := time.Now()
	plan := SeasonPlan{
		KeyDates:        []KeyDate{},
		Recommendations: []string{},
	}

	// Determine current phase
	plan.CurrentPhase, plan.PhaseDescription = determineSeasonPhase(now)

	// Determine strategy based on team composition
	plan.Strategy = determineStrategy(league)

	// Build a simple schedule difficulty profile for upcoming weeks
	plan.ScheduleDifficulty, plan.ScheduleNote = buildScheduleDifficultyProfile(league, plan.Strategy)

	// Generate key dates
	plan.KeyDates = generateKeyDates(now, league.HasMatchups)

	// Calculate weeks remaining
	plan.WeeksRemaining = calculateWeeksRemaining(now, league.HasMatchups)

	// Determine trade window status
	plan.TradeWindow = determineTradeWindow(now)

	// Generate strategic recommendations
	plan.Recommendations = generateStrategicRecommendations(league, plan.Strategy, plan.CurrentPhase, plan.ScheduleDifficulty)

	// Set next milestone
	if len(plan.KeyDates) > 0 {
		plan.NextMilestone = fmt.Sprintf("%s (%d days)", plan.KeyDates[0].Label, plan.KeyDates[0].DaysAway)
	}

	return plan
}

func determineSeasonPhase(now time.Time) (string, string) {
	month := now.Month()

	if month >= time.February && month <= time.April {
		return "Offseason", "Draft prep season - evaluate prospects, plan draft strategy"
	} else if month >= time.May && month <= time.July {
		return "Rookie Draft Season", "Rookie draft time - capitalize on draft capital"
	} else if month == time.August {
		return "Preseason", "Final roster construction before season starts"
	} else if month == time.September || (month == time.October && now.Day() < 15) {
		return "Early Season", "Sample size small - avoid panic trades"
	} else if month == time.October || month == time.November {
		return "Mid Season", "Trade deadline approaching - buy or sell window"
	} else if month == time.December && now.Day() < 25 {
		return "Playoff Push", "Final push for playoff spots - all-in moves"
	} else {
		return "Playoffs/Offseason", "Championships or planning for next year"
	}
}

func determineStrategy(league LeagueData) string {
	// For dynasty leagues, use power ranking and age
	if league.IsDynasty {
		userRank := 0
		for _, pr := range league.PowerRankings {
			if pr.IsUserTeam {
				userRank = pr.ValueRank
				avgAge := pr.AvgAge

				// Competing: top 4 value, any age
				if userRank <= 4 {
					if avgAge > 27.5 {
						return "Win Now"
					}
					return "Contending"
				}

				// Rebuilding: bottom half value
				if userRank > 6 {
					if avgAge < 25.5 {
						return "Rebuild"
					}
					return "Retool"
				}

				// Middle: depends on age
				if avgAge < 26.0 {
					return "Build"
				}
				return "Retool"
			}
		}
	}

	// For redraft, use win record
	if league.WinProb != "" {
		// Parse win probability if available
		return "Compete" // Default for redraft
	}

	return "Compete"
}

func generateKeyDates(now time.Time, inSeason bool) []KeyDate {
	dates := []KeyDate{}
	year := now.Year()

	// Trade deadline (typically week 10-11 = early November)
	tradeDeadline := time.Date(year, time.November, 5, 0, 0, 0, 0, time.UTC)
	if now.Before(tradeDeadline) && inSeason {
		daysAway := int(tradeDeadline.Sub(now).Hours() / 24)
		actions := []string{"Evaluate buy/sell decisions", "Lock in playoff roster"}
		if daysAway < 14 {
			actions = append(actions, "URGENT: Final trade window closing")
		}
		dates = append(dates, KeyDate{
			Date:        tradeDeadline,
			Label:       "Trade Deadline",
			DaysAway:    daysAway,
			Importance:  "Critical",
			ActionItems: actions,
		})
	}

	// Playoff start (typically week 15 = mid December)
	playoffStart := time.Date(year, time.December, 16, 0, 0, 0, 0, time.UTC)
	if now.Before(playoffStart) && inSeason {
		daysAway := int(playoffStart.Sub(now).Hours() / 24)
		dates = append(dates, KeyDate{
			Date:        playoffStart,
			Label:       "Playoffs Begin",
			DaysAway:    daysAway,
			Importance:  "High",
			ActionItems: []string{"Finalize lineup decisions", "Check playoff matchups"},
		})
	}

	// Rookie draft (typically late April/early May)
	rookieDraft := time.Date(year, time.May, 1, 0, 0, 0, 0, time.UTC)
	if now.Before(rookieDraft) && now.Month() >= time.February && now.Month() <= time.April {
		daysAway := int(rookieDraft.Sub(now).Hours() / 24)
		dates = append(dates, KeyDate{
			Date:        rookieDraft,
			Label:       "Rookie Draft",
			DaysAway:    daysAway,
			Importance:  "High",
			ActionItems: []string{"Scout prospects", "Trade for picks", "Plan draft board"},
		})
	}

	// NFL Draft (late April)
	nflDraft := time.Date(year, time.April, 25, 0, 0, 0, 0, time.UTC)
	if now.Before(nflDraft) && now.Month() >= time.February && now.Month() <= time.April {
		daysAway := int(nflDraft.Sub(now).Hours() / 24)
		dates = append(dates, KeyDate{
			Date:        nflDraft,
			Label:       "NFL Draft",
			DaysAway:    daysAway,
			Importance:  "Medium",
			ActionItems: []string{"Watch landing spots", "Adjust rankings"},
		})
	}

	// Sort by date (soonest first)
	// dates are already in chronological order, but we only add future dates

	return dates
}

func calculateWeeksRemaining(now time.Time, inSeason bool) int {
	if !inSeason {
		return 0
	}

	// NFL regular season typically ends mid-December
	seasonEnd := time.Date(now.Year(), time.December, 15, 0, 0, 0, 0, time.UTC)
	if now.After(seasonEnd) {
		return 0
	}

	weeksRemaining := int(seasonEnd.Sub(now).Hours() / 24 / 7)
	if weeksRemaining < 0 {
		return 0
	}

	return weeksRemaining
}

func determineTradeWindow(now time.Time) string {
	month := now.Month()

	// Trade deadline typically early November
	tradeDeadline := time.Date(now.Year(), time.November, 5, 0, 0, 0, 0, time.UTC)

	if now.After(tradeDeadline) && month < time.April {
		return "Closed"
	}

	// If within 2 weeks of deadline
	if now.After(tradeDeadline.AddDate(0, 0, -14)) && now.Before(tradeDeadline) {
		return "Closing Soon"
	}

	if month >= time.February || month >= time.August {
		return "Open"
	}

	return "Open"
}

func generateStrategicRecommendations(league LeagueData, strategy string, phase string, scheduleDifficulty string) []string {
	recs := []string{}

	// Strategy-specific recommendations
	switch strategy {
	case "Win Now":
		recs = append(recs, "Trade future assets for proven veterans")
		recs = append(recs, "Target players with favorable playoff schedules")
		if phase == "Mid Season" {
			recs = append(recs, "Make aggressive moves before trade deadline")
		}

	case "Contending":
		recs = append(recs, "Balance win-now moves with long-term value")
		recs = append(recs, "Target undervalued players with upside")

	case "Retool":
		recs = append(recs, "Move aging assets for picks and youth")
		recs = append(recs, "Target 2026 rookie draft capital")

	case "Rebuild":
		recs = append(recs, "Sell all veterans for maximum draft capital")
		recs = append(recs, "Accumulate 2026 and 2027 draft picks")
		recs = append(recs, "Target players under 24 years old")

	case "Build":
		recs = append(recs, "Continue accumulating young talent")
		recs = append(recs, "Trade win-now pieces for futures")
	}

	// Phase-specific recommendations
	switch phase {
	case "Offseason":
		recs = append(recs, "Evaluate post-season trades (lowest prices)")
		recs = append(recs, "Plan for upcoming rookie draft")

	case "Rookie Draft Season":
		if league.IsDynasty {
			recs = append(recs, "Execute your draft strategy")
			recs = append(recs, "Consider trading back for 2026 picks")
		}

	case "Preseason":
		recs = append(recs, "Make final roster tweaks")
		recs = append(recs, "Monitor depth charts and injuries")

	case "Early Season":
		recs = append(recs, "Avoid panic moves - small sample size")
		recs = append(recs, "Scout emerging waiver targets")

	case "Mid Season":
		recs = append(recs, "Evaluate if you're a buyer or seller")
		recs = append(recs, "Trade deadline is your last major opportunity")

	case "Playoff Push":
		recs = append(recs, "Optimize lineup for playoff schedule")
		recs = append(recs, "Consider high-ceiling plays over consistency")
	}

	// Position-specific recommendations
	if league.IsDynasty {
		pb := league.PositionalBreakdown
		total := pb.QB + pb.RB + pb.WR + pb.TE
		if total > 0 {
			rbPct := float64(pb.RB) / float64(total) * 100
			if rbPct < 20 {
				recs = append(recs, "RB depth is critical - prioritize RB acquisitions")
			}
		}
	}

	// Difficulty profile recommendations
	switch scheduleDifficulty {
	case "Hard":
		recs = append(recs, "Prioritize floor and lineup stability for difficult upcoming matchups")
		recs = append(recs, "Add depth at fragile positions before bye/injury pressure")
	case "Moderate":
		recs = append(recs, "Mix floor and upside plays while preserving roster flexibility")
	case "Easy":
		recs = append(recs, "Use softer matchup window to test upside players and stash value")
	}

	return recs
}

func buildScheduleDifficultyProfile(league LeagueData, strategy string) (string, string) {
	if !league.HasMatchups {
		return "Offseason", "No active weekly matchups; focus on roster construction and value accumulation."
	}

	score := 1 // baseline: moderate
	winProb := parseWinProbPct(league.WinProb)
	if winProb > 0 {
		if winProb < 45 {
			score += 2
		} else if winProb <= 55 {
			score += 1
		} else if winProb >= 65 {
			score -= 1
		}
	}

	for _, pr := range league.PowerRankings {
		if !pr.IsUserTeam {
			continue
		}
		if pr.StandingRank <= 3 {
			score += 1 // contenders get targeted and face stronger resistance
		}
		if pr.ValueRank > league.LeagueSize/2 {
			score += 1 // weaker rosters have less margin
		}
		break
	}

	if strategy == "Rebuild" || strategy == "Build" {
		score -= 1
	}

	if score <= 0 {
		return "Easy", "Current matchup profile is favorable; use this window to optimize upside."
	}
	if score <= 2 {
		return "Moderate", "Balanced matchup profile; stay flexible and avoid overreacting."
	}
	return "Hard", "Upcoming stretch projects tougher; protect floor and reinforce depth."
}

func parseWinProbPct(s string) int {
	if s == "" {
		return 0
	}
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return 0
	}
	raw := strings.TrimSuffix(fields[0], "%")
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return n
}
