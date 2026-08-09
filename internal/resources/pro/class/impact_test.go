// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package class

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/impact"
)

func set(t *testing.T, vals ...string) types.Set {
	t.Helper()
	out, diags := types.SetValueFrom(context.Background(), types.StringType, vals)
	if diags.HasError() {
		t.Fatalf("building set: %v", diags)
	}
	return out
}

func nullSet() types.Set { return types.SetNull(types.StringType) }

func TestClassImpactScopeCountsMobileDeviceGroupsOnly(t *testing.T) {
	got := classImpactScope(context.Background(), &ClassResourceModel{
		MobileDeviceGroupIDs: set(t, "66"),
		StudentIDs:           set(t, "1", "2"),
		TeacherIDs:           nullSet(),
		StudentGroupIDs:      nullSet(),
		TeacherGroupIDs:      nullSet(),
	})
	if got.DeviceType != impact.DeviceTypeMobile {
		t.Fatalf("a class targets mobile devices, got %v", got.DeviceType)
	}
	if len(got.JamfProGroupIDs) != 1 || got.JamfProGroupIDs[0] != "66" {
		t.Fatalf("only the device groups may be counted: %v", got.JamfProGroupIDs)
	}
	// People can only bring more devices into play, never fewer.
	var found bool
	for _, u := range got.Unresolvable {
		if u.Path == "student_ids" {
			found = true
			if u.Effect != impact.Broadens {
				t.Fatalf("students must be classified as broadening, got %v", u.Effect)
			}
		}
	}
	if !found {
		t.Fatalf("students must be recorded: %+v", got.Unresolvable)
	}
}

func TestClassImpactScopeIgnoresAbsentPeopleCollections(t *testing.T) {
	got := classImpactScope(context.Background(), &ClassResourceModel{
		MobileDeviceGroupIDs: set(t, "66"),
		StudentIDs:           nullSet(),
		TeacherIDs:           nullSet(),
		StudentGroupIDs:      nullSet(),
		TeacherGroupIDs:      nullSet(),
	})
	if len(got.Unresolvable) != 0 {
		t.Fatalf("absent collections must not be reported: %+v", got.Unresolvable)
	}
}

func TestClassImpactScopePendingGroupReference(t *testing.T) {
	got := classImpactScope(context.Background(), &ClassResourceModel{
		MobileDeviceGroupIDs: types.SetUnknown(types.StringType),
		StudentIDs:           nullSet(),
		TeacherIDs:           nullSet(),
		StudentGroupIDs:      nullSet(),
		TeacherGroupIDs:      nullSet(),
	})
	if len(got.PendingPaths) != 1 || got.PendingPaths[0] != "mobile_device_group_ids" {
		t.Fatalf("a group created by this plan must be pending: %v", got.PendingPaths)
	}
}

func TestClassImpactScopeNilModel(t *testing.T) {
	if got := classImpactScope(context.Background(), nil); !got.Empty() {
		t.Fatalf("a nil model yields an empty scope, got %+v", got)
	}
}
