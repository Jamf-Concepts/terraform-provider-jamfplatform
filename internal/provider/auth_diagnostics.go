// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform"
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
func authFailureDiagnostic(baseURL string, err error) (summary, detail string) {
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
