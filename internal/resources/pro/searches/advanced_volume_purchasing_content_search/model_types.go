// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package advanced_volume_purchasing_content_search

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/criteria"
)

// AdvancedVolumePurchasingContentSearchResourceModel is the Terraform resource model for a
// Jamf Pro advanced volume purchasing content search. The matched-content result
// set is a server-side report concern and is intentionally not modelled — the
// resource manages only the search definition (criteria, display columns, site).
// The Pro type carries only `siteId` (no site name on the wire), so there is no
// site_name attribute.
type AdvancedVolumePurchasingContentSearchResourceModel struct {
	ID            types.String              `tfsdk:"id"`
	Name          types.String              `tfsdk:"name"`
	SiteID        types.String              `tfsdk:"site_id"`
	Criteria      []criteria.CriterionModel `tfsdk:"criteria"`
	DisplayFields types.Set                 `tfsdk:"display_fields"`
	Timeouts      resourceTimeouts.Value    `tfsdk:"timeouts"`
}

// AdvancedVolumePurchasingContentSearchDataSourceModel is the Terraform data source model.
// Either id or name must be supplied (enforced by ExactlyOneOf at config
// validation).
type AdvancedVolumePurchasingContentSearchDataSourceModel struct {
	ID            types.String              `tfsdk:"id"`
	Name          types.String              `tfsdk:"name"`
	SiteID        types.String              `tfsdk:"site_id"`
	Criteria      []criteria.CriterionModel `tfsdk:"criteria"`
	DisplayFields types.Set                 `tfsdk:"display_fields"`
	Timeouts      datasourceTimeouts.Value  `tfsdk:"timeouts"`
}

// advancedVolumePurchasingContentSearchIdentityModel is the identity object for the resource
// and list results.
type advancedVolumePurchasingContentSearchIdentityModel struct {
	ID types.String `tfsdk:"id"`
}
