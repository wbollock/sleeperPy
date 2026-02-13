// ABOUTME: News signal compression for fantasy football players
// ABOUTME: Filters and ranks news to show only top 3 most critical items for user's players

package main

import (
	"sort"
	"strings"
	"time"
)

func compressPlayerNews(allNews []PlayerNews, userPlayerNames []string, isDynasty bool) CompressedNews {
	// 1. Filter: Only news for user's players (Q11)
	userNews := []PlayerNews{}
	userPlayerMap := make(map[string]bool)
	for _, name := range userPlayerNames {
		userPlayerMap[strings.ToLower(name)] = true
	}

	for _, news := range allNews {
		if userPlayerMap[strings.ToLower(news.PlayerName)] {
			userNews = append(userNews, news)
		}
	}

	// 2. Time window: Dynamic based on season state (Q10)
	timeWindow := "This Week"
	daysBack := 7

	if isOffseason() {
		timeWindow = "Last 3 Months"
		daysBack = 90
	}

	// Filter by time window
	recentNews := []PlayerNews{}
	cutoff := time.Now().AddDate(0, 0, -daysBack)
	for _, news := range userNews {
		if news.Timestamp.After(cutoff) {
			recentNews = append(recentNews, news)
		}
	}

	// 3. Score each news item by importance
	for i := range recentNews {
		recentNews[i].ImportanceScore = calculateImportanceScore(recentNews[i])
	}

	// 4. Sort by importance, take top 3 (Q12)
	sort.Slice(recentNews, func(i, j int) bool {
		return recentNews[i].ImportanceScore > recentNews[j].ImportanceScore
	})

	topHeadlines := recentNews
	if len(topHeadlines) > 3 {
		topHeadlines = topHeadlines[:3]
	}

	return CompressedNews{
		TimeWindow:   timeWindow,
		TopHeadlines: topHeadlines,
		TotalItems:   len(recentNews),
	}
}

// Importance scoring heuristic
func calculateImportanceScore(news PlayerNews) int {
	score := 0

	// Injury status = highest priority
	if news.InjuryStatus == "Out" || news.InjuryStatus == "IR" {
		score += 100
	} else if news.InjuryStatus == "Doubtful" {
		score += 80
	} else if news.InjuryStatus == "Questionable" {
		score += 50
	}

	// Starter vs bench (user's roster context)
	if news.IsStarter {
		score += 40
	}

	// Recency (newer = higher score)
	hoursSince := time.Since(news.Timestamp).Hours()
	if hoursSince < 24 {
		score += 30
	} else if hoursSince < 72 {
		score += 15
	}

	// Keywords in news text
	keywords := []string{
		"injury", "out", "ir", "doubtful",
		"trade", "traded", "acquired",
		"suspension", "suspended",
		"promoted", "starter", "rb1", "wr1",
		"breakout", "trending", "target share",
	}
	newsLower := strings.ToLower(news.NewsText)
	for _, kw := range keywords {
		if strings.Contains(newsLower, kw) {
			score += 10
			break // Only count once
		}
	}

	return score
}

// Helper: check if offseason
func isOffseason() bool {
	now := time.Now()
	month := now.Month()

	// Offseason: February - August
	// In-season: September - January
	return month >= time.February && month <= time.August
}

// Helper: extract player names from roster
func extractPlayerNames(starters []PlayerRow, bench []PlayerRow) []string {
	names := []string{}
	for _, p := range starters {
		names = append(names, p.Name)
	}
	for _, p := range bench {
		names = append(names, p.Name)
	}
	return names
}
