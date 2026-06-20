// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package advanced_user_search

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/criteria"
)

// AdvancedUserSearchResourceModel is the Terraform resource model for a Jamf Pro
// advanced user search. Unlike computer searches, user searches have no view_as
// or sort columns (confirmed against the SDK struct and the admin UI).
type AdvancedUserSearchResourceModel struct {
	ID            types.String              `tfsdk:"id"`
	Name          types.String              `tfsdk:"name"`
	SiteID        types.String              `tfsdk:"site_id"`
	SiteName      types.String              `tfsdk:"site_name"`
	Criteria      []criteria.CriterionModel `tfsdk:"criteria"`
	DisplayFields types.Set                 `tfsdk:"display_fields"`
	Timeouts      resourceTimeouts.Value    `tfsdk:"timeouts"`
}

// AdvancedUserSearchDataSourceModel is the Terraform data source model for a Jamf
// Pro advanced user search. Either id or name must be supplied (enforced by
// ExactlyOneOf at config validation).
type AdvancedUserSearchDataSourceModel struct {
	ID            types.String              `tfsdk:"id"`
	Name          types.String              `tfsdk:"name"`
	SiteID        types.String              `tfsdk:"site_id"`
	SiteName      types.String              `tfsdk:"site_name"`
	Criteria      []criteria.CriterionModel `tfsdk:"criteria"`
	DisplayFields types.Set                 `tfsdk:"display_fields"`
	Timeouts      datasourceTimeouts.Value  `tfsdk:"timeouts"`
}

// advancedUserSearchIdentityModel is the identity object for the resource and
// list results.
type advancedUserSearchIdentityModel struct {
	ID types.String `tfsdk:"id"`
}
