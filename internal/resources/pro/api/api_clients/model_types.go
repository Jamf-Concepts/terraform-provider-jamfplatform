// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package api_clients

import (
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/filters"
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ApiClientsDataSourceModel represents the Terraform data source model for API client searches.
type ApiClientsDataSourceModel struct {
	ID         types.String                      `tfsdk:"id"`
	ApiClients []ApiClientsDataSourceResultModel `tfsdk:"api_clients"`
	Filters    []filters.FilterModel             `tfsdk:"filter"`
	Timeouts   datasourceTimeouts.Value          `tfsdk:"timeouts"`
}

// ApiClientsDataSourceResultModel represents a single API client in the search results.
// The client secret is never exposed — Jamf Pro does not return it on read.
type ApiClientsDataSourceResultModel struct {
	ID                         types.String `tfsdk:"id"`
	DisplayName                types.String `tfsdk:"display_name"`
	ApiRoles                   types.Set    `tfsdk:"api_roles"`
	Enabled                    types.Bool   `tfsdk:"enabled"`
	AccessTokenLifetimeSeconds types.Int64  `tfsdk:"access_token_lifetime_seconds"`
	ClientID                   types.String `tfsdk:"client_id"`
	AppType                    types.String `tfsdk:"app_type"`
}
