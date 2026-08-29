// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package api_client

import (
	"context"
	"strconv"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// appTypeNone is the server's app_type value for a client with no active
// credentials (never generated, or revoked by disabling the client).
const appTypeNone = pro.ApiIntegrationResponseAppTypeNone

// idString renders the integration's int ID as the canonical string Terraform
// ID / import string.
func idString(id int) string {
	return strconv.Itoa(id)
}

// assignApiClientServerFields populates the server-derived fields of a resource
// model from an ApiIntegrationResponse. It deliberately does NOT touch
// client_secret or credential_rotation — those are managed by the CRUD caller,
// which holds the rotation outcome and prior state.
func assignApiClientServerFields(ctx context.Context, state *ApiClientResourceModel, resp *pro.ApiIntegrationResponse) diag.Diagnostics {
	state.ID = types.StringValue(strconv.Itoa(resp.ID))
	state.DisplayName = types.StringValue(resp.DisplayName)
	state.Enabled = types.BoolValue(resp.Enabled)
	state.AccessTokenLifetimeSeconds = types.Int64Value(int64(resp.AccessTokenLifetimeSeconds))
	state.ClientID = types.StringValue(resp.ClientID)
	state.AppType = types.StringValue(resp.AppType)

	scopes, diags := types.SetValueFrom(ctx, types.StringType, resp.AuthorizationScopes)
	if diags.HasError() {
		return diags
	}
	state.ApiRoles = scopes
	return diags
}

// resolveStoredSecret decides the client_secret to persist after a read or a
// non-rotating update. When the server reports app_type==NONE the credentials
// were never generated or were revoked (e.g. by disabling the client), so the
// stored secret is cleared; otherwise the prior secret is carried forward
// unchanged (the API never returns it).
func resolveStoredSecret(appType string, priorSecret types.String) types.String {
	if appType == appTypeNone {
		return types.StringNull()
	}
	return priorSecret
}

// assignApiClientDataSourceModel populates a data source model from an
// ApiIntegrationResponse. client_secret is never exposed on the data source.
func assignApiClientDataSourceModel(ctx context.Context, state *ApiClientDataSourceModel, resp *pro.ApiIntegrationResponse) diag.Diagnostics {
	state.ID = types.StringValue(strconv.Itoa(resp.ID))
	state.DisplayName = types.StringValue(resp.DisplayName)
	state.Enabled = types.BoolValue(resp.Enabled)
	state.AccessTokenLifetimeSeconds = types.Int64Value(int64(resp.AccessTokenLifetimeSeconds))
	state.ClientID = types.StringValue(resp.ClientID)
	state.AppType = types.StringValue(resp.AppType)

	scopes, diags := types.SetValueFrom(ctx, types.StringType, resp.AuthorizationScopes)
	if diags.HasError() {
		return diags
	}
	state.ApiRoles = scopes
	return diags
}
