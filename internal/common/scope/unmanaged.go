// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package scope

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// This file provides the plan-time visibility companion to the granular
// ownership model: because an omitted category is preserved silently on every
// update (read-merge-write), members added in the admin UI would otherwise
// never surface in a plan. Each scope-bearing resource's ModifyPlan calls its
// shape's Unmanaged*Categories against a hydrated live scope (best-effort GET
// — a transport failure skips the check, never blocks the plan) and surfaces
// the result via WarnUnmanagedCategories.

// AppendUnmanagedCategory appends label to list when the declared plan value
// is null (unmanaged) and the live server category has members. Exported for
// resources with local scope models (ebook, restricted_software,
// patch_policy); the shared-shape helpers below are built on it.
func AppendUnmanagedCategory(list []string, label string, declared, server types.Set) []string {
	if declared.IsNull() && !server.IsNull() && len(server.Elements()) > 0 {
		return append(list, label)
	}
	return list
}

// appendUnmanagedFlag is the all-flag sibling: an undeclared flag that is
// true on the server is co-managed state worth surfacing.
func appendUnmanagedFlag(list []string, label string, declared, server types.Bool) []string {
	if declared.IsNull() && server.ValueBool() {
		return append(list, label)
	}
	return list
}

// WarnUnmanagedCategories emits one attribute warning listing the scope
// categories that have members configured outside Terraform and will be
// preserved (not managed) by this configuration. No-op for an empty list.
func WarnUnmanagedCategories(diags *diag.Diagnostics, scopePath path.Path, categories []string) {
	if len(categories) == 0 {
		return
	}
	sort.Strings(categories)
	diags.AddAttributeWarning(
		scopePath,
		"Scope has categories managed outside Terraform",
		fmt.Sprintf(
			"These scope categories have members configured outside Terraform and are not declared in this configuration: %s. "+
				"They are preserved on apply. Declare a category (for example with `[]`) to have Terraform manage it.",
			strings.Join(categories, ", "),
		),
	)
}

// UnmanagedComputerScopeCategories lists the categories of a live computer
// scope that the declared plan leaves unmanaged. plan must be non-nil; server
// is the hydrate-all flatten of the live object's scope (nil means an empty
// live scope — nothing to report).
func UnmanagedComputerScopeCategories(plan, server *ComputerScopeModel) []string {
	if plan == nil || server == nil {
		return nil
	}
	var out []string
	pt, st := plan.TargetsOrZero(), server.TargetsOrZero()
	out = unmanagedComputerTargets(out, pt, st)
	pl, sl := derefOrZero(plan.Limitations), derefOrZero(server.Limitations)
	out = AppendUnmanagedCategory(out, "limitations.network_segment_ids", pl.NetworkSegmentIDs, sl.NetworkSegmentIDs)
	out = AppendUnmanagedCategory(out, "limitations.ibeacon_ids", pl.IbeaconIDs, sl.IbeaconIDs)
	out = AppendUnmanagedCategory(out, "limitations.directory_service_or_local_user_names", pl.DirectoryServiceOrLocalUserNames, sl.DirectoryServiceOrLocalUserNames)
	out = AppendUnmanagedCategory(out, "limitations.directory_service_user_group_names", pl.DirectoryServiceUserGroupNames, sl.DirectoryServiceUserGroupNames)
	pe, se := derefOrZero(plan.Exclusions), derefOrZero(server.Exclusions)
	out = AppendUnmanagedCategory(out, "exclusions.computer_ids", pe.ComputerIDs, se.ComputerIDs)
	out = AppendUnmanagedCategory(out, "exclusions.computer_group_ids", pe.ComputerGroupIDs, se.ComputerGroupIDs)
	out = AppendUnmanagedCategory(out, "exclusions.building_ids", pe.BuildingIDs, se.BuildingIDs)
	out = AppendUnmanagedCategory(out, "exclusions.department_ids", pe.DepartmentIDs, se.DepartmentIDs)
	out = AppendUnmanagedCategory(out, "exclusions.user_ids", pe.UserIDs, se.UserIDs)
	out = AppendUnmanagedCategory(out, "exclusions.user_group_ids", pe.UserGroupIDs, se.UserGroupIDs)
	out = AppendUnmanagedCategory(out, "exclusions.network_segment_ids", pe.NetworkSegmentIDs, se.NetworkSegmentIDs)
	out = AppendUnmanagedCategory(out, "exclusions.ibeacon_ids", pe.IbeaconIDs, se.IbeaconIDs)
	out = AppendUnmanagedCategory(out, "exclusions.directory_service_or_local_user_names", pe.DirectoryServiceOrLocalUserNames, se.DirectoryServiceOrLocalUserNames)
	out = AppendUnmanagedCategory(out, "exclusions.directory_service_user_group_names", pe.DirectoryServiceUserGroupNames, se.DirectoryServiceUserGroupNames)
	return out
}

// UnmanagedComputerScopeNoIbeaconsCategories is the no-iBeacon sibling of
// UnmanagedComputerScopeCategories (mac_app_store_app).
func UnmanagedComputerScopeNoIbeaconsCategories(plan, server *ComputerScopeModelNoIbeacons) []string {
	if plan == nil || server == nil {
		return nil
	}
	var out []string
	out = unmanagedComputerTargets(out, plan.TargetsOrZero(), server.TargetsOrZero())
	pl, sl := derefOrZero(plan.Limitations), derefOrZero(server.Limitations)
	out = AppendUnmanagedCategory(out, "limitations.network_segment_ids", pl.NetworkSegmentIDs, sl.NetworkSegmentIDs)
	out = AppendUnmanagedCategory(out, "limitations.directory_service_or_local_user_names", pl.DirectoryServiceOrLocalUserNames, sl.DirectoryServiceOrLocalUserNames)
	out = AppendUnmanagedCategory(out, "limitations.directory_service_user_group_names", pl.DirectoryServiceUserGroupNames, sl.DirectoryServiceUserGroupNames)
	pe, se := derefOrZero(plan.Exclusions), derefOrZero(server.Exclusions)
	out = AppendUnmanagedCategory(out, "exclusions.computer_ids", pe.ComputerIDs, se.ComputerIDs)
	out = AppendUnmanagedCategory(out, "exclusions.computer_group_ids", pe.ComputerGroupIDs, se.ComputerGroupIDs)
	out = AppendUnmanagedCategory(out, "exclusions.building_ids", pe.BuildingIDs, se.BuildingIDs)
	out = AppendUnmanagedCategory(out, "exclusions.department_ids", pe.DepartmentIDs, se.DepartmentIDs)
	out = AppendUnmanagedCategory(out, "exclusions.user_ids", pe.UserIDs, se.UserIDs)
	out = AppendUnmanagedCategory(out, "exclusions.user_group_ids", pe.UserGroupIDs, se.UserGroupIDs)
	out = AppendUnmanagedCategory(out, "exclusions.network_segment_ids", pe.NetworkSegmentIDs, se.NetworkSegmentIDs)
	out = AppendUnmanagedCategory(out, "exclusions.directory_service_or_local_user_names", pe.DirectoryServiceOrLocalUserNames, se.DirectoryServiceOrLocalUserNames)
	out = AppendUnmanagedCategory(out, "exclusions.directory_service_user_group_names", pe.DirectoryServiceUserGroupNames, se.DirectoryServiceUserGroupNames)
	return out
}

// unmanagedComputerTargets collects the computer-platform target categories
// shared by both computer model variants.
func unmanagedComputerTargets(out []string, pt, st ComputerScopeTargetsModel) []string {
	out = appendUnmanagedFlag(out, "targets.all_computers", pt.AllComputers, st.AllComputers)
	out = appendUnmanagedFlag(out, "targets.all_jss_users", pt.AllJssUsers, st.AllJssUsers)
	out = AppendUnmanagedCategory(out, "targets.computer_ids", pt.ComputerIDs, st.ComputerIDs)
	out = AppendUnmanagedCategory(out, "targets.computer_group_ids", pt.ComputerGroupIDs, st.ComputerGroupIDs)
	out = AppendUnmanagedCategory(out, "targets.building_ids", pt.BuildingIDs, st.BuildingIDs)
	out = AppendUnmanagedCategory(out, "targets.department_ids", pt.DepartmentIDs, st.DepartmentIDs)
	out = AppendUnmanagedCategory(out, "targets.user_ids", pt.UserIDs, st.UserIDs)
	out = AppendUnmanagedCategory(out, "targets.user_group_ids", pt.UserGroupIDs, st.UserGroupIDs)
	return out
}

// unmanagedMobileTargets collects the mobile-platform target categories
// shared by both mobile model variants.
func unmanagedMobileTargets(out []string, pt, st MobileScopeTargetsModel) []string {
	out = appendUnmanagedFlag(out, "targets.all_mobile_devices", pt.AllMobileDevices, st.AllMobileDevices)
	out = appendUnmanagedFlag(out, "targets.all_jss_users", pt.AllJssUsers, st.AllJssUsers)
	out = AppendUnmanagedCategory(out, "targets.mobile_device_ids", pt.MobileDeviceIDs, st.MobileDeviceIDs)
	out = AppendUnmanagedCategory(out, "targets.mobile_device_group_ids", pt.MobileDeviceGroupIDs, st.MobileDeviceGroupIDs)
	out = AppendUnmanagedCategory(out, "targets.building_ids", pt.BuildingIDs, st.BuildingIDs)
	out = AppendUnmanagedCategory(out, "targets.department_ids", pt.DepartmentIDs, st.DepartmentIDs)
	out = AppendUnmanagedCategory(out, "targets.user_ids", pt.UserIDs, st.UserIDs)
	out = AppendUnmanagedCategory(out, "targets.user_group_ids", pt.UserGroupIDs, st.UserGroupIDs)
	return out
}

// UnmanagedMobileScopeCategories lists the categories of a live mobile scope
// that the declared plan leaves unmanaged (mobile_device_configuration_profile).
func UnmanagedMobileScopeCategories(plan, server *MobileScopeModel) []string {
	if plan == nil || server == nil {
		return nil
	}
	var out []string
	out = unmanagedMobileTargets(out, plan.TargetsOrZero(), server.TargetsOrZero())
	pl, sl := derefOrZero(plan.Limitations), derefOrZero(server.Limitations)
	out = AppendUnmanagedCategory(out, "limitations.network_segment_ids", pl.NetworkSegmentIDs, sl.NetworkSegmentIDs)
	out = AppendUnmanagedCategory(out, "limitations.ibeacon_ids", pl.IbeaconIDs, sl.IbeaconIDs)
	out = AppendUnmanagedCategory(out, "limitations.directory_service_or_local_user_names", pl.DirectoryServiceOrLocalUserNames, sl.DirectoryServiceOrLocalUserNames)
	out = AppendUnmanagedCategory(out, "limitations.directory_service_user_group_names", pl.DirectoryServiceUserGroupNames, sl.DirectoryServiceUserGroupNames)
	pe, se := derefOrZero(plan.Exclusions), derefOrZero(server.Exclusions)
	out = AppendUnmanagedCategory(out, "exclusions.mobile_device_ids", pe.MobileDeviceIDs, se.MobileDeviceIDs)
	out = AppendUnmanagedCategory(out, "exclusions.mobile_device_group_ids", pe.MobileDeviceGroupIDs, se.MobileDeviceGroupIDs)
	out = AppendUnmanagedCategory(out, "exclusions.building_ids", pe.BuildingIDs, se.BuildingIDs)
	out = AppendUnmanagedCategory(out, "exclusions.department_ids", pe.DepartmentIDs, se.DepartmentIDs)
	out = AppendUnmanagedCategory(out, "exclusions.user_ids", pe.UserIDs, se.UserIDs)
	out = AppendUnmanagedCategory(out, "exclusions.user_group_ids", pe.UserGroupIDs, se.UserGroupIDs)
	out = AppendUnmanagedCategory(out, "exclusions.network_segment_ids", pe.NetworkSegmentIDs, se.NetworkSegmentIDs)
	out = AppendUnmanagedCategory(out, "exclusions.ibeacon_ids", pe.IbeaconIDs, se.IbeaconIDs)
	out = AppendUnmanagedCategory(out, "exclusions.directory_service_or_local_user_names", pe.DirectoryServiceOrLocalUserNames, se.DirectoryServiceOrLocalUserNames)
	out = AppendUnmanagedCategory(out, "exclusions.directory_service_user_group_names", pe.DirectoryServiceUserGroupNames, se.DirectoryServiceUserGroupNames)
	return out
}

// UnmanagedMobileScopeNoIbeaconsCategories is the no-iBeacon sibling of
// UnmanagedMobileScopeCategories (mobile_device_app).
func UnmanagedMobileScopeNoIbeaconsCategories(plan, server *MobileScopeModelNoIbeacons) []string {
	if plan == nil || server == nil {
		return nil
	}
	var out []string
	out = unmanagedMobileTargets(out, plan.TargetsOrZero(), server.TargetsOrZero())
	pl, sl := derefOrZero(plan.Limitations), derefOrZero(server.Limitations)
	out = AppendUnmanagedCategory(out, "limitations.network_segment_ids", pl.NetworkSegmentIDs, sl.NetworkSegmentIDs)
	out = AppendUnmanagedCategory(out, "limitations.directory_service_or_local_user_names", pl.DirectoryServiceOrLocalUserNames, sl.DirectoryServiceOrLocalUserNames)
	out = AppendUnmanagedCategory(out, "limitations.directory_service_user_group_names", pl.DirectoryServiceUserGroupNames, sl.DirectoryServiceUserGroupNames)
	pe, se := derefOrZero(plan.Exclusions), derefOrZero(server.Exclusions)
	out = AppendUnmanagedCategory(out, "exclusions.mobile_device_ids", pe.MobileDeviceIDs, se.MobileDeviceIDs)
	out = AppendUnmanagedCategory(out, "exclusions.mobile_device_group_ids", pe.MobileDeviceGroupIDs, se.MobileDeviceGroupIDs)
	out = AppendUnmanagedCategory(out, "exclusions.building_ids", pe.BuildingIDs, se.BuildingIDs)
	out = AppendUnmanagedCategory(out, "exclusions.department_ids", pe.DepartmentIDs, se.DepartmentIDs)
	out = AppendUnmanagedCategory(out, "exclusions.user_ids", pe.UserIDs, se.UserIDs)
	out = AppendUnmanagedCategory(out, "exclusions.user_group_ids", pe.UserGroupIDs, se.UserGroupIDs)
	out = AppendUnmanagedCategory(out, "exclusions.network_segment_ids", pe.NetworkSegmentIDs, se.NetworkSegmentIDs)
	out = AppendUnmanagedCategory(out, "exclusions.directory_service_or_local_user_names", pe.DirectoryServiceOrLocalUserNames, se.DirectoryServiceOrLocalUserNames)
	out = AppendUnmanagedCategory(out, "exclusions.directory_service_user_group_names", pe.DirectoryServiceUserGroupNames, se.DirectoryServiceUserGroupNames)
	return out
}

// UnmanagedUserScopeCategories lists the categories of a live user-based
// scope that the declared plan leaves unmanaged (vpp_assignment /
// vpp_invitation).
func UnmanagedUserScopeCategories(plan, server *UserScopeModel) []string {
	if plan == nil || server == nil {
		return nil
	}
	var out []string
	pt, st := plan.TargetsOrZero(), server.TargetsOrZero()
	out = appendUnmanagedFlag(out, "targets.all_jss_users", pt.AllJssUsers, st.AllJssUsers)
	out = AppendUnmanagedCategory(out, "targets.jss_user_ids", pt.JssUserIDs, st.JssUserIDs)
	out = AppendUnmanagedCategory(out, "targets.jss_user_group_ids", pt.JssUserGroupIDs, st.JssUserGroupIDs)
	pl, sl := derefOrZero(plan.Limitations), derefOrZero(server.Limitations)
	out = AppendUnmanagedCategory(out, "limitations.directory_service_user_group_names", pl.DirectoryServiceUserGroupNames, sl.DirectoryServiceUserGroupNames)
	pe, se := derefOrZero(plan.Exclusions), derefOrZero(server.Exclusions)
	out = AppendUnmanagedCategory(out, "exclusions.jss_user_ids", pe.JssUserIDs, se.JssUserIDs)
	out = AppendUnmanagedCategory(out, "exclusions.jss_user_group_ids", pe.JssUserGroupIDs, se.JssUserGroupIDs)
	out = AppendUnmanagedCategory(out, "exclusions.directory_service_user_group_names", pe.DirectoryServiceUserGroupNames, se.DirectoryServiceUserGroupNames)
	return out
}
