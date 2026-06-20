// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package advanced_mobile_device_search

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// AdvancedMobileDeviceSearchResourceModel is the Terraform resource model for a
// Jamf Pro advanced mobile device search. The matched-device result set and the
// Reports tab (file format / scheduled email) are server-side report concerns
// and are intentionally not modelled — the resource manages only the search
// definition (criteria, display columns, site). The Pro type carries only
// `siteId` (no site name on the wire), so there is no site_name attribute.
type AdvancedMobileDeviceSearchResourceModel struct {
	ID            types.String           `tfsdk:"id"`
	Name          types.String           `tfsdk:"name"`
	SiteID        types.String           `tfsdk:"site_id"`
	Criteria      types.List             `tfsdk:"criteria"`
	DisplayFields types.Set              `tfsdk:"display_fields"`
	Timeouts      resourceTimeouts.Value `tfsdk:"timeouts"`
}

// AdvancedMobileDeviceSearchDataSourceModel is the Terraform data source model.
// Either id or name must be supplied (enforced by ExactlyOneOf at config
// validation).
type AdvancedMobileDeviceSearchDataSourceModel struct {
	ID            types.String             `tfsdk:"id"`
	Name          types.String             `tfsdk:"name"`
	SiteID        types.String             `tfsdk:"site_id"`
	Criteria      types.List               `tfsdk:"criteria"`
	DisplayFields types.Set                `tfsdk:"display_fields"`
	Timeouts      datasourceTimeouts.Value `tfsdk:"timeouts"`
}

// advancedMobileDeviceSearchIdentityModel is the identity object for the resource
// and list results.
type advancedMobileDeviceSearchIdentityModel struct {
	ID types.String `tfsdk:"id"`
}
