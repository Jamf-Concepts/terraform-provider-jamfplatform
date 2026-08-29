// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package device_group

import "testing"

// TestErrorCodes pins each restated literal against the body captured during the
// wire probe. The SDK's generated ApiErrorItemCode enum is the DNS namespace's
// error schema and carries none of these, so there is no constant to key on and a
// typo would otherwise only show up as a diagnostic that silently never fires.
func TestErrorCodes(t *testing.T) {
	cases := map[string]string{
		"codeGroupAlreadyExists": codeGroupAlreadyExists,
		"codeReservedGroupName":  codeReservedGroupName,
		"codeGroupNotFound":      codeGroupNotFound,
		"codeInvalidField":       codeInvalidField,
		"codeNotEntitled":        codeNotEntitled,
		"codeBadPermissions":     codeBadPermissions,
	}
	want := map[string]string{
		"codeGroupAlreadyExists": "GROUP_ALREADY_EXISTS",
		"codeReservedGroupName":  "RESERVED_GROUP_NAME",
		"codeGroupNotFound":      "GROUP_NOT_FOUND",
		"codeInvalidField":       "INVALID_FIELD",
		"codeNotEntitled":        "NOT_ENTITLED",
		"codeBadPermissions":     "BAD_PERMISSIONS",
	}

	for name, got := range cases {
		if got != want[name] {
			t.Errorf("%s = %q, want %q", name, got, want[name])
		}
	}
}

// TestDefaultGroupName pins the reserved name the validator compares against. Get
// this wrong and the plan-time refusal stops firing, and the mistake resurfaces as
// a mid-apply 400.
func TestDefaultGroupName(t *testing.T) {
	if defaultGroupName != "Default Group" {
		t.Errorf("defaultGroupName = %q, want %q", defaultGroupName, "Default Group")
	}
}
