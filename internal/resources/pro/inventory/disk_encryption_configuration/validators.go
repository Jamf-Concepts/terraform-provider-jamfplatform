// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package disk_encryption_configuration

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// institutionalKeyTypeRequiresIRKConfigValidator enforces the rule that
// `key_type ∈ {Institutional, Individual and Institutional}` requires
// an `institutional_recovery_key` block carrying recovery cert material
// (`data`) at plan time.
//
// Wire reference: audit §1, §4. The classic POST endpoint rejects an
// Institutional create without a usable cert with HTTP 409 ("Problem
// creating certificate"); surfacing that as a plan-time validation
// avoids round-tripping to the API for an error the schema can detect.
//
// The PKCS12-requires-password check from §4 is intentionally omitted —
// `certificate_type` is server-determined and Computed in the schema, so
// it is Unknown at plan time and we cannot validate it without round-
// tripping to the API. Documented in the schema description instead.
type institutionalKeyTypeRequiresIRKConfigValidator struct{}

// Description returns a plain-text description of the validator.
func (institutionalKeyTypeRequiresIRKConfigValidator) Description(context.Context) string {
	return "key_type=Institutional and key_type=\"Individual and Institutional\" require institutional_recovery_key.data to be set"
}

// MarkdownDescription returns the markdown description.
func (v institutionalKeyTypeRequiresIRKConfigValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateResource implements the plan-time check.
func (institutionalKeyTypeRequiresIRKConfigValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data DiskEncryptionConfigurationResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.KeyType.IsNull() || data.KeyType.IsUnknown() {
		return
	}

	kt := data.KeyType.ValueString()
	needsIRK := kt == keyTypeInstitutional || kt == keyTypeIndividualInstitutional
	if !needsIRK {
		return
	}

	// IRK block absent: error.
	if data.InstitutionalRecoveryKey == nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("institutional_recovery_key"),
			fmt.Sprintf("institutional_recovery_key required when key_type = %q", kt),
			fmt.Sprintf(
				"`key_type = %q` derives FileVault recovery keys from a stored institutional certificate, but no `institutional_recovery_key` block was supplied. Add the block with a base64-encoded `data` payload (and `password` for `.p12` uploads) to apply.",
				kt,
			),
		)
		return
	}

	// IRK block present but missing `data`: error. We tolerate Unknown
	// (treated as "user might be passing a computed value") and only fail
	// when the user provided an explicit null or empty string.
	d := data.InstitutionalRecoveryKey.Data
	if d.IsUnknown() {
		return
	}
	if d.IsNull() || strings.TrimSpace(d.ValueString()) == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("institutional_recovery_key").AtName("data"),
			fmt.Sprintf("institutional_recovery_key.data required when key_type = %q", kt),
			fmt.Sprintf(
				"`key_type = %q` requires a recovery certificate. Provide the base64-encoded cert via `institutional_recovery_key.data` (and `password` for PKCS12 uploads).",
				kt,
			),
		)
	}
}

// Compile-time interface assertion.
var _ resource.ConfigValidator = institutionalKeyTypeRequiresIRKConfigValidator{}
