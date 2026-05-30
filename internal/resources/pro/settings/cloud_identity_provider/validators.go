// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package cloud_identity_provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// providerBlockConfigValidator enforces the bidirectional cross-field rule
// between `provider_name` and the two provider blocks (`google`, `entra_id`) at
// plan time:
//
//   - provider_name = "GOOGLE"   → `google` must be set, `entra_id` absent.
//   - provider_name = "ENTRA_ID" → `entra_id` must be set, `google` absent.
//
// Off-the-shelf framework validators can express "exactly one of" but not the
// value-discriminated "the block matching provider_name is the one that must
// be present", so this is a custom resource.ConfigValidator (mirrors
// directory_binding's typeBlockConfigValidator). See STYLE_GUIDE §Cross-field
// validation.
//
// Per STYLE_GUIDE §"Config-time validators MUST defer on unknown values", the
// block presence is read from typed Object values via GetAttribute so an
// unknown block (sourced from a variable / another resource) defers rather
// than false-erroring on the required-when direction.
type providerBlockConfigValidator struct{}

// Description returns a plain-text description of the validator.
func (providerBlockConfigValidator) Description(context.Context) string {
	return "provider_name selects exactly one provider block: GOOGLE requires `google` (and forbids `entra_id`); ENTRA_ID requires `entra_id` (and forbids `google`)"
}

// MarkdownDescription returns the markdown description.
func (v providerBlockConfigValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateResource implements the plan-time cross-field check.
func (providerBlockConfigValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var providerName types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("provider_name"), &providerName)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if providerName.IsNull() || providerName.IsUnknown() {
		return
	}

	var google types.Object
	var entraID types.Object
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("google"), &google)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("entra_id"), &entraID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	switch providerName.ValueString() {
	case providerGoogle:
		requireBlock(resp, path.Root("google"), google, providerGoogle, "google")
		forbidBlock(resp, path.Root("entra_id"), entraID, providerGoogle, "entra_id", "google")
	case providerEntraID:
		requireBlock(resp, path.Root("entra_id"), entraID, providerEntraID, "entra_id")
		forbidBlock(resp, path.Root("google"), google, providerEntraID, "google", "entra_id")
	}
}

// requireBlock errors when the block matching provider_name is genuinely
// null. Defers on unknown (the value is not resolvable yet).
func requireBlock(resp *resource.ValidateConfigResponse, p path.Path, block types.Object, providerName, blockName string) {
	if block.IsUnknown() {
		return
	}
	if block.IsNull() {
		resp.Diagnostics.AddAttributeError(
			p,
			fmt.Sprintf("%s block required when provider_name = %q", blockName, providerName),
			fmt.Sprintf("`provider_name = %q` requires the `%s` block to be set.", providerName, blockName),
		)
	}
}

// forbidBlock errors when the non-matching block is present. Safe on unknown
// (a forbidden-when check fires on presence; unknown-treated-as-absent just
// defers).
func forbidBlock(resp *resource.ValidateConfigResponse, p path.Path, block types.Object, providerName, blockName, allowedBlock string) {
	if block.IsNull() || block.IsUnknown() {
		return
	}
	resp.Diagnostics.AddAttributeError(
		p,
		fmt.Sprintf("%s block forbidden when provider_name = %q", blockName, providerName),
		fmt.Sprintf("`%s` cannot be set when `provider_name = %q` — only the `%s` block applies. Remove the `%s` block, or change `provider_name`.", blockName, providerName, allowedBlock, blockName),
	)
}

// Compile-time interface assertion.
var _ resource.ConfigValidator = providerBlockConfigValidator{}
