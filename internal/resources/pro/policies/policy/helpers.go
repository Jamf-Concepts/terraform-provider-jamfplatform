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

// buildNotificationEnabled projects the schema's display_notifications bool
// into proclassic.NotificationValue carrying only the bool leg. The classic
// wire uses a single <notification>true|false</notification> element for the
// boolean; the method string travels as a sibling <notification_type>
// element on the same parent (PolicySelfService.NotificationType), modelled
// by the provider as notification_location — not as a second <notification>
// tag. Returns nil when the attribute is null/unknown.
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
// self_service.display_notifications), at the cost of not detecting
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

// accountMaintenanceSecretsForCreate builds the WriteOnly password carrier
// for Create. There is no prior state, so every plaintext sourced from cfg
// is treated as fresh and threaded to the wire. Accounts are keyed by
// username (the wire identity used by Jamf to match Create/Reset/Delete
// actions). Null cfg values produce nil emit-pointers so the SDK omits
// the corresponding XML element.
func accountMaintenanceSecretsForCreate(cfg *PolicyResourceModel) *policyAccountMaintenanceSecrets {
	out := &policyAccountMaintenanceSecrets{accountPasswords: map[string]*string{}}
	if cfg == nil {
		return out
	}
	for _, a := range cfg.LocalAccounts {
		if a.Username.IsNull() || a.Username.IsUnknown() {
			continue
		}
		out.accountPasswords[a.Username.ValueString()] = helpers.OptionalStringPointer(a.Password)
	}
	if cfg.ManagementAccount != nil {
		out.managedPassword = helpers.OptionalStringPointer(cfg.ManagementAccount.ManagedPassword)
	}
	if cfg.EfiPassword != nil {
		out.ofPassword = helpers.OptionalStringPointer(cfg.EfiPassword.OfPassword)
	}
	return out
}

// accountMaintenanceSecretsForUpdate builds the WriteOnly password carrier
// for Update. Unlike directory_binding / disk_encryption (where the
// password is a one-shot bind credential and Jamf retains its stored
// value when the wire field is omitted), Jamf classic policy actions
// embed the password in the stored policy and clients re-execute the
// action on every policy run. Omitting `<password>` on Update would
// erase the stored value and break the next Create/Reset run — the
// server enforces this and returns HTTP 409 "Problem with reset
// account fields" when a Reset entry lacks a password.
//
// Consequence: Update must always include the cfg-sourced plaintext on
// the wire (when the user has one in their config). The `*_wo_version`
// companion still exists as the rotation trigger — WriteOnly attribute
// changes alone are invisible to Terraform's plan diff, so the user must
// bump wo_version (or change another non-WriteOnly attribute) to make
// Update fire at all. But the wire payload itself does not gate on
// wo_version — that gate would break the policy at the next client run.
//
// Accounts are matched by username (the wire identity used by Jamf for
// Create/Reset/Delete/DisableFileVault).
func accountMaintenanceSecretsForUpdate(plan, state, cfg *PolicyResourceModel) *policyAccountMaintenanceSecrets {
	_ = state // intentionally unused; see doc comment for the reasoning.
	return accountMaintenanceSecretsForCreate(cfg)
}
