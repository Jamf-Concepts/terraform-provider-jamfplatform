// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package advanced_computer_search

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/criteria"
)

// AdvancedComputerSearchResourceModel is the Terraform resource model for a
// Jamf Pro advanced computer search.
type AdvancedComputerSearchResourceModel struct {
	ID            types.String              `tfsdk:"id"`
	Name          types.String              `tfsdk:"name"`
	SiteID        types.String              `tfsdk:"site_id"`
	SiteName      types.String              `tfsdk:"site_name"`
	ViewAs        types.String              `tfsdk:"view_as"`
	Sort1         types.String              `tfsdk:"sort_1"`
	Sort2         types.String              `tfsdk:"sort_2"`
	Sort3         types.String              `tfsdk:"sort_3"`
	Criteria      []criteria.CriterionModel `tfsdk:"criteria"`
	DisplayFields types.Set                 `tfsdk:"display_fields"`
	Timeouts      resourceTimeouts.Value    `tfsdk:"timeouts"`
}

// AdvancedComputerSearchDataSourceModel is the Terraform data source model for a
// Jamf Pro advanced computer search. Either id or name must be supplied
// (enforced by ExactlyOneOf at config validation).
type AdvancedComputerSearchDataSourceModel struct {
	ID            types.String              `tfsdk:"id"`
	Name          types.String              `tfsdk:"name"`
	SiteID        types.String              `tfsdk:"site_id"`
	SiteName      types.String              `tfsdk:"site_name"`
	ViewAs        types.String              `tfsdk:"view_as"`
	Sort1         types.String              `tfsdk:"sort_1"`
	Sort2         types.String              `tfsdk:"sort_2"`
	Sort3         types.String              `tfsdk:"sort_3"`
	Criteria      []criteria.CriterionModel `tfsdk:"criteria"`
	DisplayFields types.Set                 `tfsdk:"display_fields"`
	Timeouts      datasourceTimeouts.Value  `tfsdk:"timeouts"`
}

// advancedComputerSearchIdentityModel is the identity object for the resource
// and list results.
type advancedComputerSearchIdentityModel struct {
	ID types.String `tfsdk:"id"`
}
