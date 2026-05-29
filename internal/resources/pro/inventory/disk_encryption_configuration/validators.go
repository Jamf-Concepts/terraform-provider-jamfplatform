// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package disk_encryption_configuration

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
	// Read only the two attributes this cross-field check needs, via
	// GetAttribute, so the validator is light to unit-test (no full-model
	// fixture) and reads no more than necessary.
	var keyType types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("key_type"), &keyType)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if keyType.IsNull() || keyType.IsUnknown() {
		return
	}

	kt := keyType.ValueString()
	needsIRK := kt == keyTypeInstitutional || kt == keyTypeIndividualInstitutional
	if !needsIRK {
		return
	}

	// Read the block as a typed Object to tell "unknown" (driven by a variable /
	// for_each / another resource) apart from "absent": decoding into the Go
	// *struct collapses both to nil. Config-time validation cannot see an
	// unknown value, so defer — erroring here would break every non-literal
	// config. Only a genuinely-absent (null) block is the error.
	var irk types.Object
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("institutional_recovery_key"), &irk)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if irk.IsUnknown() {
		return
	}

	// IRK block absent: error. `data` and `certificate_type` are marked
	// Required at the schema layer so we don't double-check them here —
	// the framework rejects a partially-populated block at plan time.
	if irk.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("institutional_recovery_key"),
			fmt.Sprintf("institutional_recovery_key required when key_type = %q", kt),
			fmt.Sprintf(
				"`key_type = %q` derives FileVault recovery keys from a stored institutional certificate, but no `institutional_recovery_key` block was supplied. Add the block with `data`, `certificate_type`, and `password` (for `.p12` uploads) to apply.",
				kt,
			),
		)
	}
}

// Compile-time interface assertion.
var _ resource.ConfigValidator = institutionalKeyTypeRequiresIRKConfigValidator{}
