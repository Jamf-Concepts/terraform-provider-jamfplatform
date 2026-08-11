// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package device_group

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestGroupLabelMatchesTheAdminUIWording(t *testing.T) {
	tests := []struct {
		groupType, deviceType, want string
	}{
		{"smart", "computer", "smart computer group"},
		{"static", "computer", "static computer group"},
		{"smart", "mobile", "smart mobile device group"},
		{"static", "mobile", "static mobile device group"},
		// An unset or unexpected value must not produce nonsense.
		{"", "", "static computer group"},
	}
	for _, tc := range tests {
		got := groupLabel(DeviceGroupResourceModel{
			GroupType:  types.StringValue(tc.groupType),
			DeviceType: types.StringValue(tc.deviceType),
		})
		if got != tc.want {
			t.Errorf("groupLabel(%q,%q) = %q, want %q", tc.groupType, tc.deviceType, got, tc.want)
		}
	}
}

func TestMembershipNoun(t *testing.T) {
	if got := membershipNoun("mobile"); got != "mobile devices" {
		t.Errorf("mobile noun = %q", got)
	}
	if got := membershipNoun("computer"); got != "computers" {
		t.Errorf("computer noun = %q", got)
	}
}

func criterion(name, op, value string) DeviceGroupCriteriaModel {
	return DeviceGroupCriteriaModel{
		AttributeName:         types.StringValue(name),
		Operator:              types.StringValue(op),
		AttributeValue:        types.StringValue(value),
		JoinType:              types.StringValue("and"),
		HasOpeningParenthesis: types.BoolValue(false),
		HasClosingParenthesis: types.BoolValue(false),
	}
}

func TestCriteriaDiffer(t *testing.T) {
	base := []DeviceGroupCriteriaModel{criterion("Operating System Version", "like", "15.")}

	if criteriaDiffer(base, base) {
		t.Fatal("identical criteria must not read as changed")
	}
	if !criteriaDiffer(base, nil) {
		t.Fatal("adding a criterion must read as changed")
	}
	// The point of comparing field by field rather than by length: an edited value
	// changes membership without changing how many criteria there are.
	edited := []DeviceGroupCriteriaModel{criterion("Operating System Version", "like", "14.")}
	if !criteriaDiffer(base, edited) {
		t.Fatal("an edited value must read as changed")
	}
	reop := []DeviceGroupCriteriaModel{criterion("Operating System Version", "is", "15.")}
	if !criteriaDiffer(base, reop) {
		t.Fatal("an edited operator must read as changed")
	}
	renamed := []DeviceGroupCriteriaModel{criterion("Model", "like", "15.")}
	if !criteriaDiffer(base, renamed) {
		t.Fatal("an edited attribute must read as changed")
	}
}

func TestCriteriaDifferIgnoresOrderField(t *testing.T) {
	// order is a presentation index the provider assigns; it does not affect
	// membership on its own, so it is deliberately not compared.
	a := []DeviceGroupCriteriaModel{criterion("Model", "like", "MacBook")}
	b := []DeviceGroupCriteriaModel{criterion("Model", "like", "MacBook")}
	a[0].Order = types.Int64Value(0)
	b[0].Order = types.Int64Value(1)
	if criteriaDiffer(a, b) {
		t.Fatal("a differing order index alone must not read as a membership change")
	}
}
