package main

import (
	"net/http/httptest"
	"testing"
)

func TestReadUserSettingsDefaultsWithNoCookie(t *testing.T) {
	r := httptest.NewRequest("GET", "/settings", nil)
	settings := readUserSettings(r)

	if settings.TierSource != "boris" {
		t.Errorf("expected default tier source 'boris', got %q", settings.TierSource)
	}
	if settings.ActionFeedLimit != MAX_ACTIONS {
		t.Errorf("expected default action feed limit %d, got %d", MAX_ACTIONS, settings.ActionFeedLimit)
	}
	if settings.OffseasonOverride != "auto" {
		t.Errorf("expected default offseason override 'auto', got %q", settings.OffseasonOverride)
	}
}

func TestWriteReadUserSettingsRoundTrip(t *testing.T) {
	w := httptest.NewRecorder()
	original := UserSettings{
		ScoringOverrides:  map[string]string{"league123": "Half PPR"},
		TierSource:        "both",
		ActionCategories:  map[string]bool{"trade": false},
		ActionFeedLimit:   3,
		OffseasonOverride: "force_on",
		FavoriteLeagues:   []string{"league123"},
		HiddenLeagues:     []string{"league456"},
	}
	writeUserSettings(w, original)

	r := httptest.NewRequest("GET", "/settings", nil)
	for _, c := range w.Result().Cookies() {
		r.AddCookie(c)
	}

	got := readUserSettings(r)
	if got.ScoringOverrides["league123"] != "Half PPR" {
		t.Errorf("expected scoring override to round-trip, got %+v", got.ScoringOverrides)
	}
	if got.TierSource != "both" {
		t.Errorf("expected tier source 'both', got %q", got.TierSource)
	}
	if got.ActionFeedLimit != 3 {
		t.Errorf("expected action feed limit 3, got %d", got.ActionFeedLimit)
	}
	if got.OffseasonOverride != "force_on" {
		t.Errorf("expected offseason override 'force_on', got %q", got.OffseasonOverride)
	}
	if !isLeagueFavorite("league123", got) {
		t.Errorf("expected league123 to be a favorite")
	}
	if !isLeagueHidden("league456", got) {
		t.Errorf("expected league456 to be hidden")
	}
	if isActionCategoryEnabled("trade", got) {
		t.Errorf("expected 'trade' category to be disabled")
	}
	if !isActionCategoryEnabled("swap", got) {
		t.Errorf("expected 'swap' category to default to enabled")
	}
}

func TestResolveScoringFormatOverridesAutoDetection(t *testing.T) {
	league := map[string]interface{}{
		"scoring_settings": map[string]interface{}{"rec": 1.0}, // auto-detects as PPR
	}
	settings := UserSettings{ScoringOverrides: map[string]string{"lg1": "Standard"}}

	if got := resolveScoringFormat(league, "lg1", settings); got != "Standard" {
		t.Errorf("expected override 'Standard' to win, got %q", got)
	}
	if got := resolveScoringFormat(league, "lg-other", settings); got != "PPR" {
		t.Errorf("expected auto-detection for a league with no override, got %q", got)
	}
}

func TestResolveOffseasonModeOverride(t *testing.T) {
	regularSeasonState := map[string]interface{}{"season_type": "regular", "week": 5.0}

	if !resolveOffseasonMode(regularSeasonState, UserSettings{OffseasonOverride: "force_on"}) {
		t.Errorf("expected force_on to report offseason even mid-regular-season")
	}
	if resolveOffseasonMode(regularSeasonState, UserSettings{OffseasonOverride: "force_off"}) {
		t.Errorf("expected force_off to never report offseason")
	}
	if resolveOffseasonMode(regularSeasonState, UserSettings{OffseasonOverride: "auto"}) {
		t.Errorf("expected auto to defer to isOffseasonMode (false for regular season)")
	}
}

func TestApplyActionSettingsFiltersAndLimits(t *testing.T) {
	actions := []Action{
		{Category: "swap", Description: "a"},
		{Category: "waiver", Description: "b"},
		{Category: "bye", Description: "c"},
		{Category: "injury", Description: "d"},
	}
	settings := UserSettings{
		ActionCategories: map[string]bool{"waiver": false},
		ActionFeedLimit:  2,
	}

	got := applyActionSettings(actions, settings)
	if len(got) != 2 {
		t.Fatalf("expected limit of 2 actions, got %d: %+v", len(got), got)
	}
	for _, a := range got {
		if a.Category == "waiver" {
			t.Errorf("expected waiver category to be filtered out, got %+v", got)
		}
	}
}

func TestApplyActionSettingsZeroLimitFallsBackToDefault(t *testing.T) {
	actions := make([]Action, MAX_ACTIONS+3)
	for i := range actions {
		actions[i] = Action{Category: "swap", Description: "x"}
	}
	got := applyActionSettings(actions, UserSettings{})
	if len(got) != MAX_ACTIONS {
		t.Fatalf("expected fallback to MAX_ACTIONS (%d), got %d", MAX_ACTIONS, len(got))
	}
}

func TestSaveSettingsFormPersistsPerLeagueFields(t *testing.T) {
	form := httptest.NewRequest("POST", "/settings", nil)
	form.Form = map[string][]string{
		"tier_source":        {"adp"},
		"offseason_override": {"force_off"},
		"action_feed_limit":  {"4"},
		"cat_bye":            {"on"},
		"cat_injury":         {"on"},
		"cat_swap":           {"on"},
		"cat_waiver":         {"on"},
		"cat_handcuff":       {"on"},
		// "cat_trade" intentionally omitted -- should end up disabled
		"league_ids":   {"lg1,lg2"},
		"scoring_lg1":  {"Standard"},
		"favorite_lg1": {"on"},
		"hidden_lg2":   {"on"},
	}

	w := httptest.NewRecorder()
	saveSettingsForm(w, form)

	r2 := httptest.NewRequest("GET", "/settings", nil)
	for _, c := range w.Result().Cookies() {
		r2.AddCookie(c)
	}
	saved := readUserSettings(r2)

	if saved.TierSource != "adp" {
		t.Errorf("expected tier_source 'adp', got %q", saved.TierSource)
	}
	if saved.ScoringOverrides["lg1"] != "Standard" {
		t.Errorf("expected lg1 scoring override 'Standard', got %+v", saved.ScoringOverrides)
	}
	if !isLeagueFavorite("lg1", saved) {
		t.Errorf("expected lg1 to be favorited")
	}
	if !isLeagueHidden("lg2", saved) {
		t.Errorf("expected lg2 to be hidden")
	}
	if isActionCategoryEnabled("trade", saved) {
		t.Errorf("expected 'trade' to be disabled since its checkbox was omitted")
	}
	if !isActionCategoryEnabled("bye", saved) {
		t.Errorf("expected 'bye' to stay enabled since its checkbox was present")
	}
}
