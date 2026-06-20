// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mobile_device_prestage_enrollment

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// minEnforcedTemporarySessionTimeout is the Jamf Pro admin-UI minimum for a
// temporary session timeout when enforcement is enabled. Below this, Jamf Pro
// silently nulls the value (§F11); the validator surfaces it at plan time.
const minEnforcedTemporarySessionTimeout = 30

// The IANA timezone validator formerly defined here was promoted to the shared
// package as validators.IANATimeZone() (internal/common/validators/timezone.go)
// when the Jamf Teacher settings resource became its second consumer; the probe
// evidence travelled with it.

// singleNameRequiresSingleDeviceName fires when names.assign_names_using is
// "Single Name" but names.single_device_name is empty. Provider-side courtesy
// validator (the server is permissive — §F11); mirrors the prefillType
// value-specific validator shape from the computer sibling.
func singleNameRequiresSingleDeviceName() validator.String {
	return singleNameRequiresSingleDeviceNameValidator{}
}

type singleNameRequiresSingleDeviceNameValidator struct{}

func (singleNameRequiresSingleDeviceNameValidator) Description(_ context.Context) string {
	return `names.assign_names_using = "Single Name" requires names.single_device_name to be set.`
}

func (v singleNameRequiresSingleDeviceNameValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (singleNameRequiresSingleDeviceNameValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if req.ConfigValue.ValueString() != "Single Name" {
		return
	}

	sibling := req.Path.ParentPath().AtName("single_device_name")
	var single types.String
	if d := req.Config.GetAttribute(ctx, sibling, &single); d.HasError() {
		resp.Diagnostics.Append(d...)
		return
	}
	if single.IsUnknown() {
		return
	}
	if single.IsNull() || single.ValueString() == "" {
		resp.Diagnostics.AddAttributeError(
			sibling,
			`single_device_name is required when assign_names_using = "Single Name"`,
			`Supply a non-empty single_device_name or choose a different assign_names_using mode.`,
		)
	}
}

// storageQuotaConflictsWithTemporarySession fires when both
// use_storage_quota_size and temporary_session_only are set true. They are the
// two mutually-exclusive Shared-iPad storage modes — Jamf Pro silently forces
// use_storage_quota_size to false when temporary_session_only is true (§F9), so
// this is a provider-side courtesy guard surfacing the conflict at plan time.
// Attached to use_storage_quota_size so it fires once. Mirrors the computer
// sibling's recoveryLockPasswordTypeRandomConflictsWithPassword shape.
func storageQuotaConflictsWithTemporarySession() validator.Bool {
	return storageQuotaConflictsWithTemporarySessionValidator{}
}

type storageQuotaConflictsWithTemporarySessionValidator struct{}

func (storageQuotaConflictsWithTemporarySessionValidator) Description(_ context.Context) string {
	return "use_storage_quota_size cannot be true when temporary_session_only is true (mutually-exclusive Shared iPad storage modes)."
}

func (v storageQuotaConflictsWithTemporarySessionValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (storageQuotaConflictsWithTemporarySessionValidator) ValidateBool(ctx context.Context, req validator.BoolRequest, resp *validator.BoolResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() || !req.ConfigValue.ValueBool() {
		return
	}

	companion := path.Root("temporary_session_only")
	var temp types.Bool
	if d := req.Config.GetAttribute(ctx, companion, &temp); d.HasError() {
		resp.Diagnostics.Append(d...)
		return
	}
	if temp.IsNull() || temp.IsUnknown() || !temp.ValueBool() {
		return
	}
	resp.Diagnostics.AddAttributeError(
		req.Path,
		"use_storage_quota_size conflicts with temporary_session_only",
		"use_storage_quota_size and temporary_session_only are mutually-exclusive Shared iPad storage modes; set at most one to true.",
	)
}

// multiUserRequiresPreventActivationLock fires when multi_user (Enable Shared
// iPad) is true but prevent_activation_lock is not also true. Jamf Pro enforces
// this as a hard prerequisite — the server rejects the write with
// `400 PREREQUISITE_NOT_MET: "Prevent activation lock needs to be enabled in
// order to enable Shared iPad"` — so this surfaces it at plan time. The rule is
// value-specific (fires only when multi_user is true), so a custom validator is
// required per STYLE_GUIDE §Cross-field validation; off-the-shelf
// `boolvalidator.AlsoRequires` would fire when multi_user is false too. Attached
// to multi_user so it fires once; the error attaches to prevent_activation_lock
// (the field the user must set). Mirrors storageQuotaConflictsWithTemporarySession.
func multiUserRequiresPreventActivationLock() validator.Bool {
	return multiUserRequiresPreventActivationLockValidator{}
}

type multiUserRequiresPreventActivationLockValidator struct{}

func (multiUserRequiresPreventActivationLockValidator) Description(_ context.Context) string {
	return "multi_user (Enable Shared iPad) requires prevent_activation_lock to be true."
}

func (v multiUserRequiresPreventActivationLockValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (multiUserRequiresPreventActivationLockValidator) ValidateBool(ctx context.Context, req validator.BoolRequest, resp *validator.BoolResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() || !req.ConfigValue.ValueBool() {
		return
	}

	companion := path.Root("prevent_activation_lock")
	var pal types.Bool
	if d := req.Config.GetAttribute(ctx, companion, &pal); d.HasError() {
		resp.Diagnostics.Append(d...)
		return
	}
	// Defer on unknown (STYLE_GUIDE §Config-time validators MUST defer on
	// unknown): the companion may come from a variable/for_each.
	if pal.IsUnknown() {
		return
	}
	if !pal.IsNull() && pal.ValueBool() {
		return // prerequisite satisfied
	}
	resp.Diagnostics.AddAttributeError(
		companion,
		"prevent_activation_lock required for Shared iPad",
		"multi_user = true (Enable Shared iPad) requires prevent_activation_lock = true; Jamf Pro rejects the configuration otherwise.",
	)
}

// multiUserRequiresSupervised fires when multi_user (Enable Shared iPad) is
// true but supervised is not also true. Unlike prevent_activation_lock (which
// Jamf Pro rejects with a hard 400), the server SILENTLY coerces multi_user
// back to false when the device isn't supervised — which surfaces in Terraform
// as a "provider produced inconsistent result after apply" crash (planned
// multi_user=true, server-echoed false). This validator turns that opaque crash
// into a clear plan-time error. Wire-confirmed: every stored multiUser=true
// record in local-testing probes (get-id3-shared, post-quota-request) had
// supervised=true. Apple requires Shared iPad devices to be supervised, so
// there is no valid multi_user=true + supervised=false config to false-reject.
// Value-specific per STYLE_GUIDE §Cross-field validation; attached to
// multi_user, error attaches to supervised. Mirrors
// multiUserRequiresPreventActivationLock.
func multiUserRequiresSupervised() validator.Bool {
	return multiUserRequiresSupervisedValidator{}
}

type multiUserRequiresSupervisedValidator struct{}

func (multiUserRequiresSupervisedValidator) Description(_ context.Context) string {
	return "multi_user (Enable Shared iPad) requires supervised to be true."
}

func (v multiUserRequiresSupervisedValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (multiUserRequiresSupervisedValidator) ValidateBool(ctx context.Context, req validator.BoolRequest, resp *validator.BoolResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() || !req.ConfigValue.ValueBool() {
		return
	}

	companion := path.Root("supervised")
	var sup types.Bool
	if d := req.Config.GetAttribute(ctx, companion, &sup); d.HasError() {
		resp.Diagnostics.Append(d...)
		return
	}
	if sup.IsUnknown() {
		return // defer (STYLE_GUIDE §Config-time validators MUST defer on unknown)
	}
	if !sup.IsNull() && sup.ValueBool() {
		return // prerequisite satisfied
	}
	resp.Diagnostics.AddAttributeError(
		companion,
		"supervised required for Shared iPad",
		"multi_user = true (Enable Shared iPad) requires supervised = true; otherwise Jamf Pro silently disables Shared iPad (Apple requires Shared iPad devices to be supervised).",
	)
}

// temporarySessionTimeoutMinimum fires when enforce_temporary_session_timeout
// is true and temporary_session_timeout is set below the Jamf Pro minimum of
// 30. Below the minimum Jamf Pro silently nulls the value (§F11); the validator
// surfaces it at plan time. Attached to temporary_session_timeout.
func temporarySessionTimeoutMinimum() validator.Int64 {
	return temporarySessionTimeoutMinimumValidator{}
}

type temporarySessionTimeoutMinimumValidator struct{}

func (temporarySessionTimeoutMinimumValidator) Description(_ context.Context) string {
	return "temporary_session_timeout must be at least 30 when enforce_temporary_session_timeout is true."
}

func (v temporarySessionTimeoutMinimumValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (temporarySessionTimeoutMinimumValidator) ValidateInt64(ctx context.Context, req validator.Int64Request, resp *validator.Int64Response) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if req.ConfigValue.ValueInt64() >= minEnforcedTemporarySessionTimeout {
		return
	}

	companion := path.Root("enforce_temporary_session_timeout")
	var enforce types.Bool
	if d := req.Config.GetAttribute(ctx, companion, &enforce); d.HasError() {
		resp.Diagnostics.Append(d...)
		return
	}
	if enforce.IsNull() || enforce.IsUnknown() || !enforce.ValueBool() {
		return
	}
	resp.Diagnostics.AddAttributeError(
		req.Path,
		"temporary_session_timeout below minimum",
		// Wrap-safe: no whitespace at the ~80-col wrap point (memory:
		// ExpectError regex line-wrap). "at-least-30" is a single token.
		"temporary_session_timeout must be at-least-30 when enforce_temporary_session_timeout is true; Jamf Pro discards smaller values.",
	)
}
