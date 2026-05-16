package hydra

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type LoginRequest struct {
	Challenge                    string   `json:"challenge"`
	RequestURL                   string   `json:"request_url"`
	RequestedScope               []string `json:"requested_scope"`
	RequestedAccessTokenAudience []string `json:"requested_access_token_audience"`
	Skip                         bool     `json:"skip"`
	Subject                      string   `json:"subject"`
}

type Client struct {
	adminURL   string
	httpClient *http.Client
}

func NewClient(adminURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{adminURL: strings.TrimSuffix(adminURL, "/"), httpClient: httpClient}
}

// GetLoginRequest calls GET /admin/oauth2/auth/requests/login?login_challenge={challengeID}.
func (c *Client) GetLoginRequest(ctx context.Context, challengeID string) (*LoginRequest, error) {
	url := c.adminURL + "/admin/oauth2/auth/requests/login?login_challenge=" + challengeID
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("hydra: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("hydra: GET login request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<10))
		return nil, fmt.Errorf("hydra: GET login request: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var lr LoginRequest
	if err := json.NewDecoder(resp.Body).Decode(&lr); err != nil {
		return nil, fmt.Errorf("hydra: decode login request: %w", err)
	}
	return &lr, nil
}
