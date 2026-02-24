// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// defaultHTTPTimeout is the default timeout for HTTP requests.
const defaultHTTPTimeout = 30 * time.Second

// userAgentTransport wraps an http.RoundTripper to add a User-Agent header to all requests.
type userAgentTransport struct {
	base      http.RoundTripper
	userAgent string
}

// RoundTrip adds the User-Agent header and delegates to the base transport.
func (t *userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req2 := req.Clone(req.Context())
	req2.Header.Set("User-Agent", t.userAgent)
	return t.base.RoundTrip(req2)
}

// newOAuth2Client creates an HTTP client with automatic OAuth2 token management.
func newOAuth2Client(config *clientcredentials.Config, userAgent string) (oauthClient *http.Client, baseClient *http.Client) {
	base := &http.Client{Timeout: defaultHTTPTimeout}

	transport := http.DefaultTransport
	if userAgent != "" {
		transport = &userAgentTransport{
			base:      http.DefaultTransport,
			userAgent: userAgent,
		}
	}
	base.Transport = transport

	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, base)
	return config.Client(ctx), base
}

// wrapWithOAuth2 wraps a base HTTP client with OAuth2 token management.
func wrapWithOAuth2(config *clientcredentials.Config, base *http.Client) *http.Client {
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, base)
	return config.Client(ctx)
}

// validateCredentials tests authentication by requesting an OAuth token.
func validateCredentials(ctx context.Context, config *clientcredentials.Config, baseClient *http.Client) error {
	ctx = context.WithValue(ctx, oauth2.HTTPClient, baseClient)
	ts := config.TokenSource(ctx)
	_, err := ts.Token()
	return err
}
