// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package device_group

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// groupNameValidator rejects the two names Jamf Security Cloud will not store as
// authored, at plan time rather than mid-apply.
//
// Both rules are wire-probed (EU sandbox, 2026-08-29) and both need catching
// before the write for different reasons:
//
// Surrounding whitespace — the server silently TRIMS it. A config of
// `name = "  example  "` applies, reads back as "example", and Terraform then
// fails the apply with "Provider produced inconsistent result after apply". The
// alternative fix, trimming in the input builder, does not help: state would
// still disagree with config. Trimming in a plan modifier would work but rewrites
// what the user typed into a diff they did not author, so the validator refuses
// instead and names the constraint.
//
// The reserved name — creating or renaming a group to "Default Group" is refused
// with 400 RESERVED_GROUP_NAME. The server compares case-insensitively and after
// trimming, so both halves of this validator use the trimmed value.
//
// Null and unknown values are deferred to the server per STYLE_GUIDE
// §Config-time validators; blankness is already covered by the schema's
// LengthAtLeast(1), and a whitespace-only name is caught by the trim rule here.
type groupNameValidator struct{}

// GroupName returns a validator enforcing the two plan-time name rules.
func GroupName() validator.String {
	return groupNameValidator{}
}

// Description returns a plain-text description of the validator.
func (groupNameValidator) Description(_ context.Context) string {
	return fmt.Sprintf("must not begin or end with whitespace, and must not be %q", defaultGroupName)
}

// MarkdownDescription returns the markdown description of the validator.
func (v groupNameValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateString implements validator.String.
func (groupNameValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()
	trimmed := strings.TrimSpace(value)

	if trimmed != value {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Group name has surrounding whitespace",
			fmt.Sprintf(
				"Jamf Security Cloud removes leading and trailing whitespace from a group name, so %q would be "+
					"stored as %q and Terraform would report the difference as an inconsistent result. Remove the "+
					"surrounding whitespace.",
				value, trimmed,
			),
		)
		return
	}

	if strings.EqualFold(trimmed, defaultGroupName) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Group name is reserved",
			fmt.Sprintf(
				"%q is the name of the built-in group every tenant already has, and Jamf Security Cloud reserves it "+
					"regardless of capitalisation. Choose a different name. The built-in group cannot be managed by "+
					"Terraform — it has no identifier — but it is reported by the "+
					"`jamfplatform_security_cloud_device_groups` data source with `built_in` set to `true`.",
				defaultGroupName,
			),
		)
	}
}
