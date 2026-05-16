// Package oidc is a minimal OIDC discovery client for apps/api.
//
// Contract: docs/specs/00-architecture/system-overview.md (OV-01, OV-02, OV-03).
// First consumer: docs/specs/10-flows/signup-first-login.md (SIGNUP-01).
//
// Scope of this iteration: fetch the discovery document and expose the
// fields SIGNUP-01..14 will read. No caching, no JWKS rotation — those
// land with `30-cross-cutting/discovery-and-jwks.md` (TODO).
package oidc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Document is the subset of the OpenID Provider metadata that apps/api
// consumes. Additional fields will be added as later SIGNUP-NN need them.
type Document struct {
	Issuer                string   `json:"issuer"`
	AuthorizationEndpoint string   `json:"authorization_endpoint"`
	TokenEndpoint         string   `json:"token_endpoint"`
	JWKSURI               string   `json:"jwks_uri"`
	CodeChallengeMethods  []string `json:"code_challenge_methods_supported"`
}

// Client fetches the OpenID Provider configuration from a known issuer.
type Client struct {
	httpClient *http.Client
	issuerURL  string
}

// NewClient binds the client to an issuer URL. The trailing slash is
// optional — both "http://host/" and "http://host" work.
//
// If httpClient is nil, http.DefaultClient is used.
func NewClient(issuerURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{httpClient: httpClient, issuerURL: issuerURL}
}

// Fetch loads {issuer}/.well-known/openid-configuration and returns the
// decoded metadata. It errors if the response is non-2xx, the body is
// not valid JSON, or the document lacks authorization_endpoint (which
// SIGNUP-01 directly depends on).
func (c *Client) Fetch(ctx context.Context) (*Document, error) {
	if strings.TrimSpace(c.issuerURL) == "" {
		return nil, errors.New("oidc: issuer URL is empty")
	}
	url := strings.TrimSuffix(c.issuerURL, "/") + "/.well-known/openid-configuration"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("oidc: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("oidc: GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<10))
		return nil, fmt.Errorf("oidc: GET %s: status %d: %s", url, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var doc Document
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, fmt.Errorf("oidc: decode %s: %w", url, err)
	}
	if doc.AuthorizationEndpoint == "" {
		return nil, fmt.Errorf("oidc: %s missing authorization_endpoint", url)
	}
	return &doc, nil
}
