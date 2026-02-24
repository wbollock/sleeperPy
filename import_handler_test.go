package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestImportHandlerRequiresProviderAndLeagueID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/import?provider=espn", nil)
	rr := httptest.NewRecorder()
	importHandler(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestImportHandlerRejectsUnknownProvider(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/import?provider=unknown&league_id=123", nil)
	rr := httptest.NewRecorder()
	importHandler(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown provider, got %d", rr.Code)
	}
}
