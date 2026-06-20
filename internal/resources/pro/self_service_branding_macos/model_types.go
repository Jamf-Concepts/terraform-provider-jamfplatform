// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package self_service_branding_macos

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// SelfServiceBrandingMacosResourceModel is the Terraform resource model for the
// Self Service macOS branding singleton. Attribute names follow the Jamf Pro UI
// (Settings > Self Service > Branding > macOS Branding), not the wire field
// names (see the schema descriptions for the mapping).
type SelfServiceBrandingMacosResourceModel struct {
	ID                 types.String           `tfsdk:"id"`
	ApplicationHeader  types.String           `tfsdk:"application_header"`   // wire: applicationName
	SidebarHeading     types.String           `tfsdk:"sidebar_heading"`      // wire: brandingName
	SidebarSubheading  types.String           `tfsdk:"sidebar_subheading"`   // wire: brandingNameSecondary
	HomePageHeading    types.String           `tfsdk:"home_page_heading"`    // wire: homeHeading
	HomePageSubheading types.String           `tfsdk:"home_page_subheading"` // wire: homeSubheading
	IconID             types.Int64            `tfsdk:"icon_id"`              // wire: iconId
	BannerImageID      types.Int64            `tfsdk:"banner_image_id"`      // wire: brandingHeaderImageId
	Timeouts           resourceTimeouts.Value `tfsdk:"timeouts"`
}

// SelfServiceBrandingMacosDataSourceModel is the read-only data source model.
type SelfServiceBrandingMacosDataSourceModel struct {
	ID                 types.String             `tfsdk:"id"`
	ApplicationHeader  types.String             `tfsdk:"application_header"`
	SidebarHeading     types.String             `tfsdk:"sidebar_heading"`
	SidebarSubheading  types.String             `tfsdk:"sidebar_subheading"`
	HomePageHeading    types.String             `tfsdk:"home_page_heading"`
	HomePageSubheading types.String             `tfsdk:"home_page_subheading"`
	IconID             types.Int64              `tfsdk:"icon_id"`
	BannerImageID      types.Int64              `tfsdk:"banner_image_id"`
	Timeouts           datasourceTimeouts.Value `tfsdk:"timeouts"`
}
