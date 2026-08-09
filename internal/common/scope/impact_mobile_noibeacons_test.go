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

// Helpers are deliberately local to this file rather than shared with
// impact_test.go, so the two files can evolve independently.

func noIbeaconsSet(t *testing.T, vals ...string) types.Set {
	t.Helper()
	out, diags := types.SetValueFrom(context.Background(), types.StringType, vals)
	if diags.HasError() {
		t.Fatalf("building set: %v", diags)
	}
	return out
}

func noIbeaconsNullSet() types.Set { return types.SetNull(types.StringType) }

func noIbeaconsEffects(s impact.Scope) map[string]impact.Effect {
	out := make(map[string]impact.Effect, len(s.Unresolvable))
	for _, u := range s.Unresolvable {
		out[u.Path] = u.Effect
	}
	return out
}

func TestMobileImpactScopeNoIbeaconsVariant(t *testing.T) {
	m := &MobileScopeModelNoIbeacons{
		Targets: &MobileScopeTargetsModel{
			AllMobileDevices:     types.BoolNull(),
			AllJssUsers:          types.BoolNull(),
			MobileDeviceIDs:      noIbeaconsSet(t, "31"),
			MobileDeviceGroupIDs: noIbeaconsSet(t, "66"),
			BuildingIDs:          noIbeaconsNullSet(),
			DepartmentIDs:        noIbeaconsNullSet(),
			UserIDs:              noIbeaconsNullSet(),
			UserGroupIDs:         noIbeaconsNullSet(),
		},
		Limitations: &MobileScopeLimitationsModelNoIbeacons{
			NetworkSegmentIDs:                noIbeaconsSet(t, "7"),
			DirectoryServiceOrLocalUserNames: noIbeaconsNullSet(),
			DirectoryServiceUserGroupNames:   noIbeaconsNullSet(),
		},
		Exclusions: &MobileScopeExclusionsModelNoIbeacons{
			MobileDeviceIDs:                  noIbeaconsSet(t, "32"),
			MobileDeviceGroupIDs:             noIbeaconsSet(t, "67"),
			BuildingIDs:                      noIbeaconsNullSet(),
			DepartmentIDs:                    noIbeaconsNullSet(),
			UserIDs:                          noIbeaconsNullSet(),
			UserGroupIDs:                     noIbeaconsNullSet(),
			NetworkSegmentIDs:                noIbeaconsNullSet(),
			DirectoryServiceOrLocalUserNames: noIbeaconsNullSet(),
			DirectoryServiceUserGroupNames:   noIbeaconsNullSet(),
		},
	}

	got := MobileImpactScopeNoIbeacons(context.Background(), m)

	if got.DeviceType != impact.DeviceTypeMobile {
		t.Fatalf("this variant is mobile-scoped, got %v", got.DeviceType)
	}
	if len(got.DeviceIDs) != 1 || got.DeviceIDs[0] != "31" {
		t.Fatalf("mobile device targets wrong: %v", got.DeviceIDs)
	}
	if len(got.ProGroups) != 1 || got.ProGroups[0].ID != "66" {
		t.Fatalf("mobile device group targets wrong: %v", got.ProGroups)
	}
	if got.ProGroups[0].DeviceType != impact.DeviceTypeMobile {
		t.Fatalf("group refs must carry the mobile estate, got %v", got.ProGroups[0].DeviceType)
	}
	if len(got.ExcludedDeviceIDs) != 1 || got.ExcludedDeviceIDs[0] != "32" {
		t.Fatalf("excluded devices must be carried as data: %v", got.ExcludedDeviceIDs)
	}
	if len(got.ExcludedProGroups) != 1 || got.ExcludedProGroups[0].ID != "67" ||
		got.ExcludedProGroups[0].DeviceType != impact.DeviceTypeMobile {
		t.Fatalf("excluded groups must be carried as mobile-estate data: %+v", got.ExcludedProGroups)
	}

	eff := noIbeaconsEffects(got)
	if eff["limitations.network_segment_ids"] != impact.Narrows {
		t.Fatalf("a network segment limitation narrows, got %v", eff)
	}

	// The point of the variant: the schema carries no iBeacon scope, so no
	// iBeacon path may ever be reported, resolvable or otherwise.
	for _, u := range got.Unresolvable {
		if strings.Contains(u.Path, "ibeacon") {
			t.Fatalf("this variant has no iBeacon scope, reported %q", u.Path)
		}
	}
	for _, p := range got.PendingPaths {
		if strings.Contains(p, "ibeacon") {
			t.Fatalf("this variant has no iBeacon scope, pending %q", p)
		}
	}
}

func TestMobileImpactScopeNoIbeaconsNilModel(t *testing.T) {
	got := MobileImpactScopeNoIbeacons(context.Background(), nil)
	if got.DeviceType != impact.DeviceTypeMobile {
		t.Fatalf("even an absent scope block is mobile-estate, got %v", got.DeviceType)
	}
	if !got.Empty() {
		t.Fatalf("an absent scope block must yield an empty scope, got %+v", got)
	}
}
