// Copyright 2025 Jamf Software LLC.
// https://developer.jamf.com/platform-api/docs/getting-started-with-the-platform-api

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// TokenResponse represents the OAuth token response
type TokenResponse struct {
	AccessToken      string `json:"access_token"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
	TokenType        string `json:"token_type"`
	NotBeforePolicy  int    `json:"not-before-policy"`
	Scope            string `json:"scope"`
}

// OAuthConfig holds the OAuth configuration
type OAuthConfig struct {
	TokenURL     string
	ClientID     string
	ClientSecret string
	Scopes       []string
}

// OAuthClient handles OAuth authentication and token management
type OAuthClient struct {
	config        OAuthConfig
	httpClient    *http.Client
	token         *TokenResponse
	tokenExpiry   time.Time
	GrantedScopes []string
	mutex         sync.RWMutex
}

// NewOAuthClient creates a new OAuth client
func NewOAuthClient(config OAuthConfig) *OAuthClient {
	return &OAuthClient{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetHTTPClient allows setting a custom HTTP client
func (c *OAuthClient) SetHTTPClient(client *http.Client) {
	c.httpClient = client
}

// GetValidToken returns a valid access token, refreshing if necessary
func (c *OAuthClient) GetValidToken(ctx context.Context) (string, error) {
	c.mutex.RLock()
	if c.token != nil && time.Now().Before(c.tokenExpiry) {
		token := c.token.AccessToken
		c.mutex.RUnlock()
		return token, nil
	}
	c.mutex.RUnlock()

	return c.refreshToken(ctx)
}

// refreshToken obtains a new access token using client credentials
func (c *OAuthClient) refreshToken(ctx context.Context) (string, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.token != nil && time.Now().Before(c.tokenExpiry) {
		return c.token.AccessToken, nil
	}

	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", c.config.ClientID)
	data.Set("client_secret", c.config.ClientSecret)

	if len(c.config.Scopes) > 0 {
		data.Set("scope", strings.Join(c.config.Scopes, " "))
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.config.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make token request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("warning: error closing response body: %v\n", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse token response: %w", err)
	}

	// Parse granted scopes from token response
	c.GrantedScopes = nil
	if tokenResp.Scope != "" {
		c.GrantedScopes = strings.Fields(tokenResp.Scope)
	}

	expiryDuration := time.Duration(tokenResp.ExpiresIn-60) * time.Second
	if expiryDuration <= 0 {
		expiryDuration = 5 * time.Minute
	}

	c.token = &tokenResp
	c.tokenExpiry = time.Now().Add(expiryDuration)

	return tokenResp.AccessToken, nil
}

// GetGrantedScopes returns the scopes granted by the OAuth server for the current token
func (c *OAuthClient) GetGrantedScopes() []string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return append([]string(nil), c.GrantedScopes...)
}

// IsTokenValid checks if the current token is valid
func (c *OAuthClient) IsTokenValid() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.token != nil && time.Now().Before(c.tokenExpiry)
}

// ClearToken clears the stored token (useful for logout or error scenarios)
func (c *OAuthClient) ClearToken() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.token = nil
	c.tokenExpiry = time.Time{}
}

// GetTokenInfo returns current token information (for debugging/monitoring)
func (c *OAuthClient) GetTokenInfo() (accessToken string, expiresAt time.Time, isValid bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if c.token == nil {
		return "", time.Time{}, false
	}

	return c.token.AccessToken, c.tokenExpiry, time.Now().Before(c.tokenExpiry)
}

// AuthenticatedRequest creates an HTTP request with OAuth authentication
func (c *OAuthClient) AuthenticatedRequest(ctx context.Context, method, url string, body io.Reader) (*http.Request, error) {
	token, err := c.GetValidToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get valid token: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	return req, nil
}

// Do performs an authenticated HTTP request
func (c *OAuthClient) Do(ctx context.Context, method, url string, body io.Reader) (*http.Response, error) {
	req, err := c.AuthenticatedRequest(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusUnauthorized {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("warning: error closing response body: %v\n", err)
		}

		c.ClearToken()

		token, err := c.GetValidToken(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to refresh token after 401: %w", err)
		}

		req, err = http.NewRequestWithContext(ctx, method, url, body)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err = c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
	}

	return resp, nil
}
