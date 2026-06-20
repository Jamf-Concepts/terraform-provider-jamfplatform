// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package jamf_connect

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// versionDeploymentTypeValidator enforces, at plan time, the version ↔
// auto_deployment_type dependency Jamf Connect enforces on the wire:
//
//   - When auto_deployment_type is NONE, version is ignored and cleared by the
//     server, so it must NOT be set (matches the admin UI, which hides the
//     version field when automatic deployment is off).
//   - When auto_deployment_type is anything else, a version is required — the
//     server rejects a full-replace write that omits it (a non-NONE type with
//     no version returns HTTP 400 "not a valid semantic version").
//
// The validator defers (no error) whenever auto_deployment_type or version is
// unknown, per the convention that config-time validators must not fire on
// unresolved plan values.
type versionDeploymentTypeValidator struct{}

// Description returns a plain-text description of the validator.
func (versionDeploymentTypeValidator) Description(context.Context) string {
	return "version is required unless auto_deployment_type is NONE, and must be omitted when it is NONE."
}

// MarkdownDescription returns the markdown description.
func (v versionDeploymentTypeValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateResource implements the cross-field check.
func (versionDeploymentTypeValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var deploymentType, version types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("auto_deployment_type"), &deploymentType)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("version"), &version)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// auto_deployment_type is Optional+Computed with a NONE default. A null
	// config value resolves to NONE, so treat null as NONE; defer only on
	// unknown. Defer when version is unknown.
	if deploymentType.IsUnknown() || version.IsUnknown() {
		return
	}

	isNone := deploymentType.IsNull() || deploymentType.ValueString() == autoDeploymentNone
	hasVersion := !version.IsNull() && version.ValueString() != ""

	switch {
	case isNone && hasVersion:
		resp.Diagnostics.AddAttributeError(
			path.Root("version"),
			"version not allowed when auto_deployment_type is NONE",
			"Jamf Connect ignores the version when automatic deployment is off. Remove version, or set auto_deployment_type to a deployment type that uses it.",
		)
	case !isNone && !hasVersion:
		resp.Diagnostics.AddAttributeError(
			path.Root("version"),
			"version required",
			"A version must be set when auto_deployment_type is not NONE (the chosen version is what Jamf Connect deploys).",
		)
	}
}
