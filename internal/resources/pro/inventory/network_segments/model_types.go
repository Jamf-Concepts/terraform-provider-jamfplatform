// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package network_segments

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/filters"
)

// NetworkSegmentsDataSourceModel represents the Terraform data source model for
// network segment searches.
type NetworkSegmentsDataSourceModel struct {
	ID              types.String                           `tfsdk:"id"`
	NetworkSegments []NetworkSegmentsDataSourceResultModel `tfsdk:"network_segments"`
	Filter          *filters.ClassicFilterModel            `tfsdk:"filter"`
	Timeouts        datasourceTimeouts.Value               `tfsdk:"timeouts"`
}

// NetworkSegmentsDataSourceResultModel represents a single network segment in the
// search results. Only the fields the classic list endpoint actually returns are
// exposed — id, name, and the IP range. Per-item fields beyond these require a
// singular data source lookup.
type NetworkSegmentsDataSourceResultModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	StartingAddress types.String `tfsdk:"starting_address"`
	EndingAddress   types.String `tfsdk:"ending_address"`
}
