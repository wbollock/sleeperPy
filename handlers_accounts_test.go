package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReadSavedUsernames(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: "sleeper_usernames", Value: "wboll, testuser,WBOLL,,demo"})

	got := readSavedUsernames(req)
	if len(got) != 3 {
		t.Fatalf("expected 3 usernames, got %d", len(got))
	}
	if got[0] != "wboll" || got[1] != "testuser" || got[2] != "demo" {
		t.Fatalf("unexpected username order/content: %#v", got)
	}
}

func TestWriteSavedUsernamesPrependsAndLimits(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: "sleeper_usernames", Value: "a,b,c,d,e"})
	rr := httptest.NewRecorder()

	writeSavedUsernames(rr, req, "z")

	cookies := rr.Result().Cookies()
	var saved string
	for _, c := range cookies {
		if c.Name == "sleeper_usernames" {
			saved = c.Value
		}
	}
	if saved == "" {
		t.Fatalf("expected sleeper_usernames cookie to be set")
	}
	// Should prepend new name and keep max 5.
	if saved != "z,a,b,c,d" {
		t.Fatalf("unexpected saved usernames value: %q", saved)
	}
}
