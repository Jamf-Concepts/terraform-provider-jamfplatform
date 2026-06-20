// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package computer_inventory_collection_settings

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// accountCollectionParentPath is the toggle that gates the account sub-options.
var accountCollectionParentPath = path.Root("collect_local_user_accounts")

// requiresAccountCollection is the attribute-level validator for the two sub-options of
// collect_local_user_accounts (include_home_directory_sizes, include_hidden_accounts).
// It rejects a sub-option set to true while account collection is explicitly false — the
// combination the Jamf Pro admin UI greys out and the server refuses (the server
// silently forces the sub-option off, which would otherwise surface as a confusing
// "inconsistent result after apply").
//
// This is a value-specific "forbidden-when-true" rule, so it is a custom validator.Bool
// (off-the-shelf ConflictsWith would also fire when the sub-option is false). A plan
// modifier cannot enforce it: the framework forbids overriding an explicitly-configured
// value ("planned value does not match config value"), and an unset sub-option already
// resolves to the server's false harmlessly.
type requiresAccountCollection struct{}

var _ validator.Bool = requiresAccountCollection{}

func (v requiresAccountCollection) Description(_ context.Context) string {
	return "may only be true when collect_local_user_accounts is true"
}

func (v requiresAccountCollection) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v requiresAccountCollection) ValidateBool(ctx context.Context, req validator.BoolRequest, resp *validator.BoolResponse) {
	// Forbidden-when-true: only fires when this sub-option is known to be true.
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() || !req.ConfigValue.ValueBool() {
		return
	}

	var parent types.Bool
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, accountCollectionParentPath, &parent)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Defer on unknown/absent (a variable- or resource-sourced parent, or an omitted
	// Optional attribute, is not resolvable here); error only when the parent is
	// genuinely false.
	if parent.IsNull() || parent.IsUnknown() || parent.ValueBool() {
		return
	}

	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Invalid Attribute Combination",
		fmt.Sprintf("%s requires collect_local_user_accounts to be true. Set collect_local_user_accounts = true, or remove this attribute (or set it to false).", req.Path),
	)
}
