// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package policy

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// minutesPerDay is the wire-required granularity for
// user_interaction.allow_deferral_minutes. The classic API enforces
// "allow_deferral_minutes must be a multiple of 1440 (minutes in day)" and
// returns HTTP 409 on any non-multiple.
const minutesPerDay = 1440

// multipleOfInt64Validator enforces that an Int64 attribute is a positive
// multiple of a fixed divisor. The classic /policies endpoint applies this to
// user_interaction.allow_deferral_minutes (divisor 1440 — one day). A
// dedicated validator is needed because terraform-plugin-framework-validators
// only ships int64validator.OneOf (enumerable) and int64validator.Between
// (range) — neither expresses "any multiple of N".
type multipleOfInt64Validator struct {
	divisor int64
}

// MultipleOfInt64 returns a validator.Int64 enforcing multiple-of-divisor.
func MultipleOfInt64(divisor int64) validator.Int64 {
	return multipleOfInt64Validator{divisor: divisor}
}

// Description returns a plain-text description of the validator.
func (v multipleOfInt64Validator) Description(context.Context) string {
	return fmt.Sprintf("value must be a multiple of %d", v.divisor)
}

// MarkdownDescription returns the markdown description.
func (v multipleOfInt64Validator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateInt64 implements validator.Int64.
func (v multipleOfInt64Validator) ValidateInt64(ctx context.Context, req validator.Int64Request, resp *validator.Int64Response) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueInt64()
	if v.divisor == 0 || value%v.divisor != 0 {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid value",
			fmt.Sprintf("Expected a multiple of %d, got %d.", v.divisor, value),
		)
	}
}

// userInteractionConfigValidator enforces the two cross-field constraints the
// classic /policies endpoint applies to <user_interaction>:
//
//  1. When allow_users_to_defer = false, neither allow_deferral_until_utc nor
//     allow_deferral_minutes may be set. The server returns 409 with `Error:
//     When 'allow_users_to_defer' is false, 'allow_deferral_until_utc' and
//     'allow_deferral_minutes' cannot be configured`.
//  2. allow_deferral_until_utc and allow_deferral_minutes are mutually
//     exclusive. The server returns 409 with `Error: You cannot use both
//     'allow_deferral_until_utc' and 'allow_deferral_minutes'`. Note that
//     omitting one field on an Update does not clear a previously-stored
//     value (Optional+Computed semantics keep prior state); transitioning
//     between the two deferral forms therefore requires destroy+recreate.
//
// The third documented wire constraint (allow_deferral_minutes must be a
// multiple of 1440) is enforced inline on the attribute via MultipleOfInt64.
type userInteractionConfigValidator struct{}

// UserInteractionConfigValidator returns the resource.ConfigValidator wiring
// the user_interaction cross-field rules into plan-time validation.
func UserInteractionConfigValidator() resource.ConfigValidator {
	return userInteractionConfigValidator{}
}

// Description returns a plain-text description of the validator.
func (userInteractionConfigValidator) Description(context.Context) string {
	return "user_interaction: allow_users_to_defer=false forbids deferral fields; allow_deferral_until_utc and allow_deferral_minutes are mutually exclusive"
}

// MarkdownDescription returns the markdown description.
func (v userInteractionConfigValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateResource implements the plan-time cross-field check.
func (userInteractionConfigValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data PolicyResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() || data.UserInteraction == nil {
		return
	}
	ui := data.UserInteraction

	untilSet := !ui.AllowDeferralUntilUtc.IsNull() && !ui.AllowDeferralUntilUtc.IsUnknown()
	minutesSet := !ui.AllowDeferralMinutes.IsNull() && !ui.AllowDeferralMinutes.IsUnknown()

	// Rule 1: when allow_users_to_defer is explicitly false, deferral fields
	// are forbidden. A null/Unknown allow_users_to_defer is treated as
	// "unset", which the server accepts.
	if !ui.AllowUsersToDefer.IsNull() && !ui.AllowUsersToDefer.IsUnknown() && !ui.AllowUsersToDefer.ValueBool() {
		if untilSet {
			resp.Diagnostics.AddAttributeError(
				path.Root("user_interaction").AtName("allow_deferral_until_utc"),
				"allow_deferral_until_utc forbidden when allow_users_to_defer = false",
				"Set `allow_users_to_defer = true` to use a deferral cut-off, or remove `allow_deferral_until_utc`. The classic API rejects this combination with HTTP 409 (`Error: When 'allow_users_to_defer' is false, 'allow_deferral_until_utc' and 'allow_deferral_minutes' cannot be configured`).",
			)
		}
		if minutesSet {
			resp.Diagnostics.AddAttributeError(
				path.Root("user_interaction").AtName("allow_deferral_minutes"),
				"allow_deferral_minutes forbidden when allow_users_to_defer = false",
				"Set `allow_users_to_defer = true` to use a deferral duration, or remove `allow_deferral_minutes`. The classic API rejects this combination with HTTP 409 (`Error: When 'allow_users_to_defer' is false, 'allow_deferral_until_utc' and 'allow_deferral_minutes' cannot be configured`).",
			)
		}
	}

	// Rule 2: until_utc and minutes are mutually exclusive.
	if untilSet && minutesSet {
		resp.Diagnostics.AddAttributeError(
			path.Root("user_interaction").AtName("allow_deferral_minutes"),
			"allow_deferral_until_utc and allow_deferral_minutes are mutually exclusive",
			"Set exactly one of `allow_deferral_until_utc` (a cut-off date) or `allow_deferral_minutes` (a duration). The classic API rejects both being set with HTTP 409 (`Error: You cannot use both 'allow_deferral_until_utc' and 'allow_deferral_minutes'`). Note: omitting one on an Update does not clear a previously-set value — transitioning between forms requires destroy+recreate.",
		)
	}
}
