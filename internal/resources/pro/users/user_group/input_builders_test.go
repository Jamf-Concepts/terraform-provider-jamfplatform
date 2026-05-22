// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package user_group

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildUserGroupInput_Smart_PrioritiesAndDefaults(t *testing.T) {
	plan := UserGroupResourceModel{
		Name:                     types.StringValue("Marketing Apple IDs"),
		GroupType:                types.StringValue("smart"),
		NotifyOnMembershipChange: types.BoolValue(false),
		SiteID:                   types.StringValue("-1"),
		Criteria: []UserGroupCriterionModel{
			{
				Name:                  types.StringValue("User Group"),
				SearchType:            types.StringValue("member of"),
				Value:                 types.StringValue("All Managed Apple IDs"),
				Priority:              types.Int64Null(),
				AndOr:                 types.StringNull(),
				HasOpeningParenthesis: types.BoolNull(),
				HasClosingParenthesis: types.BoolNull(),
			},
			{
				Name:       types.StringValue("VPP Invitation Status"),
				SearchType: types.StringValue("is"),
				Value:      types.StringValue("Associated"),
				Priority:   types.Int64Value(1),
				AndOr:      types.StringValue("and"),
			},
		},
		Members: types.SetNull(types.StringType),
	}

	out, diags := buildUserGroupInput(context.Background(), plan)
	if diags.HasError() {
		t.Fatalf("diagnostics: %v", diags)
	}
	if out == nil {
		t.Fatal("nil output")
	}
	if out.IsSmart == nil || *out.IsSmart != true {
		t.Errorf("IsSmart must be true for smart group")
	}
	if out.Users != nil {
		t.Errorf("Users must be nil for smart group, got %v", out.Users)
	}
	if out.Criteria == nil || out.Criteria.Criterion == nil {
		t.Fatal("Criteria wrapper missing")
	}
	crits := *out.Criteria.Criterion
	if len(crits) != 2 {
		t.Fatalf("expected 2 criteria, got %d", len(crits))
	}
	// Priority filled from index when omitted.
	if crits[0].Priority == nil || *crits[0].Priority != 0 {
		t.Errorf("priority[0] expected 0 (index default), got %v", crits[0].Priority)
	}
	if crits[1].Priority == nil || *crits[1].Priority != 1 {
		t.Errorf("priority[1] expected 1, got %v", crits[1].Priority)
	}
	// AndOr default "and" when null.
	if crits[0].AndOr == nil || *crits[0].AndOr != "and" {
		t.Errorf("and_or default 'and' expected, got %v", crits[0].AndOr)
	}
	// Site sentinel passed through.
	if out.Site == nil || out.Site.ID == nil || *out.Site.ID != -1 {
		t.Errorf("Site=-1 expected, got %v", out.Site)
	}
}

func TestBuildUserGroupInput_Static_WithMembers(t *testing.T) {
	members, diags := types.SetValueFrom(context.Background(), types.StringType, []string{"9", "12"})
	if diags.HasError() {
		t.Fatalf("set value: %v", diags)
	}
	plan := UserGroupResourceModel{
		Name:                     types.StringValue("Excluded Users"),
		GroupType:                types.StringValue("static"),
		NotifyOnMembershipChange: types.BoolValue(true),
		SiteID:                   types.StringValue("-1"),
		Members:                  members,
	}

	out, diags := buildUserGroupInput(context.Background(), plan)
	if diags.HasError() {
		t.Fatalf("diagnostics: %v", diags)
	}
	if out == nil {
		t.Fatal("nil output")
	}
	if out.IsSmart == nil || *out.IsSmart != false {
		t.Errorf("IsSmart must be false for static group")
	}
	if out.Criteria != nil {
		t.Errorf("Criteria must be nil for static group, got %v", out.Criteria)
	}
	if out.Users == nil || out.Users.User == nil {
		t.Fatal("Users wrapper missing")
	}
	got := *out.Users.User
	if len(got) != 2 {
		t.Fatalf("expected 2 members, got %d", len(got))
	}
	for _, u := range got {
		if u.ID == nil {
			t.Errorf("member ID must not be nil")
		}
	}
}

func TestBuildUserGroupInput_Static_NullMembers_LeavesUsersNil(t *testing.T) {
	plan := UserGroupResourceModel{
		Name:      types.StringValue("untouched"),
		GroupType: types.StringValue("static"),
		Members:   types.SetNull(types.StringType),
	}
	out, diags := buildUserGroupInput(context.Background(), plan)
	if diags.HasError() {
		t.Fatalf("diagnostics: %v", diags)
	}
	if out.Users != nil {
		t.Errorf("Users must be nil when members is null (do not touch server-side membership), got %v", out.Users)
	}
}

func TestBuildUserGroupInput_Static_NonIntMember_Errors(t *testing.T) {
	members, diags := types.SetValueFrom(context.Background(), types.StringType, []string{"not-an-int"})
	if diags.HasError() {
		t.Fatalf("set value: %v", diags)
	}
	plan := UserGroupResourceModel{
		Name:      types.StringValue("bad"),
		GroupType: types.StringValue("static"),
		Members:   members,
	}
	_, diags = buildUserGroupInput(context.Background(), plan)
	if !diags.HasError() {
		t.Fatal("expected error for non-integer member")
	}
}

func TestBuildCriteriaWrapper_SortsByPriority(t *testing.T) {
	crits := []UserGroupCriterionModel{
		{Name: types.StringValue("b"), SearchType: types.StringValue("is"), Value: types.StringValue("y"), Priority: types.Int64Value(2)},
		{Name: types.StringValue("a"), SearchType: types.StringValue("is"), Value: types.StringValue("x"), Priority: types.Int64Value(0)},
		{Name: types.StringValue("c"), SearchType: types.StringValue("is"), Value: types.StringValue("z"), Priority: types.Int64Value(1)},
	}
	w := buildCriteriaWrapper(crits)
	if w == nil || w.Criterion == nil {
		t.Fatal("nil wrapper")
	}
	got := *w.Criterion
	priorities := make([]int, len(got))
	for i, c := range got {
		if c.Priority == nil {
			t.Fatalf("priority[%d] is nil", i)
		}
		priorities[i] = *c.Priority
	}
	for i := 1; i < len(priorities); i++ {
		if priorities[i-1] > priorities[i] {
			t.Errorf("not sorted: %v", priorities)
			break
		}
	}
}
