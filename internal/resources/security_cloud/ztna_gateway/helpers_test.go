// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ztna_gateway

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
)

// apiError builds the wrapped error shape the SDK returns, so the test exercises
// the same unwrap path the CRUD callers do.
func apiError(status int, code, description string) error {
	return fmt.Errorf("CreateZtnaGatewayV1: %w", &jamfplatform.APIResponseError{
		StatusCode: status,
		Method:     "POST",
		URL:        "/api/securitycloud/v1/ztna/gateways",
		Errors: []jamfplatform.ErrorDetail{
			{Code: code, Description: description},
		},
	})
}

func TestAppendWriteDiagnostics_MapsCodes(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantPath *path.Path
		wantText string
	}{
		{
			name:     "type change explains that replacement is required",
			err:      apiError(400, codeGatewayTypeChangeNotSupported, "Gateway type cannot be changed."),
			wantText: "has to be replaced",
		},
		{
			name:     "secret clear points at the secret",
			err:      apiError(400, codeIPSecSecretClearNotSupported, "Clearing the IPSec pre-shared key is not supported."),
			wantPath: new(path.Root("ipsec").AtName("jamf_side").AtName("authentication_secret")),
			wantText: "rotated but never removed",
		},
		{
			name:     "unknown id points at tenant_ids",
			err:      apiError(400, codeBadRequest, "No mapping found for one of the supplied ids"),
			wantPath: new(path.Root("tenant_ids")),
			wantText: "same organization",
		},
		{
			// A quota, not a mistake: no attribute path, because the addresses are
			// computed and there is no input an edit could fix.
			name:     "dedicated IP limit carries no attribute path",
			err:      apiError(409, codeDedicatedIPsLimit, "The dedicated IP address limit has been reached (5 of 5)."),
			wantText: "no dedicated IP addresses left",
		},
		{
			name:     "not entitled names the entitlement",
			err:      apiError(403, codeNotEntitled, "Not entitled."),
			wantText: "does not have the ZTNA surface enabled",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var diags diag.Diagnostics
			if !appendWriteDiagnostics(&diags, tc.err) {
				t.Fatalf("expected the error to be recognised, diagnostics: %v", diags)
			}
			if !diags.HasError() {
				t.Fatal("expected an error diagnostic")
			}
			if !strings.Contains(diags[0].Detail(), tc.wantText) {
				t.Errorf("detail %q does not explain the fix (looking for %q)", diags[0].Detail(), tc.wantText)
			}
			withPath, hasPath := diags[0].(diag.DiagnosticWithPath)
			if tc.wantPath == nil {
				if hasPath {
					t.Errorf("diagnostic should not attach to a single attribute, got %s", withPath.Path())
				}
				return
			}
			if !hasPath {
				t.Fatalf("diagnostic %T carries no path; it must point at the attribute that caused it", diags[0])
			}
			if !withPath.Path().Equal(*tc.wantPath) {
				t.Errorf("path = %s, want %s", withPath.Path(), *tc.wantPath)
			}
		})
	}
}

// TestAppendWriteDiagnostics_UnrecognisedErrorsFallThrough is the important
// negative case: an unmapped code must be reported false so the caller emits the
// raw API error instead of swallowing it.
func TestAppendWriteDiagnostics_UnrecognisedErrorsFallThrough(t *testing.T) {
	var diags diag.Diagnostics

	if appendWriteDiagnostics(&diags, apiError(500, "SOMETHING_NEW", "boom")) {
		t.Error("an unmapped error code must not be treated as handled")
	}
	if appendWriteDiagnostics(&diags, errors.New("connection reset")) {
		t.Error("a non-API error must not be treated as handled")
	}
	if diags.HasError() {
		t.Errorf("unhandled errors must add no diagnostics of their own, got %v", diags)
	}
}

// TestAppendDeleteDiagnostics_ConflictExplainsOrdering covers the delete refusal.
// The server sends a bare 409 with no structured detail, so the diagnostic has to
// supply the whole explanation itself.
func TestAppendDeleteDiagnostics_ConflictExplainsOrdering(t *testing.T) {
	var diags diag.Diagnostics
	err := fmt.Errorf("DeleteZtnaGatewayV1: %w", &jamfplatform.APIResponseError{
		StatusCode: http.StatusConflict,
		Method:     "DELETE",
		URL:        "/api/securitycloud/v1/ztna/gateways/c08e",
	})

	if !appendDeleteDiagnostics(&diags, err) {
		t.Fatalf("a 409 on delete must be recognised, diagnostics: %v", diags)
	}
	if !diags.HasError() {
		t.Fatal("expected an error diagnostic")
	}
	for _, want := range []string{"separate apply", "grouped gateway", "DNS zone"} {
		if !strings.Contains(diags[0].Detail(), want) {
			t.Errorf("detail does not mention %q:\n%s", want, diags[0].Detail())
		}
	}
}

// TestAppendDeleteDiagnostics_OtherStatusesFallThrough keeps the 404 path with the
// caller, which removes the resource from state rather than erroring.
func TestAppendDeleteDiagnostics_OtherStatusesFallThrough(t *testing.T) {
	var diags diag.Diagnostics
	if appendDeleteDiagnostics(&diags, apiError(http.StatusNotFound, "NOT_FOUND", "gone")) {
		t.Error("a 404 must not be handled here")
	}
	if appendDeleteDiagnostics(&diags, errors.New("connection reset")) {
		t.Error("a non-API error must not be handled here")
	}
}

// TestAppendDeleteDiagnostics_NamesAccessPolicies pins the referrer the operator
// cannot resolve by reordering applies.
//
// The provider does not manage ZTNA access policies, so a gateway referenced by one
// has no Terraform-visible dependency edge and no apply ordering releases it. The
// diagnostic used to list only zones and grouped gateways, sending the operator to
// check two things that were not the cause.
func TestAppendDeleteDiagnostics_NamesAccessPolicies(t *testing.T) {
	var diags diag.Diagnostics

	if !appendDeleteDiagnostics(&diags, apiError(http.StatusConflict, "", "")) {
		t.Fatal("a 409 must be recognised")
	}
	if !strings.Contains(diags[0].Detail(), "access polic") {
		t.Errorf("delete diagnostic must name access policies; got %q", diags[0].Detail())
	}
}

// TestAppendDeleteDiagnostics_SurfacesDetailWhenPresent pins that a structured
// detail is passed through rather than contradicted.
//
// The probed referrer cases answered with a bare 409, but the bundled spec
// documents per-referrer codes with remediation text. If the endpoint starts
// sending one, the operator must see it — the old wording asserted the body said
// nothing while never reading it.
func TestAppendDeleteDiagnostics_SurfacesDetailWhenPresent(t *testing.T) {
	var diags diag.Diagnostics

	if !appendDeleteDiagnostics(&diags, apiError(http.StatusConflict, "REFERENCED_BY_ACCESS_POLICIES",
		"Disconnect this from all policies, then try again.")) {
		t.Fatal("a 409 must be recognised")
	}
	if !strings.Contains(diags[0].Detail(), "Disconnect this from all policies") {
		t.Errorf("delete diagnostic must surface the server's own remedy; got %q", diags[0].Detail())
	}
}

// TestReportedDetails covers the shapes the delete conflict can carry, including
// the bare 409 the wire has actually been seen to send.
func TestReportedDetails(t *testing.T) {
	bare := jamfplatform.AsAPIError(apiError(http.StatusConflict, "", ""))
	if got := reportedDetails(bare); got != "" {
		t.Errorf("a detail with no description must add nothing, got %q", got)
	}

	withDetail := jamfplatform.AsAPIError(apiError(http.StatusConflict, "SOME_CODE", "Because reasons."))
	if got := reportedDetails(withDetail); !strings.Contains(got, "Because reasons.") {
		t.Errorf("reportedDetails = %q, want it to carry the description", got)
	}
}
