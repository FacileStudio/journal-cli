package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListKeys(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/apikeys" || r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]any{
				{
					"id":              1,
					"app":             "evelynecrea",
					"kind":            "secret",
					"prefix":          "journal_evelynecrea_abc",
					"allowed_origins": []string{},
					"daily_quota":     0,
					"used_today":      42,
					"created_at":      "2026-09-01T15:00:00Z",
				},
			},
		})
	}))
	defer server.Close()

	c := New(server.URL, "test-token")
	keys, err := c.ListKeys(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
	if keys[0].App != "evelynecrea" || keys[0].UsedToday != 42 {
		t.Fatalf("unexpected key content: %+v", keys[0])
	}
}

func TestCreateKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/apikeys" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		var req CreateKeyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(CreateKeyResponse{
			Key: APIKey{
				ID:             2,
				App:            req.App,
				Kind:           req.Kind,
				Prefix:         "journal_pub_evelynecrea_xyz",
				AllowedOrigins: req.AllowedOrigins,
				DailyQuota:     req.DailyQuota,
				CreatedAt:      "2026-09-01T15:00:00Z",
			},
			Token: "journal_pub_evelynecrea_xyz123456",
		})
	}))
	defer server.Close()

	c := New(server.URL, "test-token")
	resp, err := c.CreateKey(context.Background(), CreateKeyRequest{
		App:            "evelynecrea",
		Kind:           "public",
		AllowedOrigins: []string{"https://evelynecreation.fr"},
		DailyQuota:     5000,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Token != "journal_pub_evelynecrea_xyz123456" {
		t.Fatalf("expected token, got %s", resp.Token)
	}
	if resp.Key.App != "evelynecrea" || resp.Key.DailyQuota != 5000 {
		t.Fatalf("unexpected key payload: %+v", resp.Key)
	}
}

func TestRevokeKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/apikeys/2" || r.Method != http.MethodDelete {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	c := New(server.URL, "test-token")
	err := c.RevokeKey(context.Background(), 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
