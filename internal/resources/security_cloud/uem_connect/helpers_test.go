// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package uem_connect

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// apiError builds an error shaped the way the SDK surfaces a Jamf Security Cloud
// failure, so the translation can be tested without a live tenant.
func apiError(status int, code, description string) error {
	return &jamfplatform.APIResponseError{
		StatusCode: status,
		Errors:     []jamfplatform.ErrorDetail{{Code: code, Description: description}},
	}
}

func TestAppendCreateDiagnostics(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		wantMatched   bool
		wantSummary   string
		wantAttribute string
		wantDetail    []string
	}{
		{
			name:        "one per tenant",
			err:         apiError(http.StatusConflict, codeConfigAlreadyExists, "A connector already exists for the customer with an incompatible UEM vendor."),
			wantMatched: true,
			wantSummary: "A UEM Connect integration already exists for this tenant",
			// The server blames a vendor mismatch, which is not the cause; the
			// diagnostic has to say so or the user goes looking for the wrong thing.
			wantDetail: []string{"Import the existing integration", "whatever vendor"},
		},
		{
			name:        "connection failure names all three causes",
			err:         apiError(http.StatusBadRequest, codeConnectionFailed, "UEM connection failed."),
			wantMatched: true,
			wantSummary: "Jamf Security Cloud could not reach the Jamf Pro instance",
			wantDetail:  []string{"address is wrong", "not reachable", "credentials are not valid"},
		},
		{
			name:          "group identifier format is attributed to the attribute",
			err:           apiError(http.StatusUnprocessableEntity, codeValidationFailed, "Group ID of JamfPro must start with 'mobile_' or 'computer_' followed by numbers. Invalid group ID: 999999"),
			wantMatched:   true,
			wantSummary:   "Invalid Jamf Pro group identifier",
			wantAttribute: "group_membership_mapping.mappings",
		},
		{
			name:        "other validation failures fall through to a generic error",
			err:         apiError(http.StatusUnprocessableEntity, codeValidationFailed, ": invalid auth configuration for Jamf PRO"),
			wantMatched: true,
			wantSummary: "Jamf Security Cloud rejected the UEM Connect integration",
		},
		{
			name:        "not entitled",
			err:         apiError(http.StatusForbidden, codeNotEntitled, "Not entitled."),
			wantMatched: true,
			wantSummary: "Tenant not entitled to Jamf Security Cloud UEM Connect",
		},
		{
			name:        "an unrecognised code is left to the caller",
			err:         apiError(http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred"),
			wantMatched: false,
		},
		{
			name:        "a non-API error is left to the caller",
			err:         errors.New("dial tcp: connection refused"),
			wantMatched: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var diags diag.Diagnostics
			matched := appendCreateDiagnostics(&diags, tc.err)

			if matched != tc.wantMatched {
				t.Fatalf("matched = %v, want %v", matched, tc.wantMatched)
			}
			if !tc.wantMatched {
				if diags.HasError() {
					t.Errorf("diagnostics were added for an unrecognised error: %v", diags)
				}
				return
			}
			if !diags.HasError() {
				t.Fatal("no diagnostic was added")
			}

			d := diags.Errors()[0]
			if d.Summary() != tc.wantSummary {
				t.Errorf("summary = %q, want %q", d.Summary(), tc.wantSummary)
			}
			for _, fragment := range tc.wantDetail {
				if !strings.Contains(d.Detail(), fragment) {
					t.Errorf("detail does not mention %q:\n%s", fragment, d.Detail())
				}
			}
			if tc.wantAttribute != "" {
				withPath, ok := d.(diag.DiagnosticWithPath)
				if !ok {
					t.Fatalf("diagnostic is not attached to an attribute; want %s", tc.wantAttribute)
				}
				if got := withPath.Path().String(); got != tc.wantAttribute {
					t.Errorf("attribute path = %q, want %q", got, tc.wantAttribute)
				}
			}
			// Every translated diagnostic quotes what the server said, so an
			// unexpected variant is still traceable from the error text.
			if !strings.Contains(d.Detail(), "Reported by Jamf Security Cloud:") {
				t.Errorf("detail does not quote the server's own message:\n%s", d.Detail())
			}
		})
	}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"404 status", apiError(http.StatusNotFound, codeNotFound, "Config with ID '0' doesn't exist"), true},
		{"code without the status", apiError(http.StatusBadRequest, codeNotFound, "gone"), true},
		{"a conflict is not a not-found", apiError(http.StatusConflict, codeConfigAlreadyExists, "exists"), false},
		{"a transport error is not a not-found", errors.New("connection refused"), false},
		{"nil", nil, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isNotFound(tc.err); got != tc.want {
				t.Errorf("isNotFound = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestReverseMapping(t *testing.T) {
	in := map[string]string{"a": "1", "b": "2"}
	out := reverseMapping(in)

	if len(out) != 2 || out["1"] != "a" || out["2"] != "b" {
		t.Errorf("reverseMapping = %+v", out)
	}
}

func TestSortedMapKeys(t *testing.T) {
	got := sortedMapKeys(map[string]string{"c": "", "a": "", "b": ""})
	want := []string{"a", "b", "c"}

	if len(got) != len(want) {
		t.Fatalf("got %d keys, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("keys = %v, want %v", got, want)
			break
		}
	}
}
