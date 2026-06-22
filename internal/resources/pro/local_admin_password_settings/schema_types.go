// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package local_admin_password_settings

import (
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// localAdminPasswordSettingsTimeoutAttributeTypes defines the timeout attribute types.
var localAdminPasswordSettingsTimeoutAttributeTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"update": types.StringType,
	"delete": types.StringType,
}

// rotationIntervalNever is the rotation_interval value that turns automatic
// password rotation off. The UI presents it as the first dropdown entry; behind
// the scenes it corresponds to automatic rotation being disabled.
const rotationIntervalNever = "Never"

// rotationAfterViewingToDuration maps each "Rotation after viewing interval"
// dropdown preset to the duration the server stores for it. Mirrors the Jamf Pro
// UI dropdown exactly (Settings → Computer Management → Security). The server
// stores arbitrary durations and does not enforce these presets, so the
// provider's OneOf validator is the sole guard on the configured value.
var rotationAfterViewingToDuration = map[string]int{
	"1 hour":   3600,
	"3 hours":  10800,
	"12 hours": 43200,
	"1 day":    86400,
	"3 days":   259200,
	"7 days":   604800,
}

// validRotationAfterViewingInterval lists the accepted rotation_after_viewing_interval
// values in UI dropdown order.
var validRotationAfterViewingInterval = []string{
	"1 hour", "3 hours", "12 hours", "1 day", "3 days", "7 days",
}

// rotationIntervalDurationToValue maps each non-"Never" "Rotation interval"
// dropdown preset to the duration the server stores for it.
var rotationIntervalDurationToValue = map[string]int{
	"7 days":   604800,
	"30 days":  2592000,
	"60 days":  5184000,
	"180 days": 15552000,
}

// validRotationIntervalDurations lists the non-"Never" rotation_interval presets
// in UI dropdown order — used in the diagnostic emitted when the tenant holds an
// unsupported duration while automatic rotation is on.
var validRotationIntervalDurations = []string{"7 days", "30 days", "60 days", "180 days"}

// validRotationInterval lists every accepted rotation_interval value
// ("Never" + the duration presets) in UI dropdown order.
var validRotationInterval = append([]string{rotationIntervalNever}, validRotationIntervalDurations...)

// durationToRotationAfterViewing and durationToRotationInterval are the reverse
// lookups used when mapping the stored durations back to dropdown labels on read.
var (
	durationToRotationAfterViewing = invertDurationMap(rotationAfterViewingToDuration)
	durationToRotationInterval     = invertDurationMap(rotationIntervalDurationToValue)
)

// defaultPasswordRotationDuration / defaultAutoRotateExpirationDuration are
// non-zero fallbacks (7 days). The server rejects either duration when it is 0,
// even while automatic rotation is off, so the input builder must always send a
// value greater than zero. These are only reached if a future refactor failed to
// adopt the live value; the normal path carries the existing (always > 0)
// duration forward. 7 days is a valid dropdown preset.
const (
	defaultPasswordRotationDuration     = 604800
	defaultAutoRotateExpirationDuration = 604800
)

// markdownValueList renders a slice of enum values as a backticked,
// comma-separated list for use in MarkdownDescription strings. Deriving the
// documented allowed values from the same slice the OneOf validator uses keeps
// the docs and the validator from drifting apart.
func markdownValueList(vals []string) string {
	quoted := make([]string, len(vals))
	for i, v := range vals {
		quoted[i] = "`" + v + "`"
	}
	return strings.Join(quoted, ", ")
}

// invertDurationMap builds the duration -> label reverse of a label -> duration map.
func invertDurationMap(m map[string]int) map[int]string {
	out := make(map[int]string, len(m))
	for label, dur := range m {
		out[dur] = label
	}
	return out
}
