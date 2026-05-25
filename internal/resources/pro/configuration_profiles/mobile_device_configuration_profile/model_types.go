// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mobile_device_configuration_profile

import (
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ResourceModel is the Terraform model for jamfplatform_pro_mobile_device_configuration_profile.
type ResourceModel struct {
	ID          types.String           `tfsdk:"id"`
	General     *GeneralModel          `tfsdk:"general"`
	Scope       *ScopeModel            `tfsdk:"scope"`
	SelfService *SelfServiceModel      `tfsdk:"self_service"`
	Timeouts    resourceTimeouts.Value `tfsdk:"timeouts"`
}

// GeneralModel models <configuration_profile><general>. `level` carries the
// UI-canonical value ("Device Level" / "User Level"); mappings.go translates
// to wire-write ("Device" / "User") on input and from wire-read ("System" /
// "User") on read.
type GeneralModel struct {
	ID                                   types.String `tfsdk:"id"`
	Name                                 types.String `tfsdk:"name"`
	Description                          types.String `tfsdk:"description"`
	Level                                types.String `tfsdk:"level"`
	DistributionMethod                   types.String `tfsdk:"distribution_method"`
	RedeployOnUpdate                     types.String `tfsdk:"redeploy_on_update"`
	RedeployDaysBeforeCertificateExpires types.Int64  `tfsdk:"redeploy_days_before_certificate_expires"`
	UUID                                 types.String `tfsdk:"uuid"`
	Payloads                             types.String `tfsdk:"payloads"`
	CategoryID                           types.String `tfsdk:"category_id"`
	CategoryName                         types.String `tfsdk:"category_name"`
	SiteID                               types.String `tfsdk:"site_id"`
	SiteName                             types.String `tfsdk:"site_name"`
}

// ScopeModel models <configuration_profile><scope>. Mobile-device side only —
// uses mobile_devices / mobile_device_groups, not computers / computer_groups.
// User IDs use UI-canonical names; wire elements are <jss_users> and <jss_user_groups>.
type ScopeModel struct {
	AllMobileDevices     types.Bool             `tfsdk:"all_mobile_devices"`
	AllJssUsers          types.Bool             `tfsdk:"all_jss_users"`
	MobileDeviceIDs      types.Set              `tfsdk:"mobile_device_ids"`
	MobileDeviceGroupIDs types.Set              `tfsdk:"mobile_device_group_ids"`
	BuildingIDs          types.Set              `tfsdk:"building_ids"`
	DepartmentIDs        types.Set              `tfsdk:"department_ids"`
	UserIDs              types.Set              `tfsdk:"user_ids"`
	UserGroupIDs         types.Set              `tfsdk:"user_group_ids"`
	Limitations          *ScopeLimitationsModel `tfsdk:"limitations"`
	Exclusions           *ScopeExclusionsModel  `tfsdk:"exclusions"`
}

// ScopeLimitationsModel models <scope><limitations>.
type ScopeLimitationsModel struct {
	NetworkSegmentIDs                types.Set `tfsdk:"network_segment_ids"`
	IbeaconIDs                       types.Set `tfsdk:"ibeacon_ids"`
	DirectoryServiceOrLocalUserNames types.Set `tfsdk:"directory_service_or_local_user_names"`
	DirectoryServiceUserGroupNames   types.Set `tfsdk:"directory_service_user_group_names"`
}

// ScopeExclusionsModel models <scope><exclusions>.
type ScopeExclusionsModel struct {
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

// SelfServiceModel models <configuration_profile><self_service>. Mobile
// profiles omit the notification block, install_button_text, display_name,
// and force_users_to_view_description that macOS profiles carry.
type SelfServiceModel struct {
	SelfServiceDescription types.String              `tfsdk:"self_service_description"`
	FeatureOnMainPage      types.Bool                `tfsdk:"feature_on_main_page"`
	RemovalDisallowed      types.String              `tfsdk:"removal_disallowed"`
	AuthorizationPassword  types.String              `tfsdk:"authorization_password"`
	Categories             []SelfServiceCategoryItem `tfsdk:"categories"`
}

// SelfServiceCategoryItem models a single <category> inside
// <self_service_categories>. Mobile wire carries only ID and Name — no
// display_in / feature_in (unlike macOS profiles).
type SelfServiceCategoryItem struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

// identityModel is the identity object for resource imports + list results.
type identityModel struct {
	ID types.String `tfsdk:"id"`
}

// DataSourceModel is the read-only data-source projection of the resource model.
type DataSourceModel struct {
	ID          types.String      `tfsdk:"id"`
	Name        types.String      `tfsdk:"name"`
	General     *GeneralModel     `tfsdk:"general"`
	Scope       *ScopeModel       `tfsdk:"scope"`
	SelfService *SelfServiceModel `tfsdk:"self_service"`
}
