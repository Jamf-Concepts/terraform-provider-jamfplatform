// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package scope

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/impact"
)

// This file adapts the shared scope block to the device-type-neutral shape the
// impact package counts. It lives here rather than in internal/common/impact so
// that package stays free of any resource's model types — blueprints and
// compliance benchmarks target device groups without using this scope block at
// all.
//
// The mapping encodes which side of Jamf Pro's scope model each attribute sits
// on, because that decides which way an uncountable input moves the true figure:
//
//   - Targets build the audience. Everything countable here (the all-flag,
//     device ids, group ids) is counted; the user-based targets and the
//     building/department targets are recorded as broadening, since they can
//     only add devices the calculation has not seen.
//   - Limitations narrow the audience. None of them can be evaluated ahead of
//     time, so all are recorded as narrowing.
//   - Exclusions remove from the audience. They are recorded as narrowing rather
//     than subtracted: the target figure already sums group counts without
//     deduplicating overlap, and subtracting a partly-overlapping exclusion
//     count from an inflated total can understate the result in a way that is
//     harder to reason about than an upper bound.

// impactSection names the three scope tabs as they appear in configuration, used
// to build the attribute paths shown in an impact alert.
const (
	sectionTargets     = "targets"
	sectionLimitations = "limitations"
	sectionExclusions  = "exclusions"
)

// addTargetExtras records the target-side inputs that broaden the audience by an
// amount the calculation cannot determine.
func addTargetExtras(b *impact.ScopeBuilder, allJssUsers types.Bool, buildingIDs, departmentIDs, userIDs, userGroupIDs types.Set) {
	b.BroadensIf(sectionTargets+".all_jss_users", allJssUsers, impact.ReasonUserTarget)
	b.Broadens(sectionTargets+".building_ids", buildingIDs, impact.ReasonNotCounted)
	b.Broadens(sectionTargets+".department_ids", departmentIDs, impact.ReasonNotCounted)
	b.Broadens(sectionTargets+".user_ids", userIDs, impact.ReasonUserTarget)
	b.Broadens(sectionTargets+".user_group_ids", userGroupIDs, impact.ReasonUserTarget)
}

// addLimitations records the limitation-side inputs, all of which narrow.
func addLimitations(b *impact.ScopeBuilder, networkSegmentIDs, ibeaconIDs, userNames, dsGroupNames types.Set) {
	b.Narrows(sectionLimitations+".network_segment_ids", networkSegmentIDs, impact.ReasonNetworkSegment)
	b.Narrows(sectionLimitations+".ibeacon_ids", ibeaconIDs, impact.ReasonIbeacon)
	b.Narrows(sectionLimitations+".directory_service_or_local_user_names", userNames, impact.ReasonUserName)
	b.Narrows(sectionLimitations+".directory_service_user_group_names", dsGroupNames, impact.ReasonDirectoryServiceGroup)
}

// addExclusionDevices records the countable exclusion categories as narrowing.
func addExclusionDevices(b *impact.ScopeBuilder, deviceAttr string, deviceIDs, groupIDs types.Set) {
	b.Narrows(sectionExclusions+"."+deviceAttr, deviceIDs, impact.ReasonExclusion)
	b.Narrows(sectionExclusions+".computer_group_ids", groupIDs, impact.ReasonExclusion)
}

// addExclusionOther records the remaining exclusion categories as narrowing.
func addExclusionOther(b *impact.ScopeBuilder, buildingIDs, departmentIDs, userIDs, userGroupIDs, networkSegmentIDs, ibeaconIDs, userNames, dsGroupNames types.Set) {
	b.Narrows(sectionExclusions+".building_ids", buildingIDs, impact.ReasonExclusion)
	b.Narrows(sectionExclusions+".department_ids", departmentIDs, impact.ReasonExclusion)
	b.Narrows(sectionExclusions+".user_ids", userIDs, impact.ReasonExclusion)
	b.Narrows(sectionExclusions+".user_group_ids", userGroupIDs, impact.ReasonExclusion)
	b.Narrows(sectionExclusions+".network_segment_ids", networkSegmentIDs, impact.ReasonNetworkSegment)
	b.Narrows(sectionExclusions+".ibeacon_ids", ibeaconIDs, impact.ReasonIbeacon)
	b.Narrows(sectionExclusions+".directory_service_or_local_user_names", userNames, impact.ReasonUserName)
	b.Narrows(sectionExclusions+".directory_service_user_group_names", dsGroupNames, impact.ReasonDirectoryServiceGroup)
}

// ComputerImpactScope converts a computer scope block into the shape the impact
// package counts. A nil model yields an empty scope, which reports nothing.
func ComputerImpactScope(ctx context.Context, m *ComputerScopeModel) impact.Scope {
	b := impact.NewScopeBuilder(ctx, impact.DeviceTypeComputer)
	if m == nil {
		return b.Scope()
	}
	t := m.TargetsOrZero()
	b.All(t.AllComputers).
		Devices(sectionTargets+".computer_ids", t.ComputerIDs).
		JamfProGroups(sectionTargets+".computer_group_ids", t.ComputerGroupIDs)
	addTargetExtras(b, t.AllJssUsers, t.BuildingIDs, t.DepartmentIDs, t.UserIDs, t.UserGroupIDs)
	if l := m.Limitations; l != nil {
		addLimitations(b, l.NetworkSegmentIDs, l.IbeaconIDs, l.DirectoryServiceOrLocalUserNames, l.DirectoryServiceUserGroupNames)
	}
	if e := m.Exclusions; e != nil {
		addExclusionDevices(b, "computer_ids", e.ComputerIDs, e.ComputerGroupIDs)
		addExclusionOther(b, e.BuildingIDs, e.DepartmentIDs, e.UserIDs, e.UserGroupIDs,
			e.NetworkSegmentIDs, e.IbeaconIDs, e.DirectoryServiceOrLocalUserNames, e.DirectoryServiceUserGroupNames)
	}
	return b.Scope()
}

// ComputerImpactScopeNoIbeacons converts the no-iBeacon computer scope variant.
func ComputerImpactScopeNoIbeacons(ctx context.Context, m *ComputerScopeModelNoIbeacons) impact.Scope {
	b := impact.NewScopeBuilder(ctx, impact.DeviceTypeComputer)
	if m == nil {
		return b.Scope()
	}
	t := m.TargetsOrZero()
	b.All(t.AllComputers).
		Devices(sectionTargets+".computer_ids", t.ComputerIDs).
		JamfProGroups(sectionTargets+".computer_group_ids", t.ComputerGroupIDs)
	addTargetExtras(b, t.AllJssUsers, t.BuildingIDs, t.DepartmentIDs, t.UserIDs, t.UserGroupIDs)
	if l := m.Limitations; l != nil {
		addLimitations(b, l.NetworkSegmentIDs, types.SetNull(types.StringType), l.DirectoryServiceOrLocalUserNames, l.DirectoryServiceUserGroupNames)
	}
	if e := m.Exclusions; e != nil {
		addExclusionDevices(b, "computer_ids", e.ComputerIDs, e.ComputerGroupIDs)
		addExclusionOther(b, e.BuildingIDs, e.DepartmentIDs, e.UserIDs, e.UserGroupIDs,
			e.NetworkSegmentIDs, types.SetNull(types.StringType), e.DirectoryServiceOrLocalUserNames, e.DirectoryServiceUserGroupNames)
	}
	return b.Scope()
}

// MobileImpactScope converts a mobile device scope block.
func MobileImpactScope(ctx context.Context, m *MobileScopeModel) impact.Scope {
	b := impact.NewScopeBuilder(ctx, impact.DeviceTypeMobile)
	if m == nil {
		return b.Scope()
	}
	t := m.TargetsOrZero()
	b.All(t.AllMobileDevices).
		Devices(sectionTargets+".mobile_device_ids", t.MobileDeviceIDs).
		JamfProGroups(sectionTargets+".mobile_device_group_ids", t.MobileDeviceGroupIDs)
	addTargetExtras(b, t.AllJssUsers, t.BuildingIDs, t.DepartmentIDs, t.UserIDs, t.UserGroupIDs)
	if l := m.Limitations; l != nil {
		addLimitations(b, l.NetworkSegmentIDs, l.IbeaconIDs, l.DirectoryServiceOrLocalUserNames, l.DirectoryServiceUserGroupNames)
	}
	if e := m.Exclusions; e != nil {
		b.Narrows(sectionExclusions+".mobile_device_ids", e.MobileDeviceIDs, impact.ReasonExclusion)
		b.Narrows(sectionExclusions+".mobile_device_group_ids", e.MobileDeviceGroupIDs, impact.ReasonExclusion)
		addExclusionOther(b, e.BuildingIDs, e.DepartmentIDs, e.UserIDs, e.UserGroupIDs,
			e.NetworkSegmentIDs, e.IbeaconIDs, e.DirectoryServiceOrLocalUserNames, e.DirectoryServiceUserGroupNames)
	}
	return b.Scope()
}
