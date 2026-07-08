// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package scope

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestUnmanagedComputerScopeNoIbeaconsCategories(t *testing.T) {
	plan := &ComputerScopeModelNoIbeacons{
		Targets: &ComputerScopeTargetsModel{
			ComputerGroupIDs: newStringSet(t, []string{"10"}), // declared: not reported
			BuildingIDs:      newStringSet(t, nil),            // declared []: not reported
		},
	}
	server := &ComputerScopeModelNoIbeacons{
		Targets: &ComputerScopeTargetsModel{
			AllComputers:     types.BoolValue(false), // false flag: not reported
			ComputerGroupIDs: newStringSet(t, []string{"99"}),
			BuildingIDs:      newStringSet(t, []string{"7"}),
			DepartmentIDs:    newStringSet(t, []string{"3"}), // undeclared + members: reported
			UserIDs:          EmptyStringSet(),               // undeclared + empty: not reported
		},
		Exclusions: &ComputerScopeExclusionsModelNoIbeacons{
			BuildingIDs: newStringSet(t, []string{"55"}), // undeclared + members: reported
		},
	}
	got := UnmanagedComputerScopeNoIbeaconsCategories(plan, server)
	want := map[string]bool{"targets.department_ids": true, "exclusions.building_ids": true}
	if len(got) != len(want) {
		t.Fatalf("expected %d categories, got %v", len(want), got)
	}
	for _, c := range got {
		if !want[c] {
			t.Errorf("unexpected category %q in %v", c, got)
		}
	}
}

func TestUnmanagedCategories_TrueFlagReported(t *testing.T) {
	plan := &UserScopeModel{Targets: &UserScopeTargetsModel{
		JssUserIDs: newStringSet(t, []string{"1"}),
	}}
	server := &UserScopeModel{Targets: &UserScopeTargetsModel{
		AllJssUsers: types.BoolValue(true),
	}}
	got := UnmanagedUserScopeCategories(plan, server)
	if len(got) != 1 || got[0] != "targets.all_jss_users" {
		t.Fatalf("expected undeclared true flag reported, got %v", got)
	}
}

func TestUnmanagedCategories_NilInputs(t *testing.T) {
	if got := UnmanagedComputerScopeCategories(nil, &ComputerScopeModel{}); got != nil {
		t.Errorf("nil plan must report nothing, got %v", got)
	}
	if got := UnmanagedComputerScopeCategories(&ComputerScopeModel{}, nil); got != nil {
		t.Errorf("nil server must report nothing, got %v", got)
	}
}

func TestWarnUnmanagedCategories(t *testing.T) {
	var diags diag.Diagnostics
	WarnUnmanagedCategories(&diags, path.Root("scope"), nil)
	if len(diags) != 0 {
		t.Fatalf("empty list must not warn, got %v", diags)
	}
	WarnUnmanagedCategories(&diags, path.Root("scope"), []string{"targets.building_ids", "exclusions.user_ids"})
	if len(diags) != 1 {
		t.Fatalf("expected exactly one warning, got %v", diags)
	}
	if diags.ErrorsCount() != 0 {
		t.Fatal("must be a warning, not an error")
	}
}
