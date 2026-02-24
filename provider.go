package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type LeagueProvider interface {
	FetchUser(username string) (map[string]interface{}, error)
	FetchUserLeagues(userID string, season int) ([]map[string]interface{}, error)
	FetchNFLState() (map[string]interface{}, error)
	FetchLeagueRosters(leagueID string) ([]map[string]interface{}, error)
	FetchLeagueMatchups(leagueID string, week int) ([]map[string]interface{}, error)
	FetchLeagueUsers(leagueID string) ([]map[string]interface{}, error)
	FetchLeagueTradedPicks(leagueID string) ([]map[string]interface{}, error)
}

type SleeperProvider struct {
	baseURL string
	client  *http.Client
}

func NewSleeperProvider(client *http.Client) *SleeperProvider {
	if client == nil {
		client = http.DefaultClient
	}
	return &SleeperProvider{
		baseURL: "https://api.sleeper.app/v1",
		client:  client,
	}
}

var appProvider LeagueProvider = NewSleeperProvider(httpClient)

func (p *SleeperProvider) FetchUser(username string) (map[string]interface{}, error) {
	return p.fetchJSON(fmt.Sprintf("%s/user/%s", p.baseURL, username))
}

func (p *SleeperProvider) FetchUserLeagues(userID string, season int) ([]map[string]interface{}, error) {
	return p.fetchJSONArray(fmt.Sprintf("%s/user/%s/leagues/nfl/%d", p.baseURL, userID, season))
}

func (p *SleeperProvider) FetchNFLState() (map[string]interface{}, error) {
	return p.fetchJSON(fmt.Sprintf("%s/state/nfl", p.baseURL))
}

func (p *SleeperProvider) FetchLeagueRosters(leagueID string) ([]map[string]interface{}, error) {
	return p.fetchJSONArray(fmt.Sprintf("%s/league/%s/rosters", p.baseURL, leagueID))
}

func (p *SleeperProvider) FetchLeagueMatchups(leagueID string, week int) ([]map[string]interface{}, error) {
	return p.fetchJSONArray(fmt.Sprintf("%s/league/%s/matchups/%d", p.baseURL, leagueID, week))
}

func (p *SleeperProvider) FetchLeagueUsers(leagueID string) ([]map[string]interface{}, error) {
	return p.fetchJSONArray(fmt.Sprintf("%s/league/%s/users", p.baseURL, leagueID))
}

func (p *SleeperProvider) FetchLeagueTradedPicks(leagueID string) ([]map[string]interface{}, error) {
	return p.fetchJSONArray(fmt.Sprintf("%s/league/%s/traded_picks", p.baseURL, leagueID))
}

func (p *SleeperProvider) fetchJSON(url string) (map[string]interface{}, error) {
	resp, err := p.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&out)
	return out, err
}

func (p *SleeperProvider) fetchJSONArray(url string) ([]map[string]interface{}, error) {
	resp, err := p.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&out)
	return out, err
}
