package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSleeperProviderFetchUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user/wboll" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"user_id":"u1","username":"wboll"}`))
	}))
	defer server.Close()

	p := NewSleeperProvider(server.Client())
	p.baseURL = server.URL

	user, err := p.FetchUser("wboll")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user["user_id"] != "u1" {
		t.Fatalf("expected user_id u1, got %v", user["user_id"])
	}
}

func TestSleeperProviderFetchLeagueEndpoints(t *testing.T) {
	var seen []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/state/nfl" {
			_, _ = w.Write([]byte(`{"week":1}`))
			return
		}
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	p := NewSleeperProvider(server.Client())
	p.baseURL = server.URL

	_, _ = p.FetchUserLeagues("u1", 2026)
	_, _ = p.FetchNFLState()
	_, _ = p.FetchLeagueRosters("l1")
	_, _ = p.FetchLeagueMatchups("l1", 2)
	_, _ = p.FetchLeagueUsers("l1")
	_, _ = p.FetchLeagueTradedPicks("l1")

	expected := []string{
		"/user/u1/leagues/nfl/2026",
		"/state/nfl",
		"/league/l1/rosters",
		"/league/l1/matchups/2",
		"/league/l1/users",
		"/league/l1/traded_picks",
	}
	if len(seen) != len(expected) {
		t.Fatalf("expected %d requests, got %d (%v)", len(expected), len(seen), seen)
	}
	for i := range expected {
		if seen[i] != expected[i] {
			t.Fatalf("request %d path mismatch: got %s want %s", i, seen[i], expected[i])
		}
	}
}
