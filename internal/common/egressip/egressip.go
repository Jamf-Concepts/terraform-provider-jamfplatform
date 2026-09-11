// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// Package egressip reports the host's public source address, which is the
// address a Jamf-side IP allowlist or WAF rule is written against.
//
// It exists so the provider has one implementation and one cache. Two
// diagnostics need the address — provider configuration, when the token
// exchange is blocked, and any resource whose API call an edge page answered —
// and they run at different times in the same process.
package egressip

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// LookupURL echoes the caller's public source address as a bare line of text.
// Chosen because the response is a single IP and nothing else, so there is no
// parsing to get wrong and no JSON contract to drift.
const LookupURL = "https://checkip.amazonaws.com"

// lookupTimeout bounds one attempt. Every caller is on a path where the
// operator is already waiting on a failure, so a slow or unreachable echo
// service must not add to the delay — a missing address costs them one
// copy-pasteable command, whereas a hang costs them the error message itself.
const lookupTimeout = 3 * time.Second

// maxResponseBytes caps the read. The response should be one short line, and
// this must not become a way for an intercepting proxy to stream an unbounded
// body into a Terraform error message.
const maxResponseBytes = 64

// Lookup reports this host's public source address, or "" if it cannot be
// determined, and performs at most one request per process.
//
// The cache is what makes this safe to call from a resource diagnostic. An edge
// block fails every in-flight resource at once — Terraform applies ten in
// parallel by default — and an uncached lookup would turn one outage into a
// burst of identical calls to a third-party service, each adding its timeout to
// an error the operator is already waiting on. The address cannot meaningfully
// change within one apply, so caching costs nothing.
//
// A failure is cached too, deliberately: whatever stopped the first attempt
// (no general internet egress, a proxy that only permits the Jamf host) will
// stop the rest, and retrying it once per failed resource is the cost this
// exists to avoid.
var Lookup = onceFrom(LookupURL)

// onceFrom builds a process-lifetime cache over one lookup URL. Separate from
// Lookup so a test can bind a fresh cache to a stub server; sync.OnceValue
// cannot be reset, so a test sharing the production cache would either poison it
// for the rest of the run or pass only when it ran first.
func onceFrom(url string) func() string {
	return sync.OnceValue(func() string { return FetchFrom(url) })
}

// FetchFrom performs one uncached lookup against an explicit URL, so a test can
// point it at a local server.
//
// Every failure is silent. This only enriches an error that is already being
// returned, so a failed lookup must degrade to omitting one line rather than
// replacing the real diagnostic with a complaint about the lookup. Callers
// substitute an equivalent shell command when this returns "".
func FetchFrom(url string) string {
	ctx, cancel := context.WithTimeout(context.Background(), lookupTimeout)
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

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(body))
}
