// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package scripts

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/filters"
)

// ScriptsDataSourceModel represents the Terraform data source model for script searches.
type ScriptsDataSourceModel struct {
	ID       types.String                   `tfsdk:"id"`
	Scripts  []ScriptsDataSourceResultModel `tfsdk:"scripts"`
	Filters  []filters.FilterModel          `tfsdk:"filter"`
	Timeouts datasourceTimeouts.Value       `tfsdk:"timeouts"`
}

// ScriptsDataSourceResultModel represents a single script in the search results.
type ScriptsDataSourceResultModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	CategoryID     types.String `tfsdk:"category_id"`
	CategoryName   types.String `tfsdk:"category_name"`
	Info           types.String `tfsdk:"info"`
	Notes          types.String `tfsdk:"notes"`
	OsRequirements types.String `tfsdk:"os_requirements"`
	Priority       types.String `tfsdk:"priority"`
	Parameter4     types.String `tfsdk:"parameter_4"`
	Parameter5     types.String `tfsdk:"parameter_5"`
	Parameter6     types.String `tfsdk:"parameter_6"`
	Parameter7     types.String `tfsdk:"parameter_7"`
	Parameter8     types.String `tfsdk:"parameter_8"`
	Parameter9     types.String `tfsdk:"parameter_9"`
	Parameter10    types.String `tfsdk:"parameter_10"`
	Parameter11    types.String `tfsdk:"parameter_11"`
	ScriptContents types.String `tfsdk:"script_contents"`
}
