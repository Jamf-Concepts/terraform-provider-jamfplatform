// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package api_client

import (
	"context"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// buildApiClientInput converts the Terraform plan model into an SDK
// ApiIntegrationRequest payload. Update is a full-replace PUT, so the whole
// payload is always sent. `enabled` and `access_token_lifetime_seconds` are
// sent as nil pointers when unknown/null so Jamf Pro applies its server
// defaults (enabled=false, lifetime=300).
func buildApiClientInput(ctx context.Context, plan ApiClientResourceModel) (*pro.ApiIntegrationRequest, diag.Diagnostics) {
	scopes, diags := helpers.SetToStringSlice(ctx, plan.ApiRoles)
	if diags.HasError() {
		return nil, diags
	}

	req := &pro.ApiIntegrationRequest{
		DisplayName:         plan.DisplayName.ValueString(),
		AuthorizationScopes: scopes,
	}

	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		enabled := plan.Enabled.ValueBool()
		req.Enabled = &enabled
	}
	if !plan.AccessTokenLifetimeSeconds.IsNull() && !plan.AccessTokenLifetimeSeconds.IsUnknown() {
		lifetime := int(plan.AccessTokenLifetimeSeconds.ValueInt64())
		req.AccessTokenLifetimeSeconds = &lifetime
	}

	return req, diags
}
