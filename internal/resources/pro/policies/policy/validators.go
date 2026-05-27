// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package policy

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// deferralUntilUtcPattern matches the classic /policies wire form for
// <allow_deferral_until_utc>: ISO-8601 with millisecond precision and a
// four-digit numeric offset, e.g. "2027-01-01T01:00:00.000+0000". The
// classic API rejects any other format with HTTP 409.
var deferralUntilUtcPattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}[+-]\d{4}$`)

// activationExpirationDatePattern matches the classic /policies wire form
// for <activation_date> and <expiration_date>: 24-hour `YYYY-MM-DD HH:MM:SS`
// with a single space separator (e.g. "2027-06-01 14:30:00"). Wire-probed
// 2026-05-27 against /JSSResource/policies/7029: send `5:00 PM` style and
// the value is silently dropped; send `17:00` and the server returns HTTP
// 409 ("Problem with date_time_limitations"). The companion epoch / UTC
// echoes (`*_epoch`, `*_utc`) are derived server-side and not modelled.
var activationExpirationDatePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$`)

// noExecuteTimePattern matches the classic /policies wire form for
// <no_execute_start> and <no_execute_end>: 12-hour `h:MM AM` / `h:MM PM`
// with the hour 1-12 (no leading zero) and a literal space before the
// meridiem (e.g. "1:00 AM", "12:30 PM"). Wire-probed 2026-05-27: 24-hour
// HH:MM returns HTTP 409; AM/PM with no day set silently drops; AM/PM with
// no_execute_on day set persists exactly as sent. Anything outside the
// 12-hour AM/PM shape is unsafe.
var noExecuteTimePattern = regexp.MustCompile(`^(1[0-2]|[1-9]):[0-5]\d (AM|PM)$`)

// deferralTypeCompanionsValidator enforces the cross-field shape of the
// `user_interaction.deferral_type` enum against its type-specific siblings
// `deferral_until_utc` and `deferral_days`:
//
//   - `deferral_type = "none"`     forbids both siblings.
//   - `deferral_type = "date"`     requires `deferral_until_utc`, forbids `deferral_days`.
//   - `deferral_type = "duration"` requires `deferral_days` (>=1), forbids `deferral_until_utc`.
//
// The rule is value-discriminated (different value of `deferral_type`
// implies a different companion set), so per STYLE_GUIDE.md §Cross-field
// validation a custom `validator.String` is the right tool — off-the-shelf
// `AlsoRequires` / `ConflictsWith` fire on any value and cannot express
// "required only when this string equals X". Errors attach to the
// companion's path so the user looks at the field they need to fix.
type deferralTypeCompanionsValidator struct{}

// DeferralTypeCompanionsValidator constructs the validator.
func DeferralTypeCompanionsValidator() validator.String {
	return deferralTypeCompanionsValidator{}
}

// Description returns a plain-text description of the validator.
func (deferralTypeCompanionsValidator) Description(_ context.Context) string {
	return "user_interaction.deferral_type controls which of deferral_until_utc / deferral_days must be present"
}

// MarkdownDescription returns the markdown description.
func (v deferralTypeCompanionsValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateString implements validator.String.
func (deferralTypeCompanionsValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	untilPath := req.Path.ParentPath().AtName("deferral_until_utc")
	daysPath := req.Path.ParentPath().AtName("deferral_days")

	var untilVal types.String
	var daysVal types.Int64
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, untilPath, &untilVal)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, daysPath, &daysVal)...)
	if resp.Diagnostics.HasError() {
		return
	}

	untilSet := !untilVal.IsNull() && !untilVal.IsUnknown()
	daysSet := !daysVal.IsNull() && !daysVal.IsUnknown()

	// When deferral_type is null/unknown we still reject orphaned siblings —
	// the trio is a single Optional concept and a companion without its
	// discriminator is meaningless.
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		if untilSet {
			resp.Diagnostics.AddAttributeError(
				untilPath,
				"deferral_until_utc requires deferral_type = \"date\"",
				"Set `deferral_type = \"date\"` to use a deferral cut-off, or remove `deferral_until_utc`.",
			)
		}
		if daysSet {
			resp.Diagnostics.AddAttributeError(
				daysPath,
				"deferral_days requires deferral_type = \"duration\"",
				"Set `deferral_type = \"duration\"` to use a deferral duration, or remove `deferral_days`.",
			)
		}
		return
	}

	switch req.ConfigValue.ValueString() {
	case "none":
		if untilSet {
			resp.Diagnostics.AddAttributeError(
				untilPath,
				"deferral_until_utc forbidden when deferral_type = \"none\"",
				"Remove `deferral_until_utc`, or set `deferral_type = \"date\"`.",
			)
		}
		if daysSet {
			resp.Diagnostics.AddAttributeError(
				daysPath,
				"deferral_days forbidden when deferral_type = \"none\"",
				"Remove `deferral_days`, or set `deferral_type = \"duration\"`.",
			)
		}
	case "date":
		if !untilSet {
			resp.Diagnostics.AddAttributeError(
				untilPath,
				"deferral_until_utc required when deferral_type = \"date\"",
				"Provide a UTC ISO-8601 cut-off (e.g. `2027-01-01T01:00:00.000+0000`).",
			)
		}
		if daysSet {
			resp.Diagnostics.AddAttributeError(
				daysPath,
				"deferral_days forbidden when deferral_type = \"date\"",
				"Remove `deferral_days`, or set `deferral_type = \"duration\"`.",
			)
		}
	case "duration":
		if !daysSet {
			resp.Diagnostics.AddAttributeError(
				daysPath,
				"deferral_days required when deferral_type = \"duration\"",
				"Provide a positive day count (e.g. `deferral_days = 3`).",
			)
		} else if daysVal.ValueInt64() < 1 {
			resp.Diagnostics.AddAttributeError(
				daysPath,
				"deferral_days must be >= 1",
				fmt.Sprintf("Got %d. The classic API stores this as minutes — the provider multiplies by 1440.", daysVal.ValueInt64()),
			)
		}
		if untilSet {
			resp.Diagnostics.AddAttributeError(
				untilPath,
				"deferral_until_utc forbidden when deferral_type = \"duration\"",
				"Remove `deferral_until_utc`, or set `deferral_type = \"date\"`.",
			)
		}
	}
}
