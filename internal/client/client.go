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
	}
}

// SetHTTPClient allows setting a custom HTTP client
func (c *Client) SetHTTPClient(httpClient *http.Client) {
	c.oauthClient.SetHTTPClient(httpClient)
}

// makeRequest is a helper method for making authenticated API requests
func (c *Client) makeRequest(ctx context.Context, method, endpoint string, body interface{}) (*http.Response, error) {
	var requestBody io.Reader

	url := fmt.Sprintf("%s%s", c.baseURL, endpoint)

	if body != nil {
		requestBodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		requestBody = bytes.NewReader(requestBodyBytes)
	}

	// Create authenticated request
	req, err := c.oauthClient.AuthenticatedRequest(ctx, method, url, requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create authenticated request: %w", err)
	}

	// Set Content-Type for JSON requests
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Use the OAuth client's HTTP client directly since we already have an authenticated request
	// The Do method on OAuthClient expects to handle the entire flow including authentication
	resp, err := c.oauthClient.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}

	// Handle 401 errors by clearing token and retrying once
	if resp.StatusCode == http.StatusUnauthorized {
		if closeErr := resp.Body.Close(); closeErr != nil {
			fmt.Printf("warning: error closing response body: %v\n", closeErr)
		}

		c.oauthClient.ClearToken()

		// Retry with fresh token
		req, err = c.oauthClient.AuthenticatedRequest(ctx, method, url, requestBody)
		if err != nil {
			return nil, fmt.Errorf("failed to create authenticated request after 401: %w", err)
		}

		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err = c.oauthClient.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("API request failed on retry: %w", err)
		}
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
