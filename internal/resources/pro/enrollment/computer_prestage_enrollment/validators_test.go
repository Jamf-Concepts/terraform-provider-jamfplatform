// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package computer_prestage_enrollment

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// unknownString / unknownBool are sentinels for buildConfig signalling that an
// attribute should be set to an UNKNOWN value of the given primitive type.
type unknownString struct{}

type unknownBool struct{}

// nullBool is a sentinel for buildConfig signalling a null value typed as Bool
// (the bare nil sentinel defaults to a null String).
type nullBool struct{}

// buildConfig synthesises a *tfsdk.Config from a small attribute map for
// use by validators that call req.Config.GetAttribute(...).
func buildConfig(t *testing.T, attrs map[string]any) tfsdk.Config {
	t.Helper()
	typeMap := map[string]tftypes.Type{}
	valMap := map[string]tftypes.Value{}
	tfsdkAttrs := map[string]schema.Attribute{}

	for k, raw := range attrs {
		switch v := raw.(type) {
		case string:
			typeMap[k] = tftypes.String
			valMap[k] = tftypes.NewValue(tftypes.String, v)
			tfsdkAttrs[k] = schema.StringAttribute{Optional: true}
		case bool:
			typeMap[k] = tftypes.Bool
			valMap[k] = tftypes.NewValue(tftypes.Bool, v)
			tfsdkAttrs[k] = schema.BoolAttribute{Optional: true}
		case unknownString:
			typeMap[k] = tftypes.String
			valMap[k] = tftypes.NewValue(tftypes.String, tftypes.UnknownValue)
			tfsdkAttrs[k] = schema.StringAttribute{Optional: true}
		case unknownBool:
			typeMap[k] = tftypes.Bool
			valMap[k] = tftypes.NewValue(tftypes.Bool, tftypes.UnknownValue)
			tfsdkAttrs[k] = schema.BoolAttribute{Optional: true}
		case nullBool:
			typeMap[k] = tftypes.Bool
			valMap[k] = tftypes.NewValue(tftypes.Bool, nil)
			tfsdkAttrs[k] = schema.BoolAttribute{Optional: true}
		case nil:
			typeMap[k] = tftypes.String
			valMap[k] = tftypes.NewValue(tftypes.String, nil)
			tfsdkAttrs[k] = schema.StringAttribute{Optional: true}
		default:
			t.Fatalf("buildConfig: unsupported type for %q: %T", k, raw)
		}
	}

	objType := tftypes.Object{AttributeTypes: typeMap}
	cfg := tfsdk.Config{
		Schema: schema.Schema{Attributes: tfsdkAttrs},
		Raw:    tftypes.NewValue(objType, valMap),
	}
	return cfg
}

func runStringValidator(t *testing.T, v validator.String, cfg tfsdk.Config, attrName string, value attr.Value) []string {
	t.Helper()
	req := validator.StringRequest{
		Path:        path.Root(attrName),
		ConfigValue: value.(types.String),
		Config:      cfg,
	}
	var resp validator.StringResponse
	v.ValidateString(context.Background(), req, &resp)
	out := make([]string, 0, len(resp.Diagnostics))
	for _, d := range resp.Diagnostics {
		out = append(out, d.Detail())
	}
	return out
}

func TestRecoveryLockPasswordTypeRandomConflictsWithPassword_NoConflict(t *testing.T) {
	cfg := buildConfig(t, map[string]any{
		"recovery_lock_password_type": "MANUAL",
		"recovery_lock_password":      "pwd",
	})
	out := runStringValidator(t, recoveryLockPasswordTypeRandomConflictsWithPassword(),
		cfg, "recovery_lock_password_type", types.StringValue("MANUAL"))
	if len(out) != 0 {
		t.Errorf("MANUAL+pwd should NOT trip the RANDOM-conflict validator, got %v", out)
	}
}

func TestRecoveryLockPasswordTypeRandomConflictsWithPassword_FiresOnConflict(t *testing.T) {
	cfg := buildConfig(t, map[string]any{
		"recovery_lock_password_type": "RANDOM",
		"recovery_lock_password":      "Sup3r",
	})
	out := runStringValidator(t, recoveryLockPasswordTypeRandomConflictsWithPassword(),
		cfg, "recovery_lock_password_type", types.StringValue("RANDOM"))
	if len(out) == 0 {
		t.Errorf("RANDOM + non-empty password should fire validator")
	}
}

func TestRecoveryLockPasswordRequiresManualAndEnabled_FiresWhenDisabled(t *testing.T) {
	cfg := buildConfig(t, map[string]any{
		"enable_recovery_lock":        false,
		"recovery_lock_password_type": "MANUAL",
	})
	out := runStringValidator(t, recoveryLockPasswordRequiresManualAndEnabled(),
		cfg, "recovery_lock_password", types.StringValue("pwd"))
	if len(out) == 0 {
		t.Errorf("pwd + enable=false should fire validator")
	}
}

func TestRecoveryLockPasswordRequiresManualAndEnabled_FiresWhenRandom(t *testing.T) {
	cfg := buildConfig(t, map[string]any{
		"enable_recovery_lock":        true,
		"recovery_lock_password_type": "RANDOM",
	})
	out := runStringValidator(t, recoveryLockPasswordRequiresManualAndEnabled(),
		cfg, "recovery_lock_password", types.StringValue("pwd"))
	if len(out) == 0 {
		t.Errorf("pwd + type=RANDOM should fire validator")
	}
}

func TestRecoveryLockPasswordRequiresManualAndEnabled_NoError(t *testing.T) {
	cfg := buildConfig(t, map[string]any{
		"enable_recovery_lock":        true,
		"recovery_lock_password_type": "MANUAL",
	})
	out := runStringValidator(t, recoveryLockPasswordRequiresManualAndEnabled(),
		cfg, "recovery_lock_password", types.StringValue("pwd"))
	if len(out) != 0 {
		t.Errorf("happy-path config should not fire validator, got %v", out)
	}
}

// Regression for the config-time validators-defer-on-unknown bug: when the
// companion enable_recovery_lock is UNKNOWN (e.g. wired from a variable), the
// validator must defer rather than false-error.
func TestRecoveryLockPasswordRequiresManualAndEnabled_DefersWhenEnableUnknown(t *testing.T) {
	cfg := buildConfig(t, map[string]any{
		"enable_recovery_lock":        unknownBool{},
		"recovery_lock_password_type": "MANUAL",
	})
	out := runStringValidator(t, recoveryLockPasswordRequiresManualAndEnabled(),
		cfg, "recovery_lock_password", types.StringValue("pwd"))
	if len(out) != 0 {
		t.Errorf("pwd + enable=UNKNOWN must defer (no diagnostic), got %v", out)
	}
}

func TestRecoveryLockPasswordRequiresManualAndEnabled_FiresWhenEnableNull(t *testing.T) {
	cfg := buildConfig(t, map[string]any{
		"enable_recovery_lock":        nullBool{},
		"recovery_lock_password_type": "MANUAL",
	})
	out := runStringValidator(t, recoveryLockPasswordRequiresManualAndEnabled(),
		cfg, "recovery_lock_password", types.StringValue("pwd"))
	if len(out) == 0 {
		t.Errorf("pwd + enable null should fire validator")
	}
}

func TestPrefillTypeCustomRequiresFullAndUserNames_DefersWhenCompanionUnknown(t *testing.T) {
	// prefill_type=CUSTOM with a required companion UNKNOWN must defer.
	cfg := buildConfig(t, map[string]any{
		"prefill_account_full_name": unknownString{},
		"prefill_account_user_name": "jdoe",
	})
	req := validator.StringRequest{
		Path:        path.Root("prefill_type"),
		ConfigValue: types.StringValue("CUSTOM"),
		Config:      cfg,
	}
	var resp validator.StringResponse
	prefillTypeCustomRequiresFullAndUserNames().ValidateString(context.Background(), req, &resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("CUSTOM + unknown companion must defer (no diagnostic), got %v", resp.Diagnostics)
	}
}

func TestPrefillTypeCustomRequiresFullAndUserNames_FiresWhenCompanionNull(t *testing.T) {
	// prefill_type=CUSTOM with a required companion null/absent must error.
	cfg := buildConfig(t, map[string]any{
		"prefill_account_user_name": "jdoe",
		"prefill_account_full_name": nil, // null String
	})
	req := validator.StringRequest{
		Path:        path.Root("prefill_type"),
		ConfigValue: types.StringValue("CUSTOM"),
		Config:      cfg,
	}
	var resp validator.StringResponse
	prefillTypeCustomRequiresFullAndUserNames().ValidateString(context.Background(), req, &resp)
	if !resp.Diagnostics.HasError() {
		t.Errorf("CUSTOM + null companion should fire validator")
	}
}

func TestPrefillTypeCustomRequiresFullAndUserNames_NoErrorWhenBothSet(t *testing.T) {
	cfg := buildConfig(t, map[string]any{
		"prefill_account_full_name": "John Doe",
		"prefill_account_user_name": "jdoe",
	})
	req := validator.StringRequest{
		Path:        path.Root("prefill_type"),
		ConfigValue: types.StringValue("CUSTOM"),
		Config:      cfg,
	}
	var resp validator.StringResponse
	prefillTypeCustomRequiresFullAndUserNames().ValidateString(context.Background(), req, &resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("CUSTOM + both companions set should not fire validator, got %v", resp.Diagnostics)
	}
}

// prefill_type validator reads two siblings under the same parent block.
// Building a stub Config that exercises req.Path.ParentPath().AtName(...) is
// considerably more involved than a flat config, so this case is covered
// end-to-end by the acc test
// TestAccResource_ProComputerPrestageEnrollment_ExpectError_PrefillCustomRequiresNames.
func TestPrefillTypeCustomRequiresFullAndUserNames_SkipsWhenNotCustom(t *testing.T) {
	// Direct unit check of the early-return guards is enough — the heavy
	// path is exercised by the acceptance ExpectError test.
	cfg := buildConfig(t, map[string]any{})
	req := validator.StringRequest{
		Path:        path.Root("prefill_type"),
		ConfigValue: types.StringValue("DEVICE_OWNER"),
		Config:      cfg,
	}
	var resp validator.StringResponse
	prefillTypeCustomRequiresFullAndUserNames().ValidateString(context.Background(), req, &resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("DEVICE_OWNER must not trip CUSTOM-only validator, got %v", resp.Diagnostics)
	}
}
