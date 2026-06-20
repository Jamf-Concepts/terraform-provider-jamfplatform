// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package advanced_volume_purchasing_content_search

import (
	"slices"
	"testing"
)

// TestValidOperators_ContentSubset pins the Volume-Purchasing-Content operator
// vocabulary to the 8 string + numeric operators the UI offers, and asserts the
// 13 membership/date/inequality operators that never apply to content attributes
// are excluded.
func TestValidOperators_ContentSubset(t *testing.T) {
	// Order follows the canonical criteria.Operators ordering preserved by Without.
	want := []string{
		"is", "is not", "like", "not like",
		"more than", "less than",
		"matches regex", "does not match regex",
	}
	if !slices.Equal(ValidOperators, want) {
		t.Fatalf("ValidOperators mismatch:\n got %v\nwant %v", ValidOperators, want)
	}

	for _, excluded := range []string{
		"has", "does not have", "member of", "not member of",
		"before (yyyy-mm-dd)", "after (yyyy-mm-dd)",
		"in less than x days", "in more than x days",
		"more than x days ago", "less than x days ago",
		"greater than", "greater than or equal", "less than or equal",
	} {
		if slices.Contains(ValidOperators, excluded) {
			t.Errorf("operator %q must be excluded from the content subset", excluded)
		}
	}
}
