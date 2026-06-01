// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package api_role

import (
	"context"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// buildApiRoleInput converts the Terraform plan model into an SDK ApiRoleRequest payload.
// Update is a full-replace PUT, so the whole privileges set is always sent.
func buildApiRoleInput(ctx context.Context, plan ApiRoleResourceModel) (*pro.ApiRoleRequest, diag.Diagnostics) {
	privileges, diags := helpers.SetToStringSlice(ctx, plan.Privileges)
	if diags.HasError() {
		return nil, diags
	}
	return &pro.ApiRoleRequest{
		DisplayName: plan.DisplayName.ValueString(),
		Privileges:  privileges,
	}, diags
}
