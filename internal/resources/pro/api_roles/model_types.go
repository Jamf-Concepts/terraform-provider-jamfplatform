// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package api_roles

import (
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/filters"
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ApiRolesDataSourceModel represents the Terraform data source model for API role searches.
type ApiRolesDataSourceModel struct {
	ID       types.String                    `tfsdk:"id"`
	ApiRoles []ApiRolesDataSourceResultModel `tfsdk:"api_roles"`
	Filters  []filters.FilterModel           `tfsdk:"filter"`
	Timeouts datasourceTimeouts.Value        `tfsdk:"timeouts"`
}

// ApiRolesDataSourceResultModel represents a single API role in the search results.
type ApiRolesDataSourceResultModel struct {
	ID          types.String `tfsdk:"id"`
	DisplayName types.String `tfsdk:"display_name"`
	Privileges  types.Set    `tfsdk:"privileges"`
}
