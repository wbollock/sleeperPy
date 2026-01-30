package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Mock API handler that intercepts Sleeper API calls in test mode
func mockAPIHandler(w http.ResponseWriter, r *http.Request) {
	if !testMode {
		http.Error(w, "Not in test mode", 404)
		return
	}

	path := r.URL.Path
	log.Printf("[TEST MODE] Mock API request: %s", path)

	w.Header().Set("Content-Type", "application/json")

	// User lookup
	if strings.HasPrefix(path, "/api/mock/user/") && !strings.Contains(path, "/leagues/") {
		username := strings.TrimPrefix(path, "/api/mock/user/")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"user_id":      "test_user_" + username,
			"username":     username,
			"display_name": "Test User",
		})
		return
	}

	// Leagues
	if strings.Contains(path, "/leagues/nfl/") {
		leagues := getMockLeagues()
		json.NewEncoder(w).Encode(leagues)
		return
	}

	// NFL state (current week)
	if strings.Contains(path, "/state/nfl") {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"week":   10.0,
			"season": "2024",
		})
		return
	}

	// Players data
	if strings.Contains(path, "/players/nfl") {
		players := getMockPlayers()
		json.NewEncoder(w).Encode(players)
		return
	}

	// Rosters
	if strings.Contains(path, "/rosters") {
		rosters := getMockRosters()
		json.NewEncoder(w).Encode(rosters)
		return
	}

	// Matchups
	if strings.Contains(path, "/matchups/") {
		matchups := getMockMatchups()
		json.NewEncoder(w).Encode(matchups)
		return
	}

	http.Error(w, "Mock endpoint not found", 404)
}

// Boris Chen tier files handler for test mode
func mockBorisTiersHandler(w http.ResponseWriter, r *http.Request) {
	if !testMode {
		http.Error(w, "Not in test mode", 404)
		return
	}

	path := r.URL.Path
	log.Printf("[TEST MODE] Mock Boris tiers request: %s", path)

	// Map URL patterns to local files
	var filename string
	if strings.Contains(path, "text_QB.txt") {
		filename = "QB.txt"
	} else if strings.Contains(path, "text_RB-PPR.txt") || strings.Contains(path, "text_RB-HALF.txt") || strings.Contains(path, "text_RB.txt") {
		filename = "RB-PPR.txt" // Use PPR for all in test mode
	} else if strings.Contains(path, "text_WR") {
		filename = "WR-PPR.txt"
	} else if strings.Contains(path, "text_TE") {
		filename = "TE-PPR.txt"
	} else if strings.Contains(path, "text_FLX") {
		filename = "FLX-PPR.txt"
	} else if strings.Contains(path, "text_K.txt") {
		filename = "K.txt"
	} else if strings.Contains(path, "text_DST.txt") {
		filename = "DST.txt"
	} else {
		http.Error(w, "Unknown tier file", 404)
		return
	}

	tierPath := filepath.Join("testdata", "boris_tiers", filename)
	content, err := os.ReadFile(tierPath)
	if err != nil {
		log.Printf("[TEST MODE] Error reading tier file %s: %v", tierPath, err)
		http.Error(w, fmt.Sprintf("Tier file not found: %s", filename), 404)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write(content)
}

// Mock league data - 3 different league configurations
func getMockLeagues() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"league_id": "mock_league_1",
			"name":      "Test League - PPR with FLEX",
			"scoring_settings": map[string]interface{}{
				"rec": 1.0,
			},
			"roster_positions": []interface{}{"QB", "RB", "RB", "WR", "WR", "WR", "TE", "FLEX", "K", "DEF", "BN", "BN", "BN", "BN", "BN"},
		},
		{
			"league_id": "mock_league_2",
			"name":      "Test League - Superflex Half PPR",
			"scoring_settings": map[string]interface{}{
				"rec": 0.5,
			},
			"roster_positions": []interface{}{"QB", "RB", "RB", "WR", "WR", "TE", "SUPER_FLEX", "FLEX", "K", "DEF", "BN", "BN", "BN", "BN"},
		},
		{
			"league_id": "mock_league_3",
			"name":      "Test League - Standard No Flex",
			"scoring_settings": map[string]interface{}{
				"rec": 0.0,
			},
			"roster_positions": []interface{}{"QB", "RB", "RB", "WR", "WR", "TE", "K", "DEF", "BN", "BN", "BN", "BN", "BN", "BN"},
		},
	}
}

// Mock player database
func getMockPlayers() map[string]interface{} {
	return map[string]interface{}{
		// QBs
		"QB1": map[string]interface{}{"first_name": "Patrick", "last_name": "Mahomes", "position": "QB", "active": true, "roster_percent": 99.9},
		"QB2": map[string]interface{}{"first_name": "Josh", "last_name": "Allen", "position": "QB", "active": true, "roster_percent": 99.5},
		"QB3": map[string]interface{}{"first_name": "Lamar", "last_name": "Jackson", "position": "QB", "active": true, "roster_percent": 95.0},
		"QB4": map[string]interface{}{"first_name": "Justin", "last_name": "Herbert", "position": "QB", "active": true, "roster_percent": 85.0},
		"QB5": map[string]interface{}{"first_name": "Baker", "last_name": "Mayfield", "position": "QB", "active": true, "roster_percent": 65.0},
		"QB6": map[string]interface{}{"first_name": "Jared", "last_name": "Goff", "position": "QB", "active": true, "roster_percent": 70.0},

		// RBs
		"RB1":  map[string]interface{}{"first_name": "Christian", "last_name": "McCaffrey", "position": "RB", "active": true, "roster_percent": 99.9},
		"RB2":  map[string]interface{}{"first_name": "Saquon", "last_name": "Barkley", "position": "RB", "active": true, "roster_percent": 99.5},
		"RB3":  map[string]interface{}{"first_name": "Breece", "last_name": "Hall", "position": "RB", "active": true, "roster_percent": 98.0},
		"RB4":  map[string]interface{}{"first_name": "Bijan", "last_name": "Robinson", "position": "RB", "active": true, "roster_percent": 97.0},
		"RB5":  map[string]interface{}{"first_name": "Derrick", "last_name": "Henry", "position": "RB", "active": true, "roster_percent": 95.0},
		"RB6":  map[string]interface{}{"first_name": "Josh", "last_name": "Jacobs", "position": "RB", "active": true, "roster_percent": 90.0},
		"RB7":  map[string]interface{}{"first_name": "David", "last_name": "Montgomery", "position": "RB", "active": true, "roster_percent": 75.0},
		"RB8":  map[string]interface{}{"first_name": "Aaron", "last_name": "Jones", "position": "RB", "active": true, "roster_percent": 85.0},
		"RB9":  map[string]interface{}{"first_name": "Najee", "last_name": "Harris", "position": "RB", "active": true, "roster_percent": 80.0},
		"RB10": map[string]interface{}{"first_name": "Tony", "last_name": "Pollard", "position": "RB", "active": true, "roster_percent": 70.0},
		"RB11": map[string]interface{}{"first_name": "Brian", "last_name": "Robinson", "position": "RB", "active": true, "roster_percent": 65.0},
		"RB12": map[string]interface{}{"first_name": "Jahmyr", "last_name": "Gibbs", "position": "RB", "active": true, "roster_percent": 92.0},
		"RB13": map[string]interface{}{"first_name": "Jonathan", "last_name": "Taylor", "position": "RB", "active": true, "roster_percent": 88.0},

		// WRs
		"WR1":  map[string]interface{}{"first_name": "CeeDee", "last_name": "Lamb", "position": "WR", "active": true, "roster_percent": 99.9},
		"WR2":  map[string]interface{}{"first_name": "Tyreek", "last_name": "Hill", "position": "WR", "active": true, "roster_percent": 99.8},
		"WR3":  map[string]interface{}{"first_name": "Amon-Ra", "last_name": "St. Brown", "position": "WR", "active": true, "roster_percent": 99.5},
		"WR4":  map[string]interface{}{"first_name": "AJ", "last_name": "Brown", "position": "WR", "active": true, "roster_percent": 99.0},
		"WR5":  map[string]interface{}{"first_name": "Garrett", "last_name": "Wilson", "position": "WR", "active": true, "roster_percent": 95.0},
		"WR6":  map[string]interface{}{"first_name": "Brandon", "last_name": "Aiyuk", "position": "WR", "active": true, "roster_percent": 90.0},
		"WR7":  map[string]interface{}{"first_name": "Davante", "last_name": "Adams", "position": "WR", "active": true, "roster_percent": 92.0},
		"WR8":  map[string]interface{}{"first_name": "Terry", "last_name": "McLaurin", "position": "WR", "active": true, "roster_percent": 80.0},
		"WR9":  map[string]interface{}{"first_name": "Amari", "last_name": "Cooper", "position": "WR", "active": true, "roster_percent": 85.0},
		"WR10": map[string]interface{}{"first_name": "DJ", "last_name": "Moore", "position": "WR", "active": true, "roster_percent": 82.0},
		"WR11": map[string]interface{}{"first_name": "Zay", "last_name": "Flowers", "position": "WR", "active": true, "roster_percent": 75.0},
		"WR12": map[string]interface{}{"first_name": "Justin", "last_name": "Jefferson", "position": "WR", "active": true, "roster_percent": 98.0},

		// TEs
		"TE1": map[string]interface{}{"first_name": "Travis", "last_name": "Kelce", "position": "TE", "active": true, "roster_percent": 99.9},
		"TE2": map[string]interface{}{"first_name": "Sam", "last_name": "LaPorta", "position": "TE", "active": true, "roster_percent": 98.0},
		"TE3": map[string]interface{}{"first_name": "Trey", "last_name": "McBride", "position": "TE", "active": true, "roster_percent": 95.0},
		"TE4": map[string]interface{}{"first_name": "Kyle", "last_name": "Pitts", "position": "TE", "active": true, "roster_percent": 85.0},
		"TE5": map[string]interface{}{"first_name": "Jake", "last_name": "Ferguson", "position": "TE", "active": true, "roster_percent": 70.0},
		"TE6": map[string]interface{}{"first_name": "Tyler", "last_name": "Conklin", "position": "TE", "active": true, "roster_percent": 55.0},

		// Kickers
		"K1": map[string]interface{}{"first_name": "Justin", "last_name": "Tucker", "position": "K", "active": true, "roster_percent": 95.0},
		"K2": map[string]interface{}{"first_name": "Harrison", "last_name": "Butker", "position": "K", "active": true, "roster_percent": 90.0},
		"K3": map[string]interface{}{"first_name": "Tyler", "last_name": "Bass", "position": "K", "active": true, "roster_percent": 80.0},
		"K4": map[string]interface{}{"first_name": "Jake", "last_name": "Elliott", "position": "K", "active": true, "roster_percent": 70.0},

		// DST
		"DST1": map[string]interface{}{"position": "DEF", "team": "SF", "active": true, "roster_percent": 95.0},
		"DST2": map[string]interface{}{"position": "DEF", "team": "DAL", "active": true, "roster_percent": 90.0},
		"DST3": map[string]interface{}{"position": "DEF", "team": "BUF", "active": true, "roster_percent": 85.0},
		"DST4": map[string]interface{}{"position": "DEF", "team": "BAL", "active": true, "roster_percent": 80.0},

		// IR player
		"IR1": map[string]interface{}{"first_name": "Nick", "last_name": "Chubb", "position": "RB", "active": false, "roster_percent": 60.0},
	}
}

// Mock rosters for 3 leagues
func getMockRosters() []map[string]interface{} {
	return []map[string]interface{}{
		// User roster - intentionally has some suboptimal choices
		{
			"roster_id": 1.0,
			"owner_id":  "test_user_testuser",
			"starters":  []interface{}{"QB1", "RB5", "RB7", "WR1", "WR5", "WR8", "TE1", "RB6", "K1", "DST1"}, // RB7 (Montgomery) is weak
			"players":   []interface{}{"QB1", "RB5", "RB7", "WR1", "WR5", "WR8", "TE1", "RB6", "K1", "DST1", "QB4", "RB8", "WR6", "WR9", "TE2"},
			"reserve":   []interface{}{},
		},
		// Opponent roster - strong team
		{
			"roster_id": 2.0,
			"owner_id":  "opponent_1",
			"starters":  []interface{}{"QB2", "RB1", "RB2", "WR2", "WR3", "WR4", "TE2", "RB3", "K2", "DST2"},
			"players":   []interface{}{"QB2", "RB1", "RB2", "WR2", "WR3", "WR4", "TE2", "RB3", "K2", "DST2", "QB3", "RB4", "WR12"},
		},
		// Other teams (free agents will be anyone not on these rosters)
		{
			"roster_id": 3.0,
			"owner_id":  "opponent_2",
			"starters":  []interface{}{"QB5", "RB9", "RB10", "WR7", "WR10", "WR11", "TE4", "RB11", "K3", "DST3"},
			"players":   []interface{}{"QB5", "RB9", "RB10", "WR7", "WR10", "WR11", "TE4", "RB11", "K3", "DST3", "QB6", "TE5"},
		},
	}
}

// Mock matchups
func getMockMatchups() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"roster_id":  1.0,
			"matchup_id": 1.0,
			"starters":   []interface{}{"QB1", "RB5", "RB7", "WR1", "WR5", "WR8", "TE1", "RB6", "K1", "DST1"},
		},
		{
			"roster_id":  2.0,
			"matchup_id": 1.0,
			"starters":   []interface{}{"QB2", "RB1", "RB2", "WR2", "WR3", "WR4", "TE2", "RB3", "K2", "DST2"},
		},
	}
}

// Override fetchBorisTiers in test mode to use local files
func initTestMode() {
	if !testMode {
		return
	}

	log.Println("[TEST MODE] Enabled - using mock data")

	// Override HTTP client to intercept requests
	originalClient := httpClient
	httpClient = &http.Client{
		Timeout:   originalClient.Timeout,
		Transport: &mockTransport{},
	}
}

// Custom transport that intercepts HTTP requests in test mode
type mockTransport struct{}

func (t *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Intercept Sleeper API calls
	if strings.Contains(req.URL.Host, "api.sleeper.app") {
		// Rewrite to local mock server
		req.URL.Scheme = "http"
		req.URL.Host = "localhost:8080"
		req.URL.Path = "/api/mock" + req.URL.Path
		log.Printf("[TEST MODE] Intercepted Sleeper API call, redirecting to mock: %s", req.URL.String())
	}

	// Intercept Boris Chen tier calls
	if strings.Contains(req.URL.Host, "s3-us-west-1.amazonaws.com") {
		req.URL.Scheme = "http"
		req.URL.Host = "localhost:8080"
		req.URL.Path = "/boris/mock" + req.URL.Path
		log.Printf("[TEST MODE] Intercepted Boris Chen call, redirecting to mock: %s", req.URL.String())
	}

	// Use default transport for the modified request
	return http.DefaultTransport.RoundTrip(req)
}
