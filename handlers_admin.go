// ABOUTME: Admin dashboard handlers for SleeperPy
// ABOUTME: Provides real-time metrics, usage statistics, and operational visibility

package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// Admin metrics tracking (in-memory)
var adminMetrics = struct {
	sync.RWMutex
	startTime      time.Time
	pageViews      map[string]int64 // path -> count
	userAgents     map[string]int64 // user agent -> count
	activeUsers    int64             // users in last 24h
	dynastyPercent float64           // % of leagues that are dynasty
	errorLog       []ErrorLog
}{
	startTime:  time.Now(),
	pageViews:  make(map[string]int64),
	userAgents: make(map[string]int64),
	errorLog:   make([]ErrorLog, 0, 100),
}

type ErrorLog struct {
	Timestamp time.Time
	Path      string
	Error     string
	UserAgent string
}

type AdminData struct {
	// Server Info
	Uptime          string
	GoVersion       string
	Goroutines      int
	MemoryUsage     string
	ServerTime      string

	// Metrics from Prometheus
	TotalVisitors  float64
	TotalLookups   float64
	TotalLeagues   float64
	TotalTeams     float64
	TotalErrors    float64

	// Rate metrics (calculated)
	LookupsPerHour float64
	LeaguesPerHour float64

	// Additional metrics
	PageViews      map[string]int64
	TopUserAgents  []UACount
	RecentErrors   []ErrorLog
	DynastyPercent float64
}

type UACount struct {
	UserAgent string
	Count     int64
}

type PublicStatusData struct {
	Uptime         string
	UpdatedAt      string
	ServiceStatus  string
	ErrorRate      string
	TotalLookups   float64
	TotalLeagues   float64
	TotalErrors    float64
	LookupsPerHour float64
	LeaguesPerHour float64
}

func publicStatusHandler(w http.ResponseWriter, r *http.Request) {
	lookups := getMetricValue(totalLookups)
	leagues := getMetricValue(totalLeagues)
	errors := getMetricValue(totalErrors)

	uptime := time.Since(adminMetrics.startTime)
	uptimeHours := uptime.Hours()
	lookupsPerHour := 0.0
	leaguesPerHour := 0.0
	if uptimeHours > 0 {
		lookupsPerHour = lookups / uptimeHours
		leaguesPerHour = leagues / uptimeHours
	}

	denom := lookups
	if denom < 1 {
		denom = 1
	}
	errorRate := (errors / denom) * 100.0
	serviceStatus := "Operational"
	if lookups == 0 {
		serviceStatus = "Initializing"
	} else if errorRate >= 5.0 {
		serviceStatus = "Degraded"
	}

	data := PublicStatusData{
		Uptime:         uptime.Round(time.Second).String(),
		UpdatedAt:      time.Now().Format("2006-01-02 15:04:05 MST"),
		ServiceStatus:  serviceStatus,
		ErrorRate:      fmt.Sprintf("%.1f%%", errorRate),
		TotalLookups:   lookups,
		TotalLeagues:   leagues,
		TotalErrors:    errors,
		LookupsPerHour: lookupsPerHour,
		LeaguesPerHour: leaguesPerHour,
	}

	tmpl := template.Must(template.ParseFiles("templates/status.html"))
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Error rendering status page", http.StatusInternalServerError)
		log.Printf("Error rendering status.html: %v", err)
	}
}

func adminHandler(w http.ResponseWriter, r *http.Request) {
	if !adminAccessAllowed(r) {
		w.WriteHeader(http.StatusUnauthorized)
		tmpl := template.Must(template.ParseFiles("templates/admin_unauthorized.html"))
		_ = tmpl.Execute(w, map[string]string{
			"ServerTime": time.Now().Format("2006-01-02 15:04:05 MST"),
		})
		return
	}

	data := AdminData{
		Uptime:        time.Since(adminMetrics.startTime).Round(time.Second).String(),
		GoVersion:     runtime.Version(),
		Goroutines:    runtime.NumGoroutine(),
		MemoryUsage:   getMemoryUsage(),
		ServerTime:    time.Now().Format("2006-01-02 15:04:05 MST"),
		TotalVisitors: getMetricValue(totalVisitors),
		TotalLookups:  getMetricValue(totalLookups),
		TotalLeagues:  getMetricValue(totalLeagues),
		TotalTeams:    getMetricValue(totalTeams),
		TotalErrors:   getMetricValue(totalErrors),
	}

	// Calculate rate metrics
	uptimeHours := time.Since(adminMetrics.startTime).Hours()
	if uptimeHours > 0 {
		data.LookupsPerHour = data.TotalLookups / uptimeHours
		data.LeaguesPerHour = data.TotalLeagues / uptimeHours
	}

	// Add in-memory metrics
	adminMetrics.RLock()
	data.PageViews = make(map[string]int64)
	for k, v := range adminMetrics.pageViews {
		data.PageViews[k] = v
	}
	data.DynastyPercent = adminMetrics.dynastyPercent

	// Top user agents
	type kv struct {
		Key   string
		Value int64
	}
	var uaSlice []kv
	for k, v := range adminMetrics.userAgents {
		uaSlice = append(uaSlice, kv{k, v})
	}
	adminMetrics.RUnlock()

	// Sort and get top 10
	if len(uaSlice) > 10 {
		// Simple bubble sort for top 10
		for i := 0; i < 10 && i < len(uaSlice); i++ {
			for j := i + 1; j < len(uaSlice); j++ {
				if uaSlice[j].Value > uaSlice[i].Value {
					uaSlice[i], uaSlice[j] = uaSlice[j], uaSlice[i]
				}
			}
		}
		uaSlice = uaSlice[:10]
	}

	for _, ua := range uaSlice {
		data.TopUserAgents = append(data.TopUserAgents, UACount{
			UserAgent: ua.Key,
			Count:     ua.Value,
		})
	}

	// Recent errors (last 20)
	adminMetrics.RLock()
	if len(adminMetrics.errorLog) > 20 {
		data.RecentErrors = adminMetrics.errorLog[len(adminMetrics.errorLog)-20:]
	} else {
		data.RecentErrors = adminMetrics.errorLog
	}
	adminMetrics.RUnlock()

	// Render template
	tmpl := template.Must(template.ParseFiles("templates/admin.html"))
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Error rendering admin dashboard", http.StatusInternalServerError)
		log.Printf("Error rendering admin.html: %v", err)
	}
}

// Admin API endpoint for JSON metrics
func adminAPIHandler(w http.ResponseWriter, r *http.Request) {
	if !adminAccessAllowed(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	data := map[string]interface{}{
		"uptime":          time.Since(adminMetrics.startTime).Seconds(),
		"total_visitors":  getMetricValue(totalVisitors),
		"total_lookups":   getMetricValue(totalLookups),
		"total_leagues":   getMetricValue(totalLeagues),
		"total_teams":     getMetricValue(totalTeams),
		"total_errors":    getMetricValue(totalErrors),
		"goroutines":      runtime.NumGoroutine(),
		"dynasty_percent": adminMetrics.dynastyPercent,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// Helper functions

func adminAccessAllowed(r *http.Request) bool {
	adminKey := os.Getenv("ADMIN_KEY")
	if adminKey == "" {
		return false
	}
	if adminKey == "changeme" && os.Getenv("ADMIN_ALLOW_INSECURE") != "1" {
		return false
	}

	if !adminIPAllowed(r.RemoteAddr) {
		return false
	}

	providedKey, fromQuery := adminKeyFromRequest(r)
	if providedKey == "" {
		return false
	}
	if fromQuery && !allowQueryAuth(r.RemoteAddr) {
		return false
	}

	return providedKey == adminKey
}

func adminKeyFromRequest(r *http.Request) (string, bool) {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(auth, "Bearer ")), false
	}
	if key := strings.TrimSpace(r.Header.Get("X-Admin-Key")); key != "" {
		return key, false
	}
	if key := strings.TrimSpace(r.URL.Query().Get("secret")); key != "" {
		return key, true
	}
	return "", false
}

func allowQueryAuth(remoteAddr string) bool {
	if os.Getenv("ADMIN_ALLOW_QUERY") == "1" {
		return true
	}
	return isLoopbackAddr(remoteAddr)
}

func adminIPAllowed(remoteAddr string) bool {
	allowed := strings.TrimSpace(os.Getenv("ADMIN_ALLOWED_IPS"))
	if allowed == "" {
		return true
	}
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	ip := net.ParseIP(strings.TrimSpace(host))
	if ip == nil {
		return false
	}
	for _, entry := range strings.Split(allowed, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if ip.Equal(net.ParseIP(entry)) {
			return true
		}
	}
	return false
}

func isLoopbackAddr(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	ip := net.ParseIP(strings.TrimSpace(host))
	if ip == nil {
		return false
	}
	return ip.IsLoopback()
}

func getMetricValue(counter prometheus.Counter) float64 {
	metric := &dto.Metric{}
	if err := counter.Write(metric); err != nil {
		return 0
	}
	return metric.Counter.GetValue()
}

func getMemoryUsage() string {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return fmt.Sprintf("%.2f MB", float64(m.Alloc)/1024/1024)
}

// Track page views for admin dashboard
func trackPageView(path string) {
	adminMetrics.Lock()
	adminMetrics.pageViews[path]++
	adminMetrics.Unlock()
}

// Track user agent for admin dashboard
func trackUserAgent(ua string) {
	if ua == "" {
		return
	}

	// Simplify user agent (just browser/OS)
	simplified := simplifyUserAgent(ua)

	adminMetrics.Lock()
	adminMetrics.userAgents[simplified]++
	adminMetrics.Unlock()
}

// Log error for admin dashboard
func logAdminError(path string, err error, ua string) {
	adminMetrics.Lock()
	defer adminMetrics.Unlock()

	errorLog := ErrorLog{
		Timestamp: time.Now(),
		Path:      path,
		Error:     err.Error(),
		UserAgent: simplifyUserAgent(ua),
	}

	adminMetrics.errorLog = append(adminMetrics.errorLog, errorLog)

	// Keep only last 100 errors
	if len(adminMetrics.errorLog) > 100 {
		adminMetrics.errorLog = adminMetrics.errorLog[1:]
	}
}

// Simplify user agent string
func simplifyUserAgent(ua string) string {
	// Extract browser and OS
	if ua == "" {
		return "Unknown"
	}

	// Basic parsing - just get the main browser
	if contains(ua, "Chrome") {
		return "Chrome"
	} else if contains(ua, "Safari") && !contains(ua, "Chrome") {
		return "Safari"
	} else if contains(ua, "Firefox") {
		return "Firefox"
	} else if contains(ua, "Edge") {
		return "Edge"
	} else if contains(ua, "Opera") {
		return "Opera"
	}

	// Check for mobile
	if contains(ua, "Mobile") {
		return "Mobile Browser"
	}

	return "Other"
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
