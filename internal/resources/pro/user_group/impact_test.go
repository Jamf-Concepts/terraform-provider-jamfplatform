// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package user_group

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func criterion(name, searchType, value string) UserGroupCriterionModel {
	return UserGroupCriterionModel{
		Name:                  types.StringValue(name),
		SearchType:            types.StringValue(searchType),
		Value:                 types.StringValue(value),
		AndOr:                 types.StringValue("and"),
		HasOpeningParenthesis: types.BoolValue(false),
		HasClosingParenthesis: types.BoolValue(false),
	}
}

func TestCriterionListsDiffer(t *testing.T) {
	base := []UserGroupCriterionModel{criterion("Email Address", "like", "@example.com")}

	if criterionListsDiffer(base, base) {
		t.Fatal("identical criteria must not read as changed")
	}
	if !criterionListsDiffer(base, nil) {
		t.Fatal("adding a criterion must read as changed")
	}
	// Field-by-field comparison matters: an edited value changes membership while
	// leaving the number of criteria alone.
	if !criterionListsDiffer(base, []UserGroupCriterionModel{criterion("Email Address", "like", "@other.com")}) {
		t.Fatal("an edited value must read as changed")
	}
	if !criterionListsDiffer(base, []UserGroupCriterionModel{criterion("Email Address", "is", "@example.com")}) {
		t.Fatal("an edited search type must read as changed")
	}
	if !criterionListsDiffer(base, []UserGroupCriterionModel{criterion("Full Name", "like", "@example.com")}) {
		t.Fatal("an edited attribute must read as changed")
	}
}

func TestCriterionListsDifferIgnoresPriority(t *testing.T) {
	// priority is a presentation index; it does not change who is a member.
	a := []UserGroupCriterionModel{criterion("Full Name", "like", "Smith")}
	b := []UserGroupCriterionModel{criterion("Full Name", "like", "Smith")}
	a[0].Priority = types.Int64Value(0)
	b[0].Priority = types.Int64Value(1)
	if criterionListsDiffer(a, b) {
		t.Fatal("a differing priority alone must not read as a membership change")
	}
}
