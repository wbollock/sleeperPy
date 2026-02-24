package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const openRouterDefaultModel = "openai/gpt-4o-mini"

var openRouterBaseURL = "https://openrouter.ai/api/v1/chat/completions"

var openRouterHTTPClient = &http.Client{
	Timeout: 20 * time.Second,
}

type openRouterMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openRouterRequest struct {
	Model       string              `json:"model"`
	Messages    []openRouterMessage `json:"messages"`
	Temperature float64             `json:"temperature,omitempty"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
}

type openRouterResponse struct {
	Choices []struct {
		Message openRouterMessage `json:"message"`
	} `json:"choices"`
}

var llmUsageState = struct {
	sync.Mutex
	data map[string]llmUsageEntry
}{
	data: make(map[string]llmUsageEntry),
}

type llmUsageEntry struct {
	Date  string
	Count int
}

func isPremiumUsername(username string) bool {
	allowlist := strings.TrimSpace(os.Getenv("PREMIUM_USERS"))
	if allowlist == "" || username == "" {
		return false
	}
	normalized := strings.ToLower(strings.TrimSpace(username))
	for _, entry := range strings.Split(allowlist, ",") {
		if strings.ToLower(strings.TrimSpace(entry)) == normalized {
			return true
		}
	}
	return false
}

func hasOpenRouterKey() bool {
	return strings.TrimSpace(os.Getenv("OPENROUTER_API_KEY")) != ""
}

func dailyLLMLimit() int {
	raw := strings.TrimSpace(os.Getenv("PREMIUM_LLM_DAILY_LIMIT"))
	if raw == "" {
		return 12
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 12
	}
	return n
}

func llmUnitsForMode(mode string) int {
	switch mode {
	case "all", "1":
		return 2
	default:
		return 1
	}
}

// consumeLLMBudget returns allowed, remaining quota.
func consumeLLMBudget(username, llmMode string) (bool, int) {
	if strings.TrimSpace(username) == "" || strings.TrimSpace(llmMode) == "" {
		return true, dailyLLMLimit()
	}

	limit := dailyLLMLimit()
	units := llmUnitsForMode(llmMode)
	today := time.Now().Format("2006-01-02")
	key := strings.ToLower(strings.TrimSpace(username))

	llmUsageState.Lock()
	defer llmUsageState.Unlock()

	entry := llmUsageState.data[key]
	if entry.Date != today {
		entry = llmUsageEntry{Date: today, Count: 0}
	}

	if entry.Count+units > limit {
		remaining := limit - entry.Count
		if remaining < 0 {
			remaining = 0
		}
		llmUsageState.data[key] = entry
		return false, remaining
	}

	entry.Count += units
	llmUsageState.data[key] = entry
	return true, limit - entry.Count
}

func openRouterModel() string {
	if model := strings.TrimSpace(os.Getenv("OPENROUTER_MODEL")); model != "" {
		return model
	}
	return openRouterDefaultModel
}

func buildLeagueSummary(league LeagueData) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("League: %s\n", league.LeagueName))
	sb.WriteString(fmt.Sprintf("Scoring: %s\n", league.Scoring))
	sb.WriteString(fmt.Sprintf("Teams: %d\n", league.LeagueSize))
	if league.IsDynasty {
		sb.WriteString("Format: Dynasty\n")
	} else {
		sb.WriteString("Format: Redraft\n")
	}
	if league.TotalRosterValue > 0 {
		sb.WriteString(fmt.Sprintf("Roster Value: %d\n", league.TotalRosterValue))
	}
	if league.UserAvgAge > 0 {
		sb.WriteString(fmt.Sprintf("Average Age: %.1f\n", league.UserAvgAge))
	}

	if len(league.Starters) > 0 {
		sb.WriteString("Starters:\n")
		for i, p := range league.Starters {
			if i >= 10 {
				break
			}
			sb.WriteString(fmt.Sprintf("- %s %s (Tier %v", p.Pos, p.Name, p.Tier))
			if p.DynastyValue > 0 {
				sb.WriteString(fmt.Sprintf(", Value %d", p.DynastyValue))
			}
			if p.Age > 0 {
				sb.WriteString(fmt.Sprintf(", Age %d", p.Age))
			}
			sb.WriteString(")\n")
		}
	}

	if league.IsDynasty && len(league.DraftPicks) > 0 {
		acquired := 0
		for _, pick := range league.DraftPicks {
			if pick.OriginalName != "" {
				acquired++
			}
		}
		sb.WriteString(fmt.Sprintf("Draft Picks: %d total, %d acquired\n", len(league.DraftPicks), acquired))
	}

	if len(league.TradeTargets) > 0 {
		sb.WriteString("Trade Targets:\n")
		for i, t := range league.TradeTargets {
			if i >= 3 {
				break
			}
			sb.WriteString(fmt.Sprintf("- %s (%s)\n", t.TeamName, t.Reason))
		}
	}

	if len(league.RecentTransactions) > 0 {
		sb.WriteString(fmt.Sprintf("Recent Transactions: %d\n", len(league.RecentTransactions)))
	}

	priorityMatrix := buildPriorityMatrix(league)
	if priorityMatrix != "" {
		sb.WriteString("Priority Matrix:\n")
		sb.WriteString(priorityMatrix)
	}

	riskWatchlist := buildRiskWatchlist(league)
	if riskWatchlist != "" {
		sb.WriteString("Risk Watchlist:\n")
		sb.WriteString(riskWatchlist)
	}

	return sb.String()
}

func buildOverviewSummary(leagues []LeagueData) string {
	var sb strings.Builder

	sb.WriteString("All Leagues Summary:\n")
	totalRiskFlags := 0
	for _, league := range leagues {
		sb.WriteString(fmt.Sprintf("\n- %s (%s, %d teams)\n", league.LeagueName, league.Scoring, league.LeagueSize))
		if league.TotalRosterValue > 0 {
			sb.WriteString(fmt.Sprintf("  Roster Value: %d\n", league.TotalRosterValue))
		}
		if league.UserAvgAge > 0 {
			sb.WriteString(fmt.Sprintf("  Avg Age: %.1f\n", league.UserAvgAge))
		}
		if league.IsDynasty && len(league.DraftPicks) > 0 {
			sb.WriteString(fmt.Sprintf("  Draft Picks: %d\n", len(league.DraftPicks)))
		}
		if len(league.TradeTargets) > 0 {
			sb.WriteString(fmt.Sprintf("  Trade Targets: %d\n", len(league.TradeTargets)))
		}
		riskCount := countLeagueRisks(league)
		if riskCount > 0 {
			totalRiskFlags += riskCount
			sb.WriteString(fmt.Sprintf("  Risk Flags: %d\n", riskCount))
		}
	}
	sb.WriteString(fmt.Sprintf("\nAggregate Risk Flags: %d\n", totalRiskFlags))

	return sb.String()
}

func callOpenRouter(ctx context.Context, messages []openRouterMessage, maxTokens int) (string, error) {
	apiKey := strings.TrimSpace(os.Getenv("OPENROUTER_API_KEY"))
	if apiKey == "" {
		return "", fmt.Errorf("OPENROUTER_API_KEY is not set")
	}

	reqBody := openRouterRequest{
		Model:       openRouterModel(),
		Messages:    messages,
		Temperature: 0.4,
		MaxTokens:   maxTokens,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openRouterBaseURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	if ref := strings.TrimSpace(os.Getenv("OPENROUTER_REFERRER")); ref != "" {
		req.Header.Set("HTTP-Referer", ref)
	}
	if title := strings.TrimSpace(os.Getenv("OPENROUTER_TITLE")); title != "" {
		req.Header.Set("X-Title", title)
	}

	resp, err := openRouterHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("openrouter error: status %d", resp.StatusCode)
	}

	var decoded openRouterResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return "", err
	}
	if len(decoded.Choices) == 0 {
		return "", fmt.Errorf("openrouter error: empty response")
	}
	return strings.TrimSpace(decoded.Choices[0].Message.Content), nil
}

func generateTeamTalk(ctx context.Context, league LeagueData) (string, error) {
	summary := buildLeagueSummary(league)
	messages := []openRouterMessage{
		{
			Role:    "system",
			Content: "You are a fantasy football strategist. Keep responses concise, practical, and specific to the team data. Use bullet points and clear action items.",
		},
		{
			Role: "user",
			Content: "Provide a short team talk with:\n" +
				"- 3 priority actions\n" +
				"- Priority Matrix (Needs vs Surpluses)\n" +
				"- Risk Watchlist (injury/age)\n" +
				"- 2 watchlist notes\n" +
				"- 1 trade suggestion (if any)\n\n" +
				"Team data:\n" + summary,
		},
	}
	return callOpenRouter(ctx, messages, 400)
}

func generateOverview(ctx context.Context, leagues []LeagueData) (string, error) {
	summary := buildOverviewSummary(leagues)
	messages := []openRouterMessage{
		{
			Role:    "system",
			Content: "You are a fantasy football analyst. Summarize key trends across leagues, keep it concise and actionable. Use bullet points.",
		},
		{
			Role: "user",
			Content: "Provide an overview across all leagues with:\n" +
				"- Top 3 priorities across leagues\n" +
				"- Cross-League Priority Matrix themes (common needs/surpluses)\n" +
				"- Cross-League Risk Watchlist (injury/age)\n" +
				"- 2 common risks\n" +
				"- 2 quick wins\n\n" +
				"Data:\n" + summary,
		},
	}
	return callOpenRouter(ctx, messages, 450)
}

func applyTeamTalks(ctx context.Context, leagues []LeagueData) []LeagueData {
	for i := range leagues {
		leagueCtx, cancel := context.WithTimeout(ctx, 18*time.Second)
		talk, err := generateTeamTalk(leagueCtx, leagues[i])
		cancel()
		if err != nil {
			debugLog("[DEBUG] OpenRouter team talk error for %s: %v", leagues[i].LeagueName, err)
			continue
		}
		leagues[i].PremiumTeamTalk = talk
	}
	return leagues
}

func buildPriorityMatrix(league LeagueData) string {
	pb := league.PositionalBreakdown
	total := pb.QB + pb.RB + pb.WR + pb.TE
	if total == 0 {
		return ""
	}

	type posPct struct {
		pos string
		pct float64
	}
	positions := []posPct{
		{pos: "QB", pct: float64(pb.QB) / float64(total) * 100},
		{pos: "RB", pct: float64(pb.RB) / float64(total) * 100},
		{pos: "WR", pct: float64(pb.WR) / float64(total) * 100},
		{pos: "TE", pct: float64(pb.TE) / float64(total) * 100},
	}

	needs := []string{}
	surpluses := []string{}
	for _, p := range positions {
		if p.pct < 15 {
			needs = append(needs, fmt.Sprintf("%s (%.0f%%)", p.pos, p.pct))
		}
		if p.pct > 30 {
			surpluses = append(surpluses, fmt.Sprintf("%s (%.0f%%)", p.pos, p.pct))
		}
	}
	if len(needs) == 0 {
		needs = append(needs, "None")
	}
	if len(surpluses) == 0 {
		surpluses = append(surpluses, "None")
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("- Needs: %s\n", strings.Join(needs, ", ")))
	sb.WriteString(fmt.Sprintf("- Surpluses: %s\n", strings.Join(surpluses, ", ")))
	return sb.String()
}

func buildRiskWatchlist(league LeagueData) string {
	risks := []string{}
	seen := map[string]bool{}

	for _, p := range league.PlayerNewsFeed {
		if p.InjuryStatus == "" {
			continue
		}
		status := strings.ToLower(p.InjuryStatus)
		if status == "out" || status == "ir" || status == "doubtful" || status == "questionable" {
			key := "injury:" + p.PlayerName
			if seen[key] {
				continue
			}
			seen[key] = true
			risks = append(risks, fmt.Sprintf("- Injury: %s (%s)", p.PlayerName, p.InjuryStatus))
			if len(risks) >= 3 {
				break
			}
		}
	}

	for _, p := range league.AgingPlayers {
		if p.Age <= 0 {
			continue
		}
		key := "age:" + p.Name
		if seen[key] {
			continue
		}
		seen[key] = true
		risks = append(risks, fmt.Sprintf("- Age: %s (%s age %d)", p.Name, p.Pos, p.Age))
		if len(risks) >= 6 {
			break
		}
	}

	if len(risks) == 0 {
		return ""
	}
	return strings.Join(risks, "\n") + "\n"
}

func countLeagueRisks(league LeagueData) int {
	count := 0
	seen := map[string]bool{}
	for _, p := range league.PlayerNewsFeed {
		status := strings.ToLower(p.InjuryStatus)
		if status == "out" || status == "ir" || status == "doubtful" || status == "questionable" {
			key := "injury:" + p.PlayerName
			if !seen[key] {
				seen[key] = true
				count++
			}
		}
	}
	for _, p := range league.AgingPlayers {
		if p.Age > 0 {
			key := "age:" + p.Name
			if !seen[key] {
				seen[key] = true
				count++
			}
		}
	}
	return count
}
