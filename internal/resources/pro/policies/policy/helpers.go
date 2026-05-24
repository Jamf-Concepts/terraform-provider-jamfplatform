// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package policy

import (
	"strconv"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// extractPolicyID returns the top-level ID as a string from a Create response.
// Classic returns the assigned ID at <policy><id> (Policy.ID); General.ID is
// the same value echoed inside the general block. We prefer the top-level
// reading first and fall back to General.
func extractPolicyID(p *proclassic.Policy) string {
	if p == nil {
		return ""
	}
	if p.ID != nil {
		return strconv.Itoa(*p.ID)
	}
	if p.General != nil && p.General.ID != nil {
		return strconv.Itoa(*p.General.ID)
	}
	return ""
}

// buildNotificationEnabled projects the schema's notification_enabled bool
// into proclassic.NotificationValue carrying only the bool leg. The classic
// wire uses a single <notification>true|false</notification> element for the
// boolean; the method string travels as a sibling <notification_type>
// element on the same parent (PolicySelfService.NotificationType), not as
// a second <notification> tag. Returns nil when the attribute is
// null/unknown.
func buildNotificationEnabled(enabled types.Bool) *proclassic.NotificationValue {
	if !helpers.IsConfiguredValue(enabled) {
		return nil
	}
	v := enabled.ValueBool()
	return &proclassic.NotificationValue{Enabled: &v}
}

// flattenNotificationEnabled extracts the bool leg from the SDK
// NotificationValue (the boolean <notification> wire element). Follows the
// preferCurrent pattern so configured caller values stick across refreshes.
func flattenNotificationEnabled(n *proclassic.NotificationValue, current types.Bool) types.Bool {
	var apiEnabled *bool
	if n != nil {
		apiEnabled = n.Enabled
	}
	return preferCurrentBoolPointer(apiEnabled, current)
}

// preferCurrentStringPointer returns the current TF value when the caller
// already supplied one, otherwise adopts the API value (or null when both
// are absent). Designed for Optional+Computed scalar attrs nested inside
// managed sections: protects against Jamf classic API quirks where the
// server may echo a different value than what the caller wrote (e.g.
// self_service.notification_enabled), at the cost of not detecting
// server-side drift. The canary accepts the tradeoff.
func preferCurrentStringPointer(api *string, current types.String) types.String {
	if helpers.IsConfiguredValue(current) {
		return current
	}
	if api == nil {
		return types.StringNull()
	}
	return types.StringValue(*api)
}

// preferServerOrCurrentString returns the server value when present and
// non-empty, otherwise the prior state value. Used for Computed string
// attributes the server intermittently fails to echo on Update —
// substituting the prior known state value avoids a Known → null (or
// Known → "") transition that would otherwise trip the framework's
// "produced inconsistent result after apply" check when the Computed
// attribute is part of plan. The classic /policies endpoint has been
// observed returning both `nil` and `""` for the same SHA field on
// successive Update round-trips, so both cases collapse to "use prior".
func preferServerOrCurrentString(api *string, current types.String) types.String {
	if api != nil && *api != "" {
		return types.StringValue(*api)
	}
	if !current.IsNull() && !current.IsUnknown() {
		return current
	}
	return types.StringNull()
}

// preferCurrentBoolPointer is the bool sibling of preferCurrentStringPointer.
func preferCurrentBoolPointer(api *bool, current types.Bool) types.Bool {
	if helpers.IsConfiguredValue(current) {
		return current
	}
	if api == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*api)
}

// preferCurrentInt is the int64 sibling. API is *int (SDK convention).
func preferCurrentInt(api *int, current types.Int64) types.Int64 {
	if helpers.IsConfiguredValue(current) {
		return current
	}
	if api == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*api))
}

// optionalBoolPointer is the bool sibling of helpers.OptionalStringPointer.
// Mirrors the same null/unknown handling for write payloads.
func optionalBoolPointer(value types.Bool) *bool {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	v := value.ValueBool()
	return &v
}

// invertOptionalBoolPointer is optionalBoolPointer with the value negated.
// Used to translate a user-facing boolean attribute into its inverse on the
// wire (e.g. permanently_delete_home_directory ↔ archive_home_directory).
// Null/unknown still collapses to nil so the wire emits no element.
func invertOptionalBoolPointer(value types.Bool) *bool {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	v := !value.ValueBool()
	return &v
}

// invertBoolPointerValueOrNull is the state-side inverse of
// invertOptionalBoolPointer. A nil server pointer becomes null state; a
// non-nil pointer becomes its negated TF Bool value.
func invertBoolPointerValueOrNull(b *bool) types.Bool {
	if b == nil {
		return types.BoolNull()
	}
	return types.BoolValue(!*b)
}

// optionalInt64ToInt projects a TF Int64 into a *int suitable for SDK
// payloads. Returns nil for null/unknown.
func optionalInt64ToInt(value types.Int64) *int {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	v := int(value.ValueInt64())
	return &v
}

// stringIDPtr parses a TF String holding a numeric ID into *int. Returns nil
// for null/unknown/empty/un-parseable.
func stringIDPtr(value types.String) *int {
	if !helpers.IsConfiguredValue(value) {
		return nil
	}
	s := value.ValueString()
	if s == "" {
		return nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	return &v
}
