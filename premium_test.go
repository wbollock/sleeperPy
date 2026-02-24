package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestIsPremiumUsername(t *testing.T) {
	t.Setenv("PREMIUM_USERS", "wboll, Alice ,Bob")

	if !isPremiumUsername("wboll") {
		t.Fatalf("expected wboll to be premium")
	}
	if !isPremiumUsername("ALICE") {
		t.Fatalf("expected alice (case-insensitive) to be premium")
	}
	if isPremiumUsername("charlie") {
		t.Fatalf("expected charlie to be non-premium")
	}
}

func TestBuildLeagueSummaryIncludesKeyFields(t *testing.T) {
	league := LeagueData{
		LeagueName:       "Test League",
		Scoring:          "PPR",
		LeagueSize:       12,
		IsDynasty:        true,
		TotalRosterValue: 12345,
		UserAvgAge:       25.5,
		DraftPicks: []DraftPick{
			{Year: 2026, Round: 1},
			{Year: 2026, Round: 2, OriginalName: "Other Team"},
		},
		Starters: []PlayerRow{
			{Pos: "QB", Name: "Player One", Tier: 1, DynastyValue: 8000, Age: 24},
		},
	}

	out := buildLeagueSummary(league)
	mustContain := []string{
		"Test League",
		"PPR",
		"Teams: 12",
		"Roster Value: 12345",
		"Average Age: 25.5",
		"Draft Picks: 2 total, 1 acquired",
		"Player One",
	}

	for _, term := range mustContain {
		if !strings.Contains(out, term) {
			t.Fatalf("expected summary to contain %q", term)
		}
	}
}

func TestCallOpenRouter(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "test-key")
	t.Setenv("OPENROUTER_MODEL", "test-model")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("unexpected authorization header: %s", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`))
	}))
	defer server.Close()

	origURL := openRouterBaseURL
	origClient := openRouterHTTPClient
	openRouterBaseURL = server.URL
	openRouterHTTPClient = server.Client()
	defer func() {
		openRouterBaseURL = origURL
		openRouterHTTPClient = origClient
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	out, err := callOpenRouter(ctx, []openRouterMessage{{Role: "user", Content: "hello"}}, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "ok" {
		t.Fatalf("expected ok, got %q", out)
	}
}

func TestCallOpenRouterMissingKey(t *testing.T) {
	_ = os.Unsetenv("OPENROUTER_API_KEY")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := callOpenRouter(ctx, []openRouterMessage{{Role: "user", Content: "hello"}}, 10)
	if err == nil {
		t.Fatalf("expected error when OPENROUTER_API_KEY is missing")
	}
}

func TestConsumeLLMBudget(t *testing.T) {
	t.Setenv("PREMIUM_LLM_DAILY_LIMIT", "3")

	llmUsageState.Lock()
	llmUsageState.data = map[string]llmUsageEntry{}
	llmUsageState.Unlock()

	ok, rem := consumeLLMBudget("wboll", "overview")
	if !ok || rem != 2 {
		t.Fatalf("expected first call allowed with 2 remaining, got allowed=%v rem=%d", ok, rem)
	}

	ok, rem = consumeLLMBudget("wboll", "all") // costs 2 units
	if !ok || rem != 0 {
		t.Fatalf("expected second call allowed with 0 remaining, got allowed=%v rem=%d", ok, rem)
	}

	ok, rem = consumeLLMBudget("wboll", "team")
	if ok || rem != 0 {
		t.Fatalf("expected third call blocked with 0 remaining, got allowed=%v rem=%d", ok, rem)
	}
}
