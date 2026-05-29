// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package enrollment_customization

import (
	"context"
	"fmt"
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

// uniqueDisplayNameValidator enforces that each element of a
// ListNestedAttribute carries a unique `display_name` string across the list.
// The Jamf Pro panel endpoints tolerate duplicate display names server-side,
// but duplicates make admin-UI navigation ambiguous and the spike doc
// declares them invalid at the schema layer.
type uniqueDisplayNameValidator struct{}

// UniqueDisplayNameValidator constructs the validator.
func UniqueDisplayNameValidator() validator.List { return uniqueDisplayNameValidator{} }

// Description returns the validator description.
func (uniqueDisplayNameValidator) Description(_ context.Context) string {
	return "every element must have a unique display_name within the list"
}

// MarkdownDescription returns the markdown description.
func (v uniqueDisplayNameValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateList implements validator.List.
func (uniqueDisplayNameValidator) ValidateList(ctx context.Context, req validator.ListRequest, resp *validator.ListResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	elements := req.ConfigValue.Elements()
	seen := make(map[string]int, len(elements))
	for i, elem := range elements {
		obj, ok := elem.(types.Object)
		if !ok || obj.IsNull() || obj.IsUnknown() {
			continue
		}
		attrs := obj.Attributes()
		raw, ok := attrs["display_name"]
		if !ok {
			continue
		}
		name, ok := raw.(types.String)
		if !ok || name.IsNull() || name.IsUnknown() {
			continue
		}
		key := name.ValueString()
		if first, dup := seen[key]; dup {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtListIndex(i).AtName("display_name"),
				"duplicate display_name within list",
				fmt.Sprintf("display_name %q was already used at index %d. Each element must have a unique display_name.", key, first),
			)
			continue
		}
		seen[key] = i
	}
}

// Compile-time assertion.
var _ validator.List = uniqueDisplayNameValidator{}
