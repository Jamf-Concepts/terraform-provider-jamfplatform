// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package validators

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// timeOfDayRegex pins a time-of-day string to the canonical 24-hour HH:MM:SS
// form the Jamf Pro API echoes. The server parses these fields as
// java.time.LocalTime — it accepts "HH:MM" but canonicalizes it to "HH:MM:SS"
// on the wire (wire-probed on /v1/teacher-app autoClear and /v1/parent-app
// restrictedTimes, 2026-06-10), which would leave config and state permanently
// disagreeing. Pinning the canonical form at plan time avoids that mismatch;
// invalid values are a server-side 400 INVALID_FIELD anyway.
var timeOfDayRegex = regexp.MustCompile(`^(?:[01][0-9]|2[0-3]):[0-5][0-9]:[0-5][0-9]$`)

// TimeOfDayHHMMSS checks that a string attribute is a canonical 24-hour
// HH:MM:SS time of day (e.g. "17:30:00").
//
// allowEmpty controls whether "" passes: full-replace string fields use "" as
// the documented clear sentinel (STYLE_GUIDE §Full-replace endpoints,
// description convention), so clearable attributes pass true; fields where a
// present value must be a real time (e.g. required begin/end times) pass
// false. Null/unknown values are deferred per STYLE_GUIDE §Config-time
// validators.
func TimeOfDayHHMMSS(allowEmpty bool) validator.String {
	return timeOfDayValidator{allowEmpty: allowEmpty}
}

type timeOfDayValidator struct {
	allowEmpty bool
}

// Description returns a plain-text description of the validator.
func (v timeOfDayValidator) Description(_ context.Context) string {
	if v.allowEmpty {
		return `value must be a 24-hour HH:MM:SS time (e.g. "17:30:00"), or "" to clear it.`
	}
	return `value must be a 24-hour HH:MM:SS time (e.g. "17:30:00").`
}

// MarkdownDescription returns the markdown description of the validator.
func (v timeOfDayValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateString implements validator.String.
func (v timeOfDayValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	if value == "" && v.allowEmpty {
		return
	}
	if !timeOfDayRegex.MatchString(value) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid time of day",
			fmt.Sprintf("%q is not a valid time: %s", value, v.Description(ctx)),
		)
	}
}
