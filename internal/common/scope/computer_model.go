// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package scope

import "github.com/hashicorp/terraform-plugin-framework/types"

// ComputerScopeModel is the Terraform model for a Jamf classic computer-scope
// <scope> block. Field names + tfsdk tags match ComputerScopeAttributes
// (with IncludeIbeacons=true). Every target category is a flat Set<String> of
// numeric Jamf Pro IDs; directory-service categories carry names. The wire
// elements for the user categories are <jss_users>/<jss_user_groups>; the
// UI-canonical attribute names user_ids/user_group_ids are used here.
//
// Consumed by policy and os_x_configuration_profile (both iBeacon-bearing).
// Resources whose endpoint omits iBeacon scope (mac_application) cannot reuse
// this struct — the framework matches model fields to schema attributes
// exactly, so a no-iBeacon schema needs its own model. See STYLE_GUIDE.md
// §Scope helper.
type ComputerScopeModel struct {
	AllComputers     types.Bool                     `tfsdk:"all_computers"`
	AllJssUsers      types.Bool                     `tfsdk:"all_jss_users"`
	ComputerIDs      types.Set                      `tfsdk:"computer_ids"`
	ComputerGroupIDs types.Set                      `tfsdk:"computer_group_ids"`
	BuildingIDs      types.Set                      `tfsdk:"building_ids"`
	DepartmentIDs    types.Set                      `tfsdk:"department_ids"`
	UserIDs          types.Set                      `tfsdk:"user_ids"`
	UserGroupIDs     types.Set                      `tfsdk:"user_group_ids"`
	Limitations      *ComputerScopeLimitationsModel `tfsdk:"limitations"`
	Exclusions       *ComputerScopeExclusionsModel  `tfsdk:"exclusions"`
}

// ComputerScopeLimitationsModel models <scope><limitations> for an
// iBeacon-bearing computer-scoped resource.
type ComputerScopeLimitationsModel struct {
	NetworkSegmentIDs                types.Set `tfsdk:"network_segment_ids"`
	IbeaconIDs                       types.Set `tfsdk:"ibeacon_ids"`
	DirectoryServiceOrLocalUserNames types.Set `tfsdk:"directory_service_or_local_user_names"`
	DirectoryServiceUserGroupNames   types.Set `tfsdk:"directory_service_user_group_names"`
}

// ComputerScopeExclusionsModel models <scope><exclusions> for an
// iBeacon-bearing computer-scoped resource.
type ComputerScopeExclusionsModel struct {
	ComputerIDs                      types.Set `tfsdk:"computer_ids"`
	ComputerGroupIDs                 types.Set `tfsdk:"computer_group_ids"`
	BuildingIDs                      types.Set `tfsdk:"building_ids"`
	DepartmentIDs                    types.Set `tfsdk:"department_ids"`
	UserIDs                          types.Set `tfsdk:"user_ids"`
	UserGroupIDs                     types.Set `tfsdk:"user_group_ids"`
	NetworkSegmentIDs                types.Set `tfsdk:"network_segment_ids"`
	IbeaconIDs                       types.Set `tfsdk:"ibeacon_ids"`
	DirectoryServiceOrLocalUserNames types.Set `tfsdk:"directory_service_or_local_user_names"`
	DirectoryServiceUserGroupNames   types.Set `tfsdk:"directory_service_user_group_names"`
}

// ComputerScopeModelNoIbeacons is the Terraform model for a computer-scope
// <scope> block built with ComputerScopeAttributes(IncludeIbeacons=false).
// The target fields are identical to ComputerScopeModel; only the limitations
// and exclusions sub-blocks differ (no ibeacon_ids). The framework matches
// model fields to schema attributes exactly, so a no-iBeacon schema needs its
// own model rather than reusing ComputerScopeModel (which carries IbeaconIDs).
//
// Consumed by mac_app_store_app, whose /macapplications endpoint silently
// drops iBeacon limitations/exclusions (wire-probed). See STYLE_GUIDE.md
// §Scope helper.
type ComputerScopeModelNoIbeacons struct {
	AllComputers     types.Bool                               `tfsdk:"all_computers"`
	AllJssUsers      types.Bool                               `tfsdk:"all_jss_users"`
	ComputerIDs      types.Set                                `tfsdk:"computer_ids"`
	ComputerGroupIDs types.Set                                `tfsdk:"computer_group_ids"`
	BuildingIDs      types.Set                                `tfsdk:"building_ids"`
	DepartmentIDs    types.Set                                `tfsdk:"department_ids"`
	UserIDs          types.Set                                `tfsdk:"user_ids"`
	UserGroupIDs     types.Set                                `tfsdk:"user_group_ids"`
	Limitations      *ComputerScopeLimitationsModelNoIbeacons `tfsdk:"limitations"`
	Exclusions       *ComputerScopeExclusionsModelNoIbeacons  `tfsdk:"exclusions"`
}

// ComputerScopeLimitationsModelNoIbeacons models <scope><limitations> for a
// computer-scoped resource whose endpoint omits iBeacon scope.
type ComputerScopeLimitationsModelNoIbeacons struct {
	NetworkSegmentIDs                types.Set `tfsdk:"network_segment_ids"`
	DirectoryServiceOrLocalUserNames types.Set `tfsdk:"directory_service_or_local_user_names"`
	DirectoryServiceUserGroupNames   types.Set `tfsdk:"directory_service_user_group_names"`
}

// ComputerScopeExclusionsModelNoIbeacons models <scope><exclusions> for a
// computer-scoped resource whose endpoint omits iBeacon scope.
type ComputerScopeExclusionsModelNoIbeacons struct {
	ComputerIDs                      types.Set `tfsdk:"computer_ids"`
	ComputerGroupIDs                 types.Set `tfsdk:"computer_group_ids"`
	BuildingIDs                      types.Set `tfsdk:"building_ids"`
	DepartmentIDs                    types.Set `tfsdk:"department_ids"`
	UserIDs                          types.Set `tfsdk:"user_ids"`
	UserGroupIDs                     types.Set `tfsdk:"user_group_ids"`
	NetworkSegmentIDs                types.Set `tfsdk:"network_segment_ids"`
	DirectoryServiceOrLocalUserNames types.Set `tfsdk:"directory_service_or_local_user_names"`
	DirectoryServiceUserGroupNames   types.Set `tfsdk:"directory_service_user_group_names"`
}
