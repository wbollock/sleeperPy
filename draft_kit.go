// ABOUTME: Pre-draft cheat sheet page, built from Boris Chen's combined cross-position "Draft Kit" tiers
// ABOUTME: Distinct from the weekly per-position tiers used elsewhere -- this is one board ranking every skill position together

package main

import (
	"log"
	"net/http"
	"strconv"
)

var validDraftKitScoring = map[string]bool{"PPR": true, "Half PPR": true, "Standard": true}

func draftKitHandler(w http.ResponseWriter, r *http.Request) {
	trackPageView(r.URL.Path)

	username := r.URL.Query().Get("user")
	if username == "" {
		if cookie, err := r.Cookie("sleeper_username"); err == nil {
			username = cookie.Value
		}
	}

	scoring := r.URL.Query().Get("scoring")
	if !validDraftKitScoring[scoring] {
		scoring = "PPR"
	}

	adjust, err := strconv.Atoi(r.URL.Query().Get("adjust"))
	if err != nil || adjust < 0 || adjust > 2 {
		adjust = 0
	}

	rawTiers := fetchDraftKitTiers(scoring, adjust)

	page := DraftKitPage{
		Username: username,
		Scoring:  scoring,
		Adjust:   adjust,
		Empty:    len(rawTiers) == 0,
	}

	if !page.Empty {
		positionByName := buildPositionLookup()
		page.Tiers = make([]DraftKitTier, 0, len(rawTiers))
		for i, names := range rawTiers {
			tier := DraftKitTier{Number: i + 1, Players: make([]DraftKitPlayer, 0, len(names))}
			for _, name := range names {
				if name == "" {
					continue
				}
				tier.Players = append(tier.Players, DraftKitPlayer{
					Name:     name,
					Position: positionByName[normalizeName(name)],
				})
			}
			page.Tiers = append(page.Tiers, tier)
		}
	}

	if err := templates.ExecuteTemplate(w, "draft_kit.html", page); err != nil {
		log.Printf("[ERROR] Template execution error: %v", err)
		http.Error(w, "Error rendering draft kit", http.StatusInternalServerError)
	}
}

// buildPositionLookup maps normalized player name -> position, for
// annotating the combined draft kit tiers (which are plain names with no
// position attached) with each player's position. Best-effort: a name that
// fails to match just renders with no position tag.
func buildPositionLookup() map[string]string {
	players, err := fetchPlayers()
	if err != nil {
		debugLog("[DEBUG] Draft kit: could not fetch players for position lookup: %v", err)
		return nil
	}

	out := make(map[string]string, len(players))
	for _, p := range players {
		pm, ok := p.(map[string]interface{})
		if !ok {
			continue
		}
		pos, _ := pm["position"].(string)
		if pos == "" {
			continue
		}
		out[normalizeName(getPlayerName(pm))] = pos
	}
	return out
}
