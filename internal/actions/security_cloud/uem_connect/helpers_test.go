// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package uemconnectactions

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/securitycloud"
	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

func apiError(status int, code, description string) error {
	return &jamfplatform.APIResponseError{
		StatusCode: status,
		Errors:     []jamfplatform.ErrorDetail{{Code: code, Description: description}},
	}
}

func TestAppendInvokeDiagnostics(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantMatched bool
		wantSummary string
		wantDetail  []string
	}{
		{
			name:        "disabled integration names the setting to change",
			err:         apiError(http.StatusConflict, codeConnectorDisabled, "UEM integration is disabled for connector '6a91'"),
			wantMatched: true,
			wantSummary: "UEM Connect is disabled, so it cannot synchronize",
			// The server names the integration's identifier; what the caller needs
			// is the attribute that has to change.
			wantDetail: []string{"enabled = true"},
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
			name:        "a transport error is left to the caller",
			err:         errors.New("connection refused"),
			wantMatched: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var diags diag.Diagnostics
			matched := appendInvokeDiagnostics(&diags, tc.err)

			if matched != tc.wantMatched {
				t.Fatalf("matched = %v, want %v", matched, tc.wantMatched)
			}
			if !tc.wantMatched {
				if diags.HasError() {
					t.Errorf("diagnostics added for an unrecognised error: %v", diags)
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
			if !strings.Contains(d.Detail(), "Reported by Jamf Security Cloud:") {
				t.Errorf("detail does not quote the server's own message:\n%s", d.Detail())
			}
		})
	}
}

// TestNotEntitledUsesTheSDKConstant pins that the one code the SDK generates is
// taken from there rather than restated, and that the one it does not is still
// absent — so if a future spec adds it, this says so.
func TestNotEntitledUsesTheSDKConstant(t *testing.T) {
	if codeNotEntitled != securitycloud.ApiErrorItemCodeNotEntitled {
		t.Errorf("codeNotEntitled = %q; it must come from the SDK constant", codeNotEntitled)
	}

	for _, v := range securitycloud.ApiErrorItemCodeValues() {
		if v == codeConnectorDisabled {
			t.Errorf("the SDK now generates a constant for %q — use it instead of the literal", codeConnectorDisabled)
		}
	}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"404", apiError(http.StatusNotFound, "NOT_FOUND", "gone"), true},
		{"conflict is not a not-found", apiError(http.StatusConflict, codeConnectorDisabled, "disabled"), false},
		{"transport error", errors.New("connection refused"), false},
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

// TestEnsureClientFailsClosed pins that Invoke refuses to run with no client rather
// than panicking on a nil dereference.
func TestEnsureClientFailsClosed(t *testing.T) {
	var a uemConnectAction
	var resp action.InvokeResponse

	if a.ensureClient(&resp) {
		t.Error("ensureClient returned true with no client")
	}
	if !resp.Diagnostics.HasError() {
		t.Error("ensureClient added no diagnostic")
	}
}
