// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mobile_device_configuration_profile

import (
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/scope"
)

// timeoutAttributeTypes defines the timeout attribute types for the mobile
// device configuration profile resource operations. Used to build a null
// timeouts value when hydrating list results (terraform query
// -generate-config-out).
var timeoutAttributeTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"update": types.StringType,
	"delete": types.StringType,
}

// ResourceModel is the Terraform model for jamfplatform_pro_mobile_device_configuration_profile.
type ResourceModel struct {
	ID          types.String            `tfsdk:"id"`
	General     *GeneralModel           `tfsdk:"general"`
	Scope       *scope.MobileScopeModel `tfsdk:"scope"`
	SelfService *SelfServiceModel       `tfsdk:"self_service"`
	Timeouts    resourceTimeouts.Value  `tfsdk:"timeouts"`
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
