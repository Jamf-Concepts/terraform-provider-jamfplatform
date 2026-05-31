// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mobile_device_app

import (
	"strconv"
	"strings"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// deploymentTypeSelfService and deploymentTypeAutomatic are the two wire values
// the classic /mobiledeviceapplications endpoint accepts for
// general.deployment_type. The literals must match the server bytes exactly,
// including the slash.
const (
	deploymentTypeSelfService = "Make Available in Self Service"
	deploymentTypeAutomatic   = "Install Automatically/Prompt Users to Install"
)

// osTypeIOS and osTypeTVOS are the two values the server accepts for
// general.os_type. The server requires os_type on every write to an in-house
// (internal_app=true) app — which is the common case — so the provider always
// sends it. Echoed on GET once set.
const (
	osTypeIOS  = "iOS"
	osTypeTVOS = "tvOS"
)

// extractMobileAppID returns the assigned ID as a string from a Create/GET
// response. Classic returns the ID at the top level (<mobile_device_application><id>)
// and echoes it inside <general>. Prefer the top-level reading, fall back to general.
func extractMobileAppID(a *proclassic.MobileDeviceApplication) string {
	if a == nil {
		return ""
	}
	if a.ID != nil {
		return strconv.Itoa(*a.ID)
	}
	if a.General != nil && a.General.ID != nil {
		return strconv.Itoa(*a.General.ID)
	}
	return ""
}

// optionalBoolPointer is the bool sibling of helpers.OptionalStringPointer.
// Null/unknown collapses to nil so the wire omits the element.
func optionalBoolPointer(value types.Bool) *bool {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	v := value.ValueBool()
	return &v
}

// optionalIntPointer maps a configured TF Int64 to *int. Null/unknown collapses
// to nil so the wire omits the element.
func optionalIntPointer(value types.Int64) *int {
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

// stringFromIntPtr renders an *int as an *string for the preferCurrent helpers.
func stringFromIntPtr(p *int) *string {
	if p == nil {
		return nil
	}
	s := strconv.Itoa(*p)
	return &s
}

// preferCurrentStringPointer returns the caller's configured value when set,
// otherwise adopts the API value (or null when both are absent). Protects
// Optional+Computed scalars nested in managed sections against classic-API
// echo quirks, at the cost of not detecting server-side drift — the standard
// ProClassic scope/self_service tradeoff (see policy resource).
func preferCurrentStringPointer(api *string, current types.String) types.String {
	if helpers.IsConfiguredValue(current) {
		return current
	}
	if api == nil {
		return types.StringNull()
	}
	return types.StringValue(*api)
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

// serverWhenPresentString reflects the API value when the server returns one,
// otherwise retains the caller's current (configured/prior) value. Used for
// os_type, whose echo is asymmetric (wire-probed): a POST never persists or
// echoes it, and a non-internal app never carries it — but once set via a PUT to
// an internal app it is stored and echoed on every GET. So: trust the echo when
// present (authoritative; surfaces external drift), and fall back to the
// configured value when absent (the create path and non-internal apps) to avoid
// nulling a Required attribute and tripping "inconsistent result after apply".
func serverWhenPresentString(api *string, current types.String) types.String {
	if api != nil {
		return types.StringValue(*api)
	}
	return current
}

// preferCurrentInt64Pointer is the Int64 sibling of preferCurrentStringPointer.
func preferCurrentInt64Pointer(api *int, current types.Int64) types.Int64 {
	if !current.IsNull() && !current.IsUnknown() {
		return current
	}
	if api == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*api))
}

// buildMobileNotification assembles the self_service notification_enabled
// attribute into a proclassic.NotificationValue. Mobile apps carry only the
// bool form of <notification> (no method, unlike mac apps), and NotificationValue
// emits a lone bool element when only Enabled is set. Returns nil when the
// attribute is unconfigured so the SDK omits the element entirely.
func buildMobileNotification(enabled types.Bool) *proclassic.NotificationValue {
	if !helpers.IsConfiguredValue(enabled) {
		return nil
	}
	v := enabled.ValueBool()
	return &proclassic.NotificationValue{Enabled: &v}
}

// normalizeNewlines collapses CRLF (and lone CR) to LF. Jamf Pro round-trips
// app_configuration.preferences verbatim for provider-authored content (sent
// LF, got LF), but UI-authored or imported apps carry CRLF; the preferences
// plan modifier treats the two as semantically equal so an import or UI edit
// does not permadiff against an LF-authored config.
func normalizeNewlines(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
}
