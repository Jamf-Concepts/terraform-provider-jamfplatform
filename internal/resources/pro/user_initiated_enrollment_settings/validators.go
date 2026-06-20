// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package user_initiated_enrollment_settings

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// mdmSigningCertificateRequiredValidator enforces the server invariant: when
// signing_mdm_profile_enabled = true, an MDM signing certificate must be
// present. Jamf Pro rejects an apply with signing_mdm_profile_enabled = true
// and no certificate (HTTP 400 FIELD_REQUIRED mdmSigningCertificateIdentity).
//
// This is a config-only check (ValidateConfig cannot see prior state), so it
// requires the mdm_signing_certificate block in configuration whenever
// signing_mdm_profile_enabled = true. The "a certificate was uploaded on a
// previous apply" exception is handled at apply time by NOT re-uploading when
// nothing changed — not by loosening this validator.
type mdmSigningCertificateRequiredValidator struct{}

var _ resource.ConfigValidator = mdmSigningCertificateRequiredValidator{}

// Description returns the validator description.
func (mdmSigningCertificateRequiredValidator) Description(_ context.Context) string {
	return "signing_mdm_profile_enabled = true requires the mdm_signing_certificate block"
}

// MarkdownDescription returns the markdown description.
func (v mdmSigningCertificateRequiredValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateResource implements resource.ConfigValidator.
func (mdmSigningCertificateRequiredValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var enabled types.Bool
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("signing_mdm_profile_enabled"), &enabled)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if enabled.IsNull() || enabled.IsUnknown() || !enabled.ValueBool() {
		return
	}

	var cert types.Object
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("mdm_signing_certificate"), &cert)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Defer when the certificate block is unknown (e.g. driven by a variable,
	// for_each, or another resource): config-time validation cannot see its
	// eventual value, and erroring here would break every non-literal config.
	// Only a genuinely-absent (null) block is an error.
	if cert.IsUnknown() {
		return
	}
	if !cert.IsNull() {
		return
	}
	resp.Diagnostics.AddAttributeError(
		path.Root("mdm_signing_certificate"),
		"mdm_signing_certificate required when signing_mdm_profile_enabled = true",
		"Using a third-party signing certificate requires an uploaded certificate. Supply the mdm_signing_certificate block (keystore_file + keystore_password), or set signing_mdm_profile_enabled = false.",
	)
}
