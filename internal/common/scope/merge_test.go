// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package scope

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestMergeComputerScopeNoIbeacons_NilPlanStaysNil(t *testing.T) {
	if got := MergeComputerScopeNoIbeacons(nil, &ComputerScopeModelNoIbeacons{}); got != nil {
		t.Fatalf("expected nil for unmanaged scope, got %+v", got)
	}
}

func TestMergeComputerScopeNoIbeacons_DeclaredWinsOmittedPreserved(t *testing.T) {
	plan := &ComputerScopeModelNoIbeacons{
		Targets: &ComputerScopeTargetsModel{
			AllComputers:     types.BoolNull(),
			AllJssUsers:      types.BoolNull(),
			ComputerIDs:      types.SetNull(types.StringType),
			ComputerGroupIDs: newStringSet(t, []string{"10"}), // declared: wins
			BuildingIDs:      newStringSet(t, nil),            // declared []: clears
			DepartmentIDs:    types.SetNull(types.StringType), // omitted: server's
			UserIDs:          types.SetNull(types.StringType),
			UserGroupIDs:     types.SetNull(types.StringType),
		},
		// limitations + exclusions omitted entirely: preserved from server.
	}
	server := &ComputerScopeModelNoIbeacons{
		Targets: &ComputerScopeTargetsModel{
			AllComputers:     types.BoolValue(false),
			AllJssUsers:      types.BoolValue(false),
			ComputerIDs:      EmptyStringSet(),
			ComputerGroupIDs: newStringSet(t, []string{"99"}),
			BuildingIDs:      newStringSet(t, []string{"7"}),
			DepartmentIDs:    newStringSet(t, []string{"3"}),
			UserIDs:          EmptyStringSet(),
			UserGroupIDs:     EmptyStringSet(),
		},
		Exclusions: &ComputerScopeExclusionsModelNoIbeacons{
			BuildingIDs: newStringSet(t, []string{"55"}),
		},
	}

	got := MergeComputerScopeNoIbeacons(plan, server)

	if v := sortedSetValues(t, got.Targets.ComputerGroupIDs); len(v) != 1 || v[0] != "10" {
		t.Errorf("declared category should win: got %v", v)
	}
	if v := sortedSetValues(t, got.Targets.BuildingIDs); len(v) != 0 {
		t.Errorf("declared [] should clear: got %v", v)
	}
	if got.Targets.BuildingIDs.IsNull() {
		t.Error("declared [] must stay non-null (emits the empty wrapper)")
	}
	if v := sortedSetValues(t, got.Targets.DepartmentIDs); len(v) != 1 || v[0] != "3" {
		t.Errorf("omitted category should preserve server members: got %v", v)
	}
	if got.Exclusions == nil {
		t.Fatal("omitted exclusions block must still merge from server")
	}
	if v := sortedSetValues(t, got.Exclusions.BuildingIDs); len(v) != 1 || v[0] != "55" {
		t.Errorf("omitted exclusions category should preserve server members: got %v", v)
	}
	// Every merged field must be non-null so the builder emits the full
	// explicit skeleton.
	if got.Targets.ComputerIDs.IsNull() || got.Limitations == nil ||
		got.Limitations.NetworkSegmentIDs.IsNull() ||
		got.Exclusions.ComputerIDs.IsNull() {
		t.Error("merged output must have every category non-null")
	}
	if got.Targets.AllComputers.IsNull() || got.Targets.AllJssUsers.IsNull() {
		t.Error("merged all-flags must be non-null")
	}
}

func TestMergeComputerScopeNoIbeacons_NilServerTreatedAsEmpty(t *testing.T) {
	plan := &ComputerScopeModelNoIbeacons{
		Targets: &ComputerScopeTargetsModel{
			ComputerGroupIDs: newStringSet(t, []string{"10"}),
		},
	}
	got := MergeComputerScopeNoIbeacons(plan, nil)
	if v := sortedSetValues(t, got.Targets.ComputerGroupIDs); len(v) != 1 || v[0] != "10" {
		t.Errorf("expected declared members, got %v", v)
	}
	if got.Targets.DepartmentIDs.IsNull() || len(got.Targets.DepartmentIDs.Elements()) != 0 {
		t.Errorf("expected empty non-null set for omitted category with no server value")
	}
}

func TestMergeComputerScope_AllComputersTrueEmptiesConflictingTargets(t *testing.T) {
	plan := &ComputerScopeModel{
		Targets: &ComputerScopeTargetsModel{
			AllComputers: types.BoolValue(true),
		},
	}
	server := &ComputerScopeModel{
		Targets: &ComputerScopeTargetsModel{
			AllComputers:     types.BoolValue(false),
			ComputerGroupIDs: newStringSet(t, []string{"99"}),
			BuildingIDs:      newStringSet(t, []string{"7"}),
			UserIDs:          newStringSet(t, []string{"1"}),
		},
		Exclusions: &ComputerScopeExclusionsModel{
			BuildingIDs: newStringSet(t, []string{"55"}),
		},
	}
	got := MergeComputerScope(plan, server)
	if !got.Targets.AllComputers.ValueBool() {
		t.Fatal("expected all_computers=true")
	}
	// The device-side categories the flag conflicts with are emptied (the
	// wire wipes them when the flag is set)...
	if len(got.Targets.ComputerGroupIDs.Elements()) != 0 || len(got.Targets.BuildingIDs.Elements()) != 0 {
		t.Error("expected conflicting target categories emptied under all_computers=true")
	}
	// ...but user-side targets and exclusions coexist with the flag
	// (wire-probed) and are preserved.
	if v := sortedSetValues(t, got.Targets.UserIDs); len(v) != 1 || v[0] != "1" {
		t.Errorf("expected user targets preserved under all_computers=true, got %v", v)
	}
	if v := sortedSetValues(t, got.Exclusions.BuildingIDs); len(v) != 1 || v[0] != "55" {
		t.Errorf("expected exclusions preserved under all_computers=true, got %v", v)
	}
}

func TestMergeMobileScopeNoIbeacons_AllMobileDevicesPrecedence(t *testing.T) {
	plan := &MobileScopeModelNoIbeacons{
		Targets: &MobileScopeTargetsModel{
			AllMobileDevices: types.BoolValue(true),
		},
	}
	server := &MobileScopeModelNoIbeacons{
		Targets: &MobileScopeTargetsModel{
			MobileDeviceGroupIDs: newStringSet(t, []string{"12"}),
			UserGroupIDs:         newStringSet(t, []string{"4"}),
		},
	}
	got := MergeMobileScopeNoIbeacons(plan, server)
	if len(got.Targets.MobileDeviceGroupIDs.Elements()) != 0 {
		t.Error("expected device-side categories emptied under all_mobile_devices=true")
	}
	if v := sortedSetValues(t, got.Targets.UserGroupIDs); len(v) != 1 || v[0] != "4" {
		t.Errorf("expected user-side categories preserved, got %v", v)
	}
}

func TestMergeUserScope_DeclaredWinsOmittedPreserved(t *testing.T) {
	plan := &UserScopeModel{
		Targets: &UserScopeTargetsModel{
			JssUserIDs: newStringSet(t, []string{"279"}),
		},
	}
	server := &UserScopeModel{
		Targets: &UserScopeTargetsModel{
			AllJssUsers:     types.BoolValue(false),
			JssUserIDs:      newStringSet(t, []string{"1"}),
			JssUserGroupIDs: newStringSet(t, []string{"928"}),
		},
		Exclusions: &UserScopeExclusionsModel{
			JssUserIDs: newStringSet(t, []string{"280"}),
		},
	}
	got := MergeUserScope(plan, server)
	if v := sortedSetValues(t, got.Targets.JssUserIDs); len(v) != 1 || v[0] != "279" {
		t.Errorf("declared should win: got %v", v)
	}
	if v := sortedSetValues(t, got.Targets.JssUserGroupIDs); len(v) != 1 || v[0] != "928" {
		t.Errorf("omitted should preserve server: got %v", v)
	}
	if v := sortedSetValues(t, got.Exclusions.JssUserIDs); len(v) != 1 || v[0] != "280" {
		t.Errorf("omitted exclusions should preserve server: got %v", v)
	}
	if got.Limitations == nil || got.Limitations.DirectoryServiceUserGroupNames.IsNull() {
		t.Error("merged output must have every category non-null")
	}
}
