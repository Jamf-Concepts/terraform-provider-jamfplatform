// Copyright 2026 Jamf Software LLC.

package device_group

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestValidateDeviceGroupPlan_SmartGroup(t *testing.T) {
	plan := &DeviceGroupResourceModel{
		GroupType: types.StringValue("smart"),
		Criteria: []DeviceGroupCriteriaModel{
			{
				AttributeName:  types.StringValue("Device Name"),
				Operator:       types.StringValue("like"),
				AttributeValue: types.StringValue("Mac"),
				JoinType:       types.StringValue("and"),
			},
		},
		Members: types.SetNull(types.StringType),
	}

	if err := validateDeviceGroupPlan(plan); err != nil {
		t.Errorf("expected no error for valid smart group, got: %v", err)
	}
}

func TestValidateDeviceGroupPlan_SmartGroupNoCriteria(t *testing.T) {
	plan := &DeviceGroupResourceModel{
		GroupType: types.StringValue("smart"),
		Criteria:  nil,
		Members:  types.SetNull(types.StringType),
	}

	err := validateDeviceGroupPlan(plan)
	if err == nil {
		t.Fatal("expected error when smart group has no criteria")
	}
	if err.Error() != "criteria must be supplied for smart groups" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateDeviceGroupPlan_SmartGroupWithMembers(t *testing.T) {
	members, _ := types.SetValueFrom(context.TODO(), types.StringType, []string{"device-1"})
	plan := &DeviceGroupResourceModel{
		GroupType: types.StringValue("smart"),
		Criteria: []DeviceGroupCriteriaModel{
			{
				AttributeName:  types.StringValue("Device Name"),
				Operator:       types.StringValue("like"),
				AttributeValue: types.StringValue("Mac"),
			},
		},
		Members: members,
	}

	err := validateDeviceGroupPlan(plan)
	if err == nil {
		t.Fatal("expected error when smart group has members")
	}
	if err.Error() != "members cannot be set for smart groups" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateDeviceGroupPlan_StaticGroup(t *testing.T) {
	members, _ := types.SetValueFrom(context.TODO(), types.StringType, []string{"device-1"})
	plan := &DeviceGroupResourceModel{
		GroupType: types.StringValue("static"),
		Members:  members,
	}

	if err := validateDeviceGroupPlan(plan); err != nil {
		t.Errorf("expected no error for valid static group, got: %v", err)
	}
}

func TestValidateDeviceGroupPlan_StaticGroupWithCriteria(t *testing.T) {
	plan := &DeviceGroupResourceModel{
		GroupType: types.StringValue("static"),
		Criteria: []DeviceGroupCriteriaModel{
			{
				AttributeName:  types.StringValue("Device Name"),
				Operator:       types.StringValue("like"),
				AttributeValue: types.StringValue("Mac"),
			},
		},
		Members: types.SetNull(types.StringType),
	}

	err := validateDeviceGroupPlan(plan)
	if err == nil {
		t.Fatal("expected error when static group has criteria")
	}
	if err.Error() != "criteria cannot be set for static groups" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateDeviceGroupPlan_UnsupportedGroupType(t *testing.T) {
	plan := &DeviceGroupResourceModel{
		GroupType: types.StringValue("dynamic"),
		Members:  types.SetNull(types.StringType),
	}

	err := validateDeviceGroupPlan(plan)
	if err == nil {
		t.Fatal("expected error for unsupported group type")
	}
	if err.Error() != "unsupported group_type \"dynamic\"" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateDeviceGroupPlan_EmptyGroupType(t *testing.T) {
	plan := &DeviceGroupResourceModel{
		GroupType: types.StringValue(""),
		Members:  types.SetNull(types.StringType),
	}

	err := validateDeviceGroupPlan(plan)
	if err == nil {
		t.Fatal("expected error for empty group type")
	}
	if err.Error() != "group_type must be provided" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDiffStringSlices(t *testing.T) {
	tests := []struct {
		name        string
		current     []string
		desired     []string
		wantAdded   []string
		wantRemoved []string
	}{
		{
			name:        "no changes",
			current:     []string{"a", "b", "c"},
			desired:     []string{"a", "b", "c"},
			wantAdded:   nil,
			wantRemoved: nil,
		},
		{
			name:        "additions only",
			current:     []string{"a"},
			desired:     []string{"a", "b", "c"},
			wantAdded:   []string{"b", "c"},
			wantRemoved: nil,
		},
		{
			name:        "removals only",
			current:     []string{"a", "b", "c"},
			desired:     []string{"a"},
			wantAdded:   nil,
			wantRemoved: []string{"b", "c"},
		},
		{
			name:        "mixed changes",
			current:     []string{"a", "b"},
			desired:     []string{"b", "c"},
			wantAdded:   []string{"c"},
			wantRemoved: []string{"a"},
		},
		{
			name:        "empty to populated",
			current:     nil,
			desired:     []string{"a", "b"},
			wantAdded:   []string{"a", "b"},
			wantRemoved: nil,
		},
		{
			name:        "populated to empty",
			current:     []string{"a", "b"},
			desired:     nil,
			wantAdded:   nil,
			wantRemoved: []string{"a", "b"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			added, removed := diffStringSlices(tc.current, tc.desired)

			if len(added) != len(tc.wantAdded) {
				t.Errorf("added: got %v, want %v", added, tc.wantAdded)
			} else {
				for i := range added {
					if added[i] != tc.wantAdded[i] {
						t.Errorf("added[%d]: got %q, want %q", i, added[i], tc.wantAdded[i])
					}
				}
			}

			if len(removed) != len(tc.wantRemoved) {
				t.Errorf("removed: got %v, want %v", removed, tc.wantRemoved)
			} else {
				for i := range removed {
					if removed[i] != tc.wantRemoved[i] {
						t.Errorf("removed[%d]: got %q, want %q", i, removed[i], tc.wantRemoved[i])
					}
				}
			}
		})
	}
}
