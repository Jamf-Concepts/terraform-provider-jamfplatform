// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package device_group

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/providerdata"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

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
	client := testhelpers.NewMockClient(t, proIDResolverHandler(t, http.StatusOK,
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
	client := testhelpers.NewMockClient(t, proIDResolverHandler(t, http.StatusForbidden, `{"errors":[{"code":"Forbidden"}]}`))
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
	client := testhelpers.NewMockClient(t, proIDResolverHandler(t, http.StatusNotFound, `{"errors":[{"code":"NotFound"}]}`))
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

func TestResolveJamfProID_OtherError_ReturnsError(t *testing.T) {
	client := testhelpers.NewMockClient(t, proIDResolverHandler(t, http.StatusBadGateway, `{"errors":[{"code":"BadGateway"}]}`))
	pd := providerdata.New(client)
	id, diags := resolveJamfProID(context.Background(), pro.New(client), pd, "plat-uuid")
	if !diags.HasError() {
		t.Fatal("502 must produce an error diagnostic")
	}
	if !id.IsNull() {
		t.Error("error path should still null the attribute")
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
