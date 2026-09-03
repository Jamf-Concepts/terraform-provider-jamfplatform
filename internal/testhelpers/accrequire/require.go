// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package accrequire

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

// accRequiredSets names the credential sets this run must have configured, read
// from JAMFPLATFORM_ACC_REQUIRE as a comma-separated list of the `require`
// tokens declared in .github/acceptance-lanes.json.
//
// It exists because "unset credentials skip" is right locally and wrong in CI.
// A contributor with no tenant must be able to run `make testacc` and get skips
// rather than a wall of failures, so absence cannot be fatal by default. But in
// a pipeline that wires the secrets, absence means the secret is missing or
// misnamed — and a skip there is invisible: the package still prints `ok` and
// the check goes green having asserted nothing against the estate.
//
// That is not hypothetical, three times over. In the SDK, a WAF block on
// 2026-08-04 made all 146 scoped tests skip while the run reported success, and
// its organization credential set was wired in exactly zero places in its
// acceptance workflow from the day its organization client was written, so all
// ten Jamf Account tests skipped on every run for the life of the file with no
// red build. This repo had the same defect and worse: acceptance.yml referenced
// 48 secrets of which 13 existed, so 35 resolved to the empty string and their
// suites self-skipped green — including JAMFPLATFORM_AI_GOVERNANCE_ENVIRONMENT_ID,
// which 18 tests read and which no workflow referenced at all.
//
// The unit is one set per lane rather than a single boolean, because the
// acceptance matrix runs lanes that legitimately need different credentials: the
// pro lane must not fail for a missing organization secret it never uses.
// .github/workflows/acceptance.yml sets this per lane from matrix.require.
var accRequiredSets = sync.OnceValue(func() map[string]bool {
	return parseRequiredSets(AccEnv("JAMFPLATFORM_ACC_REQUIRE"))
})

// parseRequiredSets turns the raw JAMFPLATFORM_ACC_REQUIRE value into the set of
// tokens this run declares required.
//
// Split out of accRequiredSets so it can be tested: that variable is a
// sync.OnceValue reading the environment, so within one process its answer is
// fixed by whatever the first caller saw, and a test cannot vary the input. The
// normalisation is the part worth pinning. A lane row written
// `"require": "securitycloud, pro-tenant"` — a comma-separated list is exactly
// the shape that invites a space after the comma — must register `pro-tenant`
// and not " pro-tenant", or the promotion below silently misses the set and the
// lane reports success having asserted nothing: the failure this whole
// mechanism exists to close, reintroduced by two characters of whitespace.
func parseRequiredSets(raw string) map[string]bool {
	required := map[string]bool{}
	for name := range strings.SplitSeq(raw, ",") {
		if name = strings.ToLower(strings.TrimSpace(name)); name != "" {
			required[name] = true
		}
	}
	return required
}

// promotionToken reports whether a precheck's unset-credential skip must become
// a failure, and under which token.
//
// known is false for a precheck absent from accPrecheckRequireTokens, which is a
// wiring fault rather than a test condition and is fatal either way — see
// SkipOrFailUnset.
//
// Split out for the same reason as parseRequiredSets, and it carries the rule
// that matters most: the match is EXACT. `pro` must not promote `pro-tenant`,
// nor `pro-tenant` promote `pro`, because the two are different lanes on
// different credentials — a substring or prefix match would have the pro lane
// demand the tenant lane's secrets and fail every test for a configuration
// reason.
func promotionToken(precheck string, required map[string]bool) (token string, promote, known bool) {
	tokens, known := accPrecheckRequireTokens[precheck]
	if !known {
		return "", false, false
	}
	for _, token := range tokens {
		if required[token] {
			return token, true, true
		}
	}
	return "", false, true
}

// accPrecheckRequireTokens names, per precheck helper, the JAMFPLATFORM_ACC_REQUIRE
// tokens under which that helper's unset-credential skip must become a failure.
//
// This map is the single place the two vocabularies meet. A lane's `require`
// token is written twice — once in .github/acceptance-lanes.json and once here —
// and the two are not the same word: lane `account` uses token `organization`,
// lane `pro` uses `platform`. Nothing but
// internal/conformance/acc_lanes_test.go compares them, and a one-character
// disagreement is silent AND green: accRequiredSets misses the set,
// SkipOrFailUnset takes t.Skipf instead of t.Fatalf, and the lane reports
// success having run nothing — the exact skip-into-green failure the require
// mechanism exists to prevent.
//
// AccPreCheck lists every scoped lane's token because it is the generic gate
// every family but Jamf Account routes through; a new lane must be added here or
// the conformance test fails, which is the intended way to find out.
// AccPreCheckOffline lists none: the provider-defined functions run offline with
// no API client and no provider configuration, so they need no credential and
// any lane may run them.
//
// One key is not a precheck helper and must not be tidied away.
// AccTenantIDOrSkip is a mid-test gate rather than a precheck, but it skips for
// an unset variable exactly as a precheck does, so it needs the same promotion —
// and it is the gate that keeps the pro-tenant lane's one wire assertion
// honest. internal/conformance/acc_lanes_test.go's shape discovery therefore
// requires every precheck-shaped HELPER to be a key here, without requiring
// every key to be precheck-shaped.
var accPrecheckRequireTokens = map[string][]string{
	"AccPreCheck":              {"platform", "environment", "securitycloud", "aigovernance", "pro-tenant"},
	"AccPreCheckOffline":       nil,
	"AccPreCheckSecurityCloud": {"securitycloud"},
	"AccPreCheckAIGovernance":  {"aigovernance"},
	"AccPreCheckAccount":       {"organization"},
	"AccTenantIDOrSkip":        {"pro-tenant"},
}

// SkipOrFailUnset ends the calling test for a credential set, entitlement
// declaration or scope variable this run has not configured: fatally when the
// run declared that set required, otherwise as a skip.
//
// precheck names the helper making the call and must be a key in
// accPrecheckRequireTokens; reason states what is unconfigured, in the
// imperative-free form the message reads it as ("JAMFPLATFORM_BASE_URL is
// unset"). It is exported because internal/provider carries a package-local
// precheck of its own — the organization-scope test needs the credentials
// without the scope requirement AccPreCheck adds — and that helper must route
// through the same gate or it becomes a hole in it.
//
// Call this only for a genuinely UNCONFIGURED input. Credentials that were
// supplied and then refused must always fail, which is a different case and the
// one CredentialRejectedMessage carries.
func SkipOrFailUnset(t *testing.T, precheck, reason string) {
	t.Helper()

	token, promote, known := promotionToken(precheck, accRequiredSets())
	if !known {
		// A precheck absent from the map would silently lose its gate, so say so
		// rather than degrading to a skip. The conformance test makes this
		// unreachable in a green tree; it is here for the tree that is not.
		t.Fatalf("precheck %q is not declared in accPrecheckRequireTokens, so JAMFPLATFORM_ACC_REQUIRE cannot promote its skips to failures; add it there and give its lane a require token in .github/acceptance-lanes.json", precheck)
	}
	if promote {
		t.Fatalf("JAMFPLATFORM_ACC_REQUIRE names %q, but %s, so these tests would silently skip and the lane would report success having asserted nothing", token, reason)
	}
	t.Skipf("Skipping acceptance test: %s", reason)
}

// egressIP asks a public echo service for this host's outbound address, once per
// run. It is best-effort: an empty result is reported as such rather than
// retried, because the value is diagnostic and a failure to fetch it must never
// be the reason a test fails.
var egressIP = sync.OnceValue(func() string {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://checkip.amazonaws.com")
	if err != nil {
		return ""
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 128))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(body))
})

// CredentialRejectedMessage builds the report to hand Jamf Support when the
// estate refuses credentials that were supplied, mirroring what
// terraform-provider-jamfprotect and jamfplatform-go-sdk emit for the same
// failure.
//
// Supplied-and-refused is a failure, never a skip. It is a different condition
// from unset, and conflating the two is how a WAF block reads as "no
// credentials configured" and reports green.
//
// The first question support asks about a blocked request is always "from which
// IP, and when", and on a CI runner nobody can go and check afterwards — the
// egress IP is gone with the runner. Capturing it in the failure output is the
// only chance to have it. The base URL is passed in rather than read from a
// client, which was never successfully constructed. In CI it renders as *** if
// it came from a secret; that is fine, whoever opens the ticket has it.
func CredentialRejectedMessage(baseURL string, err error) string {
	ip := egressIP()
	if ip == "" {
		ip = "(unable to determine — run `curl -s https://checkip.amazonaws.com` on this host)"
	}
	return fmt.Sprintf(`acceptance credentials were supplied but the estate rejected them.
Failing rather than skipping: a skipped suite reports success while verifying nothing.

If this is an edge or WAF block rather than a bad secret, give Jamf Support:
  - Timestamp:    %s
  - Instance URL: %s
  - Egress IP:    %s

Technical details: %v`,
		time.Now().UTC().Format(time.RFC3339),
		baseURL,
		ip,
		err)
}
