// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ztna_shared_gateways

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// SharedGatewaysDataSourceModel represents the Terraform data source model for the
// Jamf-managed shared ZTNA gateway catalogue.
type SharedGatewaysDataSourceModel struct {
	ID             types.String               `tfsdk:"id"`
	SharedGateways []SharedGatewayResultModel `tfsdk:"shared_gateways"`
	Timeouts       datasourceTimeouts.Value   `tfsdk:"timeouts"`
}

// SharedGatewayResultModel represents a single shared gateway in the results.
type SharedGatewayResultModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}
