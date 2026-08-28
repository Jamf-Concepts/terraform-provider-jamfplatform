// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ztna_grouped_gateway

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
	return fmt.Errorf("CreateZtnaGroupedGatewayV1: %w", &jamfplatform.APIResponseError{
		StatusCode: status,
		Method:     "POST",
		URL:        "/api/securitycloud/v1/ztna/grouped-gateways",
		Errors: []jamfplatform.ErrorDetail{
			{Code: code, Description: description},
		},
	})
}

func TestAppendWriteDiagnostics_MapsMembershipCodes(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantPath path.Path
		wantText string
	}{
		{
			name:     "mixed tunnel types points at the member list",
			err:      apiError(422, codeMixedTunnelTypes, "All member gateways must have the same tunnel type (all IPSec or all non-IPSec)."),
			wantPath: path.Root("gateway_ids"),
			wantText: "same form",
		},
		{
			name:     "shared member points at the member list",
			err:      apiError(422, codeSharedGatewayMember, "Grouped Gateway members must be dedicated gateways."),
			wantPath: path.Root("gateway_ids"),
			wantText: "shared_gateways",
		},
		{
			// A member that does not exist and a member owned by someone else share
			// one code, so the diagnostic has to offer both readings.
			name:     "missing member points at the member list",
			err:      apiError(422, codeGatewayNotFound, "A gateway in gatewayIds does not exist or is not accessible to this customer."),
			wantPath: path.Root("gateway_ids"),
			wantText: "belongs to another customer",
		},
		{
			name:     "unknown id points at tenant_ids",
			err:      apiError(400, codeBadRequest, "No mapping found for one of the supplied ids"),
			wantPath: path.Root("tenant_ids"),
			wantText: "same organization",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var diags diag.Diagnostics
			if !appendWriteDiagnostics(&diags, tc.err) {
				t.Fatalf("expected the error to be recognised, diagnostics: %v", diags)
			}
			withPath, ok := diags[0].(diag.DiagnosticWithPath)
			if !ok {
				t.Fatalf("diagnostic %T carries no path; it must point at the attribute that caused it", diags[0])
			}
			if !withPath.Path().Equal(tc.wantPath) {
				t.Errorf("path = %s, want %s", withPath.Path(), tc.wantPath)
			}
			if !strings.Contains(diags[0].Detail(), tc.wantText) {
				t.Errorf("detail %q does not explain the fix (looking for %q)", diags[0].Detail(), tc.wantText)
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

func TestAppendDeleteDiagnostics_ConflictExplainsOrdering(t *testing.T) {
	var diags diag.Diagnostics
	err := fmt.Errorf("DeleteZtnaGroupedGatewayV1: %w", &jamfplatform.APIResponseError{
		StatusCode: http.StatusConflict,
		Method:     "DELETE",
		URL:        "/api/securitycloud/v1/ztna/grouped-gateways/b6ed74d2",
	})

	if !appendDeleteDiagnostics(&diags, err) {
		t.Fatalf("a 409 on delete must be recognised, diagnostics: %v", diags)
	}
	if !strings.Contains(diags[0].Detail(), "separate apply") {
		t.Errorf("detail does not explain the ordering fix:\n%s", diags[0].Detail())
	}
}

func TestAppendDeleteDiagnostics_OtherStatusesFallThrough(t *testing.T) {
	var diags diag.Diagnostics
	if appendDeleteDiagnostics(&diags, apiError(http.StatusNotFound, "NOT_FOUND", "gone")) {
		t.Error("a 404 must not be handled here")
	}
	if appendDeleteDiagnostics(&diags, errors.New("connection reset")) {
		t.Error("a non-API error must not be handled here")
	}
}
