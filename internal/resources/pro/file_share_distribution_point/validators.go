// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package file_share_distribution_point

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// transportConfigValidator enforces that a distribution point exposes at least
// one transport: either a file-sharing protocol (`file_sharing_connection_type`
// of AFP or SMB) or HTTPS downloads (`https_enabled = true`). Jamf Pro rejects a
// distribution point that has neither.
//
// It fires only on the unambiguous explicit case — `file_sharing_connection_type
// = NONE` together with `https_enabled` explicitly `false`. When `https_enabled`
// is omitted it defers: the attribute is Optional+Computed, so an omitted value
// may be preserved from prior state (UseStateForUnknown), which the config alone
// cannot see. The server still rejects a genuinely transport-less create.
type transportConfigValidator struct{}

func (transportConfigValidator) Description(context.Context) string {
	return "a distribution point must use a file-sharing protocol (file_sharing_connection_type AFP or SMB) or enable HTTPS downloads (https_enabled = true)"
}

func (v transportConfigValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (transportConfigValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data FileShareDistributionPointResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(validateTransport(data)...)
}

// validateTransport is the pure validation logic for transportConfigValidator.
func validateTransport(data FileShareDistributionPointResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	if data.FileSharingConnectionType.IsNull() || data.FileSharingConnectionType.IsUnknown() {
		return diags
	}
	if data.FileSharingConnectionType.ValueString() != connectionTypeNone {
		return diags
	}
	// Connection type is NONE — HTTPS must be the transport. Only error when
	// the user explicitly disabled it; defer on null/unknown.
	if data.HTTPSEnabled.IsNull() || data.HTTPSEnabled.IsUnknown() {
		return diags
	}
	if !data.HTTPSEnabled.ValueBool() {
		diags.AddAttributeError(
			path.Root("https_enabled"),
			"No distribution transport configured",
			"When file_sharing_connection_type is \"NONE\", https_enabled must be true so the distribution point can serve packages over HTTPS.",
		)
	}
	return diags
}

// loadBalancingConfigValidator enforces that randomized load sharing
// (`enable_load_balancing = true`) is only set when the failover is another
// file share distribution point — that is, `backup_distribution_point_id` is a
// real distribution point ID, not "-1" (none) or "-2" (Jamf Cloud). Jamf Pro
// rejects load balancing in both sentinel cases.
//
// It fires only when both fields are explicitly set in config; it defers when
// `backup_distribution_point_id` is omitted, since the Optional+Computed value
// may be preserved from prior state.
type loadBalancingConfigValidator struct{}

func (loadBalancingConfigValidator) Description(context.Context) string {
	return "enable_load_balancing requires a file share failover distribution point (backup_distribution_point_id other than \"-1\" or \"-2\")"
}

func (v loadBalancingConfigValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (loadBalancingConfigValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data FileShareDistributionPointResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(validateLoadBalancing(data)...)
}

// validateLoadBalancing is the pure validation logic for
// loadBalancingConfigValidator.
func validateLoadBalancing(data FileShareDistributionPointResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	if data.EnableLoadBalancing.IsNull() || data.EnableLoadBalancing.IsUnknown() {
		return diags
	}
	if !data.EnableLoadBalancing.ValueBool() {
		return diags
	}
	if data.BackupDistributionPointID.IsNull() || data.BackupDistributionPointID.IsUnknown() {
		return diags
	}
	switch data.BackupDistributionPointID.ValueString() {
	case noneBackupSentinel:
		diags.AddAttributeError(
			path.Root("enable_load_balancing"),
			"Load balancing requires a failover distribution point",
			"enable_load_balancing can only be true when backup_distribution_point_id references another file share distribution point; it is currently \"-1\" (none).",
		)
	case cloudBackupSentinel:
		diags.AddAttributeError(
			path.Root("enable_load_balancing"),
			"Load balancing is not supported with a Jamf Cloud failover",
			"enable_load_balancing can only be true when backup_distribution_point_id references another file share distribution point; it is currently \"-2\" (the Jamf Cloud distribution point).",
		)
	}
	return diags
}
