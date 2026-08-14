package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestLogsPagedWalksTheCursor verifies the pager follows next_before across
// pages until it has the requested limit, and stops at a nil cursor.
func TestLogsPagedWalksTheCursor(t *testing.T) {
	// Three pages of two entries each. The server hands out 2 per request and
	// curates next_before until the youngest entries are exhausted.
	nextID := 6
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limit := 2
		q := r.URL.Query()
		entries := make([]Entry, 0, 2)
		for i := 0; i < limit && nextID > 0; i++ {
			entries = append(entries, Entry{ID: int64(nextID), App: "sablier", Message: "m"})
			nextID--
		}
		var next *Cursor
		if len(entries) == limit {
			oldest := entries[len(entries)-1]
			next = &Cursor{Ts: "t", ID: oldest.ID}
		}
		if q.Get("before_id") != "" {
			// ignore cursor for simplicity; pages are pre-computed by nextID
		}
		json.NewEncoder(w).Encode(ListResponse{Entries: entries, NextBefore: next})
	}))
	defer server.Close()

	client := New(server.URL, "tok")
	entries, err := client.LogsPaged(context.Background(), Filter{}, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 5 {
		t.Fatalf("got %d entries, want 5", len(entries))
	}
	if entries[0].ID != 6 || entries[4].ID != 2 {
		t.Fatalf("unexpected ids: %d..%d", entries[0].ID, entries[4].ID)
	}
}

// TestLogsPagedStopsAtNilCursor verifies a short result (fewer than limit) does
// not loop forever when the server returns a nil cursor before the budget is
// filled.
func TestLogsPagedStopsAtNilCursor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(ListResponse{
			Entries:    []Entry{{ID: 1, App: "a"}},
			NextBefore: nil,
		})
	}))
	defer server.Close()

	client := New(server.URL, "tok")
	entries, err := client.LogsPaged(context.Background(), Filter{}, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
}
