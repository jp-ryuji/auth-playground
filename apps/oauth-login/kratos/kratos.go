package kratos

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Session struct {
	ID       string   `json:"id"`
	Identity Identity `json:"identity"`
}

type Identity struct {
	ID string `json:"id"`
}

type Client struct {
	publicURL  string
	httpClient *http.Client
}

func NewClient(publicURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{publicURL: strings.TrimSuffix(publicURL, "/"), httpClient: httpClient}
}

func (c *Client) PublicURL() string { return c.publicURL }

// GetSession calls GET /sessions/whoami, forwarding cookieHeader.
// Returns nil, nil when 401 (no active session).
func (c *Client) GetSession(ctx context.Context, cookieHeader string) (*Session, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.publicURL+"/sessions/whoami", nil)
	if err != nil {
		return nil, fmt.Errorf("kratos: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if cookieHeader != "" {
		req.Header.Set("Cookie", cookieHeader)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("kratos: GET /sessions/whoami: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<10))
		return nil, fmt.Errorf("kratos: GET /sessions/whoami: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var s Session
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return nil, fmt.Errorf("kratos: decode session: %w", err)
	}
	return &s, nil
}
