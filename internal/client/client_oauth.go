// Copyright 2025 Jamf Software LLC.
// https://developer.jamf.com/platform-api/docs/getting-started-with-the-platform-api

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
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

// OAuthError represents an error response from the token endpoint (RFC 6749)
type OAuthError struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
	ErrorURI         string `json:"error_uri"`
}

// OAuthConfig holds the OAuth configuration
type OAuthConfig struct {
	TokenURL     string
	ClientID     string
	ClientSecret string
	Scopes       []string
	UserAgent    string
}

// OAuthClient handles OAuth authentication and token management
type OAuthClient struct {
	config        OAuthConfig
	httpClient    *http.Client
	token         *TokenResponse
	tokenExpiry   time.Time
	GrantedScopes []string
	mutex         sync.RWMutex
	logger        *log.Logger
	userAgent     string
}

// NewOAuthClient creates a new OAuth client
func NewOAuthClient(config OAuthConfig) *OAuthClient {
	return &OAuthClient{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		userAgent: config.UserAgent,
	}
}

// SetUserAgent sets the User-Agent header value that will be sent with token
// requests and authenticated API requests. If empty, no User-Agent header is set.
func (c *OAuthClient) SetUserAgent(ua string) {
	c.mutex.Lock()
	c.userAgent = ua
	c.mutex.Unlock()
}

// SetLogger assigns a logger to the OAuthClient. If not set, warnings are
// silently discarded.
func (c *OAuthClient) SetLogger(l *log.Logger) {
	c.logger = l
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
	c.mutex.RLock()
	if c.token != nil && time.Now().Before(c.tokenExpiry) {
		token := c.token.AccessToken
		c.mutex.RUnlock()
		return token, nil
	}
	c.mutex.RUnlock()

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
	c.mutex.RLock()
	ua := c.userAgent
	c.mutex.RUnlock()
	if ua != "" {
		req.Header.Set("User-Agent", ua)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make token request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			if c.logger != nil {
				c.logger.Printf("warning: error closing response body: %v", err)
			}
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var oerr OAuthError
		if err := json.Unmarshal(body, &oerr); err == nil && oerr.Error != "" {
			if c.logger != nil {
				c.logger.Printf("token request failed with status %d: %s: %s", resp.StatusCode, oerr.Error, oerr.ErrorDescription)
			}
			return "", fmt.Errorf("token request failed with status %d: %s: %s", resp.StatusCode, oerr.Error, oerr.ErrorDescription)
		}
		return "", fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse token response: %w", err)
	}

	c.mutex.Lock()
	c.GrantedScopes = nil
	if tokenResp.Scope != "" {
		c.GrantedScopes = strings.Fields(tokenResp.Scope)
	}

	now := time.Now()
	if tokenResp.ExpiresIn <= 0 {
		c.token = &tokenResp
		c.tokenExpiry = now.Add(5 * time.Minute)
	} else {
		margin := 10 * time.Second
		expiry := now.Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
		if time.Duration(tokenResp.ExpiresIn)*time.Second > margin {
			expiry = expiry.Add(-margin)
		}
		c.token = &tokenResp
		c.tokenExpiry = expiry
	}
	c.mutex.Unlock()

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

// AuthenticatedRequest creates an HTTP request with OAuth authentication.
func (c *OAuthClient) AuthenticatedRequest(ctx context.Context, method, url string, body []byte) (*http.Request, error) {
	token, err := c.GetValidToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get valid token: %w", err)
	}

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		if method == http.MethodPatch {
			req.Header.Set("Content-Type", "application/merge-patch+json")
		} else {
			req.Header.Set("Content-Type", "application/json")
		}
	}
	c.mutex.RLock()
	ua := c.userAgent
	c.mutex.RUnlock()
	if ua != "" {
		req.Header.Set("User-Agent", ua)
	}

	return req, nil
}

// Do performs an authenticated HTTP request. The body is provided as a
// byte slice so retries can recreate the request body reader.
func (c *OAuthClient) Do(ctx context.Context, method, url string, body []byte) (*http.Response, error) {
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
			if c.logger != nil {
				c.logger.Printf("warning: error closing response body: %v", err)
			}
		}

		c.ClearToken()

		req, err = c.AuthenticatedRequest(ctx, method, url, body)
		if err != nil {
			return nil, fmt.Errorf("failed to create authenticated request after 401: %w", err)
		}

		resp, err = c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
	}

	return resp, nil
}
