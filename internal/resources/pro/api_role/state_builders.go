// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package api_role

import (
	"context"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// assignApiRoleResourceModel populates a resource model from an ApiRole response.
// Privileges are stored as a set; Jamf Pro returns them alphabetically sorted, so
// order is not significant.
func assignApiRoleResourceModel(ctx context.Context, state *ApiRoleResourceModel, role *pro.ApiRole) diag.Diagnostics {
	state.ID = types.StringValue(role.ID)
	state.DisplayName = types.StringValue(role.DisplayName)
	privileges, diags := types.SetValueFrom(ctx, types.StringType, role.Privileges)
	if diags.HasError() {
		return diags
	}
	state.Privileges = privileges
	return diags
}

// assignApiRoleDataSourceModel populates a data source model from an ApiRole response.
func assignApiRoleDataSourceModel(ctx context.Context, state *ApiRoleDataSourceModel, role *pro.ApiRole) diag.Diagnostics {
	state.ID = types.StringValue(role.ID)
	state.DisplayName = types.StringValue(role.DisplayName)
	privileges, diags := types.SetValueFrom(ctx, types.StringType, role.Privileges)
	if diags.HasError() {
		return diags
	}
	state.Privileges = privileges
	return diags
}
