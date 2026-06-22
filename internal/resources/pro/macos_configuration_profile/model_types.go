// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package macos_configuration_profile

import (
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/scope"
)

// ResourceModel is the Terraform model for jamfplatform_pro_macos_configuration_profile.
// Mirrors proclassic.OsXConfigurationProfile field-for-field with the
// following adjustments:
//
//   - Scope target sub-blocks are flattened Set<String> via internal/common/scope.
//   - self_service.display_notifications + notification_location project into
//     a single proclassic.NotificationValue (custom marshaller on the SDK side
//     emits two <notification> elements per profile).
//   - general.payloads carries the mobileconfig plist verbatim; diff
//     suppression handled by the maskPayload + lenientEqualPlist pipeline
//     (see helpers.go and plan_modifiers.go).
//   - general.uuid is server-derived (Computed-only).
type ResourceModel struct {
	ID          types.String              `tfsdk:"id"`
	General     *GeneralModel             `tfsdk:"general"`
	Scope       *scope.ComputerScopeModel `tfsdk:"scope"`
	SelfService *SelfServiceModel         `tfsdk:"self_service"`
	Timeouts    resourceTimeouts.Value    `tfsdk:"timeouts"`
}

// GeneralModel models <os_x_configuration_profile><general>. The `level`
// attribute carries the UI-canonical value (`Computer Level` / `User Level`);
// mappings.go translates to the wire-write form (`Computer` / `User`) on
// input and reads back from the wire-read form (`System` / `User`) on read.
type GeneralModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	Description        types.String `tfsdk:"description"`
	Level              types.String `tfsdk:"level"`
	DistributionMethod types.String `tfsdk:"distribution_method"`
	UserRemovable      types.Bool   `tfsdk:"user_removable"`
	RedeployOnUpdate   types.String `tfsdk:"redeploy_on_update"`
	UUID               types.String `tfsdk:"uuid"`
	Payloads           types.String `tfsdk:"payloads"`
	CategoryID         types.String `tfsdk:"category_id"`
	CategoryName       types.String `tfsdk:"category_name"`
	SiteID             types.String `tfsdk:"site_id"`
	SiteName           types.String `tfsdk:"site_name"`
}

// SelfServiceModel models <os_x_configuration_profile><self_service>. The
// wire emits two <notification> elements (bool + string) — see
// proclassic.NotificationValue for the custom marshaller. DisplayNotifications
// and NotificationLocation are the TF projection of those two elements.
type SelfServiceModel struct {
	SelfServiceDisplayName     types.String              `tfsdk:"self_service_display_name"`
	InstallButtonText          types.String              `tfsdk:"install_button_text"`
	SelfServiceDescription     types.String              `tfsdk:"self_service_description"`
	EnsureUsersViewDescription types.Bool                `tfsdk:"ensure_users_view_description"`
	FeatureOnMainPage          types.Bool                `tfsdk:"feature_on_main_page"`
	DisplayNotifications       types.Bool                `tfsdk:"display_notifications"`
	NotificationLocation       types.String              `tfsdk:"notification_location"`
	NotificationSubject        types.String              `tfsdk:"notification_subject"`
	NotificationMessage        types.String              `tfsdk:"notification_message"`
	RemovalDisallowed          types.String              `tfsdk:"removal_disallowed"`
	Categories                 []SelfServiceCategoryItem `tfsdk:"categories"`
}

// SelfServiceCategoryItem models a single <category> inside
// <self_service_categories>. The SDK exposes the wrapper as a slice
// (fixed in the scope+SS SDK handoff merged 2026-05-24).
type SelfServiceCategoryItem struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	DisplayIn types.Bool   `tfsdk:"display_in"`
	FeatureIn types.Bool   `tfsdk:"feature_in"`
}

// identityModel is the identity object for resource imports + list results.
type identityModel struct {
	ID types.String `tfsdk:"id"`
}

// DataSourceModel is the read-only data-source projection of the resource
// model. ID or Name selects the profile.
type DataSourceModel struct {
	ID          types.String              `tfsdk:"id"`
	Name        types.String              `tfsdk:"name"`
	General     *GeneralModel             `tfsdk:"general"`
	Scope       *scope.ComputerScopeModel `tfsdk:"scope"`
	SelfService *SelfServiceModel         `tfsdk:"self_service"`
}

// PluralDataSourceModel is the projection for the plural data source — a
// flat list of summary objects.
type PluralDataSourceModel struct {
	Profiles []PluralDataSourceItem `tfsdk:"profiles"`
}

// PluralDataSourceItem models a single entry in the plural projection.
type PluralDataSourceItem struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}
