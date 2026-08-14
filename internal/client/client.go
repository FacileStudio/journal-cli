// Package client talks to a Journal instance's HTTP API.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Error carries the API's own error code alongside its message, so a caller can
// branch on the code rather than parse prose.
type Error struct {
	Status  int
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Message }

// Unauthenticated reports whether the instance rejected the session.
func (e *Error) Unauthenticated() bool { return e.Status == http.StatusUnauthorized }

// Client is a connection to one instance.
type Client struct {
	BaseURL string
	Token   string
	http    *http.Client
}

// New builds a client. The timeout is generous because a slow query over a
// huge log store can take a while, but bounded so a hung server is never waited
// on forever.
func New(baseURL, token string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Token:   token,
		http:    &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *Client) request(ctx context.Context, method, path string, body any) (*http.Request, error) {
	var payload io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		payload = bytes.NewReader(encoded)
	}

	request, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, payload)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "journal-cli")
	// The session travels as a bearer token and never as a cookie. porte reads
	// the cookie first and refuses a cookie-authenticated mutating request
	// without an X-Facile-CSRF header, which would break every write while
	// leaving every read working. Nothing attaches a bearer header on the
	// caller's behalf, so bearer is exempt from that rule by construction.
	if c.Token != "" {
		request.Header.Set("Authorization", "Bearer "+c.Token)
	}
	return request, nil
}

// do performs a request and decodes the response into out, if given.
//
// The body is read as text and parsed defensively rather than through a
// streaming decoder: Journal serves its dashboard from the same origin, so a
// mistyped path returns 200 and HTML, and a bare JSON syntax error would hide
// that the URL was simply wrong.
func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	request, err := c.request(ctx, method, path, body)
	if err != nil {
		return err
	}

	response, err := c.http.Do(request)
	if err != nil {
		return fmt.Errorf("cannot reach %s — %w", c.BaseURL, err)
	}
	defer response.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(response.Body, 32<<20))
	if err != nil {
		return err
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return decodeError(raw, response.StatusCode)
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("%s answered with something that is not JSON — check the URL points at a Journal instance", c.BaseURL)
	}
	return nil
}

func decodeError(raw []byte, status int) error {
	var envelope struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(raw, &envelope) == nil && envelope.Error.Message != "" {
		return &Error{Status: status, Code: envelope.Error.Code, Message: envelope.Error.Message}
	}
	return &Error{Status: status, Code: "unknown", Message: fmt.Sprintf("HTTP %d", status)}
}

// AuthConfig is what an instance says it accepts before a login is attempted.
type AuthConfig struct {
	SSOOnly           bool `json:"sso_only"`
	OIDCEnabled       bool `json:"oidc_enabled"`
	AllowRegistration bool `json:"allow_registration"`
}

// AuthConfig asks the instance what it accepts.
func (c *Client) AuthConfig(ctx context.Context) (AuthConfig, error) {
	var out AuthConfig
	err := c.do(ctx, http.MethodGet, "/api/auth/config", nil, &out)
	return out, err
}

// Login exchanges an address and a password for a session token.
func (c *Client) Login(ctx context.Context, email, password string) (string, error) {
	var issued struct {
		Token string `json:"token"`
	}
	body := struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}{email, password}
	err := c.do(ctx, http.MethodPost, "/api/auth/login", body, &issued)
	if err != nil {
		return "", err
	}
	if issued.Token == "" {
		return "", fmt.Errorf("the instance returned an empty token for the login")
	}
	return issued.Token, nil
}

// Exchange trades a one-time porte login code for a session token. The code is
// valid for sixty seconds and works once.
func (c *Client) Exchange(ctx context.Context, code string) (string, error) {
	var exchanged struct {
		UserID string `json:"user_id"`
		Token  string `json:"token"`
	}
	err := c.do(ctx, http.MethodPost, "/api/auth/oidc/exchange", struct {
		Code string `json:"code"`
	}{code}, &exchanged)
	if err != nil {
		return "", err
	}
	if exchanged.Token == "" {
		return "", fmt.Errorf("the instance returned an empty token for the login code")
	}
	return exchanged.Token, nil
}

// Logout revokes the session server-side.
func (c *Client) Logout(ctx context.Context) error {
	return c.do(ctx, http.MethodPost, "/api/auth/logout", nil, nil)
}

// Filter is the read side of the log query. Zero values are omitted.
type Filter struct {
	App       string
	Levels    []string
	Q         string
	RequestID string
	Since     string
	Until     string
	Limit     int
	BeforeTS  string
	BeforeID  int64
}

// Entry is one log entry, shaped as the API returns it.
type Entry struct {
	ID         int64          `json:"id"`
	App        string         `json:"app"`
	Level      string         `json:"level"`
	Message    string         `json:"message"`
	Meta       map[string]any `json:"meta,omitempty"`
	CreatedAt  string         `json:"created_at"`
	ReceivedAt string         `json:"received_at"`
}

// Cursor is the opaque position used to page backwards in time.
type Cursor struct {
	Ts string `json:"ts"`
	ID int64  `json:"id"`
}

// ListResponse is a page of log entries plus the cursor for the next one.
type ListResponse struct {
	Entries    []Entry `json:"entries"`
	NextBefore *Cursor `json:"next_before"`
}

// Logs runs one query and returns the first page.
func (c *Client) Logs(ctx context.Context, filter Filter) (ListResponse, error) {
	var out ListResponse
	err := c.do(ctx, http.MethodGet, "/api/logs"+encodeFilter(filter), nil, &out)
	return out, err
}

// maxPage is the server's per-request ceiling, which the walker never exceeds
// for a single request.
const maxPage = 1000

// LogsPaged walks the keyset cursor until it has collected limit entries (or
// the stream ends), so a caller asking for more than one page gets exactly what
// it asked for instead of the first page plus a hint. Each request is bounded
// by maxPage; the walker owns the cursor and the budget.
func (c *Client) LogsPaged(ctx context.Context, filter Filter, limit int) ([]Entry, error) {
	var out []Entry
	beforeTS, beforeID := filter.BeforeTS, filter.BeforeID

	for {
		want := limit - len(out)
		if want > maxPage {
			want = maxPage
		}
		page, err := c.Logs(ctx, Filter{
			App:       filter.App,
			Levels:    filter.Levels,
			Q:         filter.Q,
			RequestID: filter.RequestID,
			Since:     filter.Since,
			Until:     filter.Until,
			Limit:     want,
			BeforeTS:  beforeTS,
			BeforeID:  beforeID,
		})
		if err != nil {
			return nil, err
		}

		beforeTS, beforeID = cursorValue(page.NextBefore)
		out = append(out, page.Entries...)
		if len(page.Entries) == 0 || len(out) >= limit || page.NextBefore == nil {
			break
		}
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// cursorValue unpacks a page cursor into the two fields the walker sends back
// on the next request. A nil cursor yields nothing, which stops the loop.
func cursorValue(cursor *Cursor) (string, int64) {
	if cursor == nil {
		return "", 0
	}
	return cursor.Ts, cursor.ID
}

// Context returns the stream around one entry, ignoring all filters.
func (c *Client) Context(ctx context.Context, id int64, before, after int) (ListResponse, error) {
	var out ListResponse
	query := url.Values{}
	if before > 0 {
		query.Set("before", strconv.Itoa(before))
	}
	if after > 0 {
		query.Set("after", strconv.Itoa(after))
	}
	path := fmt.Sprintf("/api/logs/%d/context", id)
	if len(query) > 0 {
		path += "?" + query.Encode()
	}
	err := c.do(ctx, http.MethodGet, path, nil, &out)
	return out, err
}

// AppSummary names an app and how much it has logged recently.
type AppSummary struct {
	Name     string `json:"name"`
	Count    int64  `json:"count"`
	LastSeen string `json:"last_seen"`
}

// Apps lists what the instance has seen, most recently active first.
func (c *Client) Apps(ctx context.Context) ([]AppSummary, error) {
	var out struct {
		Apps []AppSummary `json:"apps"`
	}
	err := c.do(ctx, http.MethodGet, "/api/apps", nil, &out)
	return out.Apps, err
}

func encodeFilter(filter Filter) string {
	query := url.Values{}
	if filter.App != "" {
		query.Set("app", filter.App)
	}
	if len(filter.Levels) > 0 {
		query.Set("level", strings.Join(filter.Levels, ","))
	}
	if filter.Q != "" {
		query.Set("q", filter.Q)
	}
	if filter.RequestID != "" {
		query.Set("request_id", filter.RequestID)
	}
	if filter.Since != "" {
		query.Set("since", filter.Since)
	}
	if filter.Until != "" {
		query.Set("until", filter.Until)
	}
	if filter.Limit > 0 {
		query.Set("limit", strconv.Itoa(filter.Limit))
	}
	if filter.BeforeTS != "" && filter.BeforeID > 0 {
		query.Set("before_ts", filter.BeforeTS)
		query.Set("before_id", strconv.FormatInt(filter.BeforeID, 10))
	}
	if len(query) == 0 {
		return ""
	}
	return "?" + query.Encode()
}
