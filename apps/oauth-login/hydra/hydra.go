package hydra

import (
	"bytes"
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

type AcceptLoginResponse struct {
	RedirectTo string `json:"redirect_to"`
}

// AcceptLoginRequest calls PUT /admin/oauth2/auth/requests/login/accept with subject.
func (c *Client) AcceptLoginRequest(ctx context.Context, challengeID, subject string) (*AcceptLoginResponse, error) {
	url := c.adminURL + "/admin/oauth2/auth/requests/login/accept?login_challenge=" + challengeID
	body, err := json.Marshal(map[string]string{"subject": subject})
	if err != nil {
		return nil, fmt.Errorf("hydra: marshal accept login body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("hydra: build accept login request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("hydra: PUT accept login: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<10))
		return nil, fmt.Errorf("hydra: PUT accept login: status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var out AcceptLoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("hydra: decode accept login response: %w", err)
	}
	return &out, nil
}
