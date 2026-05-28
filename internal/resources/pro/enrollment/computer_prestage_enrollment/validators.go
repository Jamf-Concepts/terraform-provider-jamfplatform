// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package computer_prestage_enrollment

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// recoveryLockPasswordTypeRandomConflictsWithPassword fires only when the
// declared type is "RANDOM" but the user also supplied a plaintext
// recovery_lock_password. Off-the-shelf ConflictsWith would over-fire when
// the type is "MANUAL".
func recoveryLockPasswordTypeRandomConflictsWithPassword() validator.String {
	return recoveryLockPasswordTypeRandomConflictsWithPasswordValidator{}
}

type recoveryLockPasswordTypeRandomConflictsWithPasswordValidator struct{}

func (recoveryLockPasswordTypeRandomConflictsWithPasswordValidator) Description(_ context.Context) string {
	return `When recovery_lock_password_type is "RANDOM", recovery_lock_password must not be set.`
}

func (v recoveryLockPasswordTypeRandomConflictsWithPasswordValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (recoveryLockPasswordTypeRandomConflictsWithPasswordValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if req.ConfigValue.ValueString() != "RANDOM" {
		return
	}
	companion := path.Root("recovery_lock_password")
	var pwd types.String
	if d := req.Config.GetAttribute(ctx, companion, &pwd); d.HasError() {
		resp.Diagnostics.Append(d...)
		return
	}
	if pwd.IsNull() || pwd.IsUnknown() {
		return
	}
	if pwd.ValueString() == "" {
		return
	}
	resp.Diagnostics.AddAttributeError(
		companion,
		"recovery_lock_password conflicts with recovery_lock_password_type = RANDOM",
		`Jamf Pro generates random Recovery Lock passwords when recovery_lock_password_type = "RANDOM"; remove recovery_lock_password from the configuration.`,
	)
}

// recoveryLockPasswordRequiresManualAndEnabled fires when the user supplies a
// recovery_lock_password without also setting enable_recovery_lock=true AND
// recovery_lock_password_type="MANUAL".
func recoveryLockPasswordRequiresManualAndEnabled() validator.String {
	return recoveryLockPasswordRequiresManualAndEnabledValidator{}
}

type recoveryLockPasswordRequiresManualAndEnabledValidator struct{}

func (recoveryLockPasswordRequiresManualAndEnabledValidator) Description(_ context.Context) string {
	return `recovery_lock_password requires enable_recovery_lock = true and recovery_lock_password_type = "MANUAL".`
}

func (v recoveryLockPasswordRequiresManualAndEnabledValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (recoveryLockPasswordRequiresManualAndEnabledValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if req.ConfigValue.ValueString() == "" {
		return
	}

	var enable types.Bool
	if d := req.Config.GetAttribute(ctx, path.Root("enable_recovery_lock"), &enable); d.HasError() {
		resp.Diagnostics.Append(d...)
		return
	}
	if enable.IsNull() || enable.IsUnknown() || !enable.ValueBool() {
		resp.Diagnostics.AddAttributeError(
			path.Root("recovery_lock_password"),
			"recovery_lock_password requires enable_recovery_lock = true",
			"Set enable_recovery_lock = true when supplying a manual recovery lock password.",
		)
		return
	}

	var pwdType types.String
	if d := req.Config.GetAttribute(ctx, path.Root("recovery_lock_password_type"), &pwdType); d.HasError() {
		resp.Diagnostics.Append(d...)
		return
	}
	if pwdType.IsNull() || pwdType.IsUnknown() {
		return
	}
	if pwdType.ValueString() != "MANUAL" {
		resp.Diagnostics.AddAttributeError(
			path.Root("recovery_lock_password"),
			`recovery_lock_password requires recovery_lock_password_type = "MANUAL"`,
			fmt.Sprintf(`recovery_lock_password_type is %q; only "MANUAL" accepts a user-supplied password.`, pwdType.ValueString()),
		)
	}
}

// prefillTypeCustomRequiresFullAndUserNames fires when account_settings.prefill_type
// is "CUSTOM" but prefill_account_full_name or prefill_account_user_name is empty.
func prefillTypeCustomRequiresFullAndUserNames() validator.String {
	return prefillTypeCustomRequiresFullAndUserNamesValidator{}
}

type prefillTypeCustomRequiresFullAndUserNamesValidator struct{}

func (prefillTypeCustomRequiresFullAndUserNamesValidator) Description(_ context.Context) string {
	return `account_settings.prefill_type = "CUSTOM" requires prefill_account_full_name and prefill_account_user_name to be set.`
}

func (v prefillTypeCustomRequiresFullAndUserNamesValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (prefillTypeCustomRequiresFullAndUserNamesValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if req.ConfigValue.ValueString() != "CUSTOM" {
		return
	}

	checkSibling := func(name string) {
		sibling := req.Path.ParentPath().AtName(name)
		var v types.String
		if d := req.Config.GetAttribute(ctx, sibling, &v); d.HasError() {
			resp.Diagnostics.Append(d...)
			return
		}
		if v.IsNull() || v.IsUnknown() || v.ValueString() == "" {
			resp.Diagnostics.AddAttributeError(
				sibling,
				fmt.Sprintf(`%s is required when prefill_type = "CUSTOM"`, name),
				`Supply a non-empty value or change prefill_type to "DEVICE_OWNER".`,
			)
		}
	}
	checkSibling("prefill_account_full_name")
	checkSibling("prefill_account_user_name")
}
