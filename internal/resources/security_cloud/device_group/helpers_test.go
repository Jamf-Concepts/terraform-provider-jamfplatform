// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package device_group

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
)

// apiError builds the wrapped error shape the SDK returns, so the test exercises
// the same unwrap path the CRUD callers do. Every body below is copied from the
// EU sandbox probe on 2026-08-29.
func apiError(status int, code, field, description string) error {
	return fmt.Errorf("CreateDeviceGroupV1: %w", &jamfplatform.APIResponseError{
		StatusCode: status,
		Method:     "POST",
		URL:        "/securitycloud/v1/groups",
		Errors: []jamfplatform.ErrorDetail{
			{Code: code, Field: field, Description: description},
		},
	})
}

func TestAppendWriteDiagnostics_MapsCodesToAttributes(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantPath path.Path
		wantText string
	}{
		{
			name:     "duplicate name points at name and says the match is exact",
			err:      apiError(409, codeGroupAlreadyExists, "name", "Group with name 'Executives' already exists for customer 9282."),
			wantPath: path.Root("name"),
			wantText: "comparison is",
		},
		{
			name:     "reserved name points at name and says the group is unmanageable",
			err:      apiError(400, codeReservedGroupName, "name", "Group name 'Default Group' is reserved and cannot be used"),
			wantPath: path.Root("name"),
			wantText: "cannot be managed by",
		},
		{
			name:     "blank name points at name and explains whitespace-only counts",
			err:      apiError(400, codeInvalidField, "name", "Group name must not be blank"),
			wantPath: path.Root("name"),
			wantText: "only of whitespace",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var diags diag.Diagnostics

			if !appendWriteDiagnostics(&diags, tc.err) {
				t.Fatalf("expected the code to be recognised")
			}
			if !diags.HasError() {
				t.Fatalf("expected an error diagnostic, got %v", diags)
			}

			found := false
			for _, d := range diags {
				withPath, ok := d.(diag.DiagnosticWithPath)
				if !ok {
					continue
				}
				if withPath.Path().Equal(tc.wantPath) && strings.Contains(d.Detail(), tc.wantText) {
					found = true
				}
			}
			if !found {
				t.Errorf("no diagnostic at %s containing %q; got %v", tc.wantPath, tc.wantText, diags)
			}
		})
	}
}

// TestAppendWriteDiagnostics_NotEntitledIsNotAttributeScoped pins that the
// entitlement failure is a whole-resource problem rather than a field one — no
// attribute the user could change would fix it.
func TestAppendWriteDiagnostics_NotEntitledIsNotAttributeScoped(t *testing.T) {
	var diags diag.Diagnostics

	if !appendWriteDiagnostics(&diags, apiError(403, codeNotEntitled, "", "Not entitled.")) {
		t.Fatal("NOT_ENTITLED must be recognised")
	}
	for _, d := range diags {
		if _, ok := d.(diag.DiagnosticWithPath); ok {
			t.Errorf("NOT_ENTITLED must not be attached to an attribute path: %v", d)
		}
	}
}

// TestAppendWriteDiagnostics_BadPermissionsIsNotTranslated pins the deliberate
// omission. The gateway returns 403 BAD_PERMISSIONS both for a genuine privilege
// gap and for a route it does not serve at all — a control probe on a bogus path
// returned the identical body — so any wording chosen here would be wrong half the
// time. The raw error is the honest surface.
func TestAppendWriteDiagnostics_BadPermissionsIsNotTranslated(t *testing.T) {
	var diags diag.Diagnostics

	if appendWriteDiagnostics(&diags, apiError(403, codeBadPermissions, "", "The given token was not authorized to access the requested resource.")) {
		t.Error("BAD_PERMISSIONS must fall through to the caller's generic error")
	}
	if diags.HasError() {
		t.Errorf("BAD_PERMISSIONS must add no diagnostics, got %v", diags)
	}
}

// TestAppendWriteDiagnostics_UnrecognisedCodeFallsThrough keeps the caller's
// generic error path reachable for anything new the server starts returning.
func TestAppendWriteDiagnostics_UnrecognisedCodeFallsThrough(t *testing.T) {
	var diags diag.Diagnostics

	if appendWriteDiagnostics(&diags, apiError(500, "SOMETHING_NEW", "", "Unexpected.")) {
		t.Error("an unrecognised code must not report a match")
	}
	if diags.HasError() {
		t.Errorf("an unrecognised code must add no diagnostics, got %v", diags)
	}
}

// TestAppendWriteDiagnostics_NonAPIError guards the nil-unwrap path: a transport
// failure carries no error details at all.
func TestAppendWriteDiagnostics_NonAPIError(t *testing.T) {
	var diags diag.Diagnostics

	if appendWriteDiagnostics(&diags, errors.New("connection reset by peer")) {
		t.Error("a non-API error must not report a match")
	}
	if diags.HasError() {
		t.Errorf("a non-API error must add no diagnostics, got %v", diags)
	}
}
