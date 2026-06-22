// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package scope

import "github.com/hashicorp/terraform-plugin-framework/types"

// MobileScopeModel is the Terraform model for a Jamf classic mobile-device-scope
// <scope> block. Field names + tfsdk tags match MobileScopeAttributes
// (with IncludeIbeacons=true). Every target category is a flat Set<String> of
// numeric Jamf Pro IDs; directory-service categories carry names. The wire
// elements for the user categories are <jss_users>/<jss_user_groups>; the
// UI-canonical attribute names user_ids/user_group_ids are used here.
//
// Consumed by mobile_device_configuration_profile (iBeacon-bearing). Resources
// whose endpoint omits iBeacon scope (mobile_device_app) cannot reuse this
// struct — the framework matches model fields to schema attributes exactly, so
// a no-iBeacon schema needs its own model. See STYLE_GUIDE.md §Scope helper.
type MobileScopeModel struct {
	Targets     *MobileScopeTargetsModel     `tfsdk:"targets"`
	Limitations *MobileScopeLimitationsModel `tfsdk:"limitations"`
	Exclusions  *MobileScopeExclusionsModel  `tfsdk:"exclusions"`
}

// MobileScopeTargetsModel models <scope> targets — the all-flags plus the
// per-category ID sets, mirroring the admin UI's Targets tab. Shared by both
// iBeacon variants (targets do not carry iBeacons).
type MobileScopeTargetsModel struct {
	AllMobileDevices     types.Bool `tfsdk:"all_mobile_devices"`
	AllJssUsers          types.Bool `tfsdk:"all_jss_users"`
	MobileDeviceIDs      types.Set  `tfsdk:"mobile_device_ids"`
	MobileDeviceGroupIDs types.Set  `tfsdk:"mobile_device_group_ids"`
	BuildingIDs          types.Set  `tfsdk:"building_ids"`
	DepartmentIDs        types.Set  `tfsdk:"department_ids"`
	UserIDs              types.Set  `tfsdk:"user_ids"`
	UserGroupIDs         types.Set  `tfsdk:"user_group_ids"`
}

// TargetsOrZero returns the targets sub-model, or a zero value with null sets
// when the block was omitted, so input-builders can read target fields without
// a nil-guard.
func (m MobileScopeModel) TargetsOrZero() MobileScopeTargetsModel {
	if m.Targets != nil {
		return *m.Targets
	}
	return zeroMobileTargets()
}

// MobileScopeLimitationsModel models <scope><limitations> for an
// iBeacon-bearing mobile-device-scoped resource.
type MobileScopeLimitationsModel struct {
	NetworkSegmentIDs                types.Set `tfsdk:"network_segment_ids"`
	IbeaconIDs                       types.Set `tfsdk:"ibeacon_ids"`
	DirectoryServiceOrLocalUserNames types.Set `tfsdk:"directory_service_or_local_user_names"`
	DirectoryServiceUserGroupNames   types.Set `tfsdk:"directory_service_user_group_names"`
}

// MobileScopeExclusionsModel models <scope><exclusions> for an
// iBeacon-bearing mobile-device-scoped resource.
type MobileScopeExclusionsModel struct {
	MobileDeviceIDs                  types.Set `tfsdk:"mobile_device_ids"`
	MobileDeviceGroupIDs             types.Set `tfsdk:"mobile_device_group_ids"`
	BuildingIDs                      types.Set `tfsdk:"building_ids"`
	DepartmentIDs                    types.Set `tfsdk:"department_ids"`
	UserIDs                          types.Set `tfsdk:"user_ids"`
	UserGroupIDs                     types.Set `tfsdk:"user_group_ids"`
	NetworkSegmentIDs                types.Set `tfsdk:"network_segment_ids"`
	IbeaconIDs                       types.Set `tfsdk:"ibeacon_ids"`
	DirectoryServiceOrLocalUserNames types.Set `tfsdk:"directory_service_or_local_user_names"`
	DirectoryServiceUserGroupNames   types.Set `tfsdk:"directory_service_user_group_names"`
}

// MobileScopeModelNoIbeacons is the Terraform model for a mobile-device-scope
// <scope> block built with MobileScopeAttributes(IncludeIbeacons=false). The
// target fields are identical to MobileScopeModel; only the limitations and
// exclusions sub-blocks differ (no ibeacon_ids). The framework matches model
// fields to schema attributes exactly, so a no-iBeacon schema needs its own
// model rather than reusing MobileScopeModel (which carries IbeaconIDs).
//
// Consumed by mobile_device_app, whose /mobiledeviceapplications endpoint omits
// iBeacon limitations/exclusions (wire-probed). See STYLE_GUIDE.md §Scope helper.
type MobileScopeModelNoIbeacons struct {
	Targets     *MobileScopeTargetsModel               `tfsdk:"targets"`
	Limitations *MobileScopeLimitationsModelNoIbeacons `tfsdk:"limitations"`
	Exclusions  *MobileScopeExclusionsModelNoIbeacons  `tfsdk:"exclusions"`
}

// TargetsOrZero returns the targets sub-model, or a zero value with null sets
// when the block was omitted. See MobileScopeModel.TargetsOrZero.
func (m MobileScopeModelNoIbeacons) TargetsOrZero() MobileScopeTargetsModel {
	if m.Targets != nil {
		return *m.Targets
	}
	return zeroMobileTargets()
}

// zeroMobileTargets builds an all-null MobileScopeTargetsModel for the
// TargetsOrZero accessors.
func zeroMobileTargets() MobileScopeTargetsModel {
	return MobileScopeTargetsModel{
		AllMobileDevices:     types.BoolNull(),
		AllJssUsers:          types.BoolNull(),
		MobileDeviceIDs:      types.SetNull(types.StringType),
		MobileDeviceGroupIDs: types.SetNull(types.StringType),
		BuildingIDs:          types.SetNull(types.StringType),
		DepartmentIDs:        types.SetNull(types.StringType),
		UserIDs:              types.SetNull(types.StringType),
		UserGroupIDs:         types.SetNull(types.StringType),
	}
}

// MobileScopeLimitationsModelNoIbeacons models <scope><limitations> for a
// mobile-device-scoped resource whose endpoint omits iBeacon scope.
type MobileScopeLimitationsModelNoIbeacons struct {
	NetworkSegmentIDs                types.Set `tfsdk:"network_segment_ids"`
	DirectoryServiceOrLocalUserNames types.Set `tfsdk:"directory_service_or_local_user_names"`
	DirectoryServiceUserGroupNames   types.Set `tfsdk:"directory_service_user_group_names"`
}

// MobileScopeExclusionsModelNoIbeacons models <scope><exclusions> for a
// mobile-device-scoped resource whose endpoint omits iBeacon scope.
type MobileScopeExclusionsModelNoIbeacons struct {
	MobileDeviceIDs                  types.Set `tfsdk:"mobile_device_ids"`
	MobileDeviceGroupIDs             types.Set `tfsdk:"mobile_device_group_ids"`
	BuildingIDs                      types.Set `tfsdk:"building_ids"`
	DepartmentIDs                    types.Set `tfsdk:"department_ids"`
	UserIDs                          types.Set `tfsdk:"user_ids"`
	UserGroupIDs                     types.Set `tfsdk:"user_group_ids"`
	NetworkSegmentIDs                types.Set `tfsdk:"network_segment_ids"`
	DirectoryServiceOrLocalUserNames types.Set `tfsdk:"directory_service_or_local_user_names"`
	DirectoryServiceUserGroupNames   types.Set `tfsdk:"directory_service_user_group_names"`
}
