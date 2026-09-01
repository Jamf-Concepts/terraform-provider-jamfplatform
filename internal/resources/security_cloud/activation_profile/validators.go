// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package activation_profile

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// atLeastOneCapabilityValidator rejects a profile with no service capability
// enabled.
//
// Jamf Security Cloud enforces this, but as a business rule rather than field
// validation: the refusal arrives as a 400 in the service's own envelope rather
// than the documented error shape, so nothing in the response names a field or
// decodes into the structured error the rest of this namespace returns. Checking
// at plan time is the only way the operator sees which attribute is at fault.
type atLeastOneCapabilityValidator struct{}

// Description returns a plain-language description of the rule.
func (atLeastOneCapabilityValidator) Description(_ context.Context) string {
	return "at least one service capability must be enabled"
}

// MarkdownDescription returns the Markdown description of the rule.
func (v atLeastOneCapabilityValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateResource reports an error when every capability is disabled.
//
// Attributes are read individually with GetAttribute rather than decoding the
// whole config: a single unknown value anywhere in the configuration makes a
// whole-object Get fail, which would silently disable this check on any plan that
// derives an attribute from another resource.
//
// The block itself is checked for unknown before its members, and that order is
// load-bearing. Reading a child of an unknown parent yields **null**, not
// unknown, so a `capabilities` block taken wholesale from another resource's
// computed attribute would present as two disabled capabilities and fail a
// configuration that is perfectly valid.
func (v atLeastOneCapabilityValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var capabilities types.Object
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("capabilities"), &capabilities)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if capabilities.IsUnknown() || capabilities.IsNull() {
		return
	}

	contentControls := boolAt(ctx, req, resp, path.Root("capabilities").AtName("content_controls"))
	networkSecurity := boolAt(ctx, req, resp, path.Root("capabilities").AtName("network_security"))
	if resp.Diagnostics.HasError() {
		return
	}
	if contentControls.IsUnknown() || networkSecurity.IsUnknown() {
		return
	}
	if contentControls.ValueBool() || networkSecurity.ValueBool() {
		return
	}
	resp.Diagnostics.AddAttributeError(
		path.Root("capabilities"),
		"No service capability enabled",
		"An activation profile must enable at least one service capability. Set `content_controls`, "+
			"`network_security`, or both.",
	)
}

// boolAt reads one boolean attribute from the configuration.
func boolAt(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse, at path.Path) types.Bool {
	var value types.Bool
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, at, &value)...)
	return value
}
