// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package categories

import (
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/filters"
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// CategoriesDataSourceModel represents the Terraform data source model for category searches.
type CategoriesDataSourceModel struct {
	ID         types.String                      `tfsdk:"id"`
	Categories []CategoriesDataSourceResultModel `tfsdk:"categories"`
	Filters    []filters.FilterModel             `tfsdk:"filter"`
	Timeouts   datasourceTimeouts.Value          `tfsdk:"timeouts"`
}

// CategoriesDataSourceResultModel represents a single category in the search results.
type CategoriesDataSourceResultModel struct {
	ID       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	Priority types.Int64  `tfsdk:"priority"`
}
