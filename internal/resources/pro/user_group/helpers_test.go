// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package user_group

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestValidateUserGroupPlan_Smart_RequiresCriteria(t *testing.T) {
	plan := &UserGroupResourceModel{
		GroupType: types.StringValue("smart"),
		Members:   types.SetNull(types.StringType),
	}
	if err := validateUserGroupPlan(plan); err == nil {
		t.Fatal("expected error when smart group has no criteria")
	}
}

func TestValidateUserGroupPlan_Smart_RejectsMembers(t *testing.T) {
	members, _ := types.SetValueFrom(context.Background(), types.StringType, []string{"1"})
	plan := &UserGroupResourceModel{
		GroupType: types.StringValue("smart"),
		Members:   members,
		Criteria: []UserGroupCriterionModel{
			{Name: types.StringValue("x"), SearchType: types.StringValue("is"), Value: types.StringValue("y")},
		},
	}
	if err := validateUserGroupPlan(plan); err == nil {
		t.Fatal("expected error when smart group sets members")
	}
}

func TestValidateUserGroupPlan_Static_RejectsCriteria(t *testing.T) {
	plan := &UserGroupResourceModel{
		GroupType: types.StringValue("static"),
		Members:   types.SetNull(types.StringType),
		Criteria: []UserGroupCriterionModel{
			{Name: types.StringValue("x"), SearchType: types.StringValue("is"), Value: types.StringValue("y")},
		},
	}
	if err := validateUserGroupPlan(plan); err == nil {
		t.Fatal("expected error when static group sets criteria")
	}
}

func TestValidateUserGroupPlan_Static_AllowsOmittedMembers(t *testing.T) {
	plan := &UserGroupResourceModel{
		GroupType: types.StringValue("static"),
		Members:   types.SetNull(types.StringType),
	}
	if err := validateUserGroupPlan(plan); err != nil {
		t.Fatalf("static + null members must validate, got %v", err)
	}
}

func TestValidateUserGroupPlan_EmptyGroupType_Errors(t *testing.T) {
	plan := &UserGroupResourceModel{GroupType: types.StringValue("")}
	if err := validateUserGroupPlan(plan); err == nil {
		t.Fatal("expected error for empty group_type")
	}
}

func TestValidateUserGroupPlan_UnknownGroupType_Errors(t *testing.T) {
	plan := &UserGroupResourceModel{GroupType: types.StringValue("dynamic")}
	if err := validateUserGroupPlan(plan); err == nil {
		t.Fatal("expected error for unsupported group_type")
	}
}
