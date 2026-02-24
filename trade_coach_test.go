package main

import (
	"os"
	"strings"
	"testing"
)

func TestGenerateLLMTradeMessageFallsBackWithoutKey(t *testing.T) {
	_ = os.Unsetenv("OPENROUTER_API_KEY")

	proposal := TradeProposal{
		TargetTeamName: "Other Team",
		YourOffer: []ProposalPlayer{
			{Name: "Player One"},
		},
		TheirReturn: []ProposalPlayer{
			{Name: "Player Two"},
		},
		Rationale: "Helps both teams",
		Fairness:  "Fair",
		RiskLevel: "Low",
	}

	msg := generateLLMTradeMessage(proposal, "WR", "RB")
	if !strings.Contains(msg, "I can send: Player One") {
		t.Fatalf("expected fallback trade text, got %q", msg)
	}
	if !strings.Contains(msg, "surplus at WR") {
		t.Fatalf("expected surplus context in fallback, got %q", msg)
	}
}
