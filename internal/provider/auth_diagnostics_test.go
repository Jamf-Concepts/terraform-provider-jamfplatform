// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform"
)

const testBaseURL = "https://us.apigw.jamf.com"

// stubEgressIP swaps the egress lookup for the duration of a test so the blocked
// branch is exercised without touching the network.
func stubEgressIP(t *testing.T, ip string) {
	t.Helper()
	original := egressIPLookup
	egressIPLookup = func() string { return ip }
	t.Cleanup(func() { egressIPLookup = original })
}

// A rejected credential must keep pointing at the credentials. This is the
// common case and the branch must not regress into the blocked-request wording.
func TestAuthFailureDiagnostic_CredentialErrorBlamesCredentials(t *testing.T) {
	stubEgressIP(t, "203.0.113.10")

	summary, detail := authFailureDiagnostic(testBaseURL, errors.New("oauth2: 401 invalid_client"))

	if summary != "Authentication Failed" {
		t.Errorf("summary = %q, want %q", summary, "Authentication Failed")
	}
	if !strings.Contains(detail, "verify your credentials") {
		t.Errorf("detail should point at the credentials, got:\n%s", detail)
	}
	// The egress IP is a support-escalation detail for a network block. Leaking it
	// into a plain bad-secret error is exactly the conflation this change removes.
	if strings.Contains(detail, "203.0.113.10") || strings.Contains(detail, "Egress IP") {
		t.Errorf("credential failure must not carry the egress-IP support block, got:\n%s", detail)
	}
	if !strings.Contains(detail, "oauth2: 401 invalid_client") {
		t.Errorf("detail should preserve the underlying error, got:\n%s", detail)
	}
}

// The sentinel must flip the diagnostic to the network-block reading, and must
// state that the credentials are not the problem — the whole point is to stop
// the user hunting a secret that is fine.
func TestAuthFailureDiagnostic_SentinelBlamesNetworkPolicy(t *testing.T) {
	stubEgressIP(t, "203.0.113.10")

	err := fmt.Errorf("%w: 403 Forbidden <html>nginx</html>", jamfplatform.ErrUnexpectedResponse)
	summary, detail := authFailureDiagnostic(testBaseURL, err)

	if summary != "Jamf Platform API Request Blocked" {
		t.Errorf("summary = %q, want %q", summary, "Jamf Platform API Request Blocked")
	}
	if strings.Contains(detail, "verify your credentials") {
		t.Errorf("blocked request must not blame the credentials, got:\n%s", detail)
	}
	if !strings.Contains(detail, "not a credential problem") {
		t.Errorf("detail should say outright that credentials are not the cause, got:\n%s", detail)
	}
	for _, want := range []string{"203.0.113.10", "Egress IP", "Timestamp", testBaseURL, "Jamf Support"} {
		if !strings.Contains(detail, want) {
			t.Errorf("detail missing %q, got:\n%s", want, detail)
		}
	}
}

// A wrapped sentinel must still be recognised: the SDK returns it wrapped, and
// callers may wrap it further, so matching has to be errors.Is and not equality.
func TestAuthFailureDiagnostic_SentinelDetectedWhenDoubleWrapped(t *testing.T) {
	stubEgressIP(t, "198.51.100.7")

	inner := fmt.Errorf("%w: 200 OK <html>login</html>", jamfplatform.ErrUnexpectedResponse)
	err := fmt.Errorf("validating credentials: %w", inner)

	summary, _ := authFailureDiagnostic(testBaseURL, err)
	if summary != "Jamf Platform API Request Blocked" {
		t.Errorf("summary = %q, want the blocked-request summary for a wrapped sentinel", summary)
	}
}

// A failed egress lookup must degrade to a runnable command, never to an empty
// field or a complaint that replaces the real diagnostic.
func TestAuthFailureDiagnostic_FallsBackWhenEgressLookupFails(t *testing.T) {
	stubEgressIP(t, "")

	err := fmt.Errorf("%w: 503", jamfplatform.ErrUnexpectedResponse)
	_, detail := authFailureDiagnostic(testBaseURL, err)

	if !strings.Contains(detail, "unable to determine") {
		t.Errorf("detail should note the lookup failed, got:\n%s", detail)
	}
	if !strings.Contains(detail, egressIPLookupURL) {
		t.Errorf("detail should suggest the manual lookup command, got:\n%s", detail)
	}
	if strings.Contains(detail, "Egress IP:    \n") {
		t.Errorf("detail should never render an empty egress-IP line, got:\n%s", detail)
	}
}

// The lookup trims the trailing newline the echo service sends, so the IP does
// not break the alignment of the support block.
func TestLookupPublicEgressIP_TrimsResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("203.0.113.10\n"))
	}))
	t.Cleanup(server.Close)

	got := fetchEgressIP(server.URL)
	if got != "203.0.113.10" {
		t.Errorf("fetchEgressIP() = %q, want %q", got, "203.0.113.10")
	}
}

// An intercepting proxy may answer the lookup with an arbitrarily long body. The
// read is capped so it cannot be pasted wholesale into a Terraform error.
func TestLookupPublicEgressIP_CapsOversizedBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("A", 4096)))
	}))
	t.Cleanup(server.Close)

	got := fetchEgressIP(server.URL)
	if len(got) > 64 {
		t.Errorf("fetchEgressIP() returned %d bytes, want the read capped at 64", len(got))
	}
}

// An unreachable lookup returns "" rather than an error, so the caller can
// substitute the manual command.
func TestLookupPublicEgressIP_UnreachableReturnsEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := server.URL
	server.Close() // nothing is listening now

	if got := fetchEgressIP(url); got != "" {
		t.Errorf("fetchEgressIP() = %q, want empty string for an unreachable host", got)
	}
}

// End-to-end guard: the branch above is only worth having if the SDK really
// raises the sentinel for an HTML token response. Without this, a change in the
// SDK's detection would silently turn the blocked-request diagnostic into dead
// code and every block would quietly revert to reading as a bad credential.
func TestValidateCredentials_HTMLTokenResponseYieldsSentinel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("<html><head><title>403 Forbidden</title></head><body>nginx</body></html>"))
	}))
	t.Cleanup(server.Close)

	client := jamfplatform.NewClient(server.URL, "test-id", "test-secret")
	err := client.ValidateCredentials(context.Background())
	if err == nil {
		t.Fatal("ValidateCredentials succeeded against an HTML 403")
	}
	if !errors.Is(err, jamfplatform.ErrUnexpectedResponse) {
		t.Fatalf("error does not carry ErrUnexpectedResponse, so the blocked-request branch is unreachable: %v", err)
	}

	stubEgressIP(t, "203.0.113.10")
	if summary, _ := authFailureDiagnostic(server.URL, err); summary != "Jamf Platform API Request Blocked" {
		t.Errorf("summary = %q, want the blocked-request summary", summary)
	}
}

// The counterpart: a JSON credential rejection must NOT carry the sentinel, or
// every bad secret would be misreported as a network block.
func TestValidateCredentials_JSONRejectionHasNoSentinel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid_client"}`))
	}))
	t.Cleanup(server.Close)

	client := jamfplatform.NewClient(server.URL, "test-id", "test-secret")
	err := client.ValidateCredentials(context.Background())
	if err == nil {
		t.Fatal("ValidateCredentials succeeded against a 401 invalid_client")
	}
	if errors.Is(err, jamfplatform.ErrUnexpectedResponse) {
		t.Fatalf("a JSON credential rejection must not carry ErrUnexpectedResponse: %v", err)
	}

	stubEgressIP(t, "203.0.113.10")
	if summary, _ := authFailureDiagnostic(server.URL, err); summary != "Authentication Failed" {
		t.Errorf("summary = %q, want the credential-failure summary", summary)
	}
}
