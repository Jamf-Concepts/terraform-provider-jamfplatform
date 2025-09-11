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
)

// Client represents the main API client for Jamf Platform
type Client struct {
	oauthClient *OAuthClient
	baseURL     string
}

type ApiError struct {
	HTTPStatus int     `json:"httpStatus"`
	TraceID    string  `json:"traceId"`
	Errors     []Error `json:"errors"`
}

// Error represents an error response from the API
type Error struct {
	ID          string `json:"id,omitempty"`
	Code        string `json:"code"`
	Field       string `json:"field"`
	Description string `json:"description"`
}

// NewClient creates a new Jamf Platform API client.
func NewClient(baseURL, clientID, clientSecret string) *Client {
	tokenURL := baseURL + "/auth/token"

	config := OAuthConfig{
		TokenURL:     tokenURL,
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}

	if config.UserAgent == "" {
		config.UserAgent = "terraform-provider-jamfplatform"
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

// SetUserAgent sets the User-Agent header value used for token and API requests.
func (c *Client) SetUserAgent(ua string) {
	if c.oauthClient != nil {
		c.oauthClient.SetUserAgent(ua)
	}
}

// makeRequest is a helper method for making authenticated API requests
func (c *Client) makeRequest(ctx context.Context, method, endpoint string, body interface{}) (*http.Response, error) {
	var requestBodyBytes []byte

	var fullURL string
	if len(endpoint) > 0 && endpoint[0] == '/' {
		fullURL = c.baseURL + endpoint
	} else {
		fullURL = c.baseURL + "/" + endpoint
	}

	if body != nil {
		var err error
		requestBodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		requestBody = bytes.NewReader(requestBodyBytes)
	}

	req, err := c.oauthClient.AuthenticatedRequest(ctx, method, fullURL, requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create authenticated request: %w", err)
	}

	if body != nil {
		if method == http.MethodPatch {
			req.Header.Set("Content-Type", "application/merge-patch+json")
		} else {
			req.Header.Set("Content-Type", "application/json")
		}
	}

	resp, err := c.oauthClient.Do(ctx, method, fullURL, requestBodyBytes)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		if closeErr := resp.Body.Close(); closeErr != nil {
			fmt.Printf("warning: error closing response body: %v\n", closeErr)
		}

		c.oauthClient.ClearToken()

		req, err = c.oauthClient.AuthenticatedRequest(ctx, method, fullURL, requestBody)
		if err != nil {
			return nil, fmt.Errorf("failed to create authenticated request after 401: %w", err)
		}

		if body != nil {
			if method == http.MethodPatch {
				req.Header.Set("Content-Type", "application/merge-patch+json")
			} else {
				req.Header.Set("Content-Type", "application/json")
			}
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
			if c.oauthClient != nil && c.oauthClient.logger != nil {
				c.oauthClient.logger.Printf("warning: error closing response body: %v", err)
			} else {
				fmt.Printf("warning: error closing response body: %v\n", err)
			}
		}
	}()

	if resp.StatusCode != expectedStatus {
		body, _ := io.ReadAll(resp.Body)
		var apiErr ApiError
		if err := json.Unmarshal(body, &apiErr); err == nil && len(apiErr.Errors) > 0 {
			var details []string
			for _, e := range apiErr.Errors {
				details = append(details, fmt.Sprintf("[%s] %s: %s", e.Code, e.Field, e.Description))
			}
			return fmt.Errorf("API request failed with status %d, traceId %s: %s", apiErr.HTTPStatus, apiErr.TraceID, details)
		}
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}
