// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform"
	"golang.org/x/oauth2"
)

// egressIPLookupURL echoes the caller's public source address as a bare line of
// text. Chosen because the response is a single IP and nothing else, so there is
// no parsing to get wrong and no JSON contract to drift.
const egressIPLookupURL = "https://checkip.amazonaws.com"

// egressIPLookupTimeout bounds the lookup. This runs on a path where the user is
// already waiting on a failed provider configuration, so a slow or unreachable
// echo service must not add to the delay — a missing IP costs the user one
// copy-pasteable command, whereas a hang costs them the error message itself.
const egressIPLookupTimeout = 3 * time.Second

// egressIPLookup is the lookup used by authFailureDiagnostic, indirected so
// tests can exercise the blocked-request branch without network access.
var egressIPLookup = lookupPublicEgressIP

// authFailureDiagnostic renders a failed credential validation as a Terraform
// diagnostic summary and detail.
//
// It exists to separate two failures that need opposite remedies and that the
// raw error does not distinguish:
//
//   - A rejected credential. Jamf's token service answers in JSON, including
//     when it refuses: 401 {"error":"invalid_client"}. Remedy: fix the secret.
//   - A request that never reached Jamf. A WAF, IP allowlist or captive proxy
//     answered instead, with an HTML error page or an empty body. Remedy: get
//     this host's egress IP allowed. Nothing is wrong with the credentials.
//   - A base URL that is not the gateway root. The token endpoint is always
//     {baseURL}/auth/token, so a base URL carrying a path prefix sends the
//     exchange somewhere Jamf does not serve and it comes back 404. Remedy: fix
//     base_url. Neither the credentials nor the network is involved.
//
// The SDK marks the second case with ErrUnexpectedResponse (v0.13.0+) precisely
// so consumers can branch rather than string-match, since the condition is
// inferred from the shape of the body rather than reported by Jamf. Without the
// branch both cases rendered as "please verify your credentials are correct",
// which sends the user hunting a secret that turns out to be fine — the failure
// mode the sentinel was added to end.
//
// The status code is deliberately not consulted: a block arrives as a 403 from
// nginx, as a 200 carrying a login or SPA shell, or as a bare 503, so no single
// status identifies it.
// The 404 is separated first because it arrives carrying the same sentinel as a
// network block — the body is not JSON either way — and the blocked-request
// wording is the expensive wrong answer for it: it sends the user to their
// network team and to Jamf Support over a base-URL typo. That typo stops being
// hypothetical at the Platform API GA, which retires {region}.apigw.jamf.com in
// favour of {region}.api.jamfcloud.com and drops the /api segment, so every
// existing configuration has to be edited and some will be edited wrong.
//
// The status code is read off the oauth2 error rather than matched in the
// message because the SDK already models it: annotateTokenError wraps the
// *oauth2.RetrieveError with %w, so it stays reachable through errors.As no
// matter how many layers wrap it, whereas the guidance text it appends is prose
// and free to be reworded.
func authFailureDiagnostic(baseURL string, err error) (summary, detail string) {
	if isTokenEndpointNotFound(err) {
		return "Jamf Platform Base URL Not Found", "The Jamf Platform API returned 404 for the authentication request. " +
			"The token endpoint is always `{base_url}/auth/token`, so a 404 there means `base_url` is not the " +
			"gateway root — most often because it carries a path prefix such as `/api`, which the GA gateway does " +
			"not use.\n\n" +
			"Set `base_url` to the regional gateway root, with no path:\n" +
			"  - https://us.api.jamfcloud.com\n" +
			"  - https://eu.api.jamfcloud.com\n" +
			"  - https://apac.api.jamfcloud.com\n\n" +
			"Configured base URL: " + baseURL + "\n\n" +
			"This is neither a credential problem nor a network block: the credentials were never assessed, " +
			"because the request did not reach an endpoint that assesses them.\n\n" +
			"Technical details: " + err.Error()
	}

	if !errors.Is(err, jamfplatform.ErrUnexpectedResponse) {
		return "Authentication Failed", fmt.Sprintf(
			"Unable to authenticate with Jamf Platform API. Please verify your credentials are correct.\n\nError: %s",
			err.Error(),
		)
	}

	// Support needs to find the block in gateway logs, which means the time it
	// happened and the address it came from. Both are gathered here rather than
	// asked for later, because by the time the user opens a ticket the timestamp
	// is gone and the egress IP may have changed.
	ipLine := egressIPLookup()
	if ipLine == "" {
		ipLine = "(unable to determine — run `curl -s " + egressIPLookupURL + "` on this host)"
	}

	return "Jamf Platform API Request Blocked", "The Jamf Platform API returned a non-JSON response to the authentication " +
		"request, which means something in front of Jamf answered instead of Jamf itself. " +
		"This is typically a security policy — a WAF or an IP allowlist — blocking requests " +
		"from this host's IP address. It is not a credential problem: a rejected client ID or " +
		"secret comes back as JSON and reports itself as such.\n\n" +
		"Contact Jamf Support and provide the following:\n" +
		"  - Timestamp:    " + time.Now().UTC().Format(time.RFC3339) + "\n" +
		"  - Base URL:     " + baseURL + "\n" +
		"  - Egress IP:    " + ipLine + "\n\n" +
		"Technical details: " + err.Error()
}

// jamfGatewayHosts are the domains Jamf serves its own gateways from. A path
// prefix is only wrong beneath these: the SDK builds every request as
// baseURL + path, so a caller fronting Jamf with their own reverse proxy that
// mounts the token endpoint and the namespaces under one prefix is a supported
// configuration and must not be warned about.
var jamfGatewayHosts = []string{".jamfcloud.com", ".jamf.com", ".jamfnebula.com"}

// baseURLPathWarning reports a Jamf base URL carrying a path prefix, or ("", "")
// if there is nothing to say.
//
// This is the same defect authFailureDiagnostic's 404 branch explains after the
// fact, caught before the token exchange so the message names the cause rather
// than the symptom. It warns rather than errors for the reason above — the
// provider cannot tell a mis-set base URL from a deliberate reverse proxy except
// by the host, and being wrong in the erroring direction would make a working
// configuration unusable.
func baseURLPathWarning(baseURL string) (summary, detail string) {
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Host == "" {
		return "", ""
	}
	if trimmed := strings.Trim(parsed.Path, "/"); trimmed == "" {
		return "", ""
	}
	host := strings.ToLower(parsed.Hostname())
	isJamf := false
	for _, suffix := range jamfGatewayHosts {
		if strings.HasSuffix(host, suffix) {
			isJamf = true
			break
		}
	}
	if !isJamf {
		return "", ""
	}
	return "Base URL Carries a Path Prefix", "`base_url` is set to " + baseURL + ", which includes a path. The Jamf Platform " +
		"gateways serve the token endpoint and every API namespace at the host root, so authentication will be " +
		"attempted against " + strings.TrimRight(baseURL, "/") + "/auth/token and fail with a 404.\n\n" +
		"Set `base_url` to the host alone, for example https://eu.api.jamfcloud.com. The `/api` segment older " +
		"configurations carried is not used by the Jamf Platform API at GA."
}

// isTokenEndpointNotFound reports whether the token exchange failed with a 404,
// which identifies a base URL that is not the gateway root.
//
// It requires the sentinel as well as the status, so a 404 carrying a JSON body
// stays a credential failure. Jamf's token service answers refusals in JSON, and
// a JSON 404 from Jamf itself would be Jamf reporting something about the
// request rather than the gateway failing to route it.
func isTokenEndpointNotFound(err error) bool {
	if !errors.Is(err, jamfplatform.ErrUnexpectedResponse) {
		return false
	}
	var retrieve *oauth2.RetrieveError
	if !errors.As(err, &retrieve) {
		return false
	}
	return retrieve.Response != nil && retrieve.Response.StatusCode == http.StatusNotFound
}

// lookupPublicEgressIP reports this host's public source address, or "" if it
// cannot be determined.
func lookupPublicEgressIP() string {
	return fetchEgressIP(egressIPLookupURL)
}

// fetchEgressIP performs the lookup against an explicit URL so tests can point
// it at a local server.
//
// Every failure is deliberately silent: this only enriches an error that is
// already being returned, so a failed lookup must degrade to omitting one line
// rather than replacing the real diagnostic with a complaint about the lookup.
// The caller substitutes an equivalent shell command when this returns "".
func fetchEgressIP(url string) string {
	ctx, cancel := context.WithTimeout(context.Background(), egressIPLookupTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return ""
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ""
	}
	defer func() { _ = resp.Body.Close() }()

	// Capped read: the response should be one short line, and this path must not
	// become a way for an intercepting proxy to stream an unbounded body into a
	// Terraform error message.
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(body))
}
