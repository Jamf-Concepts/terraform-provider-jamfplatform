// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package buildings

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/filters"
)

// BuildingsDataSourceModel represents the Terraform data source model for building searches.
type BuildingsDataSourceModel struct {
	ID        types.String                     `tfsdk:"id"`
	Buildings []BuildingsDataSourceResultModel `tfsdk:"buildings"`
	Filters   []filters.FilterModel            `tfsdk:"filter"`
	Timeouts  datasourceTimeouts.Value         `tfsdk:"timeouts"`
}

// BuildingsDataSourceResultModel represents a single building in the search results.
type BuildingsDataSourceResultModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	City           types.String `tfsdk:"city"`
	Country        types.String `tfsdk:"country"`
	StateProvince  types.String `tfsdk:"state_province"`
	StreetAddress1 types.String `tfsdk:"street_address_1"`
	StreetAddress2 types.String `tfsdk:"street_address_2"`
	ZipPostalCode  types.String `tfsdk:"zip_postal_code"`
}
