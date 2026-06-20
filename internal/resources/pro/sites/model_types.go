// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package sites

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/filters"
)

// SitesDataSourceModel represents the Terraform data source model for site searches.
type SitesDataSourceModel struct {
	ID       types.String                 `tfsdk:"id"`
	Sites    []SitesDataSourceResultModel `tfsdk:"sites"`
	Filter   *filters.ClassicFilterModel  `tfsdk:"filter"`
	Timeouts datasourceTimeouts.Value     `tfsdk:"timeouts"`
}

// SitesDataSourceResultModel represents a single site in the search results.
type SitesDataSourceResultModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}
