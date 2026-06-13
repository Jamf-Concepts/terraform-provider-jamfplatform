// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package managed_software_updates

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// specificVersionRequiredValidator enforces the one cross-field rule the server enforces on
// the wire (wire-probed 2026-06-13): version_type SPECIFIC_VERSION or CUSTOM_VERSION requires
// specific_version (the API returns 400 INVALID_SPECIFIC_VERSION otherwise). The UI-implied
// couplings for max_deferrals/force_install_local_date_time are NOT server-enforced, so they
// are deliberately not validated here — the server accepts or ignores them.
type specificVersionRequiredValidator struct{}

func (v specificVersionRequiredValidator) Description(_ context.Context) string {
	return "specific_version is required when version_type is SPECIFIC_VERSION or CUSTOM_VERSION"
}

func (v specificVersionRequiredValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v specificVersionRequiredValidator) ValidateAction(ctx context.Context, req action.ValidateConfigRequest, resp *action.ValidateConfigResponse) {
	var versionType types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("version_type"), &versionType)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Defer when the discriminator is unknown/null — the server (and the OneOf validator)
	// settle it.
	if versionType.IsNull() || versionType.IsUnknown() {
		return
	}
	vt := versionType.ValueString()
	if vt != "SPECIFIC_VERSION" && vt != "CUSTOM_VERSION" {
		return
	}

	var specificVersion types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("specific_version"), &specificVersion)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if specificVersion.IsUnknown() {
		return
	}
	if specificVersion.IsNull() || specificVersion.ValueString() == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("specific_version"),
			"Missing specific_version",
			"specific_version must be set when version_type is SPECIFIC_VERSION or CUSTOM_VERSION.",
		)
	}
}
