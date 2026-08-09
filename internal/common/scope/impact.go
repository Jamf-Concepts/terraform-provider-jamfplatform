// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package scope

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/impact"
)

// This file adapts a Jamf Pro scope block to the device-type-neutral shape the
// impact package counts. It lives here rather than in internal/common/impact so
// that package stays free of any resource's model types — blueprints and
// compliance benchmarks target device groups without using this scope block at
// all.
//
// The mapping encodes which side of Jamf Pro's scope model each attribute sits
// on, because that decides which way an uncountable input moves the true figure:
//
//   - Targets build the audience. The all-flag, device ids and group ids are
//     counted; the user-based and building/department targets are recorded as
//     broadening, since they can only add devices the calculation has not seen.
//   - Limitations narrow the audience. None of them can be evaluated ahead of
//     time, so all are recorded as narrowing.
//   - Exclusions remove from the audience. Excluded groups and devices are passed
//     through as data so the resolver can subtract group membership exactly; the
//     rest cannot be enumerated ahead of time and stay narrowing.
//
// That knowledge is deliberately expressed once, in BuildImpactScope. Nine
// resources carry a scope block and six of them hand-compose their own model
// shape, so duplicating the classification per resource would be the easiest way
// to get one of them silently backwards.
//
// There is deliberately no adapter for UserScopeModel, which is why
// jamfplatform_pro_vpp_assignment and jamfplatform_pro_vpp_invitation raise no
// impact alert. A user scope targets Jamf Pro users and user groups and nothing
// else — no device or device-group category exists in it — so every input would
// be unresolvable and the alert could only ever say that the figure cannot be
// determined. An alert that never carries a number is noise, so those two
// resources stay silent until user-to-device resolution exists.

// impactSection names the three scope tabs as they appear in configuration, used
// to build the attribute paths shown in an impact alert.
const (
	sectionTargets     = "targets"
	sectionLimitations = "limitations"
	sectionExclusions  = "exclusions"
)

// ImpactInputs is a scope block flattened to its individual attributes, so any
// model shape can feed the one classification.
//
// Every field is optional: an unset collection is null, which the builder treats
// as absent. A resource therefore populates only the categories its own scope
// supports — a patch policy has no user targets, a restricted software record has
// no limitations block — and the omitted ones report nothing.
type ImpactInputs struct {
	// DeviceType is the estate the scope's percentages are measured against.
	// DeviceTypeAny is for a resource that targets both, such as an ebook.
	DeviceType impact.DeviceType

	// DeviceAttr and GroupAttr are the configuration names of the device and
	// group categories, e.g. "computer_ids" and "computer_group_ids". They appear
	// in the attribute paths an alert cites.
	DeviceAttr string
	GroupAttr  string
	// GroupEstate is the estate the group ids belong to. Defaults to DeviceType.
	GroupEstate impact.DeviceType

	// Targets.
	All              types.Bool
	DeviceIDs        types.Set
	GroupIDs         types.Set
	AllJssUsers      types.Bool
	BuildingIDs      types.Set
	DepartmentIDs    types.Set
	UserIDs          types.Set
	UserGroupIDs     types.Set
	ClassIDs         types.Set
	SecondaryDevices *SecondaryEstate

	// Limitations.
	LimitNetworkSegmentIDs types.Set
	LimitIbeaconIDs        types.Set
	LimitUserNames         types.Set
	LimitDSGroupNames      types.Set

	// Exclusions.
	ExcludeDeviceIDs         types.Set
	ExcludeGroupIDs          types.Set
	ExcludeBuildingIDs       types.Set
	ExcludeDepartmentIDs     types.Set
	ExcludeUserIDs           types.Set
	ExcludeUserGroupIDs      types.Set
	ExcludeNetworkSegmentIDs types.Set
	ExcludeIbeaconIDs        types.Set
	ExcludeUserNames         types.Set
	ExcludeDSGroupNames      types.Set
	ExcludeSecondaryDevices  *SecondaryEstate
}

// SecondaryEstate carries the second estate's device and group categories for a
// resource that targets both at once. An ebook is scoped to computers and mobile
// devices in the same block, and the two are counted together.
type SecondaryEstate struct {
	DeviceType impact.DeviceType
	DeviceAttr string
	GroupAttr  string
	All        types.Bool
	DeviceIDs  types.Set
	GroupIDs   types.Set
}

// BuildImpactScope converts a scope block into the shape the impact package
// counts.
func BuildImpactScope(ctx context.Context, in ImpactInputs) impact.Scope {
	b := impact.NewScopeBuilder(ctx, in.DeviceType)

	groupEstate := in.GroupEstate
	if groupEstate == "" && in.DeviceType != impact.DeviceTypeAny {
		groupEstate = in.DeviceType
	}

	// Targets — counted.
	b.All(in.All).
		Devices(sectionTargets+"."+in.DeviceAttr, in.DeviceIDs).
		ProGroups(sectionTargets+"."+in.GroupAttr, groupEstate, in.GroupIDs)

	if sec := in.SecondaryDevices; sec != nil {
		b.All(sec.All).
			Devices(sectionTargets+"."+sec.DeviceAttr, sec.DeviceIDs).
			ProGroups(sectionTargets+"."+sec.GroupAttr, sec.DeviceType, sec.GroupIDs)
	}

	// Targets — cannot be enumerated, and can only add devices.
	b.BroadensIf(sectionTargets+".all_jss_users", in.AllJssUsers, impact.ReasonUserTarget).
		Broadens(sectionTargets+".building_ids", in.BuildingIDs, impact.ReasonNotCounted).
		Broadens(sectionTargets+".department_ids", in.DepartmentIDs, impact.ReasonNotCounted).
		Broadens(sectionTargets+".user_ids", in.UserIDs, impact.ReasonUserTarget).
		Broadens(sectionTargets+".user_group_ids", in.UserGroupIDs, impact.ReasonUserTarget).
		Broadens(sectionTargets+".class_ids", in.ClassIDs, impact.ReasonClassTarget)

	// Limitations — none can be evaluated ahead of time, and all narrow.
	b.Narrows(sectionLimitations+".network_segment_ids", in.LimitNetworkSegmentIDs, impact.ReasonNetworkSegment).
		Narrows(sectionLimitations+".ibeacon_ids", in.LimitIbeaconIDs, impact.ReasonIbeacon).
		Narrows(sectionLimitations+".directory_service_or_local_user_names", in.LimitUserNames, impact.ReasonUserName).
		Narrows(sectionLimitations+".directory_service_user_group_names", in.LimitDSGroupNames, impact.ReasonDirectoryServiceGroup)

	// Exclusions — groups and devices as data, the rest as narrowing.
	b.ExcludedDevices(sectionExclusions+"."+in.DeviceAttr, in.ExcludeDeviceIDs).
		ExcludedProGroups(sectionExclusions+"."+in.GroupAttr, groupEstate, in.ExcludeGroupIDs)

	if sec := in.ExcludeSecondaryDevices; sec != nil {
		b.ExcludedDevices(sectionExclusions+"."+sec.DeviceAttr, sec.DeviceIDs).
			ExcludedProGroups(sectionExclusions+"."+sec.GroupAttr, sec.DeviceType, sec.GroupIDs)
	}

	b.Narrows(sectionExclusions+".building_ids", in.ExcludeBuildingIDs, impact.ReasonNotCounted).
		Narrows(sectionExclusions+".department_ids", in.ExcludeDepartmentIDs, impact.ReasonNotCounted).
		Narrows(sectionExclusions+".user_ids", in.ExcludeUserIDs, impact.ReasonUserTarget).
		Narrows(sectionExclusions+".user_group_ids", in.ExcludeUserGroupIDs, impact.ReasonUserTarget).
		Narrows(sectionExclusions+".network_segment_ids", in.ExcludeNetworkSegmentIDs, impact.ReasonNetworkSegment).
		Narrows(sectionExclusions+".ibeacon_ids", in.ExcludeIbeaconIDs, impact.ReasonIbeacon).
		Narrows(sectionExclusions+".directory_service_or_local_user_names", in.ExcludeUserNames, impact.ReasonUserName).
		Narrows(sectionExclusions+".directory_service_user_group_names", in.ExcludeDSGroupNames, impact.ReasonDirectoryServiceGroup)

	return b.Scope()
}

// ComputerImpactScope converts a computer scope block. A nil model yields an
// empty scope, which reports nothing.
func ComputerImpactScope(ctx context.Context, m *ComputerScopeModel) impact.Scope {
	if m == nil {
		return impact.Scope{DeviceType: impact.DeviceTypeComputer}
	}
	t := m.TargetsOrZero()
	in := computerTargets(t)
	if l := m.Limitations; l != nil {
		in.LimitNetworkSegmentIDs = l.NetworkSegmentIDs
		in.LimitIbeaconIDs = l.IbeaconIDs
		in.LimitUserNames = l.DirectoryServiceOrLocalUserNames
		in.LimitDSGroupNames = l.DirectoryServiceUserGroupNames
	}
	if e := m.Exclusions; e != nil {
		in.ExcludeDeviceIDs = e.ComputerIDs
		in.ExcludeGroupIDs = e.ComputerGroupIDs
		in.ExcludeBuildingIDs = e.BuildingIDs
		in.ExcludeDepartmentIDs = e.DepartmentIDs
		in.ExcludeUserIDs = e.UserIDs
		in.ExcludeUserGroupIDs = e.UserGroupIDs
		in.ExcludeNetworkSegmentIDs = e.NetworkSegmentIDs
		in.ExcludeIbeaconIDs = e.IbeaconIDs
		in.ExcludeUserNames = e.DirectoryServiceOrLocalUserNames
		in.ExcludeDSGroupNames = e.DirectoryServiceUserGroupNames
	}
	return BuildImpactScope(ctx, in)
}

// ComputerImpactScopeNoIbeacons converts the no-iBeacon computer scope variant.
func ComputerImpactScopeNoIbeacons(ctx context.Context, m *ComputerScopeModelNoIbeacons) impact.Scope {
	if m == nil {
		return impact.Scope{DeviceType: impact.DeviceTypeComputer}
	}
	t := m.TargetsOrZero()
	in := computerTargets(t)
	if l := m.Limitations; l != nil {
		in.LimitNetworkSegmentIDs = l.NetworkSegmentIDs
		in.LimitUserNames = l.DirectoryServiceOrLocalUserNames
		in.LimitDSGroupNames = l.DirectoryServiceUserGroupNames
	}
	if e := m.Exclusions; e != nil {
		in.ExcludeDeviceIDs = e.ComputerIDs
		in.ExcludeGroupIDs = e.ComputerGroupIDs
		in.ExcludeBuildingIDs = e.BuildingIDs
		in.ExcludeDepartmentIDs = e.DepartmentIDs
		in.ExcludeUserIDs = e.UserIDs
		in.ExcludeUserGroupIDs = e.UserGroupIDs
		in.ExcludeNetworkSegmentIDs = e.NetworkSegmentIDs
		in.ExcludeUserNames = e.DirectoryServiceOrLocalUserNames
		in.ExcludeDSGroupNames = e.DirectoryServiceUserGroupNames
	}
	return BuildImpactScope(ctx, in)
}

// computerTargets maps the shared computer targets model onto the inputs.
func computerTargets(t ComputerScopeTargetsModel) ImpactInputs {
	return ImpactInputs{
		DeviceType:    impact.DeviceTypeComputer,
		DeviceAttr:    "computer_ids",
		GroupAttr:     "computer_group_ids",
		All:           t.AllComputers,
		DeviceIDs:     t.ComputerIDs,
		GroupIDs:      t.ComputerGroupIDs,
		AllJssUsers:   t.AllJssUsers,
		BuildingIDs:   t.BuildingIDs,
		DepartmentIDs: t.DepartmentIDs,
		UserIDs:       t.UserIDs,
		UserGroupIDs:  t.UserGroupIDs,
	}
}

// mobileTargets maps the shared mobile targets model onto the inputs.
func mobileTargets(t MobileScopeTargetsModel) ImpactInputs {
	return ImpactInputs{
		DeviceType:    impact.DeviceTypeMobile,
		DeviceAttr:    "mobile_device_ids",
		GroupAttr:     "mobile_device_group_ids",
		All:           t.AllMobileDevices,
		DeviceIDs:     t.MobileDeviceIDs,
		GroupIDs:      t.MobileDeviceGroupIDs,
		AllJssUsers:   t.AllJssUsers,
		BuildingIDs:   t.BuildingIDs,
		DepartmentIDs: t.DepartmentIDs,
		UserIDs:       t.UserIDs,
		UserGroupIDs:  t.UserGroupIDs,
	}
}

// MobileImpactScope converts a mobile device scope block.
func MobileImpactScope(ctx context.Context, m *MobileScopeModel) impact.Scope {
	if m == nil {
		return impact.Scope{DeviceType: impact.DeviceTypeMobile}
	}
	in := mobileTargets(m.TargetsOrZero())
	if l := m.Limitations; l != nil {
		in.LimitNetworkSegmentIDs = l.NetworkSegmentIDs
		in.LimitIbeaconIDs = l.IbeaconIDs
		in.LimitUserNames = l.DirectoryServiceOrLocalUserNames
		in.LimitDSGroupNames = l.DirectoryServiceUserGroupNames
	}
	if e := m.Exclusions; e != nil {
		in.ExcludeDeviceIDs = e.MobileDeviceIDs
		in.ExcludeGroupIDs = e.MobileDeviceGroupIDs
		in.ExcludeBuildingIDs = e.BuildingIDs
		in.ExcludeDepartmentIDs = e.DepartmentIDs
		in.ExcludeUserIDs = e.UserIDs
		in.ExcludeUserGroupIDs = e.UserGroupIDs
		in.ExcludeNetworkSegmentIDs = e.NetworkSegmentIDs
		in.ExcludeIbeaconIDs = e.IbeaconIDs
		in.ExcludeUserNames = e.DirectoryServiceOrLocalUserNames
		in.ExcludeDSGroupNames = e.DirectoryServiceUserGroupNames
	}
	return BuildImpactScope(ctx, in)
}

// MobileImpactScopeNoIbeacons converts the no-iBeacon mobile scope variant.
func MobileImpactScopeNoIbeacons(ctx context.Context, m *MobileScopeModelNoIbeacons) impact.Scope {
	if m == nil {
		return impact.Scope{DeviceType: impact.DeviceTypeMobile}
	}
	in := mobileTargets(m.TargetsOrZero())
	if l := m.Limitations; l != nil {
		in.LimitNetworkSegmentIDs = l.NetworkSegmentIDs
		in.LimitUserNames = l.DirectoryServiceOrLocalUserNames
		in.LimitDSGroupNames = l.DirectoryServiceUserGroupNames
	}
	if e := m.Exclusions; e != nil {
		in.ExcludeDeviceIDs = e.MobileDeviceIDs
		in.ExcludeGroupIDs = e.MobileDeviceGroupIDs
		in.ExcludeBuildingIDs = e.BuildingIDs
		in.ExcludeDepartmentIDs = e.DepartmentIDs
		in.ExcludeUserIDs = e.UserIDs
		in.ExcludeUserGroupIDs = e.UserGroupIDs
		in.ExcludeNetworkSegmentIDs = e.NetworkSegmentIDs
		in.ExcludeUserNames = e.DirectoryServiceOrLocalUserNames
		in.ExcludeDSGroupNames = e.DirectoryServiceUserGroupNames
	}
	return BuildImpactScope(ctx, in)
}
