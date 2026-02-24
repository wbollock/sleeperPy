// ABOUTME: Shared API layer for web and CLI access to core functions
package main

import (
	"context"
	"fmt"
	"time"
)

// APIClient provides access to core SleeperPy functionality
type APIClient struct {
	provider LeagueProvider
}

// NewAPIClient creates a new API client
func NewAPIClient() *APIClient {
	return &APIClient{provider: appProvider}
}

// FetchUser fetches a Sleeper user by username
func (a *APIClient) FetchUser(ctx context.Context, username string) (map[string]interface{}, error) {
	return a.provider.FetchUser(username)
}

// FetchUserLeagues fetches leagues for a user ID
func (a *APIClient) FetchUserLeagues(ctx context.Context, userID string) ([]map[string]interface{}, error) {
	return a.provider.FetchUserLeagues(userID, time.Now().Year())
}

// FetchBorisChenTiers fetches Boris Chen tiers for a scoring format
func (a *APIClient) FetchBorisChenTiers(ctx context.Context, scoring string) (map[string][][]string, error) {
	tiers := fetchBorisTiersImpl(scoring)
	if tiers == nil {
		return nil, fmt.Errorf("failed to fetch tiers for format: %s", scoring)
	}
	return tiers, nil
}

// FetchDynastyValues fetches KTC dynasty values
func (a *APIClient) FetchDynastyValues(ctx context.Context) (map[string]interface{}, string, error) {
	rawValues, scrapeDate := fetchDynastyValues()
	if rawValues == nil {
		return nil, "", fmt.Errorf("failed to fetch dynasty values")
	}

	// Convert to map[string]interface{} for CLI
	values := make(map[string]interface{})
	for name, val := range rawValues {
		values[name] = map[string]interface{}{
			"name":        val.Name,
			"position":    val.Position,
			"value_1qb":   val.Value1QB,
			"value_2qb":   val.Value2QB,
			"scrape_date": val.ScrapeDate,
		}
	}
	return values, scrapeDate, nil
}

// FetchPlayers fetches all NFL players from Sleeper API
func (a *APIClient) FetchPlayers(ctx context.Context) (map[string]interface{}, error) {
	return fetchPlayers()
}
