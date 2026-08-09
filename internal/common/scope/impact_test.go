// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package scope

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/impact"
)

func idSet(t *testing.T, vals ...string) types.Set {
	t.Helper()
	out, diags := types.SetValueFrom(context.Background(), types.StringType, vals)
	if diags.HasError() {
		t.Fatalf("building set: %v", diags)
	}
	return out
}

func nullSet() types.Set { return types.SetNull(types.StringType) }

// effectsByPath indexes the recorded unresolvable inputs so a test can assert
// which direction each attribute was classified in.
func effectsByPath(s impact.Scope) map[string]impact.Effect {
	out := make(map[string]impact.Effect, len(s.Unresolvable))
	for _, u := range s.Unresolvable {
		out[u.Path] = u.Effect
	}
	return out
}

func TestComputerImpactScopeNilYieldsEmpty(t *testing.T) {
	if got := ComputerImpactScope(context.Background(), nil); !got.Empty() {
		t.Fatalf("an absent scope block must yield an empty scope, got %+v", got)
	}
}

func TestComputerImpactScopeCountsTargetsAndClassifiesTheRest(t *testing.T) {
	ctx := context.Background()
	m := &ComputerScopeModel{
		Targets: &ComputerScopeTargetsModel{
			AllComputers:     types.BoolValue(false),
			AllJssUsers:      types.BoolValue(true),
			ComputerIDs:      idSet(t, "5", "6"),
			ComputerGroupIDs: idSet(t, "12"),
			BuildingIDs:      idSet(t, "321"),
			DepartmentIDs:    nullSet(),
			UserIDs:          nullSet(),
			UserGroupIDs:     idSet(t, "44"),
		},
		Limitations: &ComputerScopeLimitationsModel{
			NetworkSegmentIDs:                idSet(t, "7"),
			IbeaconIDs:                       idSet(t, "4"),
			DirectoryServiceOrLocalUserNames: nullSet(),
			DirectoryServiceUserGroupNames:   idSet(t, "Engineering"),
		},
		Exclusions: &ComputerScopeExclusionsModel{
			ComputerIDs:                      idSet(t, "9"),
			ComputerGroupIDs:                 idSet(t, "13"),
			BuildingIDs:                      nullSet(),
			DepartmentIDs:                    nullSet(),
			UserIDs:                          nullSet(),
			UserGroupIDs:                     nullSet(),
			NetworkSegmentIDs:                nullSet(),
			IbeaconIDs:                       nullSet(),
			DirectoryServiceOrLocalUserNames: nullSet(),
			DirectoryServiceUserGroupNames:   nullSet(),
		},
	}

	got := ComputerImpactScope(ctx, m)

	if got.DeviceType != impact.DeviceTypeComputer {
		t.Fatalf("device type wrong: %v", got.DeviceType)
	}
	if got.All {
		t.Fatal("all_computers was false, so the scope must not be tenant-wide")
	}
	if len(got.DeviceIDs) != 2 || len(got.JamfProGroupIDs) != 1 {
		t.Fatalf("countable targets wrong: devices=%v groups=%v", got.DeviceIDs, got.JamfProGroupIDs)
	}

	eff := effectsByPath(got)

	// Excluded groups and devices are passed through as data so the resolver can
	// subtract group membership exactly, rather than reporting an unquantified
	// narrowing. They must therefore NOT appear as caveats.
	if len(got.ExcludedJamfProGroupIDs) != 1 {
		t.Fatalf("excluded groups must be carried as data: %v", got.ExcludedJamfProGroupIDs)
	}
	if len(got.ExcludedDeviceIDs) != 1 {
		t.Fatalf("excluded devices must be carried as data: %v", got.ExcludedDeviceIDs)
	}

	// The correctness point of this mapping: limitations can only reduce the
	// audience, so they must never be classified as broadening.
	for _, p := range []string{
		"limitations.network_segment_ids",
		"limitations.ibeacon_ids",
		"limitations.directory_service_user_group_names",
	} {
		e, ok := eff[p]
		if !ok {
			t.Fatalf("%s must be recorded", p)
		}
		if e != impact.Narrows {
			t.Fatalf("%s narrows the audience, was classified %v", p, e)
		}
	}

	// Target-side inputs the calculation cannot enumerate can only add devices.
	for _, p := range []string{
		"targets.all_jss_users",
		"targets.building_ids",
		"targets.user_group_ids",
	} {
		e, ok := eff[p]
		if !ok {
			t.Fatalf("%s must be recorded", p)
		}
		if e != impact.Broadens {
			t.Fatalf("%s broadens the audience, was classified %v", p, e)
		}
	}

	// Absent collections must not be reported at all.
	for _, p := range []string{"targets.department_ids", "targets.user_ids", "exclusions.building_ids"} {
		if _, ok := eff[p]; ok {
			t.Fatalf("%s was absent and must not be reported", p)
		}
	}
}

func TestComputerImpactScopeAllComputers(t *testing.T) {
	m := &ComputerScopeModel{
		Targets: &ComputerScopeTargetsModel{
			AllComputers:     types.BoolValue(true),
			AllJssUsers:      types.BoolNull(),
			ComputerIDs:      nullSet(),
			ComputerGroupIDs: nullSet(),
			BuildingIDs:      nullSet(),
			DepartmentIDs:    nullSet(),
			UserIDs:          nullSet(),
			UserGroupIDs:     nullSet(),
		},
	}
	got := ComputerImpactScope(context.Background(), m)
	if !got.All {
		t.Fatal("all_computers must set the tenant-wide flag")
	}
	if len(got.Unresolvable) != 0 {
		t.Fatalf("a tenant-wide scope with no other input has nothing unresolvable, got %+v", got.Unresolvable)
	}
}

func TestComputerImpactScopeOmittedTargetsBlockIsSafe(t *testing.T) {
	// TargetsOrZero returns all-null fields when the block is omitted; the
	// adapter must not treat that as a tenant-wide scope.
	got := ComputerImpactScope(context.Background(), &ComputerScopeModel{})
	if got.All {
		t.Fatal("an omitted targets block must not read as tenant-wide")
	}
	if !got.Empty() {
		t.Fatalf("an omitted targets block yields nothing to count, got %+v", got)
	}
}

func TestComputerImpactScopeNoIbeaconsVariant(t *testing.T) {
	m := &ComputerScopeModelNoIbeacons{
		Targets: &ComputerScopeTargetsModel{
			AllComputers:     types.BoolNull(),
			AllJssUsers:      types.BoolNull(),
			ComputerIDs:      nullSet(),
			ComputerGroupIDs: idSet(t, "12"),
			BuildingIDs:      nullSet(),
			DepartmentIDs:    nullSet(),
			UserIDs:          nullSet(),
			UserGroupIDs:     nullSet(),
		},
		Limitations: &ComputerScopeLimitationsModelNoIbeacons{
			NetworkSegmentIDs:                idSet(t, "7"),
			DirectoryServiceOrLocalUserNames: nullSet(),
			DirectoryServiceUserGroupNames:   nullSet(),
		},
	}
	got := ComputerImpactScopeNoIbeacons(context.Background(), m)
	eff := effectsByPath(got)
	if eff["limitations.network_segment_ids"] != impact.Narrows {
		t.Fatalf("network segment limitation must narrow, got %v", eff)
	}
	if _, ok := eff["limitations.ibeacon_ids"]; ok {
		t.Fatal("this variant has no iBeacon scope, so none must be reported")
	}
}

func TestMobileImpactScopeUsesMobileNamesAndDeviceType(t *testing.T) {
	m := &MobileScopeModel{
		Targets: &MobileScopeTargetsModel{
			AllMobileDevices:     types.BoolNull(),
			AllJssUsers:          types.BoolNull(),
			MobileDeviceIDs:      idSet(t, "31"),
			MobileDeviceGroupIDs: idSet(t, "66"),
			BuildingIDs:          nullSet(),
			DepartmentIDs:        nullSet(),
			UserIDs:              nullSet(),
			UserGroupIDs:         nullSet(),
		},
		Exclusions: &MobileScopeExclusionsModel{
			MobileDeviceIDs:                  idSet(t, "32"),
			MobileDeviceGroupIDs:             nullSet(),
			BuildingIDs:                      nullSet(),
			DepartmentIDs:                    nullSet(),
			UserIDs:                          nullSet(),
			UserGroupIDs:                     nullSet(),
			NetworkSegmentIDs:                nullSet(),
			IbeaconIDs:                       nullSet(),
			DirectoryServiceOrLocalUserNames: nullSet(),
			DirectoryServiceUserGroupNames:   nullSet(),
		},
	}
	got := MobileImpactScope(context.Background(), m)
	if got.DeviceType != impact.DeviceTypeMobile {
		t.Fatalf("device type wrong: %v", got.DeviceType)
	}
	if len(got.JamfProGroupIDs) != 1 || got.JamfProGroupIDs[0] != "66" {
		t.Fatalf("mobile device groups wrong: %v", got.JamfProGroupIDs)
	}
	if len(got.ExcludedDeviceIDs) != 1 || got.ExcludedDeviceIDs[0] != "32" {
		t.Fatalf("excluded mobile devices must be carried as data: %v", got.ExcludedDeviceIDs)
	}
	for _, u := range got.Unresolvable {
		if strings.HasPrefix(u.Path, "exclusions.computer") || strings.HasPrefix(u.Path, "targets.computer") {
			t.Fatalf("a mobile scope must not report computer attribute paths, got %q", u.Path)
		}
	}
}

func TestImpactScopePendingGroupReference(t *testing.T) {
	m := &ComputerScopeModel{
		Targets: &ComputerScopeTargetsModel{
			AllComputers:     types.BoolNull(),
			AllJssUsers:      types.BoolNull(),
			ComputerIDs:      nullSet(),
			ComputerGroupIDs: types.SetUnknown(types.StringType),
			BuildingIDs:      nullSet(),
			DepartmentIDs:    nullSet(),
			UserIDs:          nullSet(),
			UserGroupIDs:     nullSet(),
		},
	}
	got := ComputerImpactScope(context.Background(), m)
	if len(got.PendingPaths) != 1 || got.PendingPaths[0] != "targets.computer_group_ids" {
		t.Fatalf("a group created by this plan must be recorded as pending, got %v", got.PendingPaths)
	}
}
