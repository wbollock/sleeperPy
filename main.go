package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"sleeperpy/goapp/cli"
)

var logLevel string
var testMode bool

// HTTP client with connection pooling
var httpClient = &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
	},
}

// Cache instances (types defined in types.go)
var borisTiersCache = &tiersCache{
	data:      make(map[string]map[string][][]string),
	timestamp: make(map[string]time.Time),
	ttl:       15 * time.Minute, // Cache tiers for 15 minutes
}

var dynastyValuesCache = &dynastyCache{
	data: make(map[string]DynastyValue),
	ttl:  24 * time.Hour, // Cache for 24 hours (values don't change frequently)
}

var sleeperPlayersCache = &playersCache{
	ttl: 1 * time.Hour, // Cache players data for 1 hour
}

func debugLog(format string, v ...interface{}) {
	if logLevel == "debug" {
		log.Printf(format, v...)
	}
}

// --- Prometheus metrics ---
var (
	totalVisitors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "sleeperpy_total_visitors",
		Help: "Total number of unique visitors to the site.",
	})
	totalLookups = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "sleeperpy_total_lookups",
		Help: "Total number of /lookup requests.",
	})
	totalLeagues = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "sleeperpy_total_leagues",
		Help: "Total number of leagues processed.",
	})
	totalTeams = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "sleeperpy_total_teams",
		Help: "Total number of teams processed.",
	})
	totalErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "sleeperpy_total_errors",
		Help: "Total number of errors encountered.",
	})
)

func init() {
	prometheus.MustRegister(totalVisitors)
	prometheus.MustRegister(totalLookups)
	prometheus.MustRegister(totalLeagues)
	prometheus.MustRegister(totalTeams)
	prometheus.MustRegister(totalErrors)
}

var funcMap = template.FuncMap{
	"safe":    func(s string) template.HTML { return template.HTML(s) },
	"float64": func(i int) float64 { return float64(i) },
	"mul":     func(a, b float64) float64 { return a * b },
	"div": func(a, b float64) float64 {
		if b == 0 {
			return 0
		}
		return a / b
	},
	"parseWinProb": func(s string) int {
		// s is like "62% You 🏆" or "38% Opponent 💀"
		parts := strings.Fields(s)
		if len(parts) > 0 && strings.HasSuffix(parts[0], "%") {
			n, err := strconv.Atoi(strings.TrimSuffix(parts[0], "%"))
			if err == nil {
				return n
			}
		}
		return 50
	},
	"winProbColor": func(s string) string {
		// Green for >60, yellow for 40-60, red for <40
		p := 50
		parts := strings.Fields(s)
		if len(parts) > 0 && strings.HasSuffix(parts[0], "%") {
			n, err := strconv.Atoi(strings.TrimSuffix(parts[0], "%"))
			if err == nil {
				p = n
			}
		}
		if p > 60 {
			return "#3ae87a" // green
		} else if p < 40 {
			return "#e83a3a" // red
		}
		return "#e8c63a" // yellow
	},
	"parseWinEmoji": func(s string) string {
		// s is like "62% You 🏆" or "38% Opponent 💀"
		parts := strings.Fields(s)
		if len(parts) > 2 {
			return parts[2]
		}
		return "🤝"
	},
	"formatTime": func(t time.Time) string {
		if t.IsZero() {
			return "Unknown"
		}
		// Format as relative time
		now := time.Now()
		diff := now.Sub(t)

		if diff < time.Minute {
			return "just now"
		} else if diff < time.Hour {
			mins := int(diff.Minutes())
			if mins == 1 {
				return "1 min ago"
			}
			return fmt.Sprintf("%d mins ago", mins)
		} else if diff < 24*time.Hour {
			hours := int(diff.Hours())
			if hours == 1 {
				return "1 hour ago"
			}
			return fmt.Sprintf("%d hours ago", hours)
		} else if diff < 7*24*time.Hour {
			days := int(diff.Hours() / 24)
			if days == 1 {
				return "1 day ago"
			}
			return fmt.Sprintf("%d days ago", days)
		}
		return t.Format("Jan 2, 2006")
	},
	"add": func(a, b int) int { return a + b },
}

var templates = template.Must(template.New("").Funcs(funcMap).ParseGlob("templates/*.html"))

func main() {
	flag.StringVar(&logLevel, "log", "info", "Log level: info or debug")
	flag.BoolVar(&testMode, "test", false, "Run in test mode with mock data")
	flag.Parse()

	// Check if CLI mode
	args := flag.Args()
	if len(args) > 0 && args[0] == "cli" {
		// Run CLI mode
		os.Exit(cli.Run(args[1:]))
	}

	// Otherwise run web server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Initialize test mode if enabled
	if testMode {
		initTestMode()
		log.Printf("[TEST MODE] Mock API endpoints registered")
		http.HandleFunc("/api/mock/", mockAPIHandler)
		http.HandleFunc("/boris/mock/", mockBorisTiersHandler)
	}

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	http.HandleFunc("/", visitorLogging(indexHandler))
	http.HandleFunc("/lookup", lookupHandler)
	http.HandleFunc("/signout", signoutHandler)
	http.HandleFunc("/privacy", privacyHandler)
	http.HandleFunc("/terms", termsHandler)
	http.HandleFunc("/pricing", pricingHandler)
	http.HandleFunc("/about", aboutHandler)
	http.HandleFunc("/faq", faqHandler)
	http.HandleFunc("/demo", demoHandler)
	http.HandleFunc("/robots.txt", robotsHandler)
	http.HandleFunc("/sitemap.xml", sitemapHandler)
	http.Handle("/metrics", promhttp.Handler())

	if testMode {
		log.Printf("Server running on http://localhost:%s (log level: %s, TEST MODE ENABLED)", port, logLevel)
		log.Printf("  → Use username 'testuser' to see mock data")
		log.Printf("  → 3 test leagues will be loaded with mock tiers")
	} else {
		log.Printf("Server running on http://localhost:%s (log level: %s)", port, logLevel)
	}
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// Team mapping for DST/DEF
var TEAM_MAP = map[string]string{
	"ARI": "Arizona Cardinals", "ATL": "Atlanta Falcons", "BAL": "Baltimore Ravens", "BUF": "Buffalo Bills",
	"CAR": "Carolina Panthers", "CHI": "Chicago Bears", "CIN": "Cincinnati Bengals", "CLE": "Cleveland Browns",
	"DAL": "Dallas Cowboys", "DEN": "Denver Broncos", "DET": "Detroit Lions", "GB": "Green Bay Packers",
	"HOU": "Houston Texans", "IND": "Indianapolis Colts", "JAX": "Jacksonville Jaguars", "KC": "Kansas City Chiefs",
	"LV": "Las Vegas Raiders", "LAC": "Los Angeles Chargers", "LAR": "Los Angeles Rams", "MIA": "Miami Dolphins",
	"MIN": "Minnesota Vikings", "NE": "New England Patriots", "NO": "New Orleans Saints", "NYG": "New York Giants",
	"NYJ": "New York Jets", "PHI": "Philadelphia Eagles", "PIT": "Pittsburgh Steelers", "SEA": "Seattle Seahawks",
	"SF": "San Francisco 49ers", "TB": "Tampa Bay Buccaneers", "TEN": "Tennessee Titans", "WAS": "Washington Commanders",
}
