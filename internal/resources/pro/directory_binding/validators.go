// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package directory_binding

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// typeBlockConfigValidator enforces the bidirectional cross-field rule
// between `type` and the four per-type nested blocks
// (`active_directory`, `open_directory`, `admitmac`, `centrify`) at plan
// time, before apply.
//
// The rule, derived from the §13.2 wire audit (see
// local-testing/directorybindings/AUDIT_FINDINGS.md):
//
//   - `type = "Active Directory"` → only the `active_directory` nested
//     block may be supplied; the other three must be absent.
//   - `type = "Open Directory"`   → only `open_directory`.
//   - `type = "ADmitMac"`         → only `admitmac`.
//   - `type = "Centrify"`         → only `centrify`.
//   - `type = "PowerBroker Identity Services"` → ALL four nested blocks
//     must be absent. PowerBroker carries no per-type fields in the wire,
//     so the TF schema exposes no nested block for it; the `type`
//     attribute on its own conveys the PowerBroker identity. The input
//     builder synthesises the empty SDK struct from `type` alone.
//
// Off-the-shelf framework validators (`stringvalidator.ConflictsWith`)
// fire when the companion is set regardless of `type`'s value — that is
// the wrong shape here because the conflict is value-specific. Hence a
// custom resource.ConfigValidator. Mirrors `printer`'s
// `useGenericPPDConfigValidator`. See STYLE_GUIDE §Cross-field
// validation.
type typeBlockConfigValidator struct{}

// Description returns a plain-text description of the validator.
func (typeBlockConfigValidator) Description(context.Context) string {
	return "each value of `type` permits exactly one nested per-type block (or none, for the PowerBroker type); supplying a block whose name does not match `type` is rejected at plan time"
}

// MarkdownDescription returns the markdown description.
func (v typeBlockConfigValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateResource implements the plan-time cross-field check.
func (typeBlockConfigValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data DirectoryBindingResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.Type.IsNull() || data.Type.IsUnknown() {
		return
	}

	bindingType := data.Type.ValueString()

	// allowedBlock names the at-most-one TF attribute that may be set
	// for the given `type`. Empty string means "no block allowed" — used
	// for PowerBroker.
	var allowedBlock string
	switch bindingType {
	case typeActiveDirectory:
		allowedBlock = "active_directory"
	case typeOpenDirectory:
		allowedBlock = "open_directory"
	case typeADmitMac:
		allowedBlock = "admitmac"
	case typeCentrify:
		allowedBlock = "centrify"
	case typePowerBroker:
		// All four blocks forbidden; allowedBlock stays "".
	default:
		// stringvalidator.OneOf on the schema rejects unknown values
		// at this same plan step, so we have nothing further to add.
		return
	}

	type nestedAttr struct {
		name string
		set  bool
	}
	supplied := []nestedAttr{
		{"active_directory", data.ActiveDirectory != nil},
		{"open_directory", data.OpenDirectory != nil},
		{"admitmac", data.Admitmac != nil},
		{"centrify", data.Centrify != nil},
	}

	for _, n := range supplied {
		if !n.set {
			continue
		}
		if n.name == allowedBlock {
			continue
		}
		resp.Diagnostics.AddAttributeError(
			path.Root(n.name),
			fmt.Sprintf("%s forbidden when type = %q", n.name, bindingType),
			explainBlockForbidden(n.name, bindingType, allowedBlock),
		)
	}
}

// explainBlockForbidden produces the long-form diagnostic detail string
// used by typeBlockConfigValidator. Pulled out so the diagnostic message
// is consistent across the four "this block is wrong for that type"
// permutations.
func explainBlockForbidden(attemptedBlock, bindingType, allowedBlock string) string {
	if allowedBlock == "" {
		return fmt.Sprintf(
			"`%s` cannot be set when `type = %q` — the PowerBroker variant carries no per-type configuration on the wire, so no nested block is permitted. Remove the `%s` block to apply.",
			attemptedBlock, bindingType, attemptedBlock,
		)
	}
	return fmt.Sprintf(
		"`%s` cannot be set when `type = %q` — only the `%s` block applies to this directory binding type. Either change `type` to the value matching the block you supplied, or remove the `%s` block.",
		attemptedBlock, bindingType, allowedBlock, attemptedBlock,
	)
}

// Compile-time interface assertion.
var _ resource.ConfigValidator = typeBlockConfigValidator{}
