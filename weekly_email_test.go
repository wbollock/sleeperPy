package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBuildWeeklyEmailSummary(t *testing.T) {
	page := &DashboardPage{
		Username:     "testuser",
		TotalLeagues: 2,
		DynastyCount: 1,
		RedraftCount: 1,
		LeagueSummaries: []LeagueSummary{
			{
				LeagueName:        "Dynasty A",
				Season:            "2025",
				Scoring:           "PPR",
				IsDynasty:         true,
				LeagueSize:        12,
				Record:            "8-5",
				PlayoffStatus:     "In Hunt",
				TotalRosterValue:  8450,
				ValueRank:         3,
				ValueTrend:        "↗ +5%",
				DraftPicksSummary: "2026 1st, 2027 1st",
				ActionCount:       2,
			},
		},
	}

	body := buildWeeklyEmailSummary(page)
	if !strings.Contains(body, "SleeperPy Weekly Summary") {
		t.Fatalf("missing summary header: %s", body)
	}
	if !strings.Contains(body, "User: testuser") {
		t.Fatalf("missing user line: %s", body)
	}
	if !strings.Contains(body, "Dynasty A (2025)") {
		t.Fatalf("missing league section: %s", body)
	}
	if !strings.Contains(body, "Immediate Actions: 2") {
		t.Fatalf("missing action count: %s", body)
	}
}

func TestDeriveSummaryRisks(t *testing.T) {
	page := &DashboardPage{
		LeagueSummaries: []LeagueSummary{
			{LeagueName: "L1", PlayoffStatus: "Eliminated", ValueTrend: "↘ -3%", ActionCount: 5},
			{LeagueName: "L2", PlayoffStatus: "Eliminated", ValueTrend: "↘ -2%", ActionCount: 4},
			{LeagueName: "L3", PlayoffStatus: "Eliminated", ValueTrend: "↘ -1%", ActionCount: 6},
		},
	}
	risks := deriveSummaryRisks(page)
	if len(risks) != 5 {
		t.Fatalf("expected capped risk list of 5, got %d (%v)", len(risks), risks)
	}
}

func TestWeeklyEmailHandlerRequiresUser(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/weekly-email", nil)
	rr := httptest.NewRecorder()
	weeklyEmailHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "user is required") {
		t.Fatalf("unexpected response body: %s", rr.Body.String())
	}
}

func TestWeeklyEmailHandlerPreview(t *testing.T) {
	origBuilder := buildDashboardPageForEmail
	defer func() { buildDashboardPageForEmail = origBuilder }()

	buildDashboardPageForEmail = func(username string) (*DashboardPage, error) {
		return &DashboardPage{
			Username:     username,
			TotalLeagues: 1,
			DynastyCount: 1,
			LeagueSummaries: []LeagueSummary{
				{
					LeagueName:  "League",
					Season:      "2025",
					Scoring:     "PPR",
					IsDynasty:   true,
					LeagueSize:  12,
					ActionCount: 1,
				},
			},
		}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/weekly-email?user=testuser", nil)
	rr := httptest.NewRecorder()
	weeklyEmailHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "User: testuser") {
		t.Fatalf("missing preview summary in response: %s", rr.Body.String())
	}
}

func TestWeeklyEmailHandlerBuildFailure(t *testing.T) {
	origBuilder := buildDashboardPageForEmail
	defer func() { buildDashboardPageForEmail = origBuilder }()

	buildDashboardPageForEmail = func(username string) (*DashboardPage, error) {
		return nil, errors.New("upstream failed")
	}

	req := httptest.NewRequest(http.MethodGet, "/weekly-email?user=testuser", nil)
	rr := httptest.NewRecorder()
	weeklyEmailHandler(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", rr.Code)
	}
	if strings.Contains(rr.Body.String(), "upstream failed") {
		t.Fatalf("should not leak raw upstream error: %s", rr.Body.String())
	}
}

func TestWeeklyEmailHandlerSendRequiresRecipient(t *testing.T) {
	origBuilder := buildDashboardPageForEmail
	defer func() { buildDashboardPageForEmail = origBuilder }()
	buildDashboardPageForEmail = func(username string) (*DashboardPage, error) {
		return &DashboardPage{Username: username}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/weekly-email?user=testuser&send=1", nil)
	rr := httptest.NewRecorder()
	weeklyEmailHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestWeeklyEmailHandlerSendUsesEnvRecipient(t *testing.T) {
	origBuilder := buildDashboardPageForEmail
	origSender := sendWeeklyEmailFunc
	defer func() {
		buildDashboardPageForEmail = origBuilder
		sendWeeklyEmailFunc = origSender
	}()

	buildDashboardPageForEmail = func(username string) (*DashboardPage, error) {
		return &DashboardPage{
			Username:     username,
			TotalLeagues: 1,
		}, nil
	}

	called := false
	sendWeeklyEmailFunc = func(to, subject, body string) error {
		called = true
		if to != "test@example.com" {
			t.Fatalf("expected env recipient, got %q", to)
		}
		if !strings.Contains(subject, "testuser") {
			t.Fatalf("expected username in subject, got %q", subject)
		}
		if !strings.Contains(body, "User: testuser") {
			t.Fatalf("expected summary body, got %q", body)
		}
		return nil
	}

	t.Setenv("EMAIL_SUMMARY_TO", "test@example.com")
	req := httptest.NewRequest(http.MethodGet, "/weekly-email?user=testuser&send=1", nil)
	rr := httptest.NewRecorder()
	weeklyEmailHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if !called {
		t.Fatal("expected sendWeeklyEmail to be called")
	}
}

func TestSendWeeklyEmailRequiresSMTPConfig(t *testing.T) {
	t.Setenv("SMTP_HOST", "")
	t.Setenv("SMTP_PORT", "")
	t.Setenv("SMTP_USER", "")
	t.Setenv("SMTP_PASS", "")
	t.Setenv("SMTP_FROM", "")

	err := sendWeeklyEmail("to@example.com", "subject", "body")
	if err == nil {
		t.Fatal("expected missing smtp config error")
	}
}
