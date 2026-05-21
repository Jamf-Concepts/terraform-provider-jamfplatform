// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package departments

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/filters"
)

// DepartmentsDataSourceModel represents the Terraform data source model for department searches.
type DepartmentsDataSourceModel struct {
	ID          types.String                       `tfsdk:"id"`
	Departments []DepartmentsDataSourceResultModel `tfsdk:"departments"`
	Filters     []filters.FilterModel              `tfsdk:"filter"`
	Timeouts    datasourceTimeouts.Value           `tfsdk:"timeouts"`
}

// DepartmentsDataSourceResultModel represents a single department in the search results.
type DepartmentsDataSourceResultModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}
