// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package api_client

import (
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// validateCredentialRotationRequiresEnabled enforces the server invariant that
// client credentials can only be minted while the API client is enabled: Jamf
// Pro rejects a rotation on a disabled client with
// 400 "API Integration is disabled. Enable it first before generating a new
// client secret." When `credential_rotation` is set, `enabled` must be true.
//
// Only asserts when `enabled` is known: on create `enabled` may be unknown
// (Optional+Computed, not configured) — Create's guard and the server's clear
// 400 still catch that case.
func validateCredentialRotationRequiresEnabled(rotation types.String, enabled types.Bool, enabledPath path.Path) diag.Diagnostics {
	var diags diag.Diagnostics
	if rotation.IsNull() || rotation.IsUnknown() {
		return diags
	}
	if enabled.IsUnknown() {
		return diags
	}
	if !enabled.ValueBool() {
		diags.AddAttributeError(
			enabledPath,
			"credential_rotation requires enabled = true",
			"client_secret can only be generated while the API client is enabled. Set `enabled = true` when `credential_rotation` is set, or remove `credential_rotation`. Disabling an API client revokes its client credentials.",
		)
	}
	return diags
}
