package client

import (
	"context"
	"fmt"
	"net/http"
)

// APIKey is one registered Journal API key.
type APIKey struct {
	ID             int64    `json:"id"`
	App            string   `json:"app"`
	Kind           string   `json:"kind"`
	Prefix         string   `json:"prefix"`
	AllowedOrigins []string `json:"allowed_origins"`
	DailyQuota     int      `json:"daily_quota"`
	UsedToday      int64    `json:"used_today,omitempty"`
	CreatedAt      string   `json:"created_at"`
	RevokedAt      *string  `json:"revoked_at,omitempty"`
}

// CreateKeyRequest carries parameters to mint a new API key.
type CreateKeyRequest struct {
	App            string   `json:"app"`
	Kind           string   `json:"kind"`
	AllowedOrigins []string `json:"allowed_origins,omitempty"`
	DailyQuota     int      `json:"daily_quota,omitempty"`
}

// CreateKeyResponse carries the created key metadata and the raw one-time token.
type CreateKeyResponse struct {
	Key   APIKey `json:"key"`
	Token string `json:"token"`
}

// ListKeys returns all registered API keys.
func (c *Client) ListKeys(ctx context.Context) ([]APIKey, error) {
	var out struct {
		Keys []APIKey `json:"keys"`
	}
	err := c.do(ctx, http.MethodGet, "/api/apikeys", nil, &out)
	return out.Keys, err
}

// CreateKey mints a new secret or public API key.
func (c *Client) CreateKey(ctx context.Context, req CreateKeyRequest) (CreateKeyResponse, error) {
	var out CreateKeyResponse
	err := c.do(ctx, http.MethodPost, "/api/apikeys", req, &out)
	return out, err
}

// RevokeKey revokes an API key by its numeric ID.
func (c *Client) RevokeKey(ctx context.Context, id int64) error {
	path := fmt.Sprintf("/api/apikeys/%d", id)
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}
