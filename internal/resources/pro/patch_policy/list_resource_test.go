// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package patch_policy

import (
	"errors"
	"strings"
	"testing"
)

// TestSkippedPatchPoliciesWarningDetail is the guard on the one thing that
// makes a dropped policy visible. Enumeration runs on the Pro v2 collection and
// hydration on the ProClassic by-id path, so a policy the classic read cannot
// return is dropped from the result set; the trailing warning is the only place
// that omission is stated, and an operator can act on it only if it names the
// policy and repeats the error. A detail that merely counted the omissions
// would satisfy a "there is a warning" assertion while telling nobody which
// policy went missing.
func TestSkippedPatchPoliciesWarningDetail(t *testing.T) {
	detail := skippedPatchPoliciesWarningDetail([]skippedPatchPolicy{
		{id: "12", name: "Firefox 130", err: errors.New("404 not found")},
		{id: "34", name: "Zoom 6.1", err: errors.New("403 forbidden")},
	})

	for _, want := range []string{
		"12", "Firefox 130", "404 not found",
		"34", "Zoom 6.1", "403 forbidden",
	} {
		if !strings.Contains(detail, want) {
			t.Errorf("warning detail omits %q; an operator cannot act on it:\n%s", want, detail)
		}
	}
	if !strings.Contains(detail, "2 patch polic") {
		t.Errorf("warning detail does not state how many policies were omitted:\n%s", detail)
	}
}
