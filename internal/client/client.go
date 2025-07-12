// Copyright 2025 Jamf Software LLC.
// https://developer.jamf.com/platform-api/docs/getting-started-with-the-platform-api

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Client represents the main API client for Jamf Platform
type Client struct {
	oauthClient *OAuthClient
	baseURL     string
	httpClient  *http.Client
}

// NewClient creates a new Jamf Platform API client
// region must be one of: us, eu, apac
func NewClient(region, clientID, clientSecret string) *Client {
	region = strings.ToLower(region)
	switch region {
	case "us", "eu", "apac":
	default:
		panic("invalid region: must be one of us, eu, apac")
	}

	baseDomain := fmt.Sprintf("%s.apigw.jamf.com", region)
	baseURL := fmt.Sprintf("https://%s/api/cb/engine", baseDomain)
	tokenURL := fmt.Sprintf("https://%s/auth/token", baseDomain)

	config := OAuthConfig{
		TokenURL:     tokenURL,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       []string{"default-plan", "tenant-environment", "cb-api-product"},
	}

	return &Client{
		oauthClient: NewOAuthClient(config),
		baseURL:     baseURL,
		httpClient:  &http.Client{},
	}
}

// SetHTTPClient allows setting a custom HTTP client
func (c *Client) SetHTTPClient(httpClient *http.Client) {
	c.httpClient = httpClient
	c.oauthClient.SetHTTPClient(httpClient)
}

// makeRequest is a helper method for making authenticated API requests
func (c *Client) makeRequest(ctx context.Context, method, endpoint string, body interface{}) (*http.Response, error) {
	var requestBody []byte
	var err error

	var req *http.Request
	url := fmt.Sprintf("%s%s", c.baseURL, endpoint)

	if body != nil {
		requestBody, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		req, err = http.NewRequestWithContext(ctx, method, url, bytes.NewReader(requestBody))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequestWithContext(ctx, method, url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
	}

	token, err := c.oauthClient.GetValidToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get valid token: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}

	return resp, nil
}

// handleAPIResponse processes API responses and handles common error cases
func (c *Client) handleAPIResponse(resp *http.Response, expectedStatus int, result interface{}) error {
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("warning: error closing response body: %v\n", err)
		}
	}()

	if resp.StatusCode != expectedStatus {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}
