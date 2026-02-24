package main

import (
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"
)

var buildDashboardPageForEmail = buildDashboardPage
var sendWeeklyEmailFunc = sendWeeklyEmail

func weeklyEmailHandler(w http.ResponseWriter, r *http.Request) {
	trackPageView(r.URL.Path)
	username := strings.TrimSpace(r.URL.Query().Get("user"))
	if username == "" {
		http.Error(w, "user is required", http.StatusBadRequest)
		return
	}

	page, err := buildDashboardPageForEmail(username)
	if err != nil {
		log.Printf("weekly email build failed for user %q: %v", username, err)
		http.Error(w, "failed to build summary", http.StatusBadGateway)
		return
	}

	body := buildWeeklyEmailSummary(page)

	// Preview by default.
	if strings.TrimSpace(r.URL.Query().Get("send")) != "1" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(body))
		return
	}

	to := strings.TrimSpace(r.URL.Query().Get("to"))
	if to == "" {
		to = strings.TrimSpace(os.Getenv("EMAIL_SUMMARY_TO"))
	}
	if to == "" {
		http.Error(w, "to is required when send=1 (or set EMAIL_SUMMARY_TO)", http.StatusBadRequest)
		return
	}

	subject := fmt.Sprintf("SleeperPy Weekly Summary - %s (%s)", username, time.Now().Format("2006-01-02"))
	if err := sendWeeklyEmailFunc(to, subject, body); err != nil {
		log.Printf("weekly email send failed for user %q to %q: %v", username, to, err)
		http.Error(w, "email send failed", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("ok: weekly summary email sent"))
}

func buildWeeklyEmailSummary(page *DashboardPage) string {
	var sb strings.Builder
	now := time.Now().Format("Monday, Jan 2, 2006")
	sb.WriteString(fmt.Sprintf("SleeperPy Weekly Summary\nDate: %s\nUser: %s\n\n", now, page.Username))
	sb.WriteString(fmt.Sprintf("Leagues: %d total (%d dynasty, %d redraft)\n\n", page.TotalLeagues, page.DynastyCount, page.RedraftCount))

	totalActions := 0
	for _, l := range page.LeagueSummaries {
		totalActions += l.ActionCount
	}
	sb.WriteString(fmt.Sprintf("Action Load: %d pending actions across leagues\n\n", totalActions))

	for _, l := range page.LeagueSummaries {
		sb.WriteString(fmt.Sprintf("- %s", l.LeagueName))
		if l.Season != "" {
			sb.WriteString(fmt.Sprintf(" (%s)", l.Season))
		}
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("  Format: %s | %s | %d teams\n", ternary(l.IsDynasty, "Dynasty", "Redraft"), l.Scoring, l.LeagueSize))
		if l.Record != "" {
			sb.WriteString(fmt.Sprintf("  Record: %s\n", l.Record))
		}
		if l.PlayoffStatus != "" {
			sb.WriteString(fmt.Sprintf("  Status: %s\n", l.PlayoffStatus))
		}
		if l.TotalRosterValue > 0 {
			sb.WriteString(fmt.Sprintf("  Roster Value: %d (rank #%d/%d)\n", l.TotalRosterValue, l.ValueRank, l.LeagueSize))
		}
		if l.ValueTrend != "" {
			sb.WriteString(fmt.Sprintf("  Value Trend: %s\n", l.ValueTrend))
		}
		if l.DraftPicksSummary != "" {
			sb.WriteString(fmt.Sprintf("  Draft Capital: %s\n", l.DraftPicksSummary))
		}
		if l.ActionCount > 0 {
			sb.WriteString(fmt.Sprintf("  Immediate Actions: %d\n", l.ActionCount))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("Top Risks:\n")
	riskLines := deriveSummaryRisks(page)
	if len(riskLines) == 0 {
		sb.WriteString("- No major cross-league risks detected this week.\n")
	} else {
		for _, line := range riskLines {
			sb.WriteString("- " + line + "\n")
		}
	}

	return sb.String()
}

func deriveSummaryRisks(page *DashboardPage) []string {
	risks := []string{}
	for _, l := range page.LeagueSummaries {
		if strings.Contains(strings.ToLower(l.PlayoffStatus), "eliminated") {
			risks = append(risks, fmt.Sprintf("%s: eliminated from contention", l.LeagueName))
		}
		if strings.Contains(l.ValueTrend, "↘") {
			risks = append(risks, fmt.Sprintf("%s: negative roster value trend (%s)", l.LeagueName, l.ValueTrend))
		}
		if l.ActionCount >= 4 {
			risks = append(risks, fmt.Sprintf("%s: high pending action load (%d)", l.LeagueName, l.ActionCount))
		}
	}
	if len(risks) > 5 {
		risks = risks[:5]
	}
	return risks
}

func sendWeeklyEmail(to, subject, body string) error {
	host := strings.TrimSpace(os.Getenv("SMTP_HOST"))
	port := strings.TrimSpace(os.Getenv("SMTP_PORT"))
	user := strings.TrimSpace(os.Getenv("SMTP_USER"))
	pass := strings.TrimSpace(os.Getenv("SMTP_PASS"))
	from := strings.TrimSpace(os.Getenv("SMTP_FROM"))
	if from == "" {
		from = user
	}

	if host == "" || port == "" || user == "" || pass == "" || from == "" {
		return fmt.Errorf("missing smtp configuration (SMTP_HOST/SMTP_PORT/SMTP_USER/SMTP_PASS/SMTP_FROM)")
	}
	if _, err := strconv.Atoi(port); err != nil {
		return fmt.Errorf("invalid SMTP_PORT")
	}

	addr := host + ":" + port
	auth := smtp.PlainAuth("", user, pass, host)
	msg := []byte(
		"From: " + from + "\r\n" +
			"To: " + to + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"Content-Type: text/plain; charset=utf-8\r\n" +
			"\r\n" + body + "\r\n",
	)
	return smtp.SendMail(addr, auth, from, []string{to}, msg)
}

func ternary(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}
