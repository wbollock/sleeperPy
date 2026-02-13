// League context cards - quick insights about team position
// Shows age rank, value rank, pick count, positional scarcity

package main

import (
	"fmt"
	"sort"
)

func buildContextCards(league LeagueData, userRosterValue int, userAvgAge float64) []ContextCard {
	cards := []ContextCard{}

	// 1. Value Rank Card (dynasty only)
	if league.IsDynasty && userRosterValue > 0 {
		valueRank := 0
		for _, pr := range league.PowerRankings {
			if pr.IsUserTeam {
				valueRank = pr.ValueRank
				break
			}
		}

		rankSuffix := getRankSuffix(valueRank)
		color := "#10b981" // green
		if valueRank > league.LeagueSize/2 {
			color = "#ef4444" // red - bottom half
		} else if valueRank <= 3 {
			color = "#3b82f6" // blue - top 3
		}

		cards = append(cards, ContextCard{
			Title: "Roster Value",
			Value: fmt.Sprintf("#%d%s of %d", valueRank, rankSuffix, league.LeagueSize),
			Icon:  "💎",
			Color: color,
		})
	}

	// 2. Age Rank Card (dynasty only)
	if league.IsDynasty && userAvgAge > 0 {
		ageRank := 0
		youngestAge := 100.0
		oldestAge := 0.0

		for i, team := range league.TeamAges {
			if team.AvgAge < youngestAge {
				youngestAge = team.AvgAge
			}
			if team.AvgAge > oldestAge {
				oldestAge = team.AvgAge
			}
			if team.IsUserTeam {
				ageRank = i + 1
			}
		}

		rankSuffix := getRankSuffix(ageRank)
		color := "#10b981" // green - younger
		if userAvgAge > 27 {
			color = "#f59e0b" // orange - aging
		}
		if userAvgAge > 29 {
			color = "#ef4444" // red - old
		}

		ageDescriptor := "Middle"
		if userAvgAge == youngestAge {
			ageDescriptor = "Youngest"
		} else if userAvgAge == oldestAge {
			ageDescriptor = "Oldest"
		} else if ageRank <= 3 {
			ageDescriptor = "Young"
		} else if ageRank > league.LeagueSize-3 {
			ageDescriptor = "Aging"
		}

		cards = append(cards, ContextCard{
			Title: "Roster Age",
			Value: fmt.Sprintf("%.1f yrs (#%d%s)", userAvgAge, ageRank, rankSuffix),
			Trend: ageDescriptor,
			Icon:  "📅",
			Color: color,
		})
	}

	// 3. Draft Capital Card (dynasty only)
	if league.IsDynasty && len(league.DraftPicks) > 0 {
		firstRoundPicks := 0
		totalPicks := len(league.DraftPicks)

		for _, pick := range league.DraftPicks {
			if pick.Round == 1 && pick.IsYours {
				firstRoundPicks++
			}
		}

		color := "#10b981" // green
		if totalPicks < 3 {
			color = "#ef4444" // red - low picks
		} else if firstRoundPicks > 1 {
			color = "#3b82f6" // blue - multiple 1sts
		}

		descriptor := fmt.Sprintf("%d total", totalPicks)
		if firstRoundPicks > 0 {
			descriptor = fmt.Sprintf("%d 1st%s", firstRoundPicks, pluralize(firstRoundPicks))
		}

		cards = append(cards, ContextCard{
			Title: "Draft Capital",
			Value: descriptor,
			Icon:  "🎯",
			Color: color,
		})
	}

	// 4. Positional Scarcity Card (dynasty only)
	if league.IsDynasty {
		pb := league.PositionalBreakdown
		total := pb.QB + pb.RB + pb.WR + pb.TE
		if total > 0 {
			// Find weakest position
			rbPct := float64(pb.RB) / float64(total) * 100
			wrPct := float64(pb.WR) / float64(total) * 100
			tePct := float64(pb.TE) / float64(total) * 100

			var weakPos string
			var weakPct float64
			color := "#10b981"

			if rbPct < 20 {
				weakPos = "RB"
				weakPct = rbPct
				color = "#ef4444"
			} else if wrPct < 25 {
				weakPos = "WR"
				weakPct = wrPct
				color = "#ef4444"
			} else if tePct < 10 {
				weakPos = "TE"
				weakPct = tePct
				color = "#ef4444"
			} else {
				weakPos = "Balanced"
				color = "#10b981"
			}

			value := weakPos
			if weakPos != "Balanced" {
				value = fmt.Sprintf("%s depth low (%.0f%%)", weakPos, weakPct)
			}

			cards = append(cards, ContextCard{
				Title: "Roster Balance",
				Value: value,
				Icon:  "⚖️",
				Color: color,
			})
		}
	}

	// 5. Playoff Status Card (in-season only)
	if league.HasMatchups {
		// Parse record from power rankings
		for _, pr := range league.PowerRankings {
			if pr.IsUserTeam {
				totalGames := pr.Wins + pr.Losses
				if totalGames > 0 {
					winPct := float64(pr.Wins) / float64(totalGames)

					status := "In Hunt"
					color := "#f59e0b" // orange

					if winPct >= 0.6 {
						status = "Playoff Bound"
						color = "#10b981" // green
					} else if winPct < 0.4 {
						status = "Rebuild Mode"
						color = "#ef4444" // red
					}

					cards = append(cards, ContextCard{
						Title: "Season Status",
						Value: fmt.Sprintf("%d-%d", pr.Wins, pr.Losses),
						Trend: status,
						Icon:  "🏆",
						Color: color,
					})
				}
				break
			}
		}
	}

	return cards
}

func getRankSuffix(rank int) string {
	if rank == 1 {
		return "st"
	} else if rank == 2 {
		return "nd"
	} else if rank == 3 {
		return "rd"
	}
	return "th"
}

func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

// Sort TeamAges by age for ranking
func sortTeamsByAge(teams []TeamAgeData) []TeamAgeData {
	sorted := make([]TeamAgeData, len(teams))
	copy(sorted, teams)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].AvgAge < sorted[j].AvgAge
	})
	return sorted
}
