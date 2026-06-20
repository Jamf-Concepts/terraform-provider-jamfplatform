// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package self_service_branding_ios

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// SelfServiceBrandingIosResourceModel is the Terraform resource model for the
// Self Service iOS & iPadOS branding singleton. main_header follows the UI
// (Settings > Self Service > Branding > iOS & iPadOS Branding); the colour
// fields have no UI form control (they are wire-only) so are named after the
// wire fields with descriptions explaining what each colours.
type SelfServiceBrandingIosResourceModel struct {
	ID                        types.String           `tfsdk:"id"`
	MainHeader                types.String           `tfsdk:"main_header"`                  // wire: brandingName
	BrandingNameColorCode     types.String           `tfsdk:"branding_name_color_code"`     // wire: brandingNameColorCode
	HeaderBackgroundColorCode types.String           `tfsdk:"header_background_color_code"` // wire: headerBackgroundColorCode
	MenuIconColorCode         types.String           `tfsdk:"menu_icon_color_code"`         // wire: menuIconColorCode
	StatusBarTextColor        types.String           `tfsdk:"status_bar_text_color"`        // wire: statusBarTextColor (light|dark)
	IconID                    types.Int64            `tfsdk:"icon_id"`                      // wire: iconId
	Timeouts                  resourceTimeouts.Value `tfsdk:"timeouts"`
}

// SelfServiceBrandingIosDataSourceModel is the read-only data source model.
type SelfServiceBrandingIosDataSourceModel struct {
	ID                        types.String             `tfsdk:"id"`
	MainHeader                types.String             `tfsdk:"main_header"`
	BrandingNameColorCode     types.String             `tfsdk:"branding_name_color_code"`
	HeaderBackgroundColorCode types.String             `tfsdk:"header_background_color_code"`
	MenuIconColorCode         types.String             `tfsdk:"menu_icon_color_code"`
	StatusBarTextColor        types.String             `tfsdk:"status_bar_text_color"`
	IconID                    types.Int64              `tfsdk:"icon_id"`
	Timeouts                  datasourceTimeouts.Value `tfsdk:"timeouts"`
}
