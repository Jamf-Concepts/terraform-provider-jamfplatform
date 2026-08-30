// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package policy

import "testing"

// TestErrorCodes pins each code against the body captured during the wire probe against the EU
// sandbox on 2026-08-30. A code the platform renames breaks translation silently otherwise: the
// switch in appendWriteDiagnostics simply stops matching and the raw error surfaces instead.
func TestErrorCodes(t *testing.T) {
	want := map[string]string{
		"codeToolIDUnknown":             "TOOL_ID_UNKNOWN",
		"codeSchemaVersionUnknown":      "SCHEMA_VERSION_UNKNOWN",
		"codeSchemaValidationFailed":    "SCHEMA_VALIDATION_FAILED",
		"codeValidationFailed":          "VALIDATION_FAILED",
		"codePolicyNotFound":            "POLICY_NOT_FOUND",
		"codeNoDraftToPublish":          "NO_DRAFT_TO_PUBLISH",
		"codeRequestContextNotProvided": "REQUEST_CONTEXT_NOT_PROVIDED",
	}
	got := map[string]string{
		"codeToolIDUnknown":             codeToolIDUnknown,
		"codeSchemaVersionUnknown":      codeSchemaVersionUnknown,
		"codeSchemaValidationFailed":    codeSchemaValidationFailed,
		"codeValidationFailed":          codeValidationFailed,
		"codePolicyNotFound":            codePolicyNotFound,
		"codeNoDraftToPublish":          codeNoDraftToPublish,
		"codeRequestContextNotProvided": codeRequestContextNotProvided,
	}

	for name, expected := range want {
		if got[name] != expected {
			t.Errorf("%s = %q, want %q", name, got[name], expected)
		}
	}
	if len(got) != len(want) {
		t.Errorf("the test covers %d codes but the package declares %d", len(want), len(got))
	}
}
