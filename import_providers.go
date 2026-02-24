package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type ImportOptions struct {
	LeagueID string
	Season   int
	SWID     string
	ESPN_S2  string
}

type ImportedPlayer struct {
	Name     string `json:"name"`
	Position string `json:"position"`
}

type ImportedTeam struct {
	TeamID int              `json:"team_id"`
	Name   string           `json:"name"`
	Owner  string           `json:"owner"`
	Roster []ImportedPlayer `json:"roster"`
}

type ImportedLeague struct {
	Provider   string         `json:"provider"`
	LeagueID   string         `json:"league_id"`
	Season     int            `json:"season"`
	LeagueName string         `json:"league_name"`
	Teams      []ImportedTeam `json:"teams"`
}

type LeagueImporter interface {
	ImportLeague(ctx context.Context, opts ImportOptions) (*ImportedLeague, error)
}

func getImporter(provider string) (LeagueImporter, error) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "espn":
		return &espnImporter{client: httpClient}, nil
	case "yahoo":
		return &yahooImporter{}, nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

type espnImporter struct {
	client *http.Client
}

func (e *espnImporter) ImportLeague(ctx context.Context, opts ImportOptions) (*ImportedLeague, error) {
	season := opts.Season
	if season == 0 {
		season = timeNowYear()
	}
	if strings.TrimSpace(opts.LeagueID) == "" {
		return nil, fmt.Errorf("league_id is required")
	}

	baseURL := fmt.Sprintf(
		"https://lm-api-reads.fantasy.espn.com/apis/v3/games/ffl/seasons/%d/segments/0/leagues/%s",
		season,
		url.PathEscape(opts.LeagueID),
	)
	params := url.Values{}
	params.Add("view", "mTeam")
	params.Add("view", "mRoster")
	params.Add("view", "mSettings")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if strings.TrimSpace(opts.SWID) != "" && strings.TrimSpace(opts.ESPN_S2) != "" {
		req.Header.Set("Cookie", fmt.Sprintf("SWID=%s; espn_s2=%s", strings.TrimSpace(opts.SWID), strings.TrimSpace(opts.ESPN_S2)))
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("espn import failed: status %d", resp.StatusCode)
	}

	var payload espnLeaguePayload
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	return parseESPNLeaguePayload(payload), nil
}

type yahooImporter struct{}

func (y *yahooImporter) ImportLeague(ctx context.Context, opts ImportOptions) (*ImportedLeague, error) {
	return nil, fmt.Errorf("yahoo read-only import scaffold is in place but live import requires oauth/session setup")
}

type espnLeaguePayload struct {
	ID       int `json:"id"`
	SeasonID int `json:"seasonId"`
	Settings struct {
		Name string `json:"name"`
	} `json:"settings"`
	Members []struct {
		ID          string `json:"id"`
		DisplayName string `json:"displayName"`
		FirstName   string `json:"firstName"`
		LastName    string `json:"lastName"`
	} `json:"members"`
	Teams []struct {
		ID       int      `json:"id"`
		Location string   `json:"location"`
		Nickname string   `json:"nickname"`
		Abbrev   string   `json:"abbrev"`
		Owners   []string `json:"owners"`
		Roster   struct {
			Entries []struct {
				PlayerPoolEntry struct {
					Player struct {
						FullName          string `json:"fullName"`
						DefaultPositionID int    `json:"defaultPositionId"`
					} `json:"player"`
				} `json:"playerPoolEntry"`
			} `json:"entries"`
		} `json:"roster"`
	} `json:"teams"`
}

func parseESPNLeaguePayload(p espnLeaguePayload) *ImportedLeague {
	ownerNameByID := map[string]string{}
	for _, m := range p.Members {
		display := strings.TrimSpace(m.DisplayName)
		if display == "" {
			display = strings.TrimSpace(strings.TrimSpace(m.FirstName + " " + m.LastName))
		}
		if display == "" {
			display = "Unknown"
		}
		ownerNameByID[m.ID] = display
	}

	out := &ImportedLeague{
		Provider:   "espn",
		LeagueID:   strconv.Itoa(p.ID),
		Season:     p.SeasonID,
		LeagueName: strings.TrimSpace(p.Settings.Name),
		Teams:      []ImportedTeam{},
	}

	for _, t := range p.Teams {
		teamName := strings.TrimSpace(strings.TrimSpace(t.Location + " " + t.Nickname))
		if teamName == "" {
			teamName = strings.TrimSpace(t.Abbrev)
		}
		if teamName == "" {
			teamName = fmt.Sprintf("Team %d", t.ID)
		}

		owner := "Unknown"
		if len(t.Owners) > 0 {
			if name, ok := ownerNameByID[t.Owners[0]]; ok {
				owner = name
			}
		}

		players := make([]ImportedPlayer, 0, len(t.Roster.Entries))
		for _, entry := range t.Roster.Entries {
			player := entry.PlayerPoolEntry.Player
			name := strings.TrimSpace(player.FullName)
			if name == "" {
				continue
			}
			players = append(players, ImportedPlayer{
				Name:     name,
				Position: espnPosition(player.DefaultPositionID),
			})
		}

		out.Teams = append(out.Teams, ImportedTeam{
			TeamID: t.ID,
			Name:   teamName,
			Owner:  owner,
			Roster: players,
		})
	}

	if out.Season == 0 {
		out.Season = timeNowYear()
	}
	if out.LeagueID == "0" {
		out.LeagueID = ""
	}

	return out
}

func espnPosition(id int) string {
	switch id {
	case 1:
		return "QB"
	case 2:
		return "RB"
	case 3:
		return "WR"
	case 4:
		return "TE"
	case 5:
		return "K"
	case 16:
		return "DST"
	default:
		return "UNK"
	}
}

// Wrapped for unit tests.
var timeNowYear = func() int {
	return time.Now().Year()
}
