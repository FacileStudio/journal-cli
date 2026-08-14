package main

import (
	"encoding/json"
	"net/http"
)

func main() {
	entries := []map[string]any{
		{"id": 3, "app": "sablier", "level": "error", "message": "upload failed", "created_at": "2026-08-13T00:00:03Z"},
		{"id": 2, "app": "sablier", "level": "info", "message": "started", "created_at": "2026-08-13T00:00:02Z"},
		{"id": 1, "app": "nuage", "level": "warn", "message": "slow", "created_at": "2026-08-13T00:00:01Z"},
	}
	http.HandleFunc("/api/auth/config", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"sso_only": false, "oidc_enabled": false, "allow_registration": true})
	})
	http.HandleFunc("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"token": "tok"})
	})
	http.HandleFunc("/api/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]bool{"logged_out": true})
	})
	http.HandleFunc("/api/logs", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			http.Error(w, `{"error":{"code":"unauthenticated"}}`, 401)
			return
		}
		out := []map[string]any{entries[2], entries[1], entries[0]}
		json.NewEncoder(w).Encode(map[string]any{"entries": out, "next_before": nil})
	})
	http.HandleFunc("/api/logs/3/context", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			http.Error(w, `{"error":{"code":"unauthenticated"}}`, 401)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"entries":   []map[string]any{entries[2], entries[1], entries[0]},
			"anchor_id": 3,
		})
	})
	http.HandleFunc("/api/apps", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			http.Error(w, `{"error":{"code":"unauthenticated"}}`, 401)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"apps": []map[string]any{
			{"name": "sablier", "count": 2, "last_seen": "2026-08-13T00:00:03Z"},
			{"name": "nuage", "count": 1, "last_seen": "2026-08-13T00:00:01Z"},
		}})
	})
	http.ListenAndServe("127.0.0.1:4987", nil)
}
