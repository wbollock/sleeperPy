// ABOUTME: User-configurable settings (tiering, action feed, offseason, league favorites)
// ABOUTME: No database -- persisted entirely client-side in a single JSON cookie

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

const settingsCookieName = "sleeper_settings"

// actionCategoryList is the fixed set of weekly-action categories a user can
// toggle off. Order here is also the display order on the settings page.
var actionCategoryList = []string{"bye", "injury", "swap", "waiver", "handcuff", "trade"}

// UserSettings holds every user-configurable preference. Persisted as a
// single JSON cookie (no database exists in this app -- everything else is
// fetched live from Sleeper/Boris Chen/etc. per request), so it deliberately
// stays small and flat.
type UserSettings struct {
	// ScoringOverrides maps league_id -> "PPR"/"Half PPR"/"Standard", used
	// instead of auto-detecting scoring from the league's rec setting.
	// A missing or empty entry means "auto" (detectScoringFormat decides).
	ScoringOverrides map[string]string `json:"scoring_overrides,omitempty"`

	// TierSource selects which tier data feeds the roster tables:
	// "boris" (default, Boris Chen expert-consensus clusters), "adp"
	// (FantasyFootballCalculator live draft-market tiers), or "both"
	// (Boris Chen stays primary, ADP tier shown alongside for comparison).
	TierSource string `json:"tier_source,omitempty"`

	// ActionCategories maps a category ("swap","waiver","bye","injury",
	// "handcuff","trade") to whether it's shown in the weekly action feed.
	// A missing key defaults to enabled -- this map only needs to record
	// categories the user has turned OFF.
	ActionCategories map[string]bool `json:"action_categories,omitempty"`

	// ActionFeedLimit caps how many actions show per league. 0 means "use
	// the app default" (MAX_ACTIONS).
	ActionFeedLimit int `json:"action_feed_limit,omitempty"`

	// OffseasonOverride is "auto" (default, trust Sleeper's season_type/week),
	// "force_on", or "force_off".
	OffseasonOverride string `json:"offseason_override,omitempty"`

	// FavoriteLeagues/HiddenLeagues are league_id lists. Favorites sort
	// first on the dashboard; hidden leagues are excluded entirely from
	// both the dashboard and the per-league tiers page.
	FavoriteLeagues []string `json:"favorite_leagues,omitempty"`
	HiddenLeagues   []string `json:"hidden_leagues,omitempty"`
}

func defaultUserSettings() UserSettings {
	return UserSettings{
		TierSource:        "boris",
		ActionFeedLimit:   MAX_ACTIONS,
		OffseasonOverride: "auto",
	}
}

// readUserSettings decodes the settings cookie, falling back to defaults for
// a first-time visitor or a cookie that fails to parse (e.g. an old schema).
func readUserSettings(r *http.Request) UserSettings {
	settings := defaultUserSettings()

	cookie, err := r.Cookie(settingsCookieName)
	if err != nil || cookie.Value == "" {
		return settings
	}

	raw, err := url.QueryUnescape(cookie.Value)
	if err != nil {
		return settings
	}

	var stored UserSettings
	if err := json.Unmarshal([]byte(raw), &stored); err != nil {
		debugLog("[DEBUG] Failed to parse settings cookie, using defaults: %v", err)
		return settings
	}

	if stored.TierSource == "" {
		stored.TierSource = "boris"
	}
	if stored.ActionFeedLimit <= 0 {
		stored.ActionFeedLimit = MAX_ACTIONS
	}
	if stored.OffseasonOverride == "" {
		stored.OffseasonOverride = "auto"
	}
	return stored
}

func writeUserSettings(w http.ResponseWriter, settings UserSettings) {
	data, err := json.Marshal(settings)
	if err != nil {
		debugLog("[DEBUG] Failed to marshal settings: %v", err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     settingsCookieName,
		Value:    url.QueryEscape(string(data)),
		Path:     "/",
		MaxAge:   180 * 24 * 60 * 60,
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
	})
}

// resolveScoringFormat returns the user's manual override for a league if
// one is set, otherwise falls back to auto-detection from the league's own
// scoring settings.
func resolveScoringFormat(league map[string]interface{}, leagueID string, settings UserSettings) string {
	if override, ok := settings.ScoringOverrides[leagueID]; ok && override != "" {
		return override
	}
	return detectScoringFormat(league)
}

// resolveOffseasonMode applies the user's override on top of Sleeper's own
// season_type/week signal.
func resolveOffseasonMode(state map[string]interface{}, settings UserSettings) bool {
	switch settings.OffseasonOverride {
	case "force_on":
		return true
	case "force_off":
		return false
	default:
		return isOffseasonMode(state)
	}
}

// isLeagueHidden reports whether the user has hidden this league from the
// dashboard and tiers page.
func isLeagueHidden(leagueID string, settings UserSettings) bool {
	for _, id := range settings.HiddenLeagues {
		if id == leagueID {
			return true
		}
	}
	return false
}

func isLeagueFavorite(leagueID string, settings UserSettings) bool {
	for _, id := range settings.FavoriteLeagues {
		if id == leagueID {
			return true
		}
	}
	return false
}

// isActionCategoryEnabled reports whether a weekly-action category should be
// shown. Categories default to enabled -- the settings map only records
// what's been explicitly turned off.
func isActionCategoryEnabled(category string, settings UserSettings) bool {
	enabled, exists := settings.ActionCategories[category]
	if !exists {
		return true
	}
	return enabled
}

// applyActionSettings filters a generated action list down to the
// user's enabled categories and their configured feed limit. Kept separate
// from buildWeeklyActions so that function stays a pure, easily-testable
// generator with no settings dependency.
func applyActionSettings(actions []Action, settings UserSettings) []Action {
	filtered := make([]Action, 0, len(actions))
	for _, a := range actions {
		if isActionCategoryEnabled(a.Category, settings) {
			filtered = append(filtered, a)
		}
	}
	limit := settings.ActionFeedLimit
	if limit <= 0 {
		limit = MAX_ACTIONS
	}
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}
	return filtered
}

// settingsLeagueSummary is the minimal per-league info the settings page
// needs to render scoring-override/favorite/hidden controls -- distinct
// from the full LeagueData used by the tiers page.
type settingsLeagueSummary struct {
	LeagueID        string
	LeagueName      string
	IsDynasty       bool
	DetectedScoring string
	Override        string // current override value, "" = auto
	IsFavorite      bool
	IsHidden        bool
}

// fetchUserLeagueSummaries fetches just enough per-league data (id, name,
// dynasty flag, auto-detected scoring) to populate the settings form. Merges
// current + previous season like lookupHandler/buildDashboardPage do, so
// dynasty leagues still on last season's league_id are covered too.
func fetchUserLeagueSummaries(username string, settings UserSettings) ([]settingsLeagueSummary, error) {
	user, err := appProvider.FetchUser(username)
	if err != nil || user["user_id"] == nil {
		return nil, err
	}
	userID := user["user_id"].(string)

	year := time.Now().Year()
	leagues, err := appProvider.FetchUserLeagues(userID, year)
	if err != nil {
		debugLog("[DEBUG] Error fetching leagues for year %d: %v", year, err)
	}
	if previousYearLeagues, err := appProvider.FetchUserLeagues(userID, year-1); err == nil {
		leagues = append(leagues, previousYearLeagues...)
	}

	seen := make(map[string]bool, len(leagues))
	summaries := make([]settingsLeagueSummary, 0, len(leagues))
	for _, league := range leagues {
		leagueID, _ := league["league_id"].(string)
		if leagueID == "" || seen[leagueID] {
			continue
		}
		seen[leagueID] = true
		leagueName, _ := league["name"].(string)

		summaries = append(summaries, settingsLeagueSummary{
			LeagueID:        leagueID,
			LeagueName:      leagueName,
			IsDynasty:       isDynastyLeague(league),
			DetectedScoring: detectScoringFormat(league),
			Override:        settings.ScoringOverrides[leagueID],
			IsFavorite:      isLeagueFavorite(leagueID, settings),
			IsHidden:        isLeagueHidden(leagueID, settings),
		})
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].LeagueName < summaries[j].LeagueName
	})

	return summaries, nil
}

// settingsHandler renders the settings form (GET) and saves it (POST). There
// is no account system, so "username" is resolved the same way dashboardHandler
// does: the ?user= query param, falling back to the sleeper_username cookie.
func settingsHandler(w http.ResponseWriter, r *http.Request) {
	trackPageView(r.URL.Path)

	username := r.URL.Query().Get("user")
	if username == "" {
		if cookie, err := r.Cookie("sleeper_username"); err == nil {
			username = cookie.Value
		}
	}
	if username == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodPost {
		saveSettingsForm(w, r)
		http.Redirect(w, r, "/settings?user="+url.QueryEscape(username)+"&saved=1", http.StatusSeeOther)
		return
	}

	settings := readUserSettings(r)
	leagues, err := fetchUserLeagueSummaries(username, settings)
	if err != nil {
		renderError(w, fmt.Sprintf("Could not load leagues for \"%s\": %v", username, err))
		return
	}

	categoryEnabled := make(map[string]bool, len(actionCategoryList))
	for _, cat := range actionCategoryList {
		categoryEnabled[cat] = isActionCategoryEnabled(cat, settings)
	}

	if err := templates.ExecuteTemplate(w, "settings.html", SettingsPage{
		Username:        username,
		Leagues:         leagues,
		Settings:        settings,
		CategoryEnabled: categoryEnabled,
		Saved:           r.URL.Query().Get("saved") == "1",
	}); err != nil {
		log.Printf("[ERROR] Template execution error: %v", err)
		http.Error(w, fmt.Sprintf("Error rendering settings: %v", err), http.StatusInternalServerError)
	}
}

// saveSettingsForm parses the settings POST body into a fresh UserSettings
// and writes it back as the settings cookie. Per-league fields (scoring
// override, favorite, hidden) are only known for league IDs the form
// actually rendered, listed in the hidden "league_ids" field -- selects
// always submit even at their "Auto" default, but unchecked checkboxes
// don't submit at all, so that list is what lets an unchecked box actually
// clear a previously-set favorite/hidden flag instead of leaving it stuck.
func saveSettingsForm(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	settings := readUserSettings(r)

	if ts := r.FormValue("tier_source"); ts == "boris" || ts == "adp" || ts == "both" {
		settings.TierSource = ts
	}

	if oo := r.FormValue("offseason_override"); oo == "auto" || oo == "force_on" || oo == "force_off" {
		settings.OffseasonOverride = oo
	}

	if limitStr := r.FormValue("action_feed_limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 20 {
			settings.ActionFeedLimit = limit
		}
	}

	newCategories := make(map[string]bool)
	for _, cat := range actionCategoryList {
		if r.FormValue("cat_"+cat) != "on" {
			newCategories[cat] = false
		}
	}
	settings.ActionCategories = newCategories

	newOverrides := make(map[string]string)
	var newFavorites, newHidden []string
	for _, id := range strings.Split(r.FormValue("league_ids"), ",") {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if scoring := r.FormValue("scoring_" + id); scoring == "PPR" || scoring == "Half PPR" || scoring == "Standard" {
			newOverrides[id] = scoring
		}
		if r.FormValue("favorite_"+id) == "on" {
			newFavorites = append(newFavorites, id)
		}
		if r.FormValue("hidden_"+id) == "on" {
			newHidden = append(newHidden, id)
		}
	}
	settings.ScoringOverrides = newOverrides
	settings.FavoriteLeagues = newFavorites
	settings.HiddenLeagues = newHidden

	writeUserSettings(w, settings)
}
