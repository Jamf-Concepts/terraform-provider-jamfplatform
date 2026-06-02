// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package criteria

import (
	"slices"
	"strings"
	"testing"
)

func TestOperators_IncludesDateWindowOperators(t *testing.T) {
	// The full set (device/computer groups + searches) keeps the date-window
	// operators; only the user-group subset drops them.
	for _, want := range []string{"in less than x days", "in more than x days"} {
		if !slices.Contains(Operators, want) {
			t.Errorf("Operators missing %q", want)
		}
	}
}

func TestOperators_NoDuplicates(t *testing.T) {
	seen := map[string]bool{}
	for _, op := range Operators {
		if seen[op] {
			t.Errorf("duplicate operator %q", op)
		}
		seen[op] = true
	}
}

func TestWithout_RemovesNamedOperators(t *testing.T) {
	got := Without("in less than x days", "in more than x days")
	if len(got) != len(Operators)-2 {
		t.Fatalf("expected %d operators, got %d", len(Operators)-2, len(got))
	}
	for _, dropped := range []string{"in less than x days", "in more than x days"} {
		if slices.Contains(got, dropped) {
			t.Errorf("Without did not remove %q", dropped)
		}
	}
	// Order is preserved and a retained operator survives.
	if !slices.Contains(got, "after (yyyy-mm-dd)") {
		t.Errorf("Without dropped an operator it should have kept")
	}
}

func TestDescription_ListsEveryGivenOperatorAsInlineCode(t *testing.T) {
	subset := Without("in less than x days", "in more than x days")
	desc := Description(subset)
	for _, op := range subset {
		if !strings.Contains(desc, "`"+op+"`") {
			t.Errorf("description missing inline-coded operator %q", op)
		}
	}
	// The dropped operators must NOT appear in the subset description.
	if strings.Contains(desc, "`in less than x days`") {
		t.Errorf("description for subset must not list a removed operator")
	}
}
