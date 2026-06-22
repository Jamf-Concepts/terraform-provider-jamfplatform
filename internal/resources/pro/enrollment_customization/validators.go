// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package enrollment_customization

import (
	"context"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// hexColorPattern matches a 6-digit hexadecimal RGB value without a leading
// `#`. Used to validate every colour attribute on the branding settings.
var hexColorPattern = regexp.MustCompile(`^[0-9a-fA-F]{6}$`)

// enrollmentAccess enum values for the SSO pane.
const (
	enrollmentAccessAnyIdpUser    = "any_idp_user"
	enrollmentAccessSpecificGroup = "specific_group"
)

// accessGroupNameValidator enforces access_group_name non-empty when
// enrollment_access = "specific_group" on the same SSO pane element. The
// off-the-shelf stringvalidator.AlsoRequires cannot express the rule because
// it would fire for both enum values; only the "specific_group" branch
// requires a companion.
type accessGroupNameValidator struct{}

// AccessGroupNameValidator constructs the validator.
func AccessGroupNameValidator() validator.String { return accessGroupNameValidator{} }

// Description returns the validator description.
func (accessGroupNameValidator) Description(_ context.Context) string {
	return `access_group_name is required when enrollment_access = "specific_group"`
}

// MarkdownDescription returns the markdown description.
func (v accessGroupNameValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateString implements validator.String.
func (accessGroupNameValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	var enrollmentAccess types.String
	parent := req.Path.ParentPath()
	if d := req.Config.GetAttribute(ctx, parent.AtName("enrollment_access"), &enrollmentAccess); d.HasError() {
		return
	}
	if enrollmentAccess.IsNull() || enrollmentAccess.IsUnknown() {
		return
	}
	if enrollmentAccess.ValueString() != enrollmentAccessSpecificGroup {
		return
	}
	// Defer when the value is unknown (variable/for_each-driven): config-time
	// validation cannot see it. Error only when genuinely absent/empty.
	if req.ConfigValue.IsUnknown() {
		return
	}
	if !req.ConfigValue.IsNull() && !req.ConfigValue.IsUnknown() && req.ConfigValue.ValueString() != "" {
		return
	}
	resp.Diagnostics.AddAttributeError(
		req.Path,
		`access_group_name required when enrollment_access = "specific_group"`,
		`Supply the IdP group name allowed to enrol, or change enrollment_access to "any_idp_user".`,
	)
}

// Compile-time assertion.
var _ validator.String = accessGroupNameValidator{}
