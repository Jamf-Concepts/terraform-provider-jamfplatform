// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package user_group

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// smartStaticConfigValidator enforces the smart/static cross-field rules at
// plan-time, before apply. The apply-time helper validateUserGroupPlan in
// helpers.go is retained as defence-in-depth (catches values that only
// become known during apply).
//
// Off-the-shelf framework validators (boolvalidator.AlsoRequires etc.) do
// not support value-based discrimination: they fire when an attribute is
// set, not when it equals a specific value of another attribute. The rule
// here is "criteria required *only when* group_type == smart" — value-based
// — so we implement a custom resource.ConfigValidator. See STYLE_GUIDE
// §Cross-field validation.
type smartStaticConfigValidator struct{}

// Description returns a plain-text description of the validator.
func (smartStaticConfigValidator) Description(context.Context) string {
	return "smart user groups require criteria and forbid members; static user groups forbid criteria"
}

// MarkdownDescription returns the markdown description.
func (v smartStaticConfigValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateResource implements the plan-time cross-field check.
func (smartStaticConfigValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data UserGroupResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if data.GroupType.IsNull() || data.GroupType.IsUnknown() {
		return
	}

	switch data.GroupType.ValueString() {
	case "smart":
		if len(data.Criteria) == 0 {
			resp.Diagnostics.AddAttributeError(
				path.Root("criteria"),
				"Missing required criteria",
				"Smart user groups require at least one criterion. Supply `criteria = [...]` or change `group_type` to `\"static\"`.",
			)
		}
		if helpers.IsConfiguredValue(data.Members) {
			resp.Diagnostics.AddAttributeError(
				path.Root("members"),
				"members forbidden on smart user groups",
				"Smart user groups derive membership from `criteria` — remove the `members` attribute or change `group_type` to `\"static\"`.",
			)
		}
	case "static":
		if len(data.Criteria) > 0 {
			resp.Diagnostics.AddAttributeError(
				path.Root("criteria"),
				"criteria forbidden on static user groups",
				"Static user groups are populated via `members` — remove the `criteria` attribute or change `group_type` to `\"smart\"`.",
			)
		}
	}
}
