// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package device_group

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/providerdata"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// newProIDMockClient spins up a local HTTPS server that auto-serves OAuth tokens
// at /auth/token and forwards everything else to the supplied handler. Returned
// jamfplatform.Client is bound to the server URL. Inlined here rather than reused
// from testhelpers because the testhelpers package compiles acceptance-tagged
// files that import internal/provider, which transitively imports this package
// and creates an import cycle under `go test -tags=acceptance`.
func newProIDMockClient(t *testing.T, handler http.Handler) *jamfplatform.Client {
	t.Helper()
	wrapped := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/token" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "test-token",
				"token_type":   "Bearer",
				"expires_in":   3600,
			})
			return
		}
		handler.ServeHTTP(w, r)
	})
	server := httptest.NewServer(wrapped)
	t.Cleanup(server.Close)
	// Disable the SDK's automatic retry-on-transient-failure: several tests
	// here mock a persistent 403/502 to exercise resolveJamfProID's
	// error-handling path, which would otherwise hang for the production
	// 1s-60s backoff window on every run.
	return jamfplatform.NewClient(server.URL, "test-id", "test-secret", jamfplatform.WithRetryPolicy(0, 0, 0))
}

// proIDResolverHandler returns an http.Handler that serves the Pro
// /v2/groups/{id} endpoint with the supplied status code and JSON body.
func proIDResolverHandler(t *testing.T, status int, body string) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/groups/") {
			t.Errorf("unexpected request path %q (expected '/groups/'-containing path)", r.URL.Path)
			http.Error(w, "unexpected path", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if body != "" {
			_, _ = w.Write([]byte(body))
		}
	})
}

func countSeverity(d diag.Diagnostics, sev diag.Severity) int {
	n := 0
	for _, e := range d {
		if e.Severity() == sev {
			n++
		}
	}
	return n
}

func TestResolveJamfProID_Success(t *testing.T) {
	client := newProIDMockClient(t, proIDResolverHandler(t, http.StatusOK,
		`{"groupJamfProId":"42","groupPlatformId":"plat-uuid","groupName":"All Macs","groupType":"COMPUTER","smart":true,"membershipCount":3}`,
	))
	pd := providerdata.New(client)
	id, diags := resolveJamfProID(context.Background(), pro.New(client), pd, "plat-uuid")
	if diags.HasError() {
		t.Fatalf("unexpected error diagnostics: %v", diags)
	}
	if countSeverity(diags, diag.SeverityWarning) != 0 {
		t.Errorf("expected 0 warnings on success, got %d (%v)", countSeverity(diags, diag.SeverityWarning), diags)
	}
	if id.IsNull() {
		t.Fatal("expected non-null jamf_pro_id on success")
	}
	if id.ValueString() != "42" {
		t.Errorf("expected jamf_pro_id %q, got %q", "42", id.ValueString())
	}
}

func TestResolveJamfProID_Forbidden_NullsAndWarnsOnce(t *testing.T) {
	client := newProIDMockClient(t, proIDResolverHandler(t, http.StatusForbidden, `{"errors":[{"code":"Forbidden"}]}`))
	pd := providerdata.New(client)

	id, diags := resolveJamfProID(context.Background(), pro.New(client), pd, "plat-uuid")
	if diags.HasError() {
		t.Fatalf("403 must not produce an error diagnostic; got %v", diags)
	}
	if !id.IsNull() {
		t.Error("403 must null the jamf_pro_id attribute")
	}
	if countSeverity(diags, diag.SeverityWarning) != 1 {
		t.Fatalf("expected exactly 1 warning on first 403, got %d (%v)", countSeverity(diags, diag.SeverityWarning), diags)
	}
	if !strings.Contains(diags[0].Summary(), "Read Groups") {
		t.Errorf("warning summary should reference 'Read Groups', got %q", diags[0].Summary())
	}

	_, diags2 := resolveJamfProID(context.Background(), pro.New(client), pd, "plat-uuid-2")
	if countSeverity(diags2, diag.SeverityWarning) != 0 {
		t.Errorf("repeat 403 on the same Data should suppress the warning, got %d (%v)", countSeverity(diags2, diag.SeverityWarning), diags2)
	}
}

func TestResolveJamfProID_NotFound_NullsSilently(t *testing.T) {
	client := newProIDMockClient(t, proIDResolverHandler(t, http.StatusNotFound, `{"errors":[{"code":"NotFound"}]}`))
	pd := providerdata.New(client)
	id, diags := resolveJamfProID(context.Background(), pro.New(client), pd, "missing-uuid")
	if diags.HasError() {
		t.Fatalf("404 must not produce an error diagnostic; got %v", diags)
	}
	if !id.IsNull() {
		t.Error("404 must null the jamf_pro_id attribute")
	}
	if countSeverity(diags, diag.SeverityWarning) != 0 {
		t.Errorf("404 must not produce a warning, got %d (%v)", countSeverity(diags, diag.SeverityWarning), diags)
	}
}

func TestResolveJamfProID_OtherError_NullsAndWarnsOnce(t *testing.T) {
	client := newProIDMockClient(t, proIDResolverHandler(t, http.StatusBadGateway, `{"errors":[{"code":"BadGateway"}]}`))
	pd := providerdata.New(client)

	id, diags := resolveJamfProID(context.Background(), pro.New(client), pd, "plat-uuid")
	if diags.HasError() {
		t.Fatalf("502 must not produce an error diagnostic — it should degrade to warning to avoid orphaning a successful Platform Create; got %v", diags)
	}
	if !id.IsNull() {
		t.Error("502 must null the jamf_pro_id attribute")
	}
	if countSeverity(diags, diag.SeverityWarning) != 1 {
		t.Fatalf("expected exactly 1 warning on first transient failure, got %d (%v)", countSeverity(diags, diag.SeverityWarning), diags)
	}
	if !strings.Contains(diags[0].Summary(), "Failed to resolve") {
		t.Errorf("warning summary should mention failure to resolve, got %q", diags[0].Summary())
	}

	_, diags2 := resolveJamfProID(context.Background(), pro.New(client), pd, "plat-uuid-2")
	if countSeverity(diags2, diag.SeverityWarning) != 0 {
		t.Errorf("repeat transient failure on the same Data should suppress the warning, got %d (%v)", countSeverity(diags2, diag.SeverityWarning), diags2)
	}
}

// TestResolveJamfProID_ForbiddenWarning_LatchesAcrossSimulatedLifecycle
// confirms that when a single provider invocation hits Create, then Read, then
// Update, then a data-source Read against the same 403-returning Pro endpoint,
// the missing-privilege warning is emitted exactly once — and never as an
// error. This is the end-to-end safety net for the Create-state-leak class of
// bug: if any future change makes the resolver return an error diagnostic on a
// non-success status, this test fails before the Platform Create result can be
// silently discarded by the framework.
func TestResolveJamfProID_ForbiddenWarning_LatchesAcrossSimulatedLifecycle(t *testing.T) {
	client := newProIDMockClient(t, proIDResolverHandler(t, http.StatusForbidden, `{"errors":[{"code":"Forbidden"}]}`))
	pd := providerdata.New(client)
	proClient := pro.New(client)

	totalWarnings := 0
	totalErrors := 0
	for _, label := range []string{"create", "read", "update", "datasource_read"} {
		id, diags := resolveJamfProID(context.Background(), proClient, pd, "plat-uuid-"+label)
		if diags.HasError() {
			t.Fatalf("%s call must never return an error diagnostic (would orphan state on Create); got %v", label, diags)
		}
		if !id.IsNull() {
			t.Errorf("%s call must null jamf_pro_id on 403; got %q", label, id.ValueString())
		}
		totalWarnings += countSeverity(diags, diag.SeverityWarning)
		totalErrors += countSeverity(diags, diag.SeverityError)
	}

	if totalErrors != 0 {
		t.Fatalf("expected 0 error diagnostics across the simulated lifecycle, got %d", totalErrors)
	}
	if totalWarnings != 1 {
		t.Errorf("expected exactly 1 missing-privilege warning across the entire lifecycle (latched via FiredOnce), got %d", totalWarnings)
	}
}

func TestResolveJamfProID_NilGuards(t *testing.T) {
	id, diags := resolveJamfProID(context.Background(), nil, nil, "plat-uuid")
	if diags.HasError() || len(diags) != 0 {
		t.Fatalf("nil client/pd should yield empty diagnostics, got %v", diags)
	}
	if !id.IsNull() {
		t.Error("nil client/pd must null the attribute")
	}

	id2, diags2 := resolveJamfProID(context.Background(), nil, nil, "")
	if diags2.HasError() || len(diags2) != 0 {
		t.Fatalf("empty platform id should yield empty diagnostics, got %v", diags2)
	}
	if !id2.IsNull() {
		t.Error("empty platform id must null the attribute")
	}
}
