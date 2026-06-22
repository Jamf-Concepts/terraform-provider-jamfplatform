// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package api_client

import (
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/filters"
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ApiClientResourceModel represents the Terraform resource model for a Jamf Pro API client.
type ApiClientResourceModel struct {
	ID                         types.String           `tfsdk:"id"`
	DisplayName                types.String           `tfsdk:"display_name"`
	ApiRoles                   types.Set              `tfsdk:"api_roles"`
	Enabled                    types.Bool             `tfsdk:"enabled"`
	AccessTokenLifetimeSeconds types.Int64            `tfsdk:"access_token_lifetime_seconds"`
	ClientID                   types.String           `tfsdk:"client_id"`
	AppType                    types.String           `tfsdk:"app_type"`
	CredentialRotation         types.String           `tfsdk:"credential_rotation"`
	ClientSecret               types.String           `tfsdk:"client_secret"`
	Timeouts                   resourceTimeouts.Value `tfsdk:"timeouts"`
}

// ApiClientDataSourceModel represents the Terraform data source model for a Jamf Pro API client.
// The data source never exposes client_secret — the server never returns it on read.
type ApiClientDataSourceModel struct {
	ID                         types.String             `tfsdk:"id"`
	DisplayName                types.String             `tfsdk:"display_name"`
	ApiRoles                   types.Set                `tfsdk:"api_roles"`
	Enabled                    types.Bool               `tfsdk:"enabled"`
	AccessTokenLifetimeSeconds types.Int64              `tfsdk:"access_token_lifetime_seconds"`
	ClientID                   types.String             `tfsdk:"client_id"`
	AppType                    types.String             `tfsdk:"app_type"`
	Timeouts                   datasourceTimeouts.Value `tfsdk:"timeouts"`
}

// apiClientIdentityModel represents the identity object for API client resources and list results.
type apiClientIdentityModel struct {
	ID types.String `tfsdk:"id"`
}

// ApiClientListResourceModel represents the config model for API client list queries.
type ApiClientListResourceModel struct {
	Filters []filters.FilterModel `tfsdk:"filter"`
}

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
