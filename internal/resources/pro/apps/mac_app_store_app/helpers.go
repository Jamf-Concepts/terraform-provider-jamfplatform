// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mac_app_store_app

import (
	"strconv"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// deploymentTypeSelfService and deploymentTypeAutomatic are the two wire values
// the classic /macapplications endpoint accepts for general.deployment_type.
// The literals must match the server bytes exactly, including the slash.
const (
	deploymentTypeSelfService = "Make Available in Self Service"
	deploymentTypeAutomatic   = "Install Automatically/Prompt Users to Install"
)

// extractMacAppID returns the assigned ID as a string from a Create/GET
// response. Classic returns the ID at the top level (<mac_application><id>) and
// echoes it inside <general>. Prefer the top-level reading, fall back to general.
func extractMacAppID(a *proclassic.MacApplication) string {
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

// int64FromIntPtr maps an SDK *int (server-computed counts) into a TF Int64.
// nil becomes null. These attributes are Computed-only, so there is no caller
// value to prefer.
func int64FromIntPtr(p *int) types.Int64 {
	if p == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*p))
}

// buildMacAppNotification assembles the self_service notification_enabled /
// notification_method attributes into a single proclassic.NotificationValue.
// The classic wire carries two <notification> elements (a bool and a method
// string); NotificationValue.MarshalXML emits them method-first, the only
// order that round-trips (wire-probed). Returns nil when neither attribute is
// configured so the SDK omits the element entirely.
func buildMacAppNotification(enabled types.Bool, method types.String) *proclassic.NotificationValue {
	if !helpers.IsConfiguredValue(enabled) && !helpers.IsConfiguredValue(method) {
		return nil
	}
	n := &proclassic.NotificationValue{}
	if helpers.IsConfiguredValue(enabled) {
		v := enabled.ValueBool()
		n.Enabled = &v
	}
	if helpers.IsConfiguredValue(method) {
		m := method.ValueString()
		n.Method = &m
	}
	return n
}
