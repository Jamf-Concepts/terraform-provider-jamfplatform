// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// Package client provides a Go client for the Jamf Platform API.
//
// # Versioning Strategy
//
// This package uses explicit version suffixes (V1, V2, etc.) on types and functions
// to support multiple API versions without breaking changes. This allows:
//   - Adding V2 endpoints alongside V1 without deprecation
//   - Mixing API versions within the same domain (e.g., CBEngine uses both v1 and v2)
//   - Clear indication of which API version each function calls
//
// When API endpoints are upgraded:
//   - Create new V2 types and functions
//   - Keep V1 functions unchanged
//   - Update resources to use V2 at their own pace
//
// Example:
//
//	CreateBlueprintV1() - calls /api/blueprints/v1/blueprints
//	GetCBEngineBaselinesV1() - calls /api/cb-engine/v1/baselines
//	CreateCBEngineBenchmarkV2() - calls /api/cb-engine/v2/benchmarks
//
// https://developer.jamf.com/platform-api/docs/getting-started-with-the-platform-api

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2/clientcredentials"
)

// Logger is an interface for logging HTTP requests and responses.
type Logger interface {
	LogRequest(ctx context.Context, method, url string, body []byte)
	LogResponse(ctx context.Context, statusCode int, headers http.Header, body []byte)
}

// Client represents the main API client for Jamf Platform.
type Client struct {
	baseURL     string
	httpClient  *http.Client
	baseClient  *http.Client
	oauthConfig *clientcredentials.Config
	logger      Logger
	userAgent   string
}

// PaginatedResponseRepresentation captures pagination metadata shared by multiple endpoints.
type PaginatedResponseRepresentation struct {
	Page        int   `json:"page"`
	PageSize    int   `json:"pageSize"`
	TotalCount  int64 `json:"totalCount"`
	TotalPages  int   `json:"totalPages"`
	HasNext     bool  `json:"hasNext"`
	HasPrevious bool  `json:"hasPrevious"`
}

// ApiError represents an error response from the API.
type ApiError struct {
	HTTPStatus int     `json:"httpStatus"`
	TraceID    string  `json:"traceId"`
	Errors     []Error `json:"errors"`
}

// Error represents an individual error detail from an API response.
type Error struct {
	ID          string `json:"id,omitempty"`
	Code        string `json:"code"`
	Field       string `json:"field"`
	Description string `json:"description"`
}

// NewClient creates a new Jamf Platform API client.
func NewClient(baseURL, clientID, clientSecret string) *Client {
	oauthConfig := &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     baseURL + "/auth/token",
	}

	userAgent := "terraform-provider-jamfplatform"
	httpClient, baseClient := newOAuth2Client(oauthConfig, userAgent)

	return &Client{
		baseURL:     baseURL,
		httpClient:  httpClient,
		baseClient:  baseClient,
		oauthConfig: oauthConfig,
		userAgent:   userAgent,
	}
}

// ValidateCredentials tests authentication by requesting an OAuth token.
func (c *Client) ValidateCredentials(ctx context.Context) error {
	return validateCredentials(ctx, c.oauthConfig, c.baseClient)
}

// HTTPClient returns the underlying OAuth2-managed HTTP client for raw authenticated requests.
func (c *Client) HTTPClient() *http.Client {
	return c.httpClient
}

// SetHTTPClient sets a custom base HTTP client (useful for testing).
func (c *Client) SetHTTPClient(httpClient *http.Client) {
	c.baseClient = httpClient
	c.httpClient = wrapWithOAuth2(c.oauthConfig, httpClient)
}

// SetLogger sets the logger for the client.
func (c *Client) SetLogger(logger Logger) {
	c.logger = logger
}

// SetUserAgent sets the User-Agent header value used for token and API requests.
func (c *Client) SetUserAgent(ua string) {
	c.userAgent = ua
	c.httpClient, c.baseClient = newOAuth2Client(c.oauthConfig, ua)
}

// buildURL constructs the full API URL from a relative endpoint.
func (c *Client) buildURL(endpoint string) string {
	if len(endpoint) > 0 && endpoint[0] == '/' {
		return c.baseURL + endpoint
	}
	return c.baseURL + "/" + endpoint
}

// makeRequest is a helper method for making authenticated API requests.
func (c *Client) makeRequest(ctx context.Context, method, endpoint string, body any) (*http.Response, error) {
	return c.doRequest(ctx, method, endpoint, body, "")
}

// doRequest performs an authenticated API request with an optional content type override.
func (c *Client) doRequest(ctx context.Context, method, endpoint string, body any, contentType string) (*http.Response, error) {
	var requestBodyBytes []byte

	fullURL := c.buildURL(endpoint)

	if body != nil {
		var err error
		requestBodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	if c.logger != nil {
		c.logger.LogRequest(ctx, method, fullURL, requestBodyBytes)
	}

	var bodyReader io.Reader
	if requestBodyBytes != nil {
		bodyReader = bytes.NewReader(requestBodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if requestBodyBytes != nil {
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		} else if method == http.MethodPatch {
			req.Header.Set("Content-Type", "application/merge-patch+json")
		} else {
			req.Header.Set("Content-Type", "application/json")
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}

	return resp, nil
}

// handleAPIResponse processes API responses and handles common error cases.
func (c *Client) handleAPIResponse(ctx context.Context, resp *http.Response, expectedStatus int, result any) error {
	defer func() { _ = resp.Body.Close() }()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return fmt.Errorf("failed to read response body: %w", readErr)
	}

	if c.logger != nil {
		c.logger.LogResponse(ctx, resp.StatusCode, resp.Header, body)
	}

	if resp.StatusCode != expectedStatus {
		requestInfo := fmt.Sprintf("method=%s, url=%s", resp.Request.Method, resp.Request.URL.String())
		statusText := http.StatusText(resp.StatusCode)
		statusDetail := fmt.Sprintf("%d", resp.StatusCode)
		if statusText != "" {
			statusDetail = fmt.Sprintf("%d %s", resp.StatusCode, statusText)
		}

		var apiErr ApiError
		if err := json.Unmarshal(body, &apiErr); err == nil && len(apiErr.Errors) > 0 {
			var details []string
			for _, e := range apiErr.Errors {
				details = append(details, fmt.Sprintf("[%s] %s: %s", e.Code, e.Field, e.Description))
			}
			return fmt.Errorf("API request failed with status %d, traceId %s (%s): %s", apiErr.HTTPStatus, apiErr.TraceID, requestInfo, details)
		}

		return fmt.Errorf("API request failed with status %s (%s): %s", statusDetail, requestInfo, string(body))
	}

	if result != nil {
		if err := json.Unmarshal(body, result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}
